// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package uaa provide functionality needed to interfact with UAA OAuth provider
package uaa

import (
	"fmt"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

// GetTokens fetches the UAA access token
func GetTokens(refreshOrAPIToken, _, issuer, tokenType string) (*common.Token, error) {
	var token *common.Token
	var err error

	if tokenType == common.APITokenType {
		return nil, fmt.Errorf("api token unsupported")
	} else if tokenType == common.IDTokenType {
		loginOptions := []common.LoginOption{common.WithRefreshToken(refreshOrAPIToken), common.WithListenerPortFromEnv(constants.TanzuCLIOAuthLocalListenerPort)}
		token, err = TanzuLogin(issuer, loginOptions...)
		if err != nil {
			return nil, err
		}
	}
	return token, err
}
