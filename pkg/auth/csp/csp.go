// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package csp provide functionality needed to interfact with CSP OAuth provider
package csp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	config "github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/interfaces"
)

const (
	// StgIssuer is the VMware CSP(VCSP) staging issuer.
	StgIssuer = "https://console-stg.cloud.vmware.com/csp/gateway/am/api"

	// ProdIssuer is the VMware CSP(VCSP) issuer.
	ProdIssuer = "https://console.cloud.vmware.com/csp/gateway/am/api"

	// StgIssuerTCSP is the Tanzu CSP (TCSP) staging issuer.
	StgIssuerTCSP = "https://console-stg.tanzu.broadcom.com/csp/gateway/am/api"

	// ProdIssuerTCSP is the Tanzu CSP (TCSP) issuer
	ProdIssuerTCSP = "https://console.tanzu.broadcom.com/csp/gateway/am/api"

	// LocalTPSMIssuer is the test TPSM issuer
	LocalTPSMIssuer = "http://localhost:8080"
)

var (
	// DefaultKnownIssuers are known OAuth2 endpoints in each CSP environment.
	DefaultKnownIssuers = map[string]oauth2.Endpoint{
		StgIssuer: {
			AuthURL:   "https://console-stg.cloud.vmware.com/csp/gateway/discovery",
			TokenURL:  "https://console-stg.cloud.vmware.com/csp/gateway/am/api/auth/authorize",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		ProdIssuer: {
			AuthURL:   "https://console.cloud.vmware.com/csp/gateway/discovery",
			TokenURL:  "https://console.cloud.vmware.com/csp/gateway/am/api/auth/authorize",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		StgIssuerTCSP: {
			AuthURL:   "https://console-stg.tanzu.broadcom.com/csp/gateway/discovery",
			TokenURL:  "https://console-stg.tanzu.broadcom.com/csp/gateway/am/api/auth/authorize",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
		ProdIssuerTCSP: {
			AuthURL:   "https://console.tanzu.broadcom.com/csp/gateway/discovery",
			TokenURL:  "https://console.tanzu.broadcom.com/csp/gateway/am/api/auth/authorize",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}
	httpRestClient interfaces.HTTPClient
)

func init() {
	httpRestClient = http.DefaultClient
}

// GetAccessTokenFromAPIToken fetches CSP access token using the API-token.
func GetAccessTokenFromAPIToken(apiToken, issuer string) (*common.Token, error) {
	api := fmt.Sprintf("%s/auth/api-tokens/authorize", issuer)
	data := url.Values{}
	data.Set("refresh_token", apiToken)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", api, bytes.NewBufferString(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpRestClient.Do(req)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to obtain access token. Please provide valid VMware Cloud Services API-token")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("Failed to obtain access token. Please provide valid VMware Cloud Services API-token -- %s", string(body))
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	token := common.Token{}

	if err = json.Unmarshal(body, &token); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal auth token")
	}

	return &token, nil
}

// GetIssuer returns the appropriate CSP issuer based on the environment.
func GetIssuer(staging bool) string {
	cspMetadata := GetCSPMetadata()
	if staging {
		return cspMetadata.IssuerStaging
	}
	return cspMetadata.IssuerProduction
}

// GetTokens fetches the CSP access token
func GetTokens(refreshOrAPIToken, accessToken, issuer, tokenType string) (*common.Token, error) {
	var token *common.Token
	var err error
	var orgID string
	if tokenType == common.APITokenType {
		token, err = GetAccessTokenFromAPIToken(refreshOrAPIToken, issuer)
		return token, err
	} else if tokenType == common.IDTokenType {
		orgID, err = getOrgIDFromAccessToken(accessToken)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get the CSP OrgID from the existing access token")
		}
		loginOptions := []common.LoginOption{common.WithRefreshToken(refreshOrAPIToken), common.WithOrgID(orgID), common.WithListenerPortFromEnv(constants.TanzuCLIOAuthLocalListenerPort)}
		token, err = TanzuLogin(issuer, loginOptions...)
		if err != nil {
			return nil, err
		}
	}
	return token, err
}

// getOrgIDFromAccessToken fetches the OrgID from the access token which is available in context's auth information
func getOrgIDFromAccessToken(accessToken string) (string, error) {
	token, err := common.ParseToken(&oauth2.Token{AccessToken: accessToken}, config.CSPIdpType)
	if err != nil {
		return "", err
	}
	return token.OrgID, nil
}
