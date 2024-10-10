// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package uaa provide functionality needed to interfact with UAA OAuth provider
package uaa

import (
	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

// GetTokens fetches the UAA access token
func GetTokens(refreshOrAPIToken, _, issuer, tokenType string) (*common.Token, error) {
	clientID := tanzuCLIClientID
	if tokenType == common.APITokenType {
		clientID = GetAlternateClientID()
	}
	loginOptions := []common.LoginOption{common.WithRefreshToken(refreshOrAPIToken), common.WithListenerPortFromEnv(constants.TanzuCLIOAuthLocalListenerPort), common.WithClientIDAndSecret(clientID, GetClientSecret())}
	if tokenType == common.APITokenType {
		loginOptions = append(loginOptions, common.WithSuppressInteractive(true))
	}

	token, err := TanzuLogin(issuer, loginOptions...)
	if err != nil {
		return nil, err
	}

	return token, err
}
