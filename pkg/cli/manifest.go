// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"time"
)

const (
	// ManifestFileName is the file name for the manifest.
	ManifestFileName = "manifest.yaml"
	// PluginManifestFileName is the file name for the plugin manifest.
	PluginManifestFileName = "plugin_manifest.yaml"
	// PluginGroupManifestFileName is the file name for the plugin manifest.
	PluginGroupManifestFileName = "plugin_group_manifest.yaml"
	// PluginDescriptorFileName is the file name for the plugin descriptor.
	PluginDescriptorFileName = "plugin.yaml"
	// AllPlugins is the keyword for all plugins.
	AllPlugins = "all"
)

// Manifest is stored in the repository which gives an inventory of the artifacts.
type Manifest struct {
	// Created is the time the manifest was created.
	CreatedTime time.Time `json:"created,omitempty" yaml:"created,omitempty"`

	// Plugins is a list of plugin artifacts available.
	Plugins []Plugin `json:"plugins" yaml:"plugins"`
}

// GetCoreVersion returns the core version in a backwards compatible manner.
/*
func (m *Manifest) GetCoreVersion() string {
    if m.Version != "" {
        return m.Version
    }
    return m.CoreVersion
}
*/

// Plugin is an installable CLI plugin.
type Plugin struct {
	// Name is the name of the plugin.
	Name string `json:"name" yaml:"name"`

	// Target is the name of the plugin.
	Target string `json:"target" yaml:"target"`

	// Description is the plugin's description.
	Description string `json:"description" yaml:"description"`

	// Versions available for plugin.
	Versions []string `json:"versions" yaml:"versions"`
}

// PluginGroupManifest is used to parse metadata about Plugin Groups
type PluginGroupManifest struct {
	// Created is the time the manifest was created.
	CreatedTime time.Time `json:"created,omitempty" yaml:"created,omitempty"`

	// Plugins is a list of plugin artifacts including scope and version
	Plugins []PluginNameTargetScopeVersion `json:"plugins" yaml:"plugins"`
}

// PluginScopeMetadata is used to parse metadata about plugin and it's scope
type PluginScopeMetadata struct {
	// Plugins is a list of plugin artifacts including scope and version
	Plugins []PluginNameTargetScope `json:"plugins" yaml:"plugins"`
}

// PluginNameTargetScopeVersion defines the name, target, scope and version of a plugin
type PluginNameTargetScopeVersion struct {
	// PluginNameTargetScope
	PluginNameTargetScope `json:",inline" yaml:",inline"`

	// Version is the version of the plugin.
	Version string `json:"version" yaml:"version"`
}

// PluginNameTargetScope defines the name, target and scope of a plugin
type PluginNameTargetScope struct {
	// Name is the name of the plugin.
	Name string `json:"name" yaml:"name"`

	// Target is the name of the plugin.
	Target string `json:"target" yaml:"target"`

	// IsContextScoped determines if the plugin is context-scoped or standalone
	IsContextScoped bool `json:"isContextScoped" yaml:"isContextScoped"`
}
