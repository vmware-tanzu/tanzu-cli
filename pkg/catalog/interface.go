// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package catalog implements catalog management functions
package catalog

import (
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// PluginSupplier is responsible for keeping an inventory of installed plugins
type PluginSupplier interface {
	// GetInstalledPlugins returns a list of installed plugins
	GetInstalledPlugins() ([]*cli.PluginInfo, error)
}

// PluginCatalogUpdater is the interface to read and update a collection of installed plugins.
type PluginCatalogUpdater interface {
	PluginCatalogReader

	// Upsert inserts/updates the given plugin.
	Upsert(plugin *cli.PluginInfo) error

	// Delete deletes the given plugin from the catalog, but it does not delete the installation.
	Delete(plugin string) error

	// Unlock unlocks the catalog for other process to read/write
	Unlock()
}

// PluginCatalogReader is the interface to a read collection of installed plugins.
type PluginCatalogReader interface {
	// Get looks up the info of a plugin given its name.
	Get(pluginName string) (cli.PluginInfo, bool)

	// List returns the list of active plugins.
	// Active plugin means the plugin that are available to the user
	// based on the current logged-in server.
	List() []cli.PluginInfo
}
