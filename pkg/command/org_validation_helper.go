// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

const centralConfigTanzuApplicationPlatformScopesKey = "cli.core.tanzu_application_platform_scopes"

type tapScope struct {
	Scope string `yaml:"scope" json:"scope"`
}

type tapScopesGetter func() ([]string, error)

// validateTokenForTAPScopes validates if the token claims contains at least one of the Tanzu Platform for Kubernetes scopes listed in
// the central configuration. If the central configuration doesn't have any Tanzu Platform for Kubernetes scopes listed, it will return success.
// It will skip the validation and return success if TANZU_CLI_SKIP_TANZU_CONTEXT_TAP_SCOPES_VALIDATION environment
// variable is set to true
func validateTokenForTAPScopes(claims *common.Claims, scopesGetter tapScopesGetter) (bool, error) {
	if skipTAPScopeValidation, _ := strconv.ParseBool(os.Getenv(constants.SkipTAPScopesValidationOnTanzuContext)); skipTAPScopeValidation {
		return true, nil
	}
	if scopesGetter == nil {
		scopesGetter = getTAPScopesFromCentralConfig
	}
	tapScopes, err := scopesGetter()
	if err != nil {
		log.V(7).Error(err, "error retrieving Tanzu Platform for Kubernetes scopes from the central config")
		return false, errors.Wrap(err, "error retrieving Tanzu Platform for Kubernetes scopes from the central config")
	}

	// validate only if we get the permissions/scopes from central configuration
	if len(tapScopes) == 0 {
		return true, nil
	}

	for _, tapScope := range tapScopes {
		for _, perm := range claims.Permissions {
			tapScopeSuffix := fmt.Sprintf("/%s", tapScope)
			if strings.HasSuffix(perm, tapScopeSuffix) {
				return true, nil
			}
		}
	}

	return false, nil
}

func getTAPScopesFromCentralConfig() ([]string, error) {
	// Get the Tanzu Platform for Kubernetes scopes from the default central configuration
	var tapScopes []tapScope
	err := centralconfig.DefaultCentralConfigReader.GetCentralConfigEntry(centralConfigTanzuApplicationPlatformScopesKey, &tapScopes)
	if err != nil {
		// If the key is not found in the central config, it does not return an error because some central repositories
		// may choose not to have a central config file.
		var keyNotFoundError *centralconfig.KeyNotFoundError
		if errors.As(err, &keyNotFoundError) {
			return nil, nil
		}
		return nil, err
	}
	// extract the scope names
	var scopeNames []string
	for _, ts := range tapScopes {
		scopeNames = append(scopeNames, ts.Scope)
	}
	return scopeNames, nil
}
