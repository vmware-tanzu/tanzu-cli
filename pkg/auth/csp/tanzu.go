// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"bufio"
	"bytes"
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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"go.pinniped.dev/pkg/oidcclient/pkce"
	"go.pinniped.dev/pkg/oidcclient/state"
	"golang.org/x/oauth2"
	"golang.org/x/term"

	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	cliconfig "github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const (
	// Tanzu CLI client ID that has http://127.0.0.1/callback as the
	// only allowed Redirect URI and does not have an associated client secret.
	tanzuCLIClientID     = "tanzu-cli-client-id"
	defaultListenAddress = "127.0.0.1:0"
	defaultCallbackPath  = "/callback"

	centralConfigTanzuHubMetadata = "cli.core.tanzu_hub_metadata"
	defaultCSPDisplayName         = "Tanzu Platform"
	defaultCSPProductIdentifier   = "TANZU-SAAS"
)

// stdin returns the file descriptor for stdin as an int.
func stdin() int { return int(os.Stdin.Fd()) }

// orgInfo to decode the CSP organization API response
type orgInfo struct {
	Name string `json:"displayName"`
}

// tanzuHubMetadata to parse the hub metadata from central config
type tanzuHubMetadata struct {
	CSPProductIdentifier string `json:"cspProductIdentifier" yaml:"cspProductIdentifier"`
	CSPDisplayName       string `json:"cspDisplayName" yaml:"cspDisplayName"`
	EndpointProduction   string `json:"endpointProduction" yaml:"endpointProduction"`
	EndpointStaging      string `json:"endpointStaging" yaml:"endpointStaging"`
	UseCentralConfig     bool   `json:"useCentralConfig" yaml:"useCentralConfig"`
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
	orgName               string
	promptForValue        func(ctx context.Context, promptLabel string, out io.Writer) (string, error)
	isTTY                 func(int) bool
	callbackHandlerMutex  sync.Mutex
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

// WithListenerPort specifies a TCP listener port on localhost, which will be used for the redirect_uri and to handle the
// authorization code callback. By default, a random high port will be chosen which requires the authorization server
// to support wildcard port numbers as described by https://tools.ietf.org/html/rfc8252#section-7.3:
// Being able to designate the listener port might be advantages under some circumstances
// (e.g. for determining what to port-forward from the host where the web browser is available)
func WithListenerPort(port uint16) LoginOption {
	return func(h *cspLoginHandler) error {
		h.listenAddr = net.JoinHostPort("127.0.0.1", fmt.Sprint(port))
		return nil
	}
}

// WithListenerPortFromEnv sets the TCP listener port on localhost based on the value of the specified environment variable,
// which will be used for the redirect_uri and to handle the authorization code callback.
// By default, a random high port will be chosen which requires the authorization server
// to support wildcard port numbers as described by https://tools.ietf.org/html/rfc8252#section-7.3:
// Being able to designate the listener port might be advantages under some circumstances
// (e.g. for determining what to port-forward from the host where the web browser is available)
func WithListenerPortFromEnv(envVarName string) LoginOption {
	return func(h *cspLoginHandler) error {
		portStr := os.Getenv(envVarName)
		if portStr != "" {
			port, err := strconv.ParseUint(portStr, 10, 16)
			if err != nil {
				return errors.Wrapf(err, "failed to parse %s as uint16", envVarName)
			}
			h.listenAddr = net.JoinHostPort("127.0.0.1", fmt.Sprint(port))
		}
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
		issuer:         issuerURL,
		clientID:       tanzuCLIClientID,
		listenAddr:     defaultListenAddress,
		callbackPath:   defaultCallbackPath,
		promptForValue: promptForValue,
		isTTY:          term.IsTerminal,
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

	listener, err := net.Listen("tcp", h.listenAddr)
	if err != nil {
		log.Warning(errors.Wrap(err, "could not open callback listener").Error())
	}

	// If the listener failed to start and stdin is not a TTY, then we have no hope of succeeding,
	// since we won't be able to receive the web callback, and we can't prompt for the manual auth code, so return error
	if listener == nil && !h.isTTY(stdin()) {
		return nil, fmt.Errorf("login failed: must have either a localhost listener or stdin must be a TTY")
	}

	// update the redirect URL with the port allocated/used
	redirectURI := url.URL{Scheme: "http", Host: listener.Addr().String(), Path: h.callbackPath}
	h.oauthConfig.RedirectURL = redirectURI.String()

	h.tokenExchange, h.tokenExchangeComplete = context.WithCancel(context.TODO())

	shutdown := h.runLocalListener(listener)
	defer shutdown()

	authCodeURL := h.getAuthCodeURL()
	log.Info("Opening the browser window to complete the login\n")
	err = browser.OpenURL(authCodeURL)
	if err != nil {
		log.Warning(fmt.Sprintf("failed to open the browser for login:%v", err.Error()))
	}

	// Prompt the user to visit the authorize URL, and to paste a manually-copied auth code (if possible).
	cleanupPrompt := h.promptAndLoginWithAuthCode(h.tokenExchange, authCodeURL)
	defer cleanupPrompt()

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
func (h *cspLoginHandler) runLocalListener(listener net.Listener) func() {
	mux := http.NewServeMux()
	mux.HandleFunc(h.callbackPath, h.callbackHandler)

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
	}
}

func (h *cspLoginHandler) callbackHandler(w http.ResponseWriter, r *http.Request) {
	// Lock the mutex to ensure only one request can access/update shared state at a time
	// Note: This should be remote corner case, but we have seen cases where Chrome browser redirects
	// and make 2 back to back requests with the same URL before the local server is exited.
	// In such scenario, the second request would be blocked till the prior request is finished
	// and then gets unblocked to check if the prior request already acquired token, if so just
	// return with empty message else let the request go through for token exchange and return
	// the response which would fail anyway(context would be canceled for second request)
	h.callbackHandlerMutex.Lock()
	defer h.callbackHandlerMutex.Unlock()

	// Check if token is already set by prior request
	if h.token != nil {
		return
	}
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
	h.token, err = h.getTokenUsingAuthCode(h.tokenExchange, code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// best effort: get the organization name to show in the browser
	h.orgName, _ = h.getOrganizationName()
	printSuccessMessage(w, h.orgName)
}

func printSuccessMessage(w http.ResponseWriter, orgName string) {
	msg := "You have successfully logged in!\n\nYou can now safely close this window"
	if orgName != "" {
		msg = fmt.Sprintf("You have successfully logged into '%s' organization!\n\nYou can now safely close this window", orgName)
	}
	fmt.Fprint(w, msg)
}

func (h *cspLoginHandler) getOrganizationName() (string, error) {
	claims, err := ParseToken(&oauth2.Token{AccessToken: h.token.AccessToken})
	if err != nil {
		return "", errors.Wrap(err, "failed to parse the token")
	}
	orgName, err := GetOrgNameFromOrgID(claims.OrgID, h.token.AccessToken, h.issuer)
	if err != nil {
		return "", err
	}
	return orgName, nil
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

func (h *cspLoginHandler) promptAndLoginWithAuthCode(ctx context.Context, authorizeURL string) func() {
	_, _ = fmt.Fprintf(os.Stderr, "Log in by visiting this link:\n\n    %s\n\n", authorizeURL)

	// If stdin is not a TTY, return, as we have no way of reading it.
	if !h.isTTY(stdin()) {
		return func() {}
	}

	// Launch the manual auth code prompt in a background goroutine, which will be canceled
	// if the parent context is canceled (when the login succeeds or user interrupted).
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			// Always emit a newline so the kubectl output is visually separated from the login prompts.
			_, _ = fmt.Fprintln(os.Stderr)

			h.tokenExchangeComplete()
			wg.Done()
		}()
		code, err := h.promptForValue(ctx, "    Optionally, paste your authorization code: ", os.Stderr)
		if err != nil {
			// Print a visual marker to show the prompt is no longer waiting for user input, plus a trailing
			// newline that simulates the user having pressed "enter".
			_, _ = fmt.Fprint(os.Stderr, "[...]\n")
			if !errors.Is(err, ctx.Err()) {
				log.Info(fmt.Sprintf("failed to prompt for manual authorization code: %v", err))
			}
			return
		}

		// When a code is pasted, redeem it for a token
		token, _ := h.getTokenUsingAuthCode(ctx, code)
		h.token = token
	}()
	return wg.Wait
}

func (h *cspLoginHandler) getTokenUsingAuthCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := h.oauthConfig.Exchange(ctx, code, h.pkceCodePair.Verifier())
	if err != nil {
		errMsg := fmt.Sprintf("failed to exchange auth code for OAuth tokens, err=%v", err)
		log.Info(errMsg)
		return nil, errors.New(errMsg)
	}
	return token, nil
}

func promptForValue(ctx context.Context, promptLabel string, out io.Writer) (string, error) {
	if !term.IsTerminal(stdin()) {
		return "", errors.New("stdin is not connected to a terminal")
	}
	_, err := fmt.Fprint(out, promptLabel)
	if err != nil {
		return "", fmt.Errorf("could not print prompt to stderr: %w", err)
	}

	type readResult struct {
		text string
		err  error
	}
	readResults := make(chan readResult)
	go func() {
		text, err := bufio.NewReader(os.Stdin).ReadString('\n')
		readResults <- readResult{text, err}
		close(readResults)
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-readResults:
		return strings.TrimSpace(r.text), r.err
	}
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

// GetTanzuHubEndpoint retrieves Tanzu Hub Endpoint through the CSP API or through Central Config as fallback
func GetTanzuHubEndpoint(orgID, accessToken string, useStagingIssuer bool) (string, error) {
	var errCSP error
	var endpoint string

	hubMetadata := getTanzuHubMetadataFromCentralConfig()
	if hubMetadata.CSPDisplayName == "" {
		hubMetadata.CSPDisplayName = defaultCSPDisplayName
	}
	if hubMetadata.CSPProductIdentifier == "" {
		hubMetadata.CSPProductIdentifier = defaultCSPProductIdentifier
	}

	// If feature-flag to get endpoint from central config is not configured
	// try to use CSP api to get the endpoint
	if !hubMetadata.UseCentralConfig {
		endpoint, errCSP = getTanzuHubEndpointFromCSP(hubMetadata, orgID, accessToken, useStagingIssuer)
	}

	// If the endpoint is empty or we got error while getting endpoint from CSP
	// try to get the endpoint from central configuration with hubMetadata
	if endpoint == "" || errCSP != nil {
		endpoint = hubMetadata.EndpointProduction
		if useStagingIssuer {
			endpoint = hubMetadata.EndpointStaging
		}
	}

	// If endpoint is still empty return error
	if endpoint == "" {
		return "", errCSP
	}

	return endpoint, nil
}

// getTanzuHubEndpointFromCSP retrieves Tanzu Hub Endpoint through the CSP API
func getTanzuHubEndpointFromCSP(metadata tanzuHubMetadata, orgID, accessToken string, useStagingIssuer bool) (string, error) {
	// CSPServiceURLs stores the CSP service URL information
	type CSPServiceURLs struct {
		ServiceHome string `json:"serviceHome"`
	}

	// CSPService stores the CSP service details
	type CSPService struct {
		DisplayName         string         `json:"displayName"`
		ProductIdentifier   string         `json:"productIdentifier"`
		ServiceDefinitionID string         `json:"serviceDefinitionId"`
		ServiceUrls         CSPServiceURLs `json:"serviceUrls"`
	}

	// CSPServices stores the CSP services list
	type CSPServices struct {
		ServicesList []CSPService `json:"servicesList"`
	}

	endpoint := "https://console.cloud.vmware.com"
	if useStagingIssuer {
		endpoint = "https://console-stg.cloud.vmware.com"
	}
	api := fmt.Sprintf("%s/csp/gateway/slc/api/v2/ui/definitions/?orgId=%s", endpoint, orgID)

	data := url.Values{}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", api, bytes.NewBufferString(data.Encode()))
	req.Header.Set("authorization", "Bearer "+accessToken)

	resp, err := httpRestClient.Do(req)
	if err != nil {
		return "", errors.WithMessage(err, "Failed to obtain available services for the specified organization")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", errors.Errorf("Failed to obtain available services for the specified organization. %s", string(body))
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	services := CSPServices{}

	if err = json.Unmarshal(body, &services); err != nil {
		return "", errors.Wrap(err, "could not unmarshal the services response")
	}

	for _, s := range services.ServicesList {
		if s.ProductIdentifier == metadata.CSPProductIdentifier && strings.Contains(s.DisplayName, metadata.CSPDisplayName) {
			// Remove `www.` if present from the endpoint. Because when invoking directly through API it does not work
			tanzuHubEndpoint := strings.Replace(s.ServiceUrls.ServiceHome, "www.", "", 1)
			return tanzuHubEndpoint, nil
		}
	}

	return "", errors.Errorf("could not find '%s' service associated with the specified organization", metadata.CSPDisplayName)
}

// getTanzuHubMetadataFromCentralConfig gets the tanzu hub metadata from central config as best effort
// If cannot get the data from central config it will set few default CSP config before returning the object
func getTanzuHubMetadataFromCentralConfig() tanzuHubMetadata {
	hubMetadata := tanzuHubMetadata{}

	// We will get the central configuration from the default discovery source
	discoverySource, err := config.GetCLIDiscoverySource(cliconfig.DefaultStandaloneDiscoveryName)
	if err != nil {
		return hubMetadata
	}
	centralConfigReader := centralconfig.NewCentralConfigReader(discoverySource)

	// Get the tanzu hub metadata
	_ = centralConfigReader.GetCentralConfigEntry(centralConfigTanzuHubMetadata, &hubMetadata)
	return hubMetadata
}
