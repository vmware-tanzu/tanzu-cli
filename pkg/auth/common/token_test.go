// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var JWTHeader = `{"alg":"HS256","typ":"JWT"}`

func generateJWTToken(claims string) string {
	hm := hmac.New(sha256.New, []byte("secret"))
	_, _ = hm.Write([]byte(fmt.Sprintf(
		"%s.%s",
		base64.RawURLEncoding.EncodeToString([]byte(JWTHeader)),
		base64.RawURLEncoding.EncodeToString([]byte(claims)),
	)))
	sha := hex.EncodeToString(hm.Sum(nil))
	return fmt.Sprintf(
		"%s.%s.%s",
		base64.RawURLEncoding.EncodeToString([]byte(JWTHeader)),
		base64.RawURLEncoding.EncodeToString([]byte(claims)),
		sha,
	)
}

func TestParseToken_ParseFailure(t *testing.T) {
	assert := assert.New(t)

	// Pass in incorrectly formatted AccessToken
	tkn := oauth2.Token{
		AccessToken:  "LetMeIn",
		TokenType:    "Bearer",
		RefreshToken: "LetMeInAgain",
		Expiry:       time.Now().Add(time.Minute * 30),
	}

	claims, err := ParseToken(&tkn, config.CSPIdpType)
	assert.NotNil(err)
	assert.Contains(err.Error(), "invalid")
	assert.Nil(claims)
}

func TestParseToken_MissingUsername(t *testing.T) {
	assert := assert.New(t)

	accessToken := generateJWTToken(
		`{"sub":"1234567890","name":"John Doe","iat":1516239022}`,
	)
	tkn := oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		RefreshToken: "LetMeInAgain",
		Expiry:       time.Now().Add(time.Minute * 30),
	}

	claims, err := ParseToken(&tkn, config.CSPIdpType)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not parse username")
	assert.Nil(claims)
}

func TestParseToken_MissingContextName(t *testing.T) {
	assert := assert.New(t)

	accessToken := generateJWTToken(
		`{"sub":"1234567890","username":"John Doe","orgID":1516239022}`,
	)
	tkn := oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		RefreshToken: "LetMeInAgain",
		Expiry:       time.Now().Add(time.Minute * 30),
	}

	claims, err := ParseToken(&tkn, config.CSPIdpType)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not parse orgID")
	assert.Nil(claims)
}

func TestParseToken(t *testing.T) {
	assert := assert.New(t)

	accessToken := generateJWTToken(
		`{"sub":"1234567890","username":"John Doe","context_name":"1516239022"}`,
	)
	tkn := oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		RefreshToken: "LetMeInAgain",
		Expiry:       time.Now().Add(time.Minute * 30),
	}

	claim, err := ParseToken(&tkn, config.CSPIdpType)
	assert.Nil(err)
	assert.NotNil(claim)

	assert.Equal("John Doe", claim.Username)
	assert.Equal("1516239022", claim.OrgID)
	assert.Empty(claim.Permissions)
}

func TestParseTokenUAA(t *testing.T) {
	assert := assert.New(t)

	accessToken := generateJWTToken(
		`{"sub":"1234567890","user_name":"John Doe","scope":["openid", "roles", "ensemble:admin"]}`,
	)
	tkn := oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		RefreshToken: "LetMeInAgain",
		Expiry:       time.Now().Add(time.Minute * 30),
	}

	claim, err := ParseToken(&tkn, config.UAAIdpType)
	assert.Nil(err)
	assert.NotNil(claim)

	assert.Equal("John Doe", claim.Username)
	assert.Equal("", claim.OrgID)
	assert.ElementsMatch(claim.Permissions, []string{"openid", "roles", "ensemble:admin"})
}

func TestIsExpired(t *testing.T) {
	assert := assert.New(t)

	testTime := time.Now().Add(-time.Minute)
	assert.True(IsExpired(testTime))

	testTime = time.Now().Add(time.Minute * 30)
	assert.False(IsExpired(testTime))
}

func mockBadTokenGetter(refreshToken, _, _, _ string) (*Token, error) {
	return nil, fmt.Errorf("bad token refresh for %s", refreshToken)
}

func createMockTokenGetter(newRefreshToken string, newTokenExpirySeconds int64) func(refreshToken, accessToken, issuer, tokenType string) (*Token, error) {
	return func(refreshToken, accessToken, issuer, tokenType string) (*Token, error) {
		tok := &Token{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			ExpiresIn:    newTokenExpirySeconds,
		}
		return tok, nil
	}
}

func TestGetToken_Valid_NotExpired(t *testing.T) {
	assert := assert.New(t)

	accessToken := generateJWTToken(
		`{"sub":"1234567890","username":"joe","context_name":"1516239022"}`,
	)
	expireTime := time.Now().Add(time.Minute * 30)

	serverAuth := configtypes.GlobalServerAuth{
		Issuer:       "https://oidc.example.com",
		UserName:     "jdoe",
		AccessToken:  accessToken,
		IDToken:      "xxyyzz",
		RefreshToken: "sprite",
		Expiration:   expireTime,
		Type:         "client",
	}

	tok, err := GetToken(&serverAuth, mockBadTokenGetter, config.CSPIdpType)
	// implies mockBadTokenGetter not called
	assert.Nil(err)
	assert.NotNil(tok)
	assert.Equal(accessToken, tok.AccessToken)
	assert.Equal(expireTime, tok.Expiry)
}

func TestGetToken_Expired(t *testing.T) {
	assert := assert.New(t)

	accessToken := generateJWTToken(
		`{"sub":"1234567890","username":"joe","context_name":"1516239022"}`,
	)
	expireTime := time.Now().Add(-time.Minute * 30)

	serverAuth := configtypes.GlobalServerAuth{
		Issuer:       "https://oidc.example.com",
		UserName:     "jdoe",
		AccessToken:  accessToken,
		IDToken:      "xxyyzz",
		RefreshToken: "sprite",
		Expiration:   expireTime,
		Type:         APITokenType,
	}

	newRefreshToken := "LetMeInAgain"
	newExpiry := int64(time.Until(time.Now().Add(time.Minute * 30)).Seconds())

	tokenGetter := createMockTokenGetter(newRefreshToken, newExpiry)

	tok, err := GetToken(&serverAuth, tokenGetter, config.CSPIdpType)
	assert.Nil(err)
	assert.NotNil(tok)
	assert.Equal(tok.AccessToken, accessToken)
	assert.Equal(tok.RefreshToken, newRefreshToken)
}
