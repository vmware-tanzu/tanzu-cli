// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package uaa

import (
	"os"
	"strconv"

	"golang.org/x/term"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

const (
	// Tanzu CLI client ID for UAA that has http://127.0.0.1/callback as the
	// only allowed Redirect URI and does not have an associated client secret.
	tanzuCLIClientID = "tp_cli_app"
	// Alternate client ID for UAA associated with a longer refresh token
	// validity. Use this for CLI use cases where it is impractical to
	// interactively reauthenticate once the refresh token expires.
	tanzuCLIClientIDExtended = "tp_cli_app_ext"
	tanzuCLIClientSecret     = ""
	defaultListenAddress     = "127.0.0.1:0"
	defaultCallbackPath      = "/callback"
)

func getIssuerEndpoints(issuerURL string) common.IssuerEndPoints {
	return common.IssuerEndPoints{
		AuthURL:  issuerURL + "/oauth/authorize",
		TokenURL: issuerURL + "/oauth/token",
	}
}

func GetClientSecret() string {
	// Not really used as a secret, but specified in OAuth client to UAA in order
	// to obtain the expected token refresh behavior.
	secret := "tanzu_intentionally_not_a_secret"

	if noClientSecret, _ := strconv.ParseBool(os.Getenv(constants.UAANoClientSecret)); noClientSecret {
		secret = ""
	}
	return secret
}

func GetAlternateClientID() string {
	// Default to use the same client id, even for non-interactive login use cases.
	clientID := tanzuCLIClientID
	if useAlternateClientID, _ := strconv.ParseBool(os.Getenv(constants.UAAUseAlternateClient)); useAlternateClientID {
		// Unless the env var is set
		clientID = tanzuCLIClientIDExtended
	}
	return clientID
}

var TanzuLogin = func(issuerURL string, opts ...common.LoginOption) (*common.Token, error) {
	issuerEndpoints := getIssuerEndpoints(issuerURL)

	h := common.NewTanzuLoginHandler(issuerURL, issuerEndpoints.AuthURL, issuerEndpoints.TokenURL, tanzuCLIClientID, tanzuCLIClientSecret, defaultListenAddress, defaultCallbackPath, config.UAAIdpType, nil, nil, term.IsTerminal)
	for _, opt := range opts {
		if err := opt(h); err != nil {
			return nil, err
		}
	}

	return h.DoLogin()
}
