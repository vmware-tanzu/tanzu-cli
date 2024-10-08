// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

const (
	testTanzuCLIClientID = "test-tanzu-cli-client-id" //nolint:gosec
	fakeIssuerURL        = "https://fake.issuer.com"
)

func TestHandleTokenRefresh(t *testing.T) {
	assert := assert.New(t)

	// Mock HTTP server for token refresh
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "fake-access-token", "refresh_token": "fake-refresh-token", "expires_in": 3600, "id_token": "fake-id-token"}`))
	}))
	defer server.Close()

	// Set OAuth config to use the mock server
	lh := &TanzuLoginHandler{
		oauthConfig: &oauth2.Config{
			Endpoint: oauth2.Endpoint{
				TokenURL: server.URL,
			},
		},
		refreshToken: "fake-refresh-token",
	}

	token, err := lh.getTokenWithRefreshToken()
	assert.Nil(err)
	assert.NotNil(token)
	assert.Equal(token.AccessToken, "fake-access-token")
	assert.Equal(token.RefreshToken, "fake-refresh-token")
	assert.Equal(token.TokenType, "id-token")
	assert.Equal(token.IDToken, "fake-id-token")
	assert.Equal(token.ExpiresIn, int64(3599))
}

// test that login with refresh token completes without triggering browser
// login regardless of whether refresh succeeded or not
func TestLoginWithAPIToken(t *testing.T) {
	assert := assert.New(t)

	// Mock HTTP server for token refresh
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "refresh_token=valid-api-token") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token": "fake-access-token", "refresh_token": "fake-refresh-token", "expires_in": 3600, "id_token": "fake-id-token"}`))
			return
		}
		http.Error(w, "refresh_error", http.StatusBadRequest)
	}))
	defer server.Close()

	lh := &TanzuLoginHandler{
		oauthConfig: &oauth2.Config{
			Endpoint: oauth2.Endpoint{
				TokenURL: server.URL,
			},
		},
		refreshToken:        "valid-api-token",
		suppressInteractive: true,
	}
	token, err := lh.DoLogin()

	assert.Nil(err)
	assert.NotNil(token)
	assert.Equal(token.AccessToken, "fake-access-token")
	assert.Equal(token.RefreshToken, "fake-refresh-token")
	assert.Equal(token.TokenType, "id-token")
	assert.Equal(token.IDToken, "fake-id-token")
	assert.Equal(token.ExpiresIn, int64(3599))

	lh = &TanzuLoginHandler{
		oauthConfig: &oauth2.Config{
			Endpoint: oauth2.Endpoint{
				TokenURL: server.URL,
			},
		},
		refreshToken:        "bad-refresh-token",
		suppressInteractive: true,
	}
	token, err = lh.DoLogin()
	assert.NotNil(err)
	assert.Nil(token)
}

func TestGetAuthCodeURL_validResponse(t *testing.T) {
	assert := assert.New(t)
	var err error
	h := &TanzuLoginHandler{
		oauthConfig: &oauth2.Config{
			RedirectURL: fakeIssuerURL,
			ClientID:    testTanzuCLIClientID,
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
	assert.Equal(u.Query().Get("client_id"), testTanzuCLIClientID)
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
	assert.Equal(u.Query().Get("client_id"), testTanzuCLIClientID)
	assert.Equal(u.Query().Get("orgId"), "fake-org-id")
	assert.Equal(u.Query().Get("code_challenge_method"), "S256")
}

func TestGetTokenUsingAuthCode(t *testing.T) {
	// Mock HTTP server for token refresh
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "code=valid_auth_code") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token": "fake-access-token", "refresh_token": "fake-refresh-token", "expires_in": 3600, "id_token": "fake-id-token"}`))
			return
		}
		http.Error(w, "invalid_auth_code_fake_error", http.StatusBadRequest)
	}))
	defer server.Close()

	// Mock the necessary components
	h := &TanzuLoginHandler{
		oauthConfig: &oauth2.Config{
			RedirectURL: fakeIssuerURL,
			ClientID:    testTanzuCLIClientID,
			Endpoint: oauth2.Endpoint{
				TokenURL: server.URL,
			},
		},
	}

	// Test with a valid auth code
	token, err := h.getTokenUsingAuthCode(context.TODO(), "valid_auth_code")
	if err != nil {
		t.Errorf("getTokenUsingAuthCode returned an unexpected error: %v", err)
	}

	if token == nil || token.Extra("id_token").(string) == "" {
		t.Error("getTokenUsingAuthCode did not return the expected token")
	}
	// Test with an invalid auth code
	_, err = h.getTokenUsingAuthCode(context.TODO(), "invalid_auth_code")
	if err == nil || !strings.Contains(err.Error(), "invalid_auth_code_fake_error") {
		t.Error("getTokenUsingAuthCode did not return an error for an invalid auth code")
	}
}

func TestPromptAndLoginWithAuthCode(t *testing.T) {
	// Mock HTTP server for token refresh
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "code=valid_auth_code") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token": "fake-access-token", "refresh_token": "fake-refresh-token", "expires_in": 3600, "id_token": "fake-id-token"}`))
			return
		}
		http.Error(w, "invalid_auth_code_fake_error", http.StatusBadRequest)
	}))
	defer server.Close()

	// Mock the necessary components
	h := &TanzuLoginHandler{
		tokenExchange:         context.TODO(),
		tokenExchangeComplete: func() {},
		oauthConfig: &oauth2.Config{
			RedirectURL: fakeIssuerURL,
			ClientID:    testTanzuCLIClientID,
			Endpoint: oauth2.Endpoint{
				TokenURL: server.URL,
			},
		},
		promptForValue: func(ctx context.Context, promptLabel string, out io.Writer) (string, error) {
			return "valid_auth_code", nil
		},
		isTTY: func(_ int) bool {
			return true
		},
	}

	// Test user providing valid auth code
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wg := h.promptAndLoginWithAuthCode(ctx, "http://example.com/auth")
	// Wait for the prompt to finish
	wg()

	if h.token == nil || h.token.Extra("id_token").(string) == "" {
		t.Error("promptAndLoginWithAuthCode did not set the token")
	}

	// Test user providing invalid auth code
	h.token = nil
	h.promptForValue = func(ctx context.Context, promptLabel string, out io.Writer) (string, error) {
		return "invalid_auth_code", nil
	}
	wg = h.promptAndLoginWithAuthCode(ctx, "http://example.com/auth")
	wg()

	if h.token != nil {
		t.Error("promptAndLoginWithAuthCode set the token using invalid auth code")
	}

	// Test without a TTY
	h.token = nil
	wg = h.promptAndLoginWithAuthCode(ctx, "http://example.com/auth")
	wg()

	if h.token != nil {
		t.Error("promptAndLoginWithAuthCode set the token without a TTY")
	}

	// Test with context canceled while waiting for user input
	h.token = nil
	h.promptForValue = func(ctx context.Context, promptLabel string, out io.Writer) (string, error) {
		time.Sleep(10 * time.Second)
		return "should_not_reach_here", nil
	}
	ctx, ctxCancel := context.WithCancel(context.Background())
	_ = h.promptAndLoginWithAuthCode(ctx, "http://example.com/auth")
	ctxCancel()

	if h.token != nil {
		t.Error("promptAndLoginWithAuthCode set the token with context canceled while waiting for user input")
	}
}

func TestUpdateCertMap(t *testing.T) {
	assert := assert.New(t)
	testCertHost := "test-host"

	testCases := []struct {
		originalSkipVerify string
		originalCACertData string
		providedCACertData string
		providedSkipVerify bool
		expectError        bool
	}{
		{
			originalSkipVerify: "false",
			originalCACertData: "",
			providedSkipVerify: true,
			providedCACertData: "DUMMYDATA",
		},
		{
			originalSkipVerify: "false",
			originalCACertData: "OLDDUMMYDATA",
			providedSkipVerify: false,
			providedCACertData: "DUMMYDATA",
		},
		{
			originalSkipVerify: "true",
			originalCACertData: "",
			providedSkipVerify: false,
			providedCACertData: "DUMMYDATA",
		},
	}

	for _, tc := range testCases {
		// set up cert entry if needed
		if tc.originalSkipVerify != "false" || tc.originalCACertData != "" {
			cert := &configtypes.Cert{
				Host:           testCertHost,
				CACertData:     tc.originalCACertData,
				SkipCertVerify: tc.originalSkipVerify,
			}
			err := config.SetCert(cert)
			assert.NoError(err)
		}

		lh := &TanzuLoginHandler{
			issuer:        testCertHost,
			tlsSkipVerify: tc.providedSkipVerify,
			caCertData:    tc.providedCACertData,
		}

		lh.updateCertMap()

		cert, err := config.GetCert(testCertHost)
		assert.NoError(err)
		assert.NotNil(cert)
		assert.Equal(cert.CACertData, tc.providedCACertData)
		assert.Equal(cert.SkipCertVerify, strconv.FormatBool(tc.providedSkipVerify))

		err = config.DeleteCert(testCertHost)
		assert.NoError(err)
	}
}
