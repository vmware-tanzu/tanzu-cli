// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
)

// NewOCIDiscovery returns a new Discovery using the specified OCI image.
func NewOCIDiscovery(name, image string, options ...DiscoveryOptions) Discovery {
	// Initialize discovery options
	opts := NewDiscoveryOpts()
	for _, option := range options {
		option(opts)
	}

	discovery := newDBBackedOCIDiscovery(name, image)
	discovery.pluginCriteria = opts.PluginDiscoveryCriteria
	discovery.useLocalCacheOnly = opts.UseLocalCacheOnly
	// NOTE: the use of TEST_TANZU_CLI_USE_DB_CACHE_ONLY is for testing only
	if useCacheOnlyForTesting, _ := strconv.ParseBool(os.Getenv("TEST_TANZU_CLI_USE_DB_CACHE_ONLY")); useCacheOnlyForTesting {
		discovery.useLocalCacheOnly = true
	}
	discovery.forceRefresh = opts.ForceRefresh

	return discovery
}

// NewOCIGroupDiscovery returns a new plugn group Discovery using the specified OCI image.
func NewOCIGroupDiscovery(name, image string, options ...DiscoveryOptions) GroupDiscovery {
	// Initialize discovery options
	opts := NewDiscoveryOpts()
	for _, option := range options {
		option(opts)
	}

	discovery := newDBBackedOCIDiscovery(name, image)
	discovery.groupCriteria = opts.GroupDiscoveryCriteria
	discovery.useLocalCacheOnly = opts.UseLocalCacheOnly
	// NOTE: the use of TEST_TANZU_CLI_USE_DB_CACHE_ONLY is for testing only
	if useCacheOnlyForTesting, _ := strconv.ParseBool(os.Getenv("TEST_TANZU_CLI_USE_DB_CACHE_ONLY")); useCacheOnlyForTesting {
		discovery.useLocalCacheOnly = true
	}
	discovery.forceRefresh = opts.ForceRefresh

	return discovery
}

func newDBBackedOCIDiscovery(name, image string) *DBBackedOCIDiscovery {
	// The plugin inventory uses relative image URIs to be future-proof.
	// Determine the image prefix from the main image.
	// E.g., if the main image is at project.registry.vmware.com/tanzu-cli/plugins/plugin-inventory:latest
	// then the image prefix should be project.registry.vmware.com/tanzu-cli/plugins/
	imagePrefix := path.Dir(image)
	// The data for the inventory is stored in the cache
	pluginDataDir := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, name)

	inventory := plugininventory.NewSQLiteInventory(filepath.Join(pluginDataDir, plugininventory.SQliteDBFileName), imagePrefix)
	return &DBBackedOCIDiscovery{
		name:          name,
		image:         image,
		pluginDataDir: pluginDataDir,
		inventory:     inventory,
	}
}
