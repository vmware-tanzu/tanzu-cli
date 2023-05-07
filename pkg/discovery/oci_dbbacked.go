// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/airgapped"
	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cosignhelper/sigverifier"
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
	// pluginCriteria specifies different conditions that a plugin must respect to be discovered.
	// This allows to filter the list of plugins that will be returned.
	pluginCriteria *PluginDiscoveryCriteria
	// groupCriteria specifies different conditions that a plugin group must respect to be discovered.
	// This allows to filter the list of plugins groups that will be returned.
	groupCriteria *GroupDiscoveryCriteria
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

// GetGroups returns the plugin groups defined in the discovery
func (od *DBBackedOCIDiscovery) GetGroups() ([]*plugininventory.PluginGroup, error) {
	err := od.fetchInventoryImage()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch the inventory of discovery '%s' for groups", od.Name())
	}
	return od.listGroupsFromInventory()
}

func (od *DBBackedOCIDiscovery) listPluginsFromInventory() ([]Discovered, error) {
	var pluginEntries []*plugininventory.PluginInventoryEntry
	var err error

	shouldIncludeHidden, _ := strconv.ParseBool(os.Getenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting))
	if od.pluginCriteria == nil {
		pluginEntries, err = od.getInventory().GetPlugins(&plugininventory.PluginInventoryFilter{
			IncludeHidden: shouldIncludeHidden,
		})
		if err != nil {
			return nil, err
		}
	} else {
		pluginEntries, err = od.getInventory().GetPlugins(&plugininventory.PluginInventoryFilter{
			Name:          od.pluginCriteria.Name,
			Target:        od.pluginCriteria.Target,
			Version:       od.pluginCriteria.Version,
			OS:            od.pluginCriteria.OS,
			Arch:          od.pluginCriteria.Arch,
			IncludeHidden: shouldIncludeHidden,
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
	shouldIncludeHidden, _ := strconv.ParseBool(os.Getenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting))

	if od.groupCriteria == nil {
		return od.getInventory().GetPluginGroups(plugininventory.PluginGroupFilter{
			IncludeHidden: shouldIncludeHidden,
		})
	}

	return od.getInventory().GetPluginGroups(plugininventory.PluginGroupFilter{
		Vendor:        od.groupCriteria.Vendor,
		Publisher:     od.groupCriteria.Publisher,
		Name:          od.groupCriteria.Name,
		Version:       od.groupCriteria.Version,
		IncludeHidden: shouldIncludeHidden,
	})
}

// fetchInventoryImage downloads the OCI image containing the information about the
// inventory of this discovery and stores it in the cache directory.
func (od *DBBackedOCIDiscovery) fetchInventoryImage() error {
	// check the cache to see if downloaded plugin inventory database is up-to-date or not
	// by comparing the image digests
	newCacheHashFileForInventoryImage, newCacheHashFileForMetadataImage := od.checkImageCache()
	if newCacheHashFileForInventoryImage == "" && newCacheHashFileForMetadataImage == "" {
		// The cache can be re-used. We are done.
		return nil
	}

	// The DB has changed and needs to be updated in the cache.
	log.Infof("Reading plugin inventory for %q, this will take a few seconds.", od.image)

	// Verify the inventory image signature before downloading the plugin inventory database
	err := sigverifier.VerifyInventoryImageSignature(od.image)
	if err != nil {
		return err
	}

	// download plugin inventory image to get the 'plugin_inventory.db'
	// also handle the air-gapped scenario where additional plugin inventory metadata image is present
	err = od.downloadInventoryDatabase()
	if err != nil {
		return err
	}

	// Now that everything is ready, create the digest hash file
	if newCacheHashFileForInventoryImage != "" {
		_, _ = os.Create(newCacheHashFileForInventoryImage)
	}
	// Also create digest hash file for inventory metadata image if not empty
	if newCacheHashFileForMetadataImage != "" {
		_, _ = os.Create(newCacheHashFileForMetadataImage)
	}

	return nil
}

// downloadInventoryDatabase downloads plugin inventory image to get the 'plugin_inventory.db'
//
// Additional check for airgapped environment as below:
// Also check if plugin inventory metadata image is present or not. if present, downloads the inventory
// metadata image to get the 'plugin_inventory_metadata.db' and update the 'plugin_inventory.db'
// based on the 'plugin_inventory_metadata.db'
func (od *DBBackedOCIDiscovery) downloadInventoryDatabase() error {
	tempDir1, err := os.MkdirTemp("", "")
	if err != nil {
		return errors.Wrap(err, "unable to create temp directory")
	}
	tempDir2, err := os.MkdirTemp("", "")
	if err != nil {
		return errors.Wrap(err, "unable to create temp directory")
	}
	defer os.RemoveAll(tempDir1)
	defer os.RemoveAll(tempDir2)

	// Download the plugin inventory image and save to tempDir1
	if err := carvelhelpers.DownloadImageAndSaveFilesToDir(od.image, tempDir1); err != nil {
		return errors.Wrapf(err, "failed to download OCI image from discovery '%s'", od.Name())
	}

	inventoryDBFilePath := filepath.Join(tempDir1, plugininventory.SQliteDBFileName)
	metadataDBFilePath := filepath.Join(tempDir2, plugininventory.SQliteInventoryMetadataDBFileName)

	// Download the plugin inventory metadata image if exists and save to tempDir2
	pluginInventoryMetadataImage, _ := airgapped.GetPluginInventoryMetadataImage(od.image)
	if err := carvelhelpers.DownloadImageAndSaveFilesToDir(pluginInventoryMetadataImage, tempDir2); err == nil {
		// Update the plugin inventory database (plugin_inventory.db) based on the plugin
		// inventory metadata database (plugin_inventory_metadata.db)
		err = plugininventory.NewSQLiteInventoryMetadata(metadataDBFilePath).UpdatePluginInventoryDatabase(inventoryDBFilePath)
		if err != nil {
			return errors.Wrap(err, "error while updating inventory database based on the inventory metadata database")
		}
	}

	// Copy the inventory database file from temp directory to pluginDataDir
	return utils.CopyFile(inventoryDBFilePath, filepath.Join(od.pluginDataDir, plugininventory.SQliteDBFileName))
}

// checkImageCache will get the plugin inventory image digest as well as
// plugin inventory metadata image digest if exists for this discovery
// and check if the cache already contains the up-to-date database.
// Function returns two strings (hashFileForInventoryImage, HashFileForMetadataImage)
// It returns an empty string if the cache can be used.  Otherwise
// it returns the name of the digest file that must be created once
// the new DB image has been downloaded.
func (od *DBBackedOCIDiscovery) checkImageCache() (string, string) {
	// Get the latest digest of the discovery image.
	// If the cache already contains the image with this digest
	// we do not need to verify its signature nor to download it again.
	_, hashHexValInventoryImage, err := carvelhelpers.GetImageDigest(od.image)
	if err != nil {
		// This will happen when the user has configured an invalid image discovery URI
		log.Warningf("Unable to resolve the plugin discovery image: %v", err)
		// We force abort execution here to make sure a stale image left in the cache is not used by mistake.
		log.Fatal(nil, fmt.Sprintf("Fatal: plugins discovery image resolution failed. Please check that the repository image URL %q is correct ", od.image))
	}

	correctHashFileForInventoryImage := od.checkDigestFileExistence(hashHexValInventoryImage, "")

	pluginInventoryMetadataImage, _ := airgapped.GetPluginInventoryMetadataImage(od.image)
	_, hashHexValMetadataImage, _ := carvelhelpers.GetImageDigest(pluginInventoryMetadataImage)
	// Always store the metadata image digest file even if the image does not exists
	// If image does not exists a file named `metadata.digest.` will be stored
	// If image exists a file names `metadata.digest.<hexval>` will be stored
	// It is important to store the metadata digest file irrespective of image exists
	// or not for future comparisons and validating the cache
	// We do this, for this case:
	// 	- Point the discovery to "image-1" (which has corresponding metadata image defined) [Generally airgapped repository]
	// 	- Later, change to point to discovery "image-2" (which doesn't have corresponding metadata image present) [Generally Production repository]
	// 	The cache invalidation was not happening this time if the digest of "image-1" and "image-2" are same, but since we modify
	// 	the DB content in the air-gapped scenario, we have to invalidate the cache.
	correctHashFileForMetadataImage := od.checkDigestFileExistence(hashHexValMetadataImage, "metadata.")

	return correctHashFileForInventoryImage, correctHashFileForMetadataImage
}

// checkDigestFileExistence check the digest file already exists in the cache or not
// We store the digest hash of the cached DB as a file named "<digestPrefix>digest.<hash>.
// If this file exists, we are done. If not, we remove the current digest file
// as we are about to download a new DB and create a new digest file.
// First check any existing "<digestPrefix>digest.*" file; there should only be one, but
// to protect ourselves, we check first and if there are more then one due
// to some bug, we clean them up and invalidate the cache.
func (od *DBBackedOCIDiscovery) checkDigestFileExistence(hashHexVal, digestPrefix string) string {
	correctHashFile := filepath.Join(od.pluginDataDir, digestPrefix+"digest."+hashHexVal)
	matches, _ := filepath.Glob(filepath.Join(od.pluginDataDir, digestPrefix+"digest.*"))
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
