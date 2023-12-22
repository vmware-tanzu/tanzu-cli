// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"go.pinniped.dev/pkg/oidcclient/pkce"
	"go.pinniped.dev/pkg/oidcclient/state"
	"golang.org/x/oauth2"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const (
	// Tanzu CLI client ID that has http://127.0.0.1:5400/callback as the
	// only allowed Redirect URI and does not have an associated client secret.
	tanzuCLIClientID     = "tanzu-cli-client-id"
	defaultListenAddress = "127.0.0.1:0"
	defaultCallbackPath  = "/callback"
)

// orgInfo to decode the CSP organization API response
type orgInfo struct {
	Name string `json:"displayName"`
}

type cspLoginHandler struct {
	tokenExchange         context.Context
	tokenExchangeComplete context.CancelFunc
	issuer                string
	clientID              string
	listenAddr            string
	callbackPath          string
	oauthConfig           *oauth2.Config
	pkceCodePair          pkce.Code
	state                 state.State
	token                 *oauth2.Token
	refreshToken          string
	orgID                 string
}

// LoginOption is an optional configuration for Login().
type LoginOption func(*cspLoginHandler) error

// WithRefreshToken causes the login to use refresh token instead of interactive login.
// If the refresh token is expired or invalid, the interactive login will kick in
func WithRefreshToken(refreshToken string) LoginOption {
	return func(h *cspLoginHandler) error {
		h.refreshToken = refreshToken
		return nil
	}
}

// WithOrgID causes the login to given Organization.
func WithOrgID(orgID string) LoginOption {
	return func(h *cspLoginHandler) error {
		h.orgID = orgID
		return nil
	}
}

func (h *cspLoginHandler) handleTokenRefresh() (*Token, error) {
	refreshedToken, err := h.oauthConfig.TokenSource(context.TODO(), &oauth2.Token{RefreshToken: h.refreshToken}).Token()
	if err != nil {
		return nil, err
	}
	return &Token{
		IDToken:      refreshedToken.Extra("id_token").(string),
		AccessToken:  refreshedToken.AccessToken,
		RefreshToken: refreshedToken.RefreshToken,
		ExpiresIn:    int64(time.Until(refreshedToken.Expiry).Seconds()),
		TokenType:    IDTokenType,
	}, nil
}

func TanzuLogin(issuerURL string, opts ...LoginOption) (*Token, error) {
	h := &cspLoginHandler{
		issuer:       issuerURL,
		clientID:     tanzuCLIClientID,
		listenAddr:   defaultListenAddress,
		callbackPath: defaultCallbackPath,
	}
	h.oauthConfig = &oauth2.Config{
		RedirectURL: (&url.URL{Scheme: "http", Host: h.listenAddr, Path: h.callbackPath}).String(),
		ClientID:    h.clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  KnownIssuers[issuerURL].AuthURL,
			TokenURL: KnownIssuers[issuerURL].TokenURL,
		},
	}
	for _, opt := range opts {
		if err := opt(h); err != nil {
			return nil, err
		}
	}

	if h.refreshToken != "" {
		// handle token refresh
		token, err := h.handleTokenRefresh()
		if err == nil {
			return token, nil
		}
		// If refresh token fails, proceed with login flow through the browser
	}

	return h.handleBrowserLogin()
}

func (h *cspLoginHandler) handleBrowserLogin() (*Token, error) {
	var err error
	if h.pkceCodePair, err = pkce.Generate(); err != nil {
		return nil, errors.Wrapf(err, "failed to generate PKCE code")
	}
	h.state, err = state.Generate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate state parameter")
	}

	shutdown, err := h.runLocalListener()
	if err != nil {
		return nil, err
	}
	defer shutdown()

	authCodeURL := h.getAuthCodeURL()
	// TODO(prkalle): To update the logic to address the scenario where users attempts to login to tanzu
	// in a terminal based hosts(no browser support). The plan is to show the auth URL in the terminal and
	// request user to open the auth URL in the browser and copy the auth Code to CLI terminal and let CLI complete
	// login flow.
	log.Info("Opening the browser window to complete the login\n")
	err = browser.OpenURL(authCodeURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open the browser for login")
	}

	// wait for the token exchange to be completed
	<-h.tokenExchange.Done()

	if h.token == nil || h.token.Extra("id_token").(string) == "" {
		return nil, errors.Errorf("token issuer %s did not return expected tokens", h.issuer)
	}
	return &Token{
		IDToken:      h.token.Extra("id_token").(string),
		AccessToken:  h.token.AccessToken,
		RefreshToken: h.token.RefreshToken,
		ExpiresIn:    int64(time.Until(h.token.Expiry).Seconds()),
		TokenType:    IDTokenType,
	}, nil
}

// runLocalListener is a blocking function call that starts a local listener
// to handle auth-code flow callback to perform token exchange.
func (h *cspLoginHandler) runLocalListener() (func(), error) {
	listener, err := net.Listen("tcp", h.listenAddr)
	if err != nil {
		return func() {}, errors.Wrap(err, "could not open callback listener")
	}
	// update the redirect URL with the random port allocated
	redirectURI := url.URL{Scheme: "http", Host: listener.Addr().String(), Path: h.callbackPath}
	h.oauthConfig.RedirectURL = redirectURI.String()

	mux := http.NewServeMux()
	mux.HandleFunc(h.callbackPath, h.callbackHandler)

	h.tokenExchange, h.tokenExchangeComplete = context.WithCancel(context.TODO())

	srv := http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	// run a go routine to catch interrupt signals from the CLI to
	// gracefully shut down the local listener
	go h.interruptHandler()
	go func() { _ = srv.Serve(listener) }()
	return func() {
		_ = srv.Shutdown(h.tokenExchange)
	}, nil
}

func (h *cspLoginHandler) callbackHandler(w http.ResponseWriter, r *http.Request) {
	// token exchange should be complete once this callback handler completes execution.
	defer h.tokenExchangeComplete()
	code := r.URL.Query().Get("code")
	if code == "" {
		errMsg := fmt.Sprintf("[code] query params is required, URL %s did not have this query parameters", html.EscapeString(r.URL.String()))
		http.Error(w, errMsg, http.StatusBadRequest)
		log.Info(errMsg)
		return
	}

	// Validate OAuth2 state and fail if it's incorrect (to block CSRF).
	if err := h.state.Validate(r.URL.Query().Get("state")); err != nil {
		http.Error(w, "missing or invalid state parameter", http.StatusForbidden)
		return
	}

	var err error
	h.token, err = h.oauthConfig.Exchange(h.tokenExchange, code, h.pkceCodePair.Verifier())
	if err != nil {
		errMsg := fmt.Sprintf("failed to exchange auth code for oauth tokens, err=%v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Info(errMsg)
		return
	}
	fmt.Fprint(w, "You have successfully logged in! You can now safely close this window")
}

func (h *cspLoginHandler) interruptHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	for range ch {
		h.tokenExchangeComplete()
		log.Fatal(nil, "login flow interrupted")
	}
}

func (h *cspLoginHandler) getAuthCodeURL() string {
	opts := []oauth2.AuthCodeOption{
		h.pkceCodePair.Challenge(),
		h.pkceCodePair.Method(),
	}
	if h.orgID != "" {
		opts = append(opts, oauth2.SetAuthURLParam("orgId", h.orgID))
	}

	return h.oauthConfig.AuthCodeURL(h.state.String(), opts...)
}

// GetOrgNameFromOrgID fetches CSP Org Name given the Organization ID.
func GetOrgNameFromOrgID(orgID, accessToken, issuer string) (string, error) {
	apiURL := fmt.Sprintf("%s/orgs/%s", issuer, orgID)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", apiURL, http.NoBody)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpRestClient.Do(req)
	if err != nil {
		return "", errors.WithMessage(err, "failed to obtain the CSP organization information")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", errors.Errorf("failed to obtain the CSP organization information: %s", string(body))
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	org := orgInfo{}
	if err = json.Unmarshal(body, &org); err != nil {
		return "", errors.Wrap(err, "could not unmarshal CSP organization information")
	}

	return org.Name, nil
}
