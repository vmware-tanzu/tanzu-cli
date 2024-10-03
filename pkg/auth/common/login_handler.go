// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
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

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// stdin returns the file descriptor for stdin as an int.
func stdin() int { return int(os.Stdin.Fd()) }

type TanzuLoginHandler struct {
	tokenExchange         context.Context
	tokenExchangeComplete context.CancelFunc
	issuer                string
	issuerAuthURL         string
	issuerTokenURL        string
	clientID              string
	clientSecret          string
	listenAddr            string
	callbackPath          string
	oauthConfig           *oauth2.Config
	pkceCodePair          pkce.Code
	state                 state.State
	token                 *oauth2.Token
	refreshToken          string
	orgID                 string
	orgName               string
	orgNameGetter         func(orgID, accessToken, issuer string) (string, error)
	promptForValue        func(ctx context.Context, promptLabel string, out io.Writer) (string, error)
	isTTY                 func(int) bool
	idpType               config.IdpType
	callbackHandlerMutex  sync.Mutex
	tlsSkipVerify         bool
	caCertData            string
}

// LoginOption is an optional configuration for Login().
type LoginOption func(*TanzuLoginHandler) error

func NewTanzuLoginHandler(issuer, issuerAuthURL, issuerTokenURL, clientID, clientSecret, listenAddr, callbackPath string, idpType config.IdpType, orgNameGetter func(orgID, accessToken, issuer string) (string, error), promptForValue func(ctx context.Context, promptLabel string, out io.Writer) (string, error), isTTYFn func(int) bool) *TanzuLoginHandler {
	h := &TanzuLoginHandler{
		issuer:         issuer,
		issuerAuthURL:  issuerAuthURL,
		issuerTokenURL: issuerTokenURL,
		clientID:       clientID,
		clientSecret:   clientSecret,
		listenAddr:     listenAddr,
		callbackPath:   callbackPath,
		idpType:        idpType,
		orgNameGetter:  orgNameGetter,
		promptForValue: promptForValue,
		isTTY:          isTTYFn,
	}

	if promptForValue == nil {
		h.promptForValue = h.defaultPromptForValue
	}

	h.oauthConfig = &oauth2.Config{
		RedirectURL:  (&url.URL{Scheme: "http", Host: listenAddr, Path: callbackPath}).String(),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  issuerAuthURL,
			TokenURL: issuerTokenURL,
		},
	}
	return h
}

// WithRefreshToken causes the login to use refresh token instead of interactive login.
// If the refresh token is expired or invalid, the interactive login will kick in
func WithRefreshToken(refreshToken string) LoginOption {
	return func(h *TanzuLoginHandler) error {
		h.refreshToken = refreshToken
		return nil
	}
}

// WithOrgID causes the login to given Organization.
func WithOrgID(orgID string) LoginOption {
	return func(h *TanzuLoginHandler) error {
		h.orgID = orgID
		return nil
	}
}

// WithCertInfo customizes cert verification information
func WithCertInfo(tlsSkipVerify bool, caCertData string) LoginOption {
	return func(h *TanzuLoginHandler) error {
		h.tlsSkipVerify = tlsSkipVerify
		h.caCertData = caCertData
		return nil
	}
}

// WithClientID specifies a OAuth Client ID to use
func WithClientID(clientID string) LoginOption {
	return func(h *TanzuLoginHandler) error {
		h.clientID = clientID
		if h.oauthConfig != nil {
			h.oauthConfig.ClientID = clientID
		}
		return nil
	}
}

// WithListenerPort specifies a TCP listener port on localhost, which will be used for the redirect_uri and to handle the
// authorization code callback. By default, a random high port will be chosen which requires the authorization server
// to support wildcard port numbers as described by https://tools.ietf.org/html/rfc8252#section-7.3:
// Being able to designate the listener port might be advantages under some circumstances
// (e.g. for determining what to port-forward from the host where the web browser is available)
func WithListenerPort(port uint16) LoginOption {
	return func(h *TanzuLoginHandler) error {
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
	return func(h *TanzuLoginHandler) error {
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

func (h *TanzuLoginHandler) DoLogin() (*Token, error) {
	if h.refreshToken != "" {
		token, err := h.getTokenWithRefreshToken()
		if err == nil {
			return token, nil
		}
	}
	// If refresh token fails, proceed with login flow through the browser
	return h.browserLogin()
}

func (h *TanzuLoginHandler) getTokenWithRefreshToken() (*Token, error) {
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

// Create or update the cert map entry for the issuer if necessary
func (h *TanzuLoginHandler) updateCertMap() {
	// explicitly provided cert info was successfully used to authenticate, so as a
	// best-effort: save them in the cert map for convenience if necessary
	if h.tlsSkipVerify || h.caCertData != "" {
		u, err := url.Parse(h.issuer)
		if err != nil {
			fmt.Printf("Unable to parse issuer %s: %v\n", h.issuer, err)
			return
		}
		host := u.Hostname()
		if host == "" {
			host = h.issuer
		}

		var cert *configtypes.Cert
		found, _ := config.GetCert(host)

		tlsSkipVerifyStr := strconv.FormatBool(h.tlsSkipVerify)
		if found != nil {
			if found.CACertData != h.caCertData || found.SkipCertVerify != tlsSkipVerifyStr {
				cert = &configtypes.Cert{
					Host:           host,
					CACertData:     h.caCertData,
					SkipCertVerify: tlsSkipVerifyStr,
					Insecure:       found.Insecure,
				}
				err = config.SetCert(cert)
				if err != nil {
					log.Infof("Unable to update cert info: %v\n", err)
				}
			}
		} else {
			cert = &configtypes.Cert{
				Host:           host,
				CACertData:     h.caCertData,
				SkipCertVerify: tlsSkipVerifyStr,
			}
			if err = config.SetCert(cert); err != nil {
				log.Infof("Unable to create cert info: %v\n", err)
			}
		}
	}
}

func (h *TanzuLoginHandler) browserLogin() (*Token, error) {
	var err error
	if h.pkceCodePair, err = pkce.Generate(); err != nil {
		return nil, errors.Wrapf(err, "failed to generate PKCE code")
	}
	h.state, err = state.Generate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate state parameter")
	}

	log.V(7).Infof("listening on %v\n", h.listenAddr)
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

	h.updateCertMap()

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
func (h *TanzuLoginHandler) runLocalListener(listener net.Listener) func() {
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

func (h *TanzuLoginHandler) callbackHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *TanzuLoginHandler) getOrganizationName() (string, error) {
	if h.idpType == config.UAAIdpType || h.orgNameGetter == nil {
		return "", nil
	}

	claims, err := ParseToken(&oauth2.Token{AccessToken: h.token.AccessToken}, config.CSPIdpType)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse the token")
	}
	orgName, err := h.orgNameGetter(claims.OrgID, h.token.AccessToken, h.issuer)
	if err != nil {
		return "", err
	}
	return orgName, nil
}

func (h *TanzuLoginHandler) interruptHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	for range ch {
		h.tokenExchangeComplete()
		log.Fatal(nil, "login flow interrupted")
	}
}

func (h *TanzuLoginHandler) getAuthCodeURL() string {
	opts := []oauth2.AuthCodeOption{
		h.pkceCodePair.Challenge(),
		h.pkceCodePair.Method(),
	}
	if h.orgID != "" {
		opts = append(opts, oauth2.SetAuthURLParam("orgId", h.orgID))
	}

	return h.oauthConfig.AuthCodeURL(h.state.String(), opts...)
}

func (h *TanzuLoginHandler) promptAndLoginWithAuthCode(ctx context.Context, authorizeURL string) func() {
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

// Returns custom TLS configuration if explicitly provided to the handler,
// of if persisted cert information associated with the issuer endpoint is found,
// with the provided information taking precedence over persisted information.
func (h *TanzuLoginHandler) getTLSConfig() *tls.Config {
	return GetTLSConfig(h.issuer, h.caCertData, h.tlsSkipVerify)
}

func GetTLSConfig(endpoint, certData string, skipVerify bool) *tls.Config {
	var savedCertData string
	var savedSkipVerify bool

	c, _ := config.GetCert(endpoint)

	if c != nil {
		savedCertData = c.CACertData
		savedSkipVerify, _ = strconv.ParseBool(c.SkipCertVerify)
	}

	if savedSkipVerify || skipVerify {
		//nolint:gosec // skipTLSVerify: true is only possible if the user has explicitly enabled it
		return &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
	}

	caCertData := savedCertData
	if certData != "" {
		caCertData = certData
	}

	if caCertData != "" {
		var pool *x509.CertPool
		var err error

		decodedCACertData, err := base64.StdEncoding.DecodeString(caCertData)
		if err != nil {
			log.Infof("unable to use custom cert for '%s' endpoint. Error: %s", endpoint, err.Error())
			return nil
		}

		pool, err = x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}

		if ok := pool.AppendCertsFromPEM(decodedCACertData); !ok {
			log.Infof("unable to use custom cert for %s endpoint", endpoint)
			return nil
		}
		return &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}

	return nil
}

func (h *TanzuLoginHandler) getTokenUsingAuthCode(ctx context.Context, code string) (*oauth2.Token, error) {
	if h.idpType == config.UAAIdpType {
		tlsConfig := h.getTLSConfig()
		if tlsConfig != nil {
			tr := http.DefaultTransport.(*http.Transport).Clone()
			tr.TLSClientConfig = tlsConfig

			sslcli := &http.Client{Transport: tr}
			ctx = context.WithValue(ctx, oauth2.HTTPClient, sslcli)
		}
	}

	token, err := h.oauthConfig.Exchange(ctx, code, h.pkceCodePair.Verifier())
	if err != nil {
		errString := err.Error()
		errMsg := fmt.Sprintf("failed to exchange auth code for OAuth tokens, err=%s", errString)

		println()
		log.Info(errMsg)
		if strings.Contains(errString, "failed to verify certificate") {
			log.Info("Consider using 'tanzu config cert add' to configure certificate verification settings")
		}
		return nil, err
	}
	return token, nil
}

func (h *TanzuLoginHandler) defaultPromptForValue(ctx context.Context, promptLabel string, out io.Writer) (string, error) {
	// If stdin is not a TTY, return, as we have no way of reading it.
	if !h.isTTY(stdin()) {
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
