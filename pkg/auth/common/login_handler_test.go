// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

const (
	testTanzuCLIClientID = "test-tanzu-cli-client-id" //nolint:gosec
	fakeIssuerURL        = "https://fake.issuer.com"
)

func TestHandleTokenRefresh(t *testing.T) {
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
