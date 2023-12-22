// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestHandleTokenRefresh(t *testing.T) {
	// Mock HTTP server for token refresh
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "fake-access-token", "refresh_token": "fake-refresh-token", "expires_in": 3600, "id_token": "fake-id-token"}`))
	}))
	defer server.Close()

	// Set OAuth config to use the mock server
	lh := &cspLoginHandler{
		oauthConfig: &oauth2.Config{
			Endpoint: oauth2.Endpoint{
				TokenURL: server.URL,
			},
		},
		refreshToken: "fake-refresh-token",
	}

	token, err := lh.handleTokenRefresh()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if token == nil {
		t.Error("Expected a non-nil token, got nil")
	}
	if token != nil {
		assert.Equal(t, token.AccessToken, "fake-access-token")
		assert.Equal(t, token.RefreshToken, "fake-refresh-token")
		assert.Equal(t, token.TokenType, "id-token")
		assert.Equal(t, token.IDToken, "fake-id-token")
		assert.Equal(t, token.ExpiresIn, int64(3599))
	}
}

func TestGetOrgNameFromOrgID(t *testing.T) {
	// Mock HTTP server for org name request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/orgs/org123" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"displayName": "TestOrg"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Mock HTTP client to use the server
	httpRestClient = &http.Client{
		Transport: http.DefaultTransport,
	}

	// Test the success path
	orgName, err := GetOrgNameFromOrgID("org123", "access123", server.URL)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if orgName != "TestOrg" {
		t.Errorf("Expected org name 'TestOrg', got %s", orgName)
	}

	// Test the invalid org
	_, err = GetOrgNameFromOrgID("InvalidOrg", "access123", server.URL)
	assert.ErrorContains(t, err, "failed to obtain the CSP organization information")
}

func TestGetAuthCodeURL_validResponse(t *testing.T) {
	assert := assert.New(t)
	var err error
	fakeIssuerURL := "https://fake.issuer.com"
	h := &cspLoginHandler{
		oauthConfig: &oauth2.Config{
			RedirectURL: fakeIssuerURL,
			ClientID:    tanzuCLIClientID,
			Endpoint: oauth2.Endpoint{
				AuthURL:  fakeIssuerURL + "/oauth",
				TokenURL: fakeIssuerURL + "/token",
			},
		},
		listenAddr:   "127.0.0.1:5400",
		callbackPath: "/callback",
	}

	// Test the AuthCode URL without OrgID
	gotAuthCodeURL := h.getAuthCodeURL()
	assert.True(len(gotAuthCodeURL) != 0, "Auth code URL shouldn't be empty")
	u, err := url.Parse(gotAuthCodeURL)
	assert.NoError(err)
	assert.Equal(u.Host, "fake.issuer.com")
	assert.Equal(u.Path, "/oauth")
	assert.Equal(u.Query().Get("client_id"), tanzuCLIClientID)
	assert.Equal(u.Query().Get("code_challenge_method"), "S256")
	assert.False(u.Query().Has("orgId"))

	// Test the AuthCode URL with OrgID
	h.orgID = "fake-org-id"
	gotAuthCodeURL = h.getAuthCodeURL()
	assert.True(len(gotAuthCodeURL) != 0, "Auth code URL shouldn't be empty")
	u, err = url.Parse(gotAuthCodeURL)
	assert.NoError(err)
	assert.Equal(u.Host, "fake.issuer.com")
	assert.Equal(u.Path, "/oauth")
	assert.Equal(u.Query().Get("client_id"), tanzuCLIClientID)
	assert.Equal(u.Query().Get("orgId"), "fake-org-id")
	assert.Equal(u.Query().Get("code_challenge_method"), "S256")
}
