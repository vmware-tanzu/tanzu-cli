// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides commands
package command

import (
	"strings"

	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/datastore"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const existingEndpointUpdateVersionKey = "existingEndpointUpdateVersion"

// updateConfigWithTanzuPlatformEndpointChanges uses the central configuration to see if the endpoint
// updates are needed for the CLI configuration or not. If it is needed start the endpoint updates
// for all available tanzu contexts
func updateConfigWithTanzuPlatformEndpointChanges() {
	// Get the requested endpoint update version from the default central configuration
	requestedEndpointUpdateVersion, err := centralconfig.DefaultCentralConfigReader.GetTanzuConfigEndpointUpdateVersion()
	if err != nil || requestedEndpointUpdateVersion == "" {
		return
	}

	// Get the existing endpoint update version from the datastore
	existingEndpointUpdateVersion := ""
	_ = datastore.GetDataStoreValue(existingEndpointUpdateVersionKey, &existingEndpointUpdateVersion)
	if requestedEndpointUpdateVersion == existingEndpointUpdateVersion {
		return
	}

	cfg, err := config.GetClientConfig()
	if err != nil || cfg == nil || len(cfg.KnownContexts) == 0 {
		return
	}

	// Get the endpoint update map from the default central configuration
	endpointUpdateMap, err := centralconfig.DefaultCentralConfigReader.GetTanzuConfigEndpointUpdateMapping()
	if err != nil {
		return
	}

	updateSuccess := true
	for idx := range cfg.KnownContexts {
		ctx := cfg.KnownContexts[idx]
		// If context is not eligible for endpoint update go to the next context
		if !isValidContextForEndpointUpdates(ctx) {
			continue
		}

		// Try to update endpoints in the tanzu context as per the endpointUpdateMap
		updateEndpointsInTanzuContext(ctx, endpointUpdateMap)

		// Update the context in the tanzu configuration file
		if err := config.SetContext(ctx, false); err != nil {
			updateSuccess = false
		}
	}

	// if all the contexts are updated successfully, update the flag in the data store
	if updateSuccess {
		_ = datastore.SetDataStoreValue(existingEndpointUpdateVersionKey, &requestedEndpointUpdateVersion)
		log.Info("The CLI contexts have been updated to use the new Tanzu Platform Endpoints.")
	}
}

// isValidContextForEndpointUpdates validates if the specified context is valid tanzu/tmc context or not
func isValidContextForEndpointUpdates(ctx *configtypes.Context) bool {
	if ctx != nil && (ctx.ContextType == configtypes.ContextTypeTanzu || ctx.ContextType == configtypes.ContextTypeTMC) {
		return true
	}
	return false
}

// updateEndpointsInTanzuContext replaces the old endpoint to the new endpoint if the match is found
func updateEndpointsInTanzuContext(ctx *configtypes.Context, endpointUpdateMap map[string]string) {
	for oldEndpoint, newEndpoint := range endpointUpdateMap {
		// Update global endpoint
		if ctx.GlobalOpts != nil {
			ctx.GlobalOpts.Endpoint = strings.Replace(ctx.GlobalOpts.Endpoint, oldEndpoint, newEndpoint, 1)
		}

		// Update cluster endpoint
		if ctx.ClusterOpts != nil {
			ctx.ClusterOpts.Endpoint = strings.Replace(ctx.ClusterOpts.Endpoint, oldEndpoint, newEndpoint, 1)
		}

		// Update Hub Endpoint
		hubEndpoint, exists := ctx.AdditionalMetadata[config.TanzuHubEndpointKey]
		if exists {
			ctx.AdditionalMetadata[config.TanzuHubEndpointKey] = strings.Replace(hubEndpoint.(string), oldEndpoint, newEndpoint, 1)
		}

		// Update TMC endpoint
		tmcEndpoint, exists := ctx.AdditionalMetadata[config.TanzuMissionControlEndpointKey]
		if exists {
			ctx.AdditionalMetadata[config.TanzuMissionControlEndpointKey] = strings.Replace(tmcEndpoint.(string), oldEndpoint, newEndpoint, 1)
		}
	}
}
