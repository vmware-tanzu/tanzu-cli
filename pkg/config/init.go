// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains useful functionality for config updates
package config

import (
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

func init() {
	runInit()
}

func runInit() {
	// Configure default feature flags
	_ = config.ConfigureFeatureFlags(constants.DefaultCliFeatureFlags, config.SkipIfExists())

	// Populate contexts and servers
	_ = SyncContextsAndServers()

	// Populate default central discovery
	_ = PopulateDefaultCentralDiscovery(false)
}
