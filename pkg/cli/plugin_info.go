// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

// PluginInfo contains information about an installed plugin binary
type PluginInfo struct {
	// Name is the name of the plugin.
	Name string `json:"name" yaml:"name"`

	// Description is the plugin's description.
	Description string `json:"description" yaml:"description"`

	// Version of the plugin. Must be a valid semantic version https://semver.org/
	Version string `json:"version" yaml:"version"`

	// Command group for the plugin.
	Group plugin.CmdGroup `json:"group" yaml:"group"`

	// Hidden tells whether the plugin should be hidden from the help command.
	Hidden bool `json:"hidden,omitempty" yaml:"hidden,omitempty"`

	// Aliases are other text strings used to call this command
	Aliases []string `json:"aliases,omitempty" yaml:"aliases,omitempty"`

	// InstallationPath is the path to the plugin binary.
	InstallationPath string `json:"installationPath"`
}
