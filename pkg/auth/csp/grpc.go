// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	grpc_oauth "google.golang.org/grpc/credentials/oauth"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/interfaces"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const (
	mdKeyAuthToken   = "Authorization"
	authTokenPrefix  = "Bearer "
	mdKeyAuthIDToken = "X-User-Id"
	apiToken         = "api-token"
)

var (
	configClientWrapper interfaces.ConfigClientWrapper
)

func init() {
	configClientWrapper = interfaces.NewConfigClient()
}

// WithCredentialDiscovery returns a grpc.CallOption that adds credentials into gRPC calls.
// The credentials are loaded from the auth context found on the machine.
func WithCredentialDiscovery() (grpc.CallOption, error) {
	cfg, err := config.GetClientConfig()
	if err != nil {
		return nil, err
	}
	// Wrap our TokenSource to supply id tokens
	return grpc.PerRPCCredentials(&TokenSource{
		TokenSource: &configSource{cfg},
	}), nil
}

// WithStaticCreds will wrap a static access token into a grpc.CallOption
func WithStaticCreds(accessToken string) grpc.CallOption {
	return grpc.PerRPCCredentials(&grpc_oauth.TokenSource{
		TokenSource: oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken},
		),
	})
}

type configSource struct {
	*configtypes.ClientConfig
}

// Token fetches the token.
func (c *configSource) Token() (*oauth2.Token, error) {
	g, err := c.GetCurrentServer() //nolint:staticcheck // Deprecated
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("current server is nil")
	}
	if !g.IsGlobal() { //nolint:staticcheck // Deprecated
		return nil, fmt.Errorf("trying to fetch token for non global server")
	}
	var expiration time.Time
	token := &Token{}
	if g.GlobalOpts != nil {
		if !IsExpired(g.GlobalOpts.Auth.Expiration) {
			tok := &oauth2.Token{
				AccessToken: g.GlobalOpts.Auth.AccessToken,
				Expiry:      g.GlobalOpts.Auth.Expiration,
			}
			return tok.WithExtra(map[string]interface{}{
				ExtraIDToken: g.GlobalOpts.Auth.IDToken,
			}), nil
		}
		token, err = GetAccessTokenFromAPIToken(g.GlobalOpts.Auth.RefreshToken, ProdIssuer)
		if err != nil {
			return nil, err
		}
		g.GlobalOpts.Auth.Type = apiToken
		expiration = time.Now().Local().Add(time.Second * time.Duration(token.ExpiresIn))
		g.GlobalOpts.Auth.Expiration = expiration
		g.GlobalOpts.Auth.RefreshToken = token.RefreshToken
		g.GlobalOpts.Auth.AccessToken = token.AccessToken
		g.GlobalOpts.Auth.IDToken = token.IDToken
	}

	// Acquire tanzu config lock
	configClientWrapper.AcquireTanzuConfigLock()
	defer configClientWrapper.ReleaseTanzuConfigLock()

	// TODO: Add Read/Write locking mechanism before updating the configuration
	// Currently we are only acquiring the lock while updating the configuration
	if err := configClientWrapper.StoreClientConfig(c.ClientConfig); err != nil {
		return nil, err
	}

	tok := &oauth2.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       expiration,
	}
	return tok.WithExtra(map[string]interface{}{
		ExtraIDToken: token.IDToken,
	}), nil
}

// TokenSource supplies PerRPCCredentials from an oauth2.TokenSource using CSP as the IDP.
// It will supply access token through authorization header and id_token through user-Id header
type TokenSource struct {
	oauth2.TokenSource
}

// GetRequestMetadata gets the request metadata as a map from a TokenSource.
func (ts TokenSource) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{mdKeyAuthToken: authTokenPrefix + " " + token.AccessToken}
	idTok := IDTokenFromTokenSource(token)
	if idTok != "" {
		headers[mdKeyAuthIDToken] = idTok
	}

	return headers, nil
}

// RequireTransportSecurity indicates whether the credentials requires transport security.
func (ts TokenSource) RequireTransportSecurity() bool {
	return true
}

// GetAuthOptsOrExit returns the grpc auth options. If accessToken is not empty it uses it, else it fetches the token
// from the current auth context. If it encounters and error, it exits.
func GetAuthOptsOrExit() grpc.CallOption {
	var authOpts grpc.CallOption
	var err error
	authOpts, err = WithCredentialDiscovery()
	if err != nil {
		log.Fatal(nil, "not logged in. Please retry after logging in")
	}

	return authOpts
}
