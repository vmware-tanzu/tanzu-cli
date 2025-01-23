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
)

func getEndpointSHAWithCtxTypePrefix(plugin *cli.PluginInfo) string {
	cfg, err := configlib.GetClientConfig()
	if err != nil {
		return ""
	}
	curCtxMap, err := cfg.GetAllActiveContextsMap()
	if err != nil || curCtxMap == nil {
		return ""
	}

	return computeEndpointSHAWithCtxTypePrefix(curCtxMap, plugin.Target)
}

// computeEndpointSHAWithCtxTypePrefix computes the endpoint SHA based on the target type(context type) used with context type as prefix
func computeEndpointSHAWithCtxTypePrefix(curCtx map[configtypes.ContextType]*configtypes.Context, targetType configtypes.Target) string {
	switch targetType {
	case configtypes.TargetK8s:
		ctx, exists := curCtx[configtypes.ContextTypeK8s]
		if exists {
			return string(configtypes.ContextTypeK8s) + ":" + computeEndpointSHAForK8sContext(ctx)
		}
		// If Target is k8s and k8s context type is not active, fall back to tanzu context-type
		ctx, exists = curCtx[configtypes.ContextTypeTanzu]
		if exists {
			return string(configtypes.ContextTypeTanzu) + ":" + computeEndpointSHAForTanzuContext(ctx)
		}
		return ""

	case configtypes.TargetTMC:
		ctx, exists := curCtx[configtypes.ContextTypeTMC]
		if exists {
			return string(configtypes.ContextTypeTMC) + ":" + computeEndpointSHAForTMCContext(ctx)
		}
		return ""
	case configtypes.TargetGlobal:
		// If Target is "global" and tanzu context type is active, use it as most plugins in Tanzu Platform are global target plugins
		ctx, exists := curCtx[configtypes.ContextTypeTanzu]
		if exists {
			return string(configtypes.ContextTypeTanzu) + ":" + computeEndpointSHAForTanzuContext(ctx)
		}
		// There could be "global" plugins which could use `k8s` contexts. So get endpoint from k8s context as a fallback
		ctx, exists = curCtx[configtypes.ContextTypeK8s]
		if exists {
			return string(configtypes.ContextTypeK8s) + ":" + computeEndpointSHAForK8sContext(ctx)
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

func computeEndpointSHAForTanzuContext(ctx *configtypes.Context) string {
	// use projectID if not empty, else use projectName
	projectVal := stringValue(ctx.AdditionalMetadata[configlib.ProjectNameKey])
	projectID := stringValue(ctx.AdditionalMetadata[configlib.ProjectIDKey])
	if projectID != "" {
		projectVal = projectID
	}
	endpoint := ""
	if ctx.GlobalOpts != nil {
		endpoint = ctx.GlobalOpts.Endpoint
	}
	// returns SHA256 of concatenated string of Endpoint and orgId/Project/SpaceName
	return hashString(endpoint +
		stringValue(ctx.AdditionalMetadata[configlib.OrgIDKey]) +
		projectVal +
		stringValue(ctx.AdditionalMetadata[configlib.SpaceNameKey]) +
		stringValue(ctx.AdditionalMetadata[configlib.ClusterGroupNameKey]))
}

func computeEndpointSHAForTMCContext(ctx *configtypes.Context) string {
	// returns SHA256 of concatenated string of Endpoint and RefreshToken
	// (usually RefreshToken is valid for long duration, hence it is considered for TMC Context uniqueness for telemetry)
	if ctx.GlobalOpts == nil {
		return ""
	}
	return hashString(ctx.GlobalOpts.Endpoint + ctx.GlobalOpts.Auth.RefreshToken)
}

func computeEndpointSHAForK8sContext(ctx *configtypes.Context) string {
	// returns SHA256 of the complete context
	ctxBytes, _ := json.Marshal(ctx)
	return hashString(string(ctxBytes))
}

func stringValue(val interface{}) string {
	if val == nil {
		return ""
	}
	str, valid := val.(string)
	if !valid {
		return ""
	}
	return str
}
