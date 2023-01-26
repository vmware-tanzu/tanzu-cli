// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
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
func (od *DBBackedOCIDiscovery) List() (plugins []Discovered, err error) {
	pluginInventoryDir, err := od.fetchInventoryImage()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch the inventory of discovery '%s'", od.Name())
	}

	// The plugin inventory uses relative image URIs to be future-proof.
	// Determine the image prefix from the main image.
	// E.g., if the main image is at project.registry.vmware.com/tanzu-cli/plugins/plugin-inventory:latest
	// then the image prefix should be project.registry.vmware.com/tanzu-cli/plugins/
	imagePrefix := path.Dir(od.image)
	backend := NewSQLiteBackend(od.Name(), pluginInventoryDir, imagePrefix)

	allPluginPtrs, err := backend.GetAllPlugins()
	if err != nil {
		return nil, err
	}

	// Convert from plugin pointers to plugins
	// TODO(khouzam): continue optimizing by converting every call to using pointers
	var allPlugins []Discovered
	for _, pluginPtr := range allPluginPtrs {
		allPlugins = append(allPlugins, *pluginPtr)
	}
	return allPlugins, nil
}

// fetchInventoryImage downloads the OCI image containing the information about the
// inventory of this discovery and stores it in the cache directory.
// It returns the path to the exact directory used.
func (od *DBBackedOCIDiscovery) fetchInventoryImage() (string, error) {
	// TODO(khouzam): Improve by checking if we really need to download again or if we can use the cache
	pluginDataDir := filepath.Join(common.DefaultCacheDir, inventoryDirName)
	if err := carvelhelpers.DownloadImageAndSaveFilesToDir(od.image, pluginDataDir); err != nil {
		return "", errors.Wrapf(err, "failed to download OCI image from discovery '%s'", od.Name())
	}
	return pluginDataDir, nil
}
