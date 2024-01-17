// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package constants

// This block is for global feature constants, to allow them to be used more broadly
const (
	// FeatureContextCommand determines whether to surface the context command. This is disabled by default.
	FeatureContextCommand = "features.global.context-target-v2"

	// FeatureContextScopedPluginDiscoveryForTanzuContext determines whether to enable context-scoped plugin discovery for Tanzu context.
	// This is disabled by default
	FeatureContextScopedPluginDiscoveryForTanzuContext = "features.global.context-plugin-discovery-for-tanzu-context"
)

// DefaultCliFeatureFlags is used to populate an initially empty config file with default values for feature flags.
// The keys MUST be in the format "features.global.<feature>" or initialization will fail
//
// If a developer expects that their feature will be ready to release, they should create an entry here with a true
// value.
// If a developer has a beta feature they want to expose, but leave turned off by default, they should create
// an entry here with a false value. WE HIGHLY RECOMMEND the use of a SEPARATE flag for beta use; one that ends in "-beta".
// Thus, if you plan to eventually release a feature with a flag named "features.global.foo-bar", you should consider
// releasing the beta version with "features.global.foo-bar-beta". This will make it much easier when it comes time for
// mainstreaming the feature (with a default true value) under the flag name "features.global.foo-bar", as there will be
// no conflict with previous installs (that have a false value for the entry "features.global.foo-bar-beta").
var (
	DefaultCliFeatureFlags = map[string]bool{
		FeatureContextCommand: true,
		// Do NOT include the test feature flag to disable the central repo.
		// We don't want to publicize this feature flag.
		// It defaults to false when not specified, which is what is needed.
	}
)
