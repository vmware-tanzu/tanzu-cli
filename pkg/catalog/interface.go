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

// PluginCatalog is the interface to a collection of installed plugins.
type PluginCatalog interface {
	// Upsert inserts/updates the given plugin.
	Upsert(plugin cli.PluginInfo)

	// Get looks up the info of a plugin given its name.
	Get(pluginName string) (cli.PluginInfo, bool)

	// List returns the list of active plugins.
	// Active plugin means the plugin that are available to the user
	// based on the current logged-in server.
	List() []cli.PluginInfo

	// Delete deletes the given plugin from the catalog, but it does not delete the installation.
	Delete(plugin string)
}
