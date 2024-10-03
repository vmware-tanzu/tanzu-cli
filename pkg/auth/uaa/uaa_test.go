// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package uaa

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
)

const (
	fakeIssuerURL  = "https://auth0.com/"
	fakeAPIToken   = "fake_api_token"
	fakeCACrtPath  = "/fake/ca.crt"
	fakeSkipVerify = false
)

func TestGetAccessTokenFromAPIToken(t *testing.T) {
	assert := assert.New(t)
	fakeHTTPClient := &fakes.FakeHTTPClient{}
	responseBody := io.NopCloser(bytes.NewReader([]byte(`{
		"id_token": "abc",
		"token_type": "Test",
		"expires_in": 86400,
		"scope": "Test",
		"access_token": "LetMeIn",
		"refresh_token": "LetMeInAgain"}`)))
	fakeHTTPClient.DoReturns(&http.Response{
		StatusCode: 200,
		Body:       responseBody,
	}, nil)
	httpRestClient = fakeHTTPClient
	token, err := GetAccessTokenFromAPIToken(fakeAPIToken, fakeIssuerURL, fakeCACrtPath, fakeSkipVerify)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error...................................")
	}
	assert.Nil(err)
	assert.Equal("LetMeIn", token.AccessToken)

	req := fakeHTTPClient.DoArgsForCall(0)
	bodyBytes, _ := io.ReadAll(req.Body)
	body := string(bodyBytes)

	assert.Contains(body, "refresh_token="+fakeAPIToken)
	assert.Contains(body, "client_id="+GetAlternateClientID())
	assert.Contains(body, "grant_type=refresh_token")
}

func TestGetAccessTokenFromAPIToken_FailStatus(t *testing.T) {
	assert := assert.New(t)
	fakeHTTPClient := &fakes.FakeHTTPClient{}
	responseBody := io.NopCloser(bytes.NewReader([]byte(``)))
	fakeHTTPClient.DoReturns(&http.Response{
		StatusCode: 403,
		Body:       responseBody,
	}, nil)
	httpRestClient = fakeHTTPClient
	token, err := GetAccessTokenFromAPIToken(fakeAPIToken, fakeIssuerURL, fakeCACrtPath, fakeSkipVerify)
	assert.NotNil(err)
	assert.Contains(err.Error(), "Failed to obtain access token. Please provide valid API token")
	assert.Nil(token)
}

func TestGetAccessTokenFromAPIToken_InvalidResponse(t *testing.T) {
	assert := assert.New(t)
	fakeHTTPClient := &fakes.FakeHTTPClient{}
	responseBody := io.NopCloser(bytes.NewReader([]byte(`[{
		"id_token": "abc",
		"token_type": "Test",
		"expires_in": 86400,
		"scope": "Test",
		"access_token": "LetMeIn",
		"refresh_token": "LetMeInAgain"}]`)))
	fakeHTTPClient.DoReturns(&http.Response{
		StatusCode: 200,
		Body:       responseBody,
	}, nil)
	httpRestClient = fakeHTTPClient

	token, err := GetAccessTokenFromAPIToken(fakeAPIToken, fakeIssuerURL, fakeCACrtPath, fakeSkipVerify)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not unmarshal")
	assert.Nil(token)
}
