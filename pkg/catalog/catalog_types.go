// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// PluginAssociation is a set of plugin names and their associated installation paths.
type PluginAssociation map[string]string

// Add adds plugin entry to the map
func (pa PluginAssociation) Add(pluginName, installationPath string) {
	if pa == nil {
		pa = map[string]string{}
	}
	pa[pluginName] = installationPath
}

// Remove deletes plugin entry from the map
func (pa PluginAssociation) Remove(pluginName string) {
	delete(pa, pluginName)
}

// Get returns installation path for the plugin
// If plugin doesn't exists in map it will return empty string
func (pa PluginAssociation) Get(pluginName string) string {
	return pa[pluginName]
}

// Map returns associated list of plugins as a map
func (pa PluginAssociation) Map() map[string]string {
	return pa
}

// Catalog is the Schema for the plugin catalog data
type Catalog struct {
	// PluginInfos is a list of PluginInfo
	PluginInfos []*cli.PluginInfo `json:"pluginInfos,omitempty" yaml:"pluginInfos"`

	// IndexByPath of PluginInfos for all installed plugins by installation path.
	IndexByPath map[string]cli.PluginInfo `json:"indexByPath,omitempty"`
	// IndeByName of all plugin installation paths by name.
	IndexByName map[string][]string `json:"indexByName,omitempty"`
	// StandAlonePlugins is a set of stand-alone plugin installations aggregated across all context types.
	// Note: Shall be reduced to only those stand-alone plugins that are common to all context types.
	StandAlonePlugins PluginAssociation `json:"standAlonePlugins,omitempty"`
	// ServerPlugins links a server and a set of associated plugin installations.
	ServerPlugins map[string]PluginAssociation `json:"serverPlugins,omitempty"`
}

// CatalogList contains a list of Catalog
type CatalogList struct {
	Items []Catalog `json:"items"`
}
