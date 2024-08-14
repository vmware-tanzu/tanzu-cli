// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package uaa

import (
	"golang.org/x/term"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
)

const (
	// Tanzu CLI client ID for UAA that has http://127.0.0.1/callback as the
	// only allowed Redirect URI and does not have an associated client secret.
	tanzuCLIClientID     = "tp_cli_app"
	tanzuCLIClientSecret = ""
	defaultListenAddress = "127.0.0.1:0"
	defaultCallbackPath  = "/callback"
)

func getIssuerEndpoints(issuerURL string) common.IssuerEndPoints {
	return common.IssuerEndPoints{
		AuthURL:  issuerURL + "/oauth/authorize",
		TokenURL: issuerURL + "/oauth/token",
	}
}

func TanzuLogin(issuerURL string, opts ...common.LoginOption) (*common.Token, error) {
	issuerEndpoints := getIssuerEndpoints(issuerURL)

	h := common.NewTanzuLoginHandler(issuerURL, issuerEndpoints.AuthURL, issuerEndpoints.TokenURL, tanzuCLIClientID, tanzuCLIClientSecret, defaultListenAddress, defaultCallbackPath, config.UAAIdpType, nil, nil, term.IsTerminal)
	for _, opt := range opts {
		if err := opt(h); err != nil {
			return nil, err
		}
	}

	return h.DoLogin()
}
