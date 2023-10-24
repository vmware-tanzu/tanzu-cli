// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugininventory implements an interface to deal with a plugin inventory.
// It encapsulates the logic that deals with how plugin inventories are stored
// so that other entities can use the plugin inventory without knowing its
// implementation details.
package plugininventory

import (
	"fmt"
	"strings"

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

	// GetPluginGroups returns the plugin groups found in the inventory that match the provided filter.
	GetPluginGroups(PluginGroupFilter) ([]*PluginGroup, error)

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
	// Vendor of the plugins to look for
	Vendor string
	// IncludeHidden indicates if hidden plugins should be included
	IncludeHidden bool
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

// PluginGroupIdentifier uniquely identifies a single version of a specific plugin group
type PluginGroupIdentifier struct {
	// Vendor of the group
	Vendor string
	// Publisher of the group
	Publisher string
	// Name of the group
	Name string
	// Version of the group
	Version string
}

// PluginGroup represents a list of plugins.
// The user will specify a group using
// "<Vendor>-<Publisher>/<Name>:<Version>
// e.g., "vmware-tkg/default:v2.1.0"
type PluginGroup struct {
	// Vendor of the group
	Vendor string
	// Publisher of the group
	Publisher string
	// Name of the group
	Name string
	// Description of the group
	Description string
	// Hidden tells whether the plugin-group should be ignored by the CLI.
	Hidden bool
	// Recommended version that the Tanzu CLI should install by default.
	// The value should be a valid semantic version as defined in
	// https://semver.org/. E.g., 2.0.1
	RecommendedVersion string
	// Map of version to list of plugins
	Versions map[string][]*PluginGroupPluginEntry
}

func PluginGroupToID(pg *PluginGroup) string {
	return fmt.Sprintf("%s-%s/%s", pg.Vendor, pg.Publisher, pg.Name)
}

// PluginGroupIdentifierFromID converts a plugin group id into a
// PluginGroupIdentifier structure.
// A group id can be of the forms:
//  1. vendor-publisher/name:version
//  2. vendor-publisher/name, in which case the version field is left empty
//
// Returns nil if 'id' is not of the expected format.
func PluginGroupIdentifierFromID(id string) *PluginGroupIdentifier {
	pg := &PluginGroupIdentifier{}

	// Split into "vendor-publisher" and "name:version"
	arr := strings.Split(id, "/")
	if len(arr) != 2 {
		return nil
	}

	// Split "vendor-publisher" into "vendor" and "publisher"
	vendorPublisher := strings.Split(arr[0], "-")
	if len(vendorPublisher) != 2 {
		return nil
	}
	pg.Vendor = vendorPublisher[0]
	pg.Publisher = vendorPublisher[1]

	// Sprint "name:version" into "name" an optionally "version"
	nameVersion := strings.Split(arr[1], ":")
	pg.Name = nameVersion[0]
	if len(nameVersion) > 2 {
		return nil
	}
	if len(nameVersion) == 2 {
		pg.Version = nameVersion[1]
	}

	if pg.Name == "" || pg.Vendor == "" || pg.Publisher == "" { // It is ok for "pg.Version" to be empty
		// This can happen if the id is something like `vmware-/default` or `vmware-tkg/`
		return nil
	}
	return pg
}

// PluginGroupFilter allows to specify different criteria for
// looking up plugin group entries.
type PluginGroupFilter struct {
	// Vendor of the group to look for
	Vendor string
	// Publisher of the group to look for
	Publisher string
	// Name of the group to look for
	Name string
	// Version of the group
	Version string
	// IncludeHidden indicates if hidden plugin groups should be included
	IncludeHidden bool
}

// PluginGroupSorter sorts PluginGroup objects.
type PluginGroupSorter []*PluginGroup

func (p PluginGroupSorter) Len() int      { return len(p) }
func (p PluginGroupSorter) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p PluginGroupSorter) Less(i, j int) bool {
	return PluginGroupToID(p[i]) < PluginGroupToID(p[j])
}
