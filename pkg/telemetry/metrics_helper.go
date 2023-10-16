// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/tae"
)

func getEndpointSHA(plugin *cli.PluginInfo) string {
	cfg, err := configlib.GetClientConfig()
	if err != nil {
		return ""
	}
	curCtxMap, err := cfg.GetAllActiveContextsMap()
	if err != nil || curCtxMap == nil {
		return ""
	}

	return computeEndpointSHAForContext(curCtxMap, plugin.Target)
}

// computeEndpointSHAForContext computes the endpoint SHA for based on the target type(context type) used
func computeEndpointSHAForContext(curCtx map[configtypes.ContextType]*configtypes.Context, targetType configtypes.Target) string {
	switch targetType {
	case configtypes.TargetK8s:
		ctx, exists := curCtx[configtypes.ContextTypeK8s]
		if exists {
			return computeEndpointSHAForK8sContext(ctx)
		}
		// If Target is k8s and k8s context type is not active, fall back to TAE context-type
		ctx, exists = curCtx[configtypes.ContextTypeTAE]
		if exists {
			return computeEndpointSHAForTAEContext(ctx)
		}
		return ""

	case configtypes.TargetTMC:
		ctx, exists := curCtx[configtypes.ContextTypeTMC]
		if exists {
			computeEndpointSHAForTMCContext(ctx)
		}
		return ""
	}
	return ""
}

func hashString(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func computeEndpointSHAForTAEContext(ctx *configtypes.Context) string {
	var orgID, project, space string
	if ctx.AdditionalMetadata[tae.OrgIDKey] != nil {
		orgID = ctx.AdditionalMetadata[tae.ProjectNameKey].(string)
	}
	if ctx.AdditionalMetadata[tae.ProjectNameKey] != nil {
		project = ctx.AdditionalMetadata[tae.ProjectNameKey].(string)
	}
	if ctx.AdditionalMetadata[tae.SpaceNameKey] != nil {
		space = ctx.AdditionalMetadata[tae.SpaceNameKey].(string)
	}
	// returns SHA256 of concatenated string of Endpoint and orgId/ProjectName/SpaceName
	return hashString(ctx.GlobalOpts.Endpoint + orgID + project + space)
}

func computeEndpointSHAForTMCContext(ctx *configtypes.Context) string {
	// returns SHA256 of concatenated string of Endpoint and RefreshToken
	// (usually RefreshToken is valid for long duration, hence it is considered for TMC Context uniqueness for telemetry)
	return hashString(ctx.GlobalOpts.Endpoint + ctx.GlobalOpts.Auth.RefreshToken)
}

func computeEndpointSHAForK8sContext(ctx *configtypes.Context) string {
	// returns SHA256 of the complete context
	ctxBytes, _ := json.Marshal(ctx)
	return hashString(string(ctxBytes))
}
