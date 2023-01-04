// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"time"
)

const (
	// ManifestFileName is the file name for the manifest.
	ManifestFileName = "manifest.yaml"
	// PluginDescriptorFileName is the file name for the plugin descriptor.
	PluginDescriptorFileName = "plugin.yaml"
	// AllPlugins is the keyword for all plugins.
	AllPlugins = "all"
)

// Manifest is stored in the repository which gives an inventory of the artifacts.
type Manifest struct {
	// Created is the time the manifest was created.
	CreatedTime time.Time `json:"created" yaml:"created"`

	// Plugins is a list of plugin artifacts available.
	Plugins []Plugin `json:"plugins" yaml:"plugins"`

	// Deprecated: Version of the root CLI.
	Version string `json:"version" yaml:"version"`

	// CoreVersion of the root CLI.
	CoreVersion string `json:"coreVersion" yaml:"coreVersion"`
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

	// Description is the plugin's description.
	Description string `json:"description" yaml:"description"`

	// Versions available for plugin.
	Versions []string `json:"versions" yaml:"versions"`
}
