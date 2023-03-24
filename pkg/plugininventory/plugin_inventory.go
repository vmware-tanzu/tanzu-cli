// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugininventory implements an interface to deal with a plugin inventory.
// It encapsulates the logic that deals with how plugin inventories are stored
// so that other entities can use the plugin inventory without knowing its
// implementation details.
package plugininventory

import (
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// PluginInventory is the interface to interact with a plugin inventory.
// It can be used to get the plugin information for plugins in the
// inventory based on different criteria.
type PluginInventory interface {
	// GetAllPlugins returns all plugins found in the inventory.
	GetAllPlugins() ([]*PluginInventoryEntry, error)

	// GetPlugins returns the plugins found in the inventory that match the provided filter.
	GetPlugins(*PluginInventoryFilter) ([]*PluginInventoryEntry, error)

	// GetAllGroups returns all plugin groups found in the inventory.
	GetAllGroups() ([]*PluginGroup, error)

	// CreateSchema creates table schemas to the provided database.
	// returns error if table creation fails for any reason
	CreateSchema() error

	// InsertPlugin inserts plugin to the inventory
	InsertPlugin(*PluginInventoryEntry) error

	// InsertPluginGroup inserts plugin-group to the inventory
	// if override is true, it will update the existing plugin by
	// updating the metadata and the plugin associated with the plugin-group
	InsertPluginGroup(pg *PluginGroup, override bool) error

	// UpdatePluginActivationState updates plugin metadata to activate or deactivate plugin
	UpdatePluginActivationState(*PluginInventoryEntry) error

	// UpdatePluginGroupActivationState updates plugin-group metadata to activate or deactivate the plugin-group
	UpdatePluginGroupActivationState(*PluginGroup) error
}

// PluginInventoryEntry represents the inventory information
// about a single plugin as found by the inventory backend.
type PluginInventoryEntry struct {
	// Name of the plugin
	Name string
	// Target to which the plugin applies
	Target configtypes.Target
	// Description of the plugin
	Description string
	// Publisher is the name of the publisher of this plugin
	// (e.g., a product group within a company)
	Publisher string
	// Vendor is the name of the vendor of this plugin (e.g., a company's name)
	Vendor string
	// Recommended version that Tanzu CLI should install by default.
	// The value should be a valid semantic version as defined in
	// https://semver.org/. E.g., 2.0.1
	RecommendedVersion string
	// Hidden tells whether the plugin is marked as hidden or not.
	Hidden bool
	// Artifacts contains an artifact list for every available version.
	Artifacts distribution.Artifacts
}

// PluginInventoryFilter allows to specify different criteria for
// looking up plugin entries.
type PluginInventoryFilter struct {
	// Name of the plugin to look for
	Name string
	// Target to which the plugins apply
	Target configtypes.Target
	// Version for the plugins to look for
	Version string
	// OS of the plugin binary in `GOOS` format.
	OS string
	// Arch of the plugin binary in `GOARCH` format.
	Arch string
	// Publisher of the plugins to look for
	Publisher string
	// Vendor the plugins to look for
	Vendor string
}

// PluginIdentifier uniquely identifies a single version of a specific plugin
type PluginIdentifier struct {
	// Name is the name of the plugin
	Name string
	// Target is the target of the plugin
	Target configtypes.Target
	// Version is the version for the plugin
	Version string
}

// PluginGroupPluginEntry represents a plugin entry within a plugin group
type PluginGroupPluginEntry struct {
	// The plugin version of this plugin entry
	PluginIdentifier

	// Mandatory specifies if the plugin is required to be installed or not
	Mandatory bool
}

// PluginGroup represents a list of plugins.
// The user will specify a group using
// "<Vendor>-<Publisher>/<Name>
// e.g., "vmware-tkg/v2.1.0"
type PluginGroup struct {
	// Vendor of the group
	Vendor string
	// Publisher of the group
	Publisher string
	// Name of the group
	Name string
	// Hidden tells whether the plugin-group should be ignored by the CLI.
	Hidden bool
	// The list of plugins specified by this group
	Plugins []*PluginGroupPluginEntry
}
