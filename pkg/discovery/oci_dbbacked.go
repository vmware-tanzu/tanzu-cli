// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	newCacheHashFile := od.checkImageCache()
	if newCacheHashFile == "" {
		// The cache can be re-used.  We are done.
		return nil
	}

	// The DB has changed and needs to be updated in the cache.
	log.Infof("Updating plugin database cache for %s, this will take a few seconds.", od.image)

	// Get the custom public key path and prepare cosign verifier, if empty, cosign verifier would use embedded public key for verification
	customPublicKeyPath := os.Getenv(constants.PublicKeyPathForPluginDiscoveryImageSignature)
	cosignVerifier := cosignhelper.NewCosignVerifier(customPublicKeyPath)
	if sigVerifyErr := od.verifyInventoryImageSignature(cosignVerifier); sigVerifyErr != nil {
		log.Warningf("Unable to verify the plugins discovery image signature: %v", sigVerifyErr)
		// TODO(pkalle): Update the message to convey user to check if they could use the latest public key after we get details of the well known location of the public key
		errMsg := fmt.Sprintf("Fatal, plugins discovery image signature verification failed. The `tanzu` CLI can not ensure the integrity of the plugins to be installed. To ignore this validation please append %q to the comma-separated list in the environment variable %q.  This is NOT RECOMMENDED and could put your environment at risk!",
			od.image, constants.PluginDiscoveryImageSignatureVerificationSkipList)
		log.Fatal(nil, errMsg)
	}

	if err := carvelhelpers.DownloadImageAndSaveFilesToDir(od.image, od.pluginDataDir); err != nil {
		return errors.Wrapf(err, "failed to download OCI image from discovery '%s'", od.Name())
	}

	// Now that everything is ready, create the digest hash file
	_, _ = os.Create(newCacheHashFile)

	return nil
}

// checkImageCache will get the image digest of this discovery
// and check if the cache already contains the up-to-date image.
// It returns an empty string if the cache can be used.  Otherwise
// it returns the name of the digest file that must be created once
// the new DB image has been downloaded.
func (od *DBBackedOCIDiscovery) checkImageCache() string {
	// Get the latest digest of the discovery image.
	// If the cache already contains the image with this digest
	// we do not need to verify its signature nor to download it again.
	_, hashHexVal, err := carvelhelpers.GetImageDigest(od.image)
	if err != nil {
		// This will happen when the user has configured an invalid image discovery URI
		log.Warningf("Unable to resolve the plugin discovery image: %v", err)
		// We force abort execution here to make sure a stale image left in the cache is not used by mistake.
		log.Fatal(nil, fmt.Sprintf("Fatal: plugins discovery image resolution failed. Please check that the repository image URL %q is correct ", od.image))
	}

	// We store the digest hash of the cached DB as a file named "digest.<hash>.
	// If this file exists, we are done.  If not, we remove the current digest file
	// as we are about to download a new DB and create a new digest file.
	// First check any existing "digest.*" file; there should only be one, but
	// to protect ourselves, we check first and if there are more then one due
	// to some bug, we clean them up and invalidate the cache.
	correctHashFile := filepath.Join(od.pluginDataDir, "digest."+hashHexVal)
	matches, _ := filepath.Glob(filepath.Join(od.pluginDataDir, "digest.*"))
	if len(matches) > 1 {
		// Too many digest files.  This is a bug!  Cleanup the cache.
		log.V(4).Warningf("Too many digest files in the cache!  Invalidating the cache.")
		for _, filePath := range matches {
			os.Remove(filePath)
		}
	} else if len(matches) == 1 {
		if matches[0] == correctHashFile {
			// The hash file exists which means the DB is up-to-date.  We are done.
			return ""
		}
		// The hash file indicates a different digest hash. Remove this old hash file
		// as we will download the new DB.
		os.Remove(matches[0])
	}
	return correctHashFile
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
