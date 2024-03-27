// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/airgapped"
	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cosignhelper/sigverifier"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

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
	// useLocalCacheOnly enables to get the inventory data from the cache without first refreshing cache
	useLocalCacheOnly bool
	// forceRefresh enables to force the refresh of the cached inventory data,
	// even if the cache TTL has not expired
	forceRefresh bool
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

// List is a method of the DBBackedOCIDiscovery struct that retrieves the available plugins.
// It returns a slice of Discovered interfaces and an error if any occurs during the process.
func (od *DBBackedOCIDiscovery) List() ([]Discovered, error) {
	// If useLocalCacheOnly option is not set, fetch the inventory image
	if !od.useLocalCacheOnly {
		// Fetch the inventory image
		err := od.fetchInventoryImage()
		if err != nil {
			// Return an error if unable to fetch the inventory image for plugins
			return nil, errors.Wrapf(err, "unable to fetch the inventory of discovery '%s' for plugins", od.Name())
		}
	}

	// List and return the plugins from the inventory
	return od.listPluginsFromInventory()
}

// GetGroups is a method of the DBBackedOCIDiscovery struct that retrieves the plugin groups defined in the discovery.
// It returns a slice of PluginGroup pointers and an error if any occurs during the process.
func (od *DBBackedOCIDiscovery) GetGroups() ([]*plugininventory.PluginGroup, error) {
	// If useLocalCacheOnly option is not set, fetch the inventory image
	if !od.useLocalCacheOnly {
		// Fetch the inventory image
		err := od.fetchInventoryImage()
		if err != nil {
			// Return an error if unable to fetch the inventory image for groups
			return nil, errors.Wrapf(err, "unable to fetch the inventory of discovery '%s' for groups", od.Name())
		}
	}

	// List and return the groups from the inventory
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
	if !od.forceRefresh && !od.cacheTTLExpired() {
		// If we refreshed the inventory image recently, don't refresh again.
		// The inventory image does not need to be up-to-date by the second.
		// This avoids uselessly refreshing for commands that are run close together.
		// For example:
		//   1- installing plugins for a plugin group (it is fast when the plugins are in the cache)
		//   2- installing plugins when creating a context (it is fast when the plugins are in the cache)
		//   3- multiple "plugin search" and "plugin group search" commands in a row
		return nil
	}

	// check the cache to see if downloaded plugin inventory database is up-to-date or not
	// by comparing the image digests
	newCacheHashFileForInventoryImage, newCacheHashFileForMetadataImage, err := od.checkImageCache()
	if err != nil {
		return err
	}

	if newCacheHashFileForInventoryImage == "" && newCacheHashFileForMetadataImage == "" {
		// The cache can be re-used. We are done.
		od.resetCacheTTL()
		return nil
	}

	// The DB has changed and needs to be updated in the cache.
	log.Infof("Reading plugin inventory for %q, this will take a few seconds.", od.image)

	// Verify the inventory image signature before downloading the plugin inventory database
	err = sigverifier.VerifyInventoryImageSignature(od.image)
	if err != nil {
		return err
	}

	// download the central repository image to get the 'plugin_inventory.db' and `central_config.yaml` files.
	// Also handle the air-gapped scenario where additional plugin inventory metadata image is present
	err = od.downloadCentralRepositoryData()
	if err != nil {
		return err
	}

	// Now that the new DB has been downloaded, we can reset the TTL.
	// We do this because it is possible that only the metadata digest has changed,
	// so we must reset the TTL on the main digest file.
	od.resetCacheTTL()

	// Now that everything is ready, create the digest hash file.
	// Both the name of the file and its content are important.
	if newCacheHashFileForInventoryImage != "" {
		if file, err := os.Create(newCacheHashFileForInventoryImage); err == nil {
			// We also store the URI of the image in the digest file so that we can
			// know in the future if the URI has changed.  This has particular value
			// for images added using TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY
			// since they are not stored in the config file and therefore we cannot associate
			// a discovery name with a URI such discoveries.
			_, _ = file.WriteString(od.image)
			file.Close()
		}
	}

	// Also create digest hash file for inventory metadata image if not empty
	if newCacheHashFileForMetadataImage != "" {
		if file, err := os.Create(newCacheHashFileForMetadataImage); err == nil {
			file.Close()
		}
	}

	return nil
}

// downloadCentralRepositoryData downloads the central repository OCI image to get the
// 'plugin_inventory.db' and 'central_config.yaml' files
//
// Additional check for airgapped environment as below:
// Also check if plugin inventory metadata image is present or not. if present, downloads the inventory
// metadata image to get the 'plugin_inventory_metadata.db' and update the 'plugin_inventory.db'
// based on the 'plugin_inventory_metadata.db'
func (od *DBBackedOCIDiscovery) downloadCentralRepositoryData() error {
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

	// Download the central repo OCI image and save it to tempDir1
	if err := carvelhelpers.DownloadImageAndSaveFilesToDir(od.image, tempDir1); err != nil {
		return errors.Wrapf(err, "failed to download OCI image from discovery '%s'", od.Name())
	}

	od.setupCentralConfig(tempDir1)

	return od.setupPluginInventory(tempDir1, tempDir2)
}

func (od *DBBackedOCIDiscovery) setupCentralConfig(sourceDir string) {
	// Copy the central config file from the temp directory to pluginDataDir
	sourceCentralConfigPath := filepath.Join(sourceDir, centralconfig.CentralConfigFileName)
	destCentralConfigPath := filepath.Join(od.pluginDataDir, centralconfig.CentralConfigFileName)

	// Since the central config file is optional, check if it is present in the downloaded OCI image.
	if _, err := os.Stat(sourceCentralConfigPath); os.IsNotExist(err) {
		// The file does not exist, remove any old one from the cache
		// that might be there from a previous discovery.
		_ = os.Remove(destCentralConfigPath)
	} else {
		// The file exists, copy it to the pluginDataDir
		if err = utils.CopyFile(sourceCentralConfigPath, destCentralConfigPath); err != nil {
			// Don't fail just log a warning, as the central config file is not critical.
			log.V(6).Warningf("unable to copy central config file: %v", err)
		}
	}
}

func (od *DBBackedOCIDiscovery) setupPluginInventory(inventoryDir, metadataDir string) error {
	inventoryDBFilePath := filepath.Join(inventoryDir, plugininventory.SQliteDBFileName)
	metadataDBFilePath := filepath.Join(metadataDir, plugininventory.SQliteInventoryMetadataDBFileName)

	// Download the plugin inventory metadata image if exists and save to metadataDir
	pluginInventoryMetadataImage, _ := airgapped.GetPluginInventoryMetadataImage(od.image)
	if err := carvelhelpers.DownloadImageAndSaveFilesToDir(pluginInventoryMetadataImage, metadataDir); err == nil {
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
// the plugin inventory metadata image digest (if it exists) for this discovery.
// It will then check if the cache already contains the up-to-date database.
// The function returns two strings: hashFileForDB, hashFileForMetadata which
// are the paths to the digest files that must be created once the new DB image
// and its optional metadata image have been downloaded.
// It returns an empty string when the digest has not changed and therefore
// the file already exists.  When both strings are empty, meaning both the DB
// and its metadata image have not changed, then the cached DB can be used.
// Otherwise the new DB image and its metadata image have to be downloaded, and the
// two new digest files have to be created by the calling function.
func (od *DBBackedOCIDiscovery) checkImageCache() (string, string, error) {
	// Get the latest digest of the discovery image.
	// If the cache already contains the image with this digest
	// we do not need to verify its signature nor to download it again.
	log.Infof("Refreshing plugin inventory cache for %q, this will take a few seconds.", od.image)
	_, hashHexValInventoryImage, err := carvelhelpers.GetImageDigest(od.image)
	if err != nil {
		// This will happen when the user has configured an invalid image discovery URI
		return "", "", errors.Wrapf(err, "plugins discovery image resolution failed. Please check that the repository image URL %q is correct", od.image)
	}

	correctHashFileForInventoryImage := od.checkDigestFileExistence(hashHexValInventoryImage, "")

	pluginInventoryMetadataImage, _ := airgapped.GetPluginInventoryMetadataImage(od.image)
	_, hashHexValMetadataImage, _ := carvelhelpers.GetImageDigest(pluginInventoryMetadataImage)
	// Always store the metadata image digest file even if the image does not exists.
	// If the metadata image does not exist, a file named `metadata.digest.none` will be stored.
	// If the metadata image exists, a file named `metadata.digest.<hexval>` will be stored.
	// It is important to store the metadata digest file irrespective of if the metadata image exists
	// or not for future comparisons and validating the cache.
	// We do this, for this case:
	// 	- Point the discovery to "image-1" (which has corresponding metadata image defined) [Generally airgapped repository]
	// 	- Later, change to point to discovery "image-2" (which doesn't have a corresponding metadata image present) [Generally Production repository]
	// The cache invalidation was not happening in this case if the digest of "image-1" and "image-2" were the same
	// but since we modify the DB content in the air-gapped scenario, we have to invalidate the cache.
	correctHashFileForMetadataImage := od.checkDigestFileExistence(hashHexValMetadataImage, "metadata.")

	return correctHashFileForInventoryImage, correctHashFileForMetadataImage, nil
}

// checkDigestFileExistence check the digest file already exists in the cache or not
// We store the digest hash of the cached DB as a file named "<digestPrefix>digest.<hash>.
// If this file exists, we are done. If not, we remove the current digest file
// as we are about to download a new DB and will create a new digest file.
// First check any existing "<digestPrefix>digest.*" file; there should only be one, but
// to protect ourselves, we check first and if there are more then one due
// to some bug, we clean them up and invalidate the cache.
func (od *DBBackedOCIDiscovery) checkDigestFileExistence(hashHexVal, digestPrefix string) string {
	if hashHexVal == "" {
		// This can only be the case of the metadata file and indicates no metadata will be used.
		// We have to have a value after the "<digestPrefix>digest." because on Windows, a trailing '.'
		// is ignored and would prevent the Glob matching to occur.
		// https://github.com/vmware-tanzu/tanzu-cli/issues/392
		hashHexVal = "none"
	}

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
			if digestPrefix == "" {
				// If the main digest has not changed it means we have the DB for this OCI image URI.
				// Normally that implies the URI stored in the main digest file has not changed either.
				// However, there are corner cases where the URI of the image could still have
				// changed but the digest has not. For example, the URI in the digest
				// file can be "example.com/image:latest" and the discovery image has changed
				// to "example.com/image"; both URIs are actually the same image with the
				// same digest.  In such cases, the digest file will be retained since the image
				// does not need to be re-downloaded.  To deal with such cases, we set the
				// (possibly new) URI in the digest file each time the digest is not changed.
				if file, err := os.Create(matches[0]); err == nil {
					_, _ = file.WriteString(od.image)
					file.Close()
				}
			}
			return ""
		}
		// The hash file indicates a different digest hash. Remove this old hash file
		// as we will download the new DB.
		os.Remove(matches[0])
	}
	return correctHashFile
}

func getCacheTTLValue() int {
	cacheTTL := constants.DefaultInventoryRefreshTTLSeconds
	cacheTTLOverride := os.Getenv(constants.ConfigVariablePluginDBCacheTTLSeconds)
	if cacheTTLOverride != "" {
		cacheTTLOverrideValue, err := strconv.Atoi(cacheTTLOverride)
		if err == nil && cacheTTLOverrideValue >= 0 {
			cacheTTL = cacheTTLOverrideValue
		}
	}
	return cacheTTL
}

// cacheTTLExpired checks if the last time the cache was refreshed has passed its TTL.
func (od *DBBackedOCIDiscovery) cacheTTLExpired() bool {
	matches, _ := filepath.Glob(filepath.Join(od.pluginDataDir, "digest.*"))
	if len(matches) == 1 {
		file, err := os.Open(matches[0])
		if err != nil {
			return true
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		if scanner.Scan() {
			// Check that the discovery image URI matches what is in the digest file.
			// This is important for test discoveries which can be changed by setting
			// the TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY variable.
			// If the image URI is not correct then the cache must be refreshed.
			cachedURI := scanner.Text()
			if cachedURI == od.image {
				// The URI matches.  Now check the modification time of the digest file to see
				// if the TTL is expired.
				if stat, err := os.Stat(matches[0]); err == nil {
					return time.Since(stat.ModTime()) > time.Duration(getCacheTTLValue())*time.Second
				}
			}
		}
	}

	// We need to refresh the cache.
	return true
}

// resetCacheTTL resets the modification timestamp of the digest file to the current time.
// This is used to avoid checking the inventory image digest too often.
func (od *DBBackedOCIDiscovery) resetCacheTTL() {
	matches, _ := filepath.Glob(filepath.Join(od.pluginDataDir, "digest.*"))
	if len(matches) == 1 {
		var zeroTime time.Time
		_ = os.Chtimes(matches[0], zeroTime, time.Now())
	}
}
