// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	cliv1alpha1 "github.com/vmware-tanzu/tanzu-framework/apis/cli/v1alpha1"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

// PluginInfo contains information about a plugin binary
type PluginInfo struct {
	// Name is the name of the plugin.
	Name string `json:"name" yaml:"name"`

	// Description is the plugin's description.
	Description string `json:"description" yaml:"description"`

	// Version of the plugin. Must be a valid semantic version https://semver.org/
	Version string `json:"version" yaml:"version"`

	// BuildSHA is the git commit hash the plugin was built with.
	BuildSHA string `json:"buildSHA" yaml:"buildSHA"`

	// Digest is the SHA256 hash of the plugin binary.
	Digest string `json:"digest" yaml:"digest"`

	// Command group for the plugin.
	Group plugin.CmdGroup `json:"group" yaml:"group"`

	// DocURL for the plugin.
	DocURL string `json:"docURL" yaml:"docURL"`

	// Hidden tells whether the plugin should be hidden from the help command.
	Hidden bool `json:"hidden,omitempty" yaml:"hidden,omitempty"`

	// CompletionType determines how command line completion will be determined.
	CompletionType plugin.PluginCompletionType `json:"completionType" yaml:"completionType"`

	// CompletionArgs contains the valid command line completion values if `CompletionType`
	// is set to `StaticPluginCompletion`.
	CompletionArgs []string `json:"completionArgs,omitempty" yaml:"completionArgs,omitempty"`

	// CompletionCommand is the command to call from the plugin to retrieve a list of
	// valid completion nouns when `CompletionType` is set to `DynamicPluginCompletion`.
	CompletionCommand string `json:"completionCmd,omitempty" yaml:"completionCmd,omitempty"`

	// Aliases are other text strings used to call this command
	Aliases []string `json:"aliases,omitempty" yaml:"aliases,omitempty"`

	// InstallationPath is a relative installation path for a plugin binary.
	// E.g., cluster/v0.3.2@sha256:...
	InstallationPath string `json:"installationPath"`

	// Discovery is the name of the discovery from where
	// this plugin is discovered.
	Discovery string `json:"discovery"`

	// Scope is the scope of the plugin. Stand-Alone or Context
	Scope string `json:"scope"`

	// Status is the current plugin installation status
	Status string `json:"status"`

	// DiscoveredRecommendedVersion specifies the recommended version of the plugin that was discovered
	DiscoveredRecommendedVersion string `json:"discoveredRecommendedVersion"`

	// Target specifies the target of the plugin
	Target cliv1alpha1.Target `json:"target"`

	// PostInstallHook is function to be run post install of a plugin.
	PostInstallHook plugin.Hook `json:"-" yaml:"-"`

	// DefaultFeatureFlags is default featureflags to be configured if missing when invoking plugin
	DefaultFeatureFlags map[string]bool `json:"defaultFeatureFlags"`
}
