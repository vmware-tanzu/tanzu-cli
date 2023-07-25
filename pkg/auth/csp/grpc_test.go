// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
)

var (
	fakeHTTPClient *fakes.FakeHTTPClient
)

const accessTokenDummy = "AccessToken_dummy"
const idTokenDummy = "IDToken_dummy"

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cli/core/pkg/auth/csp Suite")
}

var _ = Describe("Unit tests for grpc", func() {
	var (
		confSource  configSource
		accessToken string
		idToken     string
	)

	Context("when token is not expired", func() {
		BeforeEach(func() {
			accessToken = accessTokenDummy
			idToken = idTokenDummy
			expiration := time.Now().Local().Add(time.Second * time.Duration(1000))
			gsa := configtypes.GlobalServerAuth{
				Expiration:  expiration,
				AccessToken: accessToken,
				IDToken:     idToken,
			}
			confSource = initializeConfigSource(gsa)

			cc := &fakes.FakeConfigClientWrapper{}
			configClientWrapper = cc
			cc.StoreClientConfigReturns(nil)
			cc.AcquireTanzuConfigLock()
		})
		It("should return current token", func() {
			token, err := confSource.Token()
			Expect(err).NotTo(HaveOccurred())
			Expect(token.AccessToken).To(Equal(accessToken))
			et := token.WithExtra(ExtraIDToken)
			Expect(et.AccessToken).To(Equal(accessToken))
		})
	})
	Context("when token is expired", func() {
		BeforeEach(func() {
			accessToken = accessTokenDummy
			idToken = idTokenDummy
			expiration := time.Now().Local().Add(time.Second * time.Duration(-1000))
			gsa := configtypes.GlobalServerAuth{
				Expiration:  expiration,
				AccessToken: accessToken,
				IDToken:     idToken,
			}
			confSource = initializeConfigSource(gsa)
			fakeHTTPClient = &fakes.FakeHTTPClient{}
			httpRestClient = fakeHTTPClient
			// successful case
			responseBody := io.NopCloser(bytes.NewReader([]byte(`{
				"id_token": "abc",
				"token_type": "Test",
				"expires_in": 86400,
				"scope": "Test",
				"access_token": "LetMeInGrpc1",
				"refresh_token": "LetMeInAgainGrpc1"}`)))

			fakeHTTPClient.DoReturns(&http.Response{
				StatusCode: 200,
				Body:       responseBody,
			}, nil)

			cc := &fakes.FakeConfigClientWrapper{}
			configClientWrapper = cc
			cc.StoreClientConfigReturns(nil)
			cc.AcquireTanzuConfigLock()
		})
		It("should return token from server", func() {
			token, err := confSource.Token()
			Expect(err).NotTo(HaveOccurred())
			Expect(token.AccessToken).To(Equal("LetMeInGrpc1"))
			Expect(token.RefreshToken).To(Equal("LetMeInAgainGrpc1"))
		})
	})
})

func initializeConfigSource(gsa configtypes.GlobalServerAuth) configSource {
	gs := configtypes.GlobalServer{
		Endpoint: "",
		Auth:     gsa,
	}
	globalServer := configtypes.Server{
		Name:       "GlobalServer",
		Type:       configtypes.GlobalServerType,
		GlobalOpts: &gs,
	}
	managementServer := configtypes.Server{
		Name: "ManagementServer",
		Type: configtypes.ManagementClusterServerType,
	}
	clientConfigObj := configtypes.ClientConfig{
		KnownServers: []*configtypes.Server{
			&globalServer,
			&managementServer,
		},
		CurrentServer: globalServer.Name,
		KnownContexts: []*configtypes.Context{
			{
				Name:   globalServer.Name,
				Target: configtypes.TargetTMC,
			},
			{
				Name:   managementServer.Name,
				Target: configtypes.TargetK8s,
			},
		},
		CurrentContext: map[string]string{
			configtypes.TargetTMC: globalServer.Name,
			configtypes.TargetK8s: managementServer.Name,
		},
	}
	return configSource{
		ClientConfig: &clientConfigObj,
	}
}
