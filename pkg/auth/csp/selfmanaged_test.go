// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/stretchr/testify/assert"
)

func TestGetAccessTokenFromSelfManagedIDP_ValidRefreshToken(t *testing.T) {
	assert := assert.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ACCESS_TOKEN",  "scope": "user", "token_type": "bearer", "refresh_token": "NEW_REFRESH_TOKEN", "id_token": "ID_TOKEN"}`))
	}))
	defer ts.Close()
	refreshToken := "OLD_REFRESH_TOKEN" //nolint:gosec
	token, err := GetAccessTokenFromSelfManagedIDP(refreshToken, ts.URL)
	assert.Nil(err)
	assert.Equal(token.RefreshToken, "NEW_REFRESH_TOKEN")
	assert.Equal(token.AccessToken, "ACCESS_TOKEN")
	assert.Equal(token.IDToken, "ID_TOKEN")
	assert.Equal(token.Scope, loginScopes)
	assert.Equal(token.TokenType, "id_token")
}

func TestGetAuthCodeURL_validResponse(t *testing.T) {
	assert := assert.New(t)
	fakeIssuerURL := "https://fake.issuer.com"
	sharedOauthConfig = &oauth2.Config{
		RedirectURL:  redirectURL,
		ClientID:     pinnipedCLIClientID,
		ClientSecret: "",
		Scopes:       []string{"openid", "offline_access", "username", "groups"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/%s", fakeIssuerURL, authorizationEndpointSuffix),
			TokenURL: fmt.Sprintf("%s/%s", fakeIssuerURL, tokenEndpointSuffix),
		},
	}
	url, err := getAuthCodeURL()
	assert.Nil(err)
	assert.Condition(func() bool { return len(url) != 0 }, "Auth code URL shouldn't be empty")
	if !strings.HasPrefix(url, "https://fake.issuer.com/oauth2/authorize?client_id=pinniped-cli&code_challenge=") {
		t.Errorf("'%s' is expected to have prefix '%s' ", url, "https://fake.issuer.com/oauth2/authorize?client_id=pinniped-cli&code_challenge=")
	}
	assert.Contains(url, "code_challenge_method=S256")
}
