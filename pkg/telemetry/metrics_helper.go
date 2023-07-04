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

func getEndpointSHA(plugin *cli.PluginInfo) string {
	cfg, err := configlib.GetClientConfig()
	if err != nil {
		return ""
	}
	curCtxMap, err := cfg.GetAllCurrentContextsMap()
	if err != nil || curCtxMap == nil {
		return ""
	}

	return computeEndpointSHAForContext(curCtxMap, plugin.Target)
}

// computeEndpointSHAForContext computes the endpoint SHA for based on the target type(context type) used
func computeEndpointSHAForContext(curCtx map[configtypes.Target]*configtypes.Context, targetType configtypes.Target) string {
	switch targetType {
	case configtypes.TargetK8s:
		ctx, exists := curCtx[configtypes.TargetK8s]
		if exists {
			// returns SHA256 of the complete context
			ctxBytes, _ := json.Marshal(ctx)
			return hashString(string(ctxBytes))
		}
		return ""

	case configtypes.TargetTMC:
		ctx, exists := curCtx[configtypes.TargetTMC]
		if exists {
			// returns SHA256 of concatenated string of Endpoint and RefreshToken
			// (usually RefreshToken is valid for long duration, hence it is considered for TMC Context uniqueness for telemetry)
			return hashString(ctx.GlobalOpts.Endpoint + ctx.GlobalOpts.Auth.RefreshToken)
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
