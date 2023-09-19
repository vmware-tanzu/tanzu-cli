// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains useful functionality for config updates
package config

import (
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

func init() {
	// Acquire tanzu config lock
	config.AcquireTanzuConfigLock()

	c, err := config.GetClientConfigNoLock()
	if err != nil {
		log.Warningf("unable to get client config: %v", err)
	}
	addedFeatureFlags := AddDefaultFeatureFlagsIfMissing(c, constants.DefaultCliFeatureFlags)
	addedEdition := addDefaultEditionIfMissing(c)
	addedBomRepo := AddBomRepoIfMissing(c)
	addedCompatabilityFile := AddCompatibilityFileIfMissing(c)
	// contexts could be lost when older plugins edit the config, so populate them from servers
	addedContexts := config.PopulateContexts(c)

	if addedFeatureFlags || addedEdition || addedCompatabilityFile || addedBomRepo || addedContexts {
		_ = config.StoreClientConfig(c)
	}

	// We need to release the config lock before calling PopulateDefaultCentralDiscovery() because
	// PopulateDefaultCentralDiscovery() handles the locking of the config file itself by
	// using the config file higher-level APIs
	config.ReleaseTanzuConfigLock()
	_ = PopulateDefaultCentralDiscovery(false)
}
