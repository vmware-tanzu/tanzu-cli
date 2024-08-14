// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package common provide functionality needed by OAuth based clients
package common

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/oauth2"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const (
	extraIDToken = "id_token"
)

const (
	APITokenType   = "api-token"
	IDTokenType    = "id-token"
	ContextTimeout = 60 * time.Second

	ClaimsContext     = "context_name"
	ClaimsPermissions = "perms"
	ClaimsScopes      = "scope"
	ClaimsCspUserName = "username"
	ClaimsUaaUserName = "user_name"
)

// Token is a useful struct for storing attributes of a context.
type Token struct {
	// IDToken from OIDC.
	IDToken string `json:"id_token"`

	// TokenType is the type of token.
	// Ex: id-token, api-token
	TokenType string `json:"token_type"`

	// ExpiresIn is expiration in seconds.
	ExpiresIn int64 `json:"expires_in"`

	// Scope of the token.
	// Ex: "openid offline_access username groups"
	Scope string `json:"scope"`

	// AccessToken.
	AccessToken string `json:"access_token"`

	// RefreshToken for use with Refresh Token grant.
	RefreshToken string `json:"refresh_token"`
}

// Claims are the jwt claims.
type Claims struct {
	Username    string
	Permissions []string
	OrgID       string
	Raw         map[string]interface{}
}

type IssuerEndPoints struct {
	AuthURL  string `json:"authURL" yaml:"authURL"`
	TokenURL string `json:"tokenURL" yaml:"tokenURL"`
}

// GetToken fetches the token.
func GetToken(g *types.GlobalServerAuth, tokenGetter func(refreshOrAPIToken, accessToken, issuer, tokenType string) (*Token, error), idpType config.IdpType) (*oauth2.Token, error) {
	var err error

	if !IsExpired(g.Expiration) {
		tok := &oauth2.Token{
			RefreshToken: g.RefreshToken,
			AccessToken:  g.AccessToken,
			Expiry:       g.Expiration,
		}
		tok = tok.WithExtra(map[string]interface{}{
			extraIDToken: g.IDToken,
		})
		return tok, nil
	}
	var token *Token

	token, err = tokenGetter(g.RefreshToken, g.AccessToken, g.Issuer, g.Type)
	if err != nil {
		return nil, err
	}

	claims, err := ParseToken(&oauth2.Token{AccessToken: token.AccessToken}, idpType)
	if err != nil {
		return nil, err
	}
	g.RefreshToken = token.RefreshToken
	g.AccessToken = token.AccessToken
	g.IDToken = token.IDToken
	expiration := time.Now().Local().Add(time.Duration(token.ExpiresIn))
	g.Expiration = expiration
	g.Permissions = claims.Permissions

	tok := &oauth2.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       expiration,
	}
	tok = tok.WithExtra(map[string]interface{}{
		extraIDToken: token.IDToken,
	})

	return tok, nil
}

// ParseToken parses the JWT payload and return the decoded information.
func ParseToken(tkn *oauth2.Token, idpType config.IdpType) (*Claims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tkn.AccessToken, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	c, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("could not parse claims")
	}

	// parse permission
	var p []interface{}
	var orgID string
	var uname string

	if idpType == config.UAAIdpType {
		// "scopes" is used instead of "perms" in uaa-based token
		if p, ok = c[ClaimsScopes].([]interface{}); !ok {
			log.Warning("could not cast scopes")
		}
		uname, ok = c[ClaimsUaaUserName].(string)
		if !ok {
			return nil, fmt.Errorf("could not parse username from token")
		}
	} else {
		if p, ok = c[ClaimsPermissions].([]interface{}); !ok {
			log.Warning("could not cast permissions")
		}
		uname, ok = c[ClaimsCspUserName].(string)
		if !ok {
			return nil, fmt.Errorf("could not parse username from token")
		}
		// orgID is required in CSP (SaaS)
		if orgID, ok = c[ClaimsContext].(string); !ok {
			return nil, fmt.Errorf("could not parse orgID from token")
		}
	}

	perm := []string{}
	for _, i := range p {
		perm = append(perm, i.(string))
	}

	claims := &Claims{
		Username:    uname,
		Permissions: perm,
		OrgID:       orgID,
		Raw:         c,
	}
	return claims, nil
}

// IsExpired checks for the token expiry and returns true if the token has expired else will return false
func IsExpired(tokenExpiry time.Time) bool {
	// refresh at half token life
	two := 2
	now := time.Now().Unix()
	halfDur := -time.Duration((tokenExpiry.Unix()-now)/int64(two)) * time.Second
	return tokenExpiry.Add(halfDur).Unix() < now
}
