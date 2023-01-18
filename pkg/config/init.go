// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains useful functionality for config updates
package config

import (
	"github.com/aunum/log"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

func init() {
	// Acquire tanzu config lock
	config.AcquireTanzuConfigLock()
	defer config.ReleaseTanzuConfigLock()

	c, err := config.GetClientConfigNoLock()
	if err != nil {
		log.Warningf("unable to get client config: %v", err)
	}
	// Note: Commenting the below line since CLI wouldn't support any default discovery going forward.
	//      Users have to add the discovery sources by using `tanzu plugin source add` command
	// TODO: update/delete the below line after CLI make changes related to centralized repository
	// addedDefaultDiscovery := populateDefaultStandaloneDiscovery(c)
	addedFeatureFlags := AddDefaultFeatureFlagsIfMissing(c, constants.DefaultCliFeatureFlags)
	addedEdition := addDefaultEditionIfMissing(c)
	addedBomRepo := AddBomRepoIfMissing(c)
	addedCompatabilityFile := AddCompatibilityFileIfMissing(c)
	// contexts could be lost when older plugins edit the config, so populate them from servers
	addedContexts := config.PopulateContexts(c)

	if addedFeatureFlags || addedEdition || addedCompatabilityFile || addedBomRepo || addedContexts {
		_ = config.StoreClientConfig(c)
	}
}
