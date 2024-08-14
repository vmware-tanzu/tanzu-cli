// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
)

const issuerURL = "https://auth0.com/"

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
	token, err := GetAccessTokenFromAPIToken("asdas", issuerURL)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error...................................")
	}
	assert.Nil(err)
	assert.Equal("LetMeIn", token.AccessToken)
}

func TestGetAccessTokenFromAPIToken_Err(t *testing.T) {
	assert := assert.New(t)

	token, err := GetAccessTokenFromAPIToken("asdas", "example.com")
	assert.NotNil(err)
	assert.Nil(token)
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
	token, err := GetAccessTokenFromAPIToken("asdas", issuerURL)
	assert.NotNil(err)
	assert.Contains(err.Error(), "obtain access token")
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

	token, err := GetAccessTokenFromAPIToken("asdas", issuerURL)
	assert.NotNil(err)
	assert.Contains(err.Error(), "could not unmarshal")
	assert.Nil(token)
}
