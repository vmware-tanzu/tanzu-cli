// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cosignhelper"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// inventoryDirName is the name of the directory where the file(s) describing
// the inventory of the discovery will be downloaded and stored.
// It should be a sub-directory of the cache directory.
const inventoryDirName = "plugin_inventory"

// DBBackedOCIDiscovery is an artifact discovery utilizing an OCI image
// which contains an SQLite database describing the content of the plugin
// discovery.
type DBBackedOCIDiscovery struct {
	// name is the name given to the discovery
	name string
	// image is an OCI compliant image. Which include DNS-compatible registry name,
	// a valid URI path (MAY contain zero or more ‘/’) and a valid tag
	// E.g., harbor.my-domain.local/tanzu-cli/plugins/plugins-inventory:latest
	// This image contains a single SQLite database file.
	image string
	// criteria specified different conditions that a plugin must respect to be discovered.
	// This allows to filter the list of plugins that will be returned.
	criteria *PluginDiscoveryCriteria
	// pluginDataDir is the location where the plugin data will be stored once
	// extracted from the OCI image
	pluginDataDir string
	// inventory is the pluginInventory to be used by this discovery.
	inventory plugininventory.PluginInventory
}

func (od *DBBackedOCIDiscovery) getInventory() plugininventory.PluginInventory {
	return od.inventory
}

// Name of the discovery.
func (od *DBBackedOCIDiscovery) Name() string {
	return od.name
}

// Type of the discovery.
func (od *DBBackedOCIDiscovery) Type() string {
	return common.DiscoveryTypeOCI
}

// List available plugins.
func (od *DBBackedOCIDiscovery) List() ([]Discovered, error) {
	err := od.fetchInventoryImage()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch the inventory of discovery '%s' for plugins", od.Name())
	}
	return od.listPluginsFromInventory()
}

// GetAllGroups returns all plugin groups defined in the discovery
func (od *DBBackedOCIDiscovery) GetAllGroups() ([]*plugininventory.PluginGroup, error) {
	err := od.fetchInventoryImage()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch the inventory of discovery '%s' for groups", od.Name())
	}
	return od.listGroupsFromInventory()
}

func (od *DBBackedOCIDiscovery) listPluginsFromInventory() ([]Discovered, error) {
	var pluginEntries []*plugininventory.PluginInventoryEntry
	var err error
	if od.criteria == nil {
		pluginEntries, err = od.getInventory().GetAllPlugins()
		if err != nil {
			return nil, err
		}
	} else {
		pluginEntries, err = od.getInventory().GetPlugins(&plugininventory.PluginInventoryFilter{
			Name:    od.criteria.Name,
			Target:  od.criteria.Target,
			Version: od.criteria.Version,
			OS:      od.criteria.OS,
			Arch:    od.criteria.Arch,
		})
		if err != nil {
			return nil, err
		}
	}

	var discoveredPlugins []Discovered
	for _, entry := range pluginEntries {
		// First build the sorted list of versions from the Artifacts map
		var versions []string
		for v := range entry.Artifacts {
			versions = append(versions, v)
		}
		if err := utils.SortVersions(versions); err != nil {
			fmt.Fprintf(os.Stderr, "error parsing versions for plugin %s: %v\n", entry.Name, err)
		}

		plugin := Discovered{
			Name:               entry.Name,
			Description:        entry.Description,
			RecommendedVersion: entry.RecommendedVersion,
			InstalledVersion:   "", // Not set when discovered, but later.
			SupportedVersions:  versions,
			Distribution:       entry.Artifacts,
			Optional:           false,
			Scope:              common.PluginScopeStandalone,
			Source:             od.name,
			ContextName:        "", // Not set when discovered.
			DiscoveryType:      common.DiscoveryTypeOCI,
			Target:             entry.Target,
			Status:             common.PluginStatusNotInstalled, // Not set yet
		}
		discoveredPlugins = append(discoveredPlugins, plugin)
	}
	return discoveredPlugins, nil
}

func (od *DBBackedOCIDiscovery) listGroupsFromInventory() ([]*plugininventory.PluginGroup, error) {
	return od.getInventory().GetAllGroups()
}

// fetchInventoryImage downloads the OCI image containing the information about the
// inventory of this discovery and stores it in the cache directory.
func (od *DBBackedOCIDiscovery) fetchInventoryImage() error {
	// TODO(khouzam): Improve by checking if we really need to download again or if we can use the cache

	// Get the custom public key path and prepare cosign verifier, if empty, cosign verifier would use embedded public key for verification
	customPublicKeyPath := os.Getenv(constants.PublicKeyPathForPluginDiscoveryImageSignature)
	cosignVerifier := cosignhelper.NewCosignVerifier(customPublicKeyPath)
	if sigVerifyErr := od.verifyInventoryImageSignature(cosignVerifier); sigVerifyErr != nil {
		// Check if the error is due to invalid image repository URL or due to actual signature verification failure and throw the appropriate error message
		var errMsg string
		_, _, ImgResolveErr := carvelhelpers.GetImageDigest(od.image)
		if ImgResolveErr != nil {
			log.Warningf("Unable to resolve the plugin discovery image: %v", ImgResolveErr)
			errMsg = fmt.Sprintf("Fatal, plugins discovery image resolution failed. Please check the repository image URL %q is correct ", od.image)
		} else {
			log.Warningf("Unable to verify the plugins discovery image signature: %v", sigVerifyErr)
			// TODO(pkalle): Update the message to convey user to check if they could use the latest public key after we get details of the well known location of the public key
			errMsg = fmt.Sprintf("Fatal, plugins discovery image signature verification failed. The `tanzu` CLI can not ensure the integrity of the plugins to be installed. To ignore this validation please append %q to the comma-separated list in the environment variable %q.  This is NOT RECOMMENDED and could put your environment at risk!",
				od.image, constants.PluginDiscoveryImageSignatureVerificationSkipList)
		}
		log.Fatal(nil, errMsg)
	}

	if err := carvelhelpers.DownloadImageAndSaveFilesToDir(od.image, od.pluginDataDir); err != nil {
		return errors.Wrapf(err, "failed to download OCI image from discovery '%s'", od.Name())
	}
	return nil
}

func (od *DBBackedOCIDiscovery) verifyInventoryImageSignature(verifier cosignhelper.Cosignhelper) error {
	signatureVerificationSkipSet := getPluginDiscoveryImagesSkippedForSignatureVerification()
	if _, exists := signatureVerificationSkipSet[strings.TrimSpace(od.image)]; exists {
		// log warning message iff user had not chosen to skip warning message for signature verification
		if skip, _ := strconv.ParseBool(os.Getenv(constants.SuppressSkipSignatureVerificationWarning)); !skip {
			log.Warningf("Skipping the plugins discovery image signature verification for %q\n ", od.image)
		}
		return nil
	}

	err := verifier.Verify(context.Background(), []string{od.image})
	if err != nil {
		return err
	}
	return nil
}

func getPluginDiscoveryImagesSkippedForSignatureVerification() map[string]struct{} {
	discoveryImages := map[string]struct{}{}
	discoveryImagesList := strings.Split(os.Getenv(constants.PluginDiscoveryImageSignatureVerificationSkipList), ",")
	for _, image := range discoveryImagesList {
		image = strings.TrimSpace(image)
		if image != "" {
			discoveryImages[image] = struct{}{}
		}
	}
	return discoveryImages
}
