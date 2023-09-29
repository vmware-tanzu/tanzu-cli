// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/pkg/errors"
	oidcapi "go.pinniped.dev/generated/latest/apis/supervisor/oidc"
	"go.pinniped.dev/pkg/oidcclient/pkce"
	"go.pinniped.dev/pkg/oidcclient/state"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const (
	tokenEndpointSuffix         = "oauth2/token" //nolint:gosec
	authorizationEndpointSuffix = "oauth2/authorize"
	redirectURL                 = "http://127.0.0.1/callback"
	// pinniped-cli is a special pinniped-supervisor client ID that has http://127.0.0.1/callback as the
	// only allowed Redirect URI and does not have an associated client secret.
	pinnipedCLIClientID = oidcapi.ClientIDPinnipedCLI
	loginScopes         = "openid offline_access username groups"

	// PinnipedSupervisorDomain is the domain name for the pinniped supervisor token issuer that is
	// deployed in TMC self-managed environment to serve as an identity broker.
	PinnipedSupervisorDomain = "pinniped-supervisor"
	// FederationDomainPath is the path in the issuer URL of the federation domain setup to work
	// with the upstream identity provider.
	// TODO(ashisham): finalize what the federation domain deployed in production will look like
	// Prod and non-prod environments can share this path as the DNS zone from which the issuer URL
	// is generated from will be different.
	FederationDomainPath = "provider/pinniped"
)

var (
	// making the context and the cancel function used for token exchange
	// accessible in the callback handler to have the local listener
	// gracefully shutdown once token exchange is completed.
	tokenExchange         context.Context
	tokenExchangeComplete context.CancelFunc
	token                 *oauth2.Token

	// share a common oauth config for the package as this is the only
	// oauth2 config used for the self-managed auth flows.
	sharedOauthConfig *oauth2.Config
	// pkce code instance to generate the challenge and verifier code pair.
	pkceCodePair pkce.Code
)

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	// token exchange should be complete once this callback handler completes execution.
	defer tokenExchangeComplete()
	code := r.URL.Query().Get("code")
	if code == "" {
		errMsg := fmt.Sprintf("[state] query params is required, URL %s did not have this query parameters", html.EscapeString(r.URL.String()))
		http.Error(w, errMsg, http.StatusBadRequest)
		log.Info(errMsg)
		return
	}

	var err error
	token, err = sharedOauthConfig.Exchange(tokenExchange, code, pkceCodePair.Verifier())
	if err != nil {
		errMsg := fmt.Sprintf("failed to exchange auth code for oauth tokens, err=%v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Info(errMsg)
		return
	}
	fmt.Fprint(w, "You have successfully logged in! You can now safely close this window")
}

func interruptHandler(d context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for range c {
		d()
		log.Fatal(nil, "login flow interrupted")
	}
}

// runLocalListener is a blocking function call that starts a local listener
// to handle auth-code flow callback to perform token exchange.
func runLocalListener() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", callbackHandler)
	tokenExchange, tokenExchangeComplete = context.WithCancel(context.TODO())
	//nolint:gosec //G112: Potential Slowloris Attack because ReadHeaderTimeout is not configured in the http.Server (gosec)
	l := http.Server{
		Addr:    "",
		Handler: mux,
	}
	// run a go routine to catch interrupt signals from the CLI to
	// gracefully shutdown the local listener
	go interruptHandler(tokenExchangeComplete)
	// run a go routine to shut down the local listener once token
	// exchange is completed
	go func() {
		<-tokenExchange.Done()
		_ = l.Shutdown(tokenExchange)
	}()
	if err := l.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Wrapf(err, "failed to run a local listener to facilitate login")
	}
	return nil
}

func getAuthCodeURL() (string, error) {
	stateVal, err := state.Generate()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate state parameter")
	}
	opts := []oauth2.AuthCodeOption{
		pkceCodePair.Challenge(),
		pkceCodePair.Method(),
	}

	return sharedOauthConfig.AuthCodeURL(stateVal.String(), opts...), nil
}

func GetAccessTokenFromSelfManagedIDP(refreshToken, issuerURL string) (*Token, error) {
	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()
	sharedOauthConfig = &oauth2.Config{
		RedirectURL:  redirectURL,
		ClientID:     pinnipedCLIClientID,
		ClientSecret: "",
		Scopes:       []string{"openid", "offline_access", "username", "groups"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/%s", issuerURL, authorizationEndpointSuffix),
			TokenURL: fmt.Sprintf("%s/%s", issuerURL, tokenEndpointSuffix),
		},
	}

	if refreshToken != "" {
		refreshedToken, err := sharedOauthConfig.TokenSource(context.TODO(), &oauth2.Token{RefreshToken: refreshToken}).Token()
		if err == nil {
			return &Token{
				IDToken:      refreshedToken.Extra("id_token").(string),
				AccessToken:  refreshedToken.AccessToken,
				RefreshToken: refreshedToken.RefreshToken,
				ExpiresIn:    int64(time.Until(refreshedToken.Expiry).Seconds()),
				Scope:        loginScopes,
				TokenType:    "id_token",
			}, nil
		}
		log.Infof("failed to refresh token, err %v", err)
		// proceed with login flow through the browser
	}
	// set the issuer package variable to be used in the callback handler
	if _, err := url.Parse(issuerURL); err != nil {
		return nil, errors.Errorf("Issuer URL [%s] is not a valid URL", issuerURL)
	}
	var err error
	if pkceCodePair, err = pkce.Generate(); err != nil {
		return nil, errors.Wrapf(err, "failed to generate pkce code pair generator")
	}
	// perform a browser based login flow if no refreshToken was supplied
	// or if token refresh failed.
	g := &errgroup.Group{}
	g.Go(
		runLocalListener,
	)
	authCodeURL, err := getAuthCodeURL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate authcode url for OIDC provider at %s, err= %v", sharedOauthConfig.Endpoint.AuthURL, err)
	}

	fmt.Printf("Please open this URL in a browser window to complete the login\n\t %s\n", authCodeURL)

	if err := g.Wait(); err != nil {
		return nil, err
	}
	if token == nil || token.Extra("id_token").(string) == "" {
		return nil, errors.Errorf("token issuer %s did not return expected tokens", issuerURL)
	}
	return &Token{
		IDToken:      token.Extra("id_token").(string),
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    int64(time.Until(token.Expiry).Seconds()),
		Scope:        loginScopes,
		TokenType:    "id_token",
	}, nil
}
