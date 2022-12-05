// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configapi "github.com/vmware-tanzu/tanzu-plugin-runtime/apis/config/v1alpha1"

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
			gsa := configapi.GlobalServerAuth{
				Expiration:  metav1.NewTime(expiration),
				AccessToken: accessToken,
				IDToken:     idToken,
			}
			confSource = initializeConfigSource(gsa)
			configFile, err := os.CreateTemp("", "test-config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", configFile.Name())
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")

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
			gsa := configapi.GlobalServerAuth{
				Expiration:  metav1.NewTime(expiration),
				AccessToken: accessToken,
				IDToken:     idToken,
			}
			confSource = initializeConfigSource(gsa)
			configFile, err := os.CreateTemp("", "test-config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", configFile.Name())
			fakeHTTPClient = &fakes.FakeHTTPClient{}
			httpRestClient = fakeHTTPClient
			// successful case
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

		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")

		})
		It("should return token from server", func() {
			token, err := confSource.Token()
			Expect(err).NotTo(HaveOccurred())
			Expect(token.AccessToken).To(Equal("LetMeIn"))
			Expect(token.RefreshToken).To(Equal("LetMeInAgain"))
		})
	})
})

func initializeConfigSource(gsa configapi.GlobalServerAuth) configSource {
	gs := configapi.GlobalServer{
		Endpoint: "",
		Auth:     gsa,
	}
	globalServer := configapi.Server{
		Name:       "GlobalServer",
		Type:       configapi.GlobalServerType,
		GlobalOpts: &gs,
	}
	managementServer := configapi.Server{
		Name: "ManagementServer",
		Type: configapi.ManagementClusterServerType,
	}
	clientConfigObj := configapi.ClientConfig{
		KnownServers: []*configapi.Server{
			&globalServer,
			&managementServer,
		},
		CurrentServer: globalServer.Name,
		KnownContexts: []*configapi.Context{
			{
				Name: globalServer.Name,
				Type: configapi.CtxTypeTMC,
			},
			{
				Name: managementServer.Name,
				Type: configapi.CtxTypeK8s,
			},
		},
		CurrentContext: map[configapi.ContextType]string{
			configapi.CtxTypeTMC: globalServer.Name,
			configapi.CtxTypeK8s: managementServer.Name,
		},
	}
	return configSource{
		ClientConfig: &clientConfigObj,
	}
}
