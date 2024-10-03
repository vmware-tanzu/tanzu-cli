// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package uaa provide functionality needed to interfact with UAA OAuth provider
package uaa

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/interfaces"
)

var (
	httpRestClient interfaces.HTTPClient
)

// GetAccessTokenFromAPIToken fetches access token using the API-token.
func GetAccessTokenFromAPIToken(apiToken, uaaEndpoint, endpointCACertPath string, skipTLSVerify bool) (*common.Token, error) {
	tokenURL := getIssuerEndpoints(uaaEndpoint).TokenURL
	data := url.Values{}
	data.Set("refresh_token", apiToken)
	data.Set("client_id", GetAlternateClientID())
	data.Set("grant_type", "refresh_token")

	req, _ := http.NewRequestWithContext(context.Background(), "POST", tokenURL, bytes.NewBufferString(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if httpRestClient == nil {
		tlsConfig := common.GetTLSConfig(uaaEndpoint, endpointCACertPath, skipTLSVerify)
		if tlsConfig == nil {
			return nil, errors.New("unable to set up tls config")
		}

		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = tlsConfig
		httpRestClient = &http.Client{Transport: tr}
	}

	resp, err := httpRestClient.Do(req)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to obtain access token. Please provide valid API token")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("Failed to obtain access token. Please provide valid API token -- %s", string(body))
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	token := common.Token{}

	if err = json.Unmarshal(body, &token); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal auth token")
	}

	return &token, nil
}

// GetTokens fetches the UAA access token
func GetTokens(refreshOrAPIToken, _, issuer, tokenType string) (*common.Token, error) {
	clientID := tanzuCLIClientID
	if tokenType == common.APITokenType {
		clientID = GetAlternateClientID()
	}
	loginOptions := []common.LoginOption{common.WithRefreshToken(refreshOrAPIToken), common.WithListenerPortFromEnv(constants.TanzuCLIOAuthLocalListenerPort), common.WithClientID(clientID)}

	token, err := TanzuLogin(issuer, loginOptions...)
	if err != nil {
		return nil, err
	}

	return token, err
}
