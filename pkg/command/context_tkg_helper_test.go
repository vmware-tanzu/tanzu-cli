// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes/helper"
)

func TestRunCommandAndGetStdOutAndErr(t *testing.T) {
	expectedStdout := []byte("fake stdout message")
	expectedStderr := []byte("fake stderr message")
	expectedError := errors.New("fake error")

	// Create a command with the mock implementation
	mockCmd := &cobra.Command{
		Use:           "mock",
		Short:         "mock command",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(os.Stdout, string(expectedStdout))
			fmt.Fprint(os.Stderr, string(expectedStderr))
			return expectedError
		},
	}

	stdoutData, stderrData, err := RunCommandAndGetStdOutAndErr(mockCmd, nil)

	// Check if the stdout and stderr data match the expected values
	if !bytes.Equal(stdoutData, expectedStdout) {
		t.Errorf("Expected stdout: %s, but got: %s \n", expectedStdout, stdoutData)
	}

	if !bytes.Equal(stderrData, expectedStderr) {
		t.Errorf("Expected stderr: %s, but got: %s \n", expectedStderr, stderrData)
	}

	if err.Error() != expectedError.Error() {
		t.Errorf("Expected error %s, but got: %s \n", expectedError.Error(), err.Error())
	}
}

func TestTKGKubeconfigFetcherOptions_getKubeconfigUsingPinnipedAuthPlugin(t *testing.T) {
	type fields struct {
		Endpoint              string
		InsecureSkipTLSVerify bool
		EndpointCACertPath    string
		CmdExecutor           func(command *cobra.Command, args []string) (stdOut []byte, stderr []byte, err error)
	}
	tests := []struct {
		name       string
		fields     fields
		want       []byte
		wantErr    assert.ErrorAssertionFunc
		wantErrMsg string
	}{
		{
			name: "when only 'endpoint' is provided and if 'pinniped-auth kubeconfig get' command returns success, it should return success",
			fields: fields{
				Endpoint: "fake.end.point",
				CmdExecutor: func(command *cobra.Command, args []string) (stdOut []byte, stderr []byte, err error) {
					assert.Contains(t, args, "--endpoint")
					return []byte("fake pinniped kubeconfig"), nil, nil
				},
			},
			want:    []byte("fake pinniped kubeconfig"),
			wantErr: assert.NoError,
		},
		{
			name: "when 'endpoint' and 'insecure-skip-tls-verify' options are provided and if 'pinniped-auth kubeconfig get' command returns success, it should return success",
			fields: fields{
				Endpoint:              "fake.end.point",
				InsecureSkipTLSVerify: true,
				CmdExecutor: func(command *cobra.Command, args []string) (stdOut []byte, stderr []byte, err error) {
					assert.Contains(t, args, "--endpoint")
					assert.Contains(t, args, "--insecure-skip-tls-verify")
					return []byte("fake pinniped kubeconfig"), nil, nil
				},
			},
			want:    []byte("fake pinniped kubeconfig"),
			wantErr: assert.NoError,
		},
		{
			name: "when invalid caCertPath is provided it should return error before executing the pinniped-auth plugin command",
			fields: fields{
				Endpoint:           "fake.end.point",
				EndpointCACertPath: "invalid-path.crt",
				CmdExecutor:        nil,
			},
			want:       nil,
			wantErr:    assert.Error,
			wantErrMsg: "error reading CA certificate file invalid-path.crt",
		},
		{
			name: "when 'pinniped-auth' plugin (command executor) return error. it should return the error",
			fields: fields{
				Endpoint:              "fake.end.point",
				InsecureSkipTLSVerify: true,
				CmdExecutor: func(command *cobra.Command, args []string) (stdOut []byte, stderr []byte, err error) {
					assert.Contains(t, args, "--endpoint")
					assert.Contains(t, args, "--insecure-skip-tls-verify")
					return nil, nil, errors.New("fake pinniped-auth plugin error")
				},
			},
			want:       nil,
			wantErr:    assert.Error,
			wantErrMsg: "fake pinniped-auth plugin error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tkfo := &TKGKubeconfigFetcherOptions{
				Endpoint:              tt.fields.Endpoint,
				InsecureSkipTLSVerify: tt.fields.InsecureSkipTLSVerify,
				EndpointCACertPath:    tt.fields.EndpointCACertPath,
				CmdExecutor:           tt.fields.CmdExecutor,
			}
			got, err := tkfo.getKubeconfigUsingPinnipedAuthPlugin()
			if !tt.wantErr(t, err, "getKubeconfigUsingPinnipedAuthPlugin()") {
				return
			}
			if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
			}
			assert.Equalf(t, tt.want, got, "getKubeconfigUsingPinnipedAuthPlugin()")
		})
	}
}

var _ = Describe("Test getKubeconfigForPriorTKGVersion()", func() {

	Describe("get pinniped kubeconfig if the 'pinniped-auth' plugin doesn't support `kubeconfig get' subcommand", func() {
		const (
			clusterName = "fake-cluster"
			issuer      = "https://fakeissuer.com"
			issuerCA    = "fakeCAData"
		)
		var (
			ep        string
			tlsServer *ghttp.Server
			servCert  *x509.Certificate
			err       error
		)
		BeforeEach(func() {
			tlsServer = ghttp.NewTLSServer()
			ep = tlsServer.URL()
			servCert = tlsServer.HTTPTestServer.Certificate()
			err = os.Setenv(constants.EULAPromptAnswer, "yes")
			err = os.Setenv(constants.CEIPOptInUserPromptAnswer, "yes")
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			resetContextCommandFlags()
			tlsServer.Close()
			os.Unsetenv(constants.EULAPromptAnswer)
			os.Unsetenv(constants.CEIPOptInUserPromptAnswer)
		})
		Context("When the given endpoint(non vSphere with Tanzu) fails to provide the pinniped info ", func() {
			It("should return error", func() {
				endpoint = ep
				tlsServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/wcp/loginbanner"),
						ghttp.RespondWith(http.StatusNotFound, "I'm a 404"),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-public/configmaps/cluster-info"),
						ghttp.RespondWith(http.StatusNotFound, "I'm a 404"),
					),
				)
				tkf := NewTKGKubeconfigFetcher(ep, "", true)
				tkfo := tkf.(*TKGKubeconfigFetcherOptions)
				_, _, err = tkfo.getKubeconfigForPriorTKGVersion()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("error creating kubeconfig with tanzu pinniped-auth login plugin"))
				Expect(err.Error()).To(ContainSubstring("failed to get cluster-info from the end-point"))
			})
		})
		Context("When the given endpoint(non vSphere with Tanzu) has the pinniped configured and 'insecureSkipTLSVerify' is enabled", func() {
			It("should create the context successfully with kubeconfig file updated with pinniped auth info", func() {
				var clusterInfo, pinnipedInfo string
				endpoint = ep
				clusterInfo = helper.GetFakeClusterInfo(endpoint, servCert)
				pinnipedInfo = helper.GetFakePinnipedInfo(
					helper.PinnipedInfo{
						ClusterName:              clusterName,
						Issuer:                   issuer,
						IssuerCABundleData:       issuerCA,
						ConciergeIsClusterScoped: true,
					})
				tlsServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/wcp/loginbanner"),
						ghttp.RespondWith(http.StatusNotFound, "I'm a 404"),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-public/configmaps/cluster-info"),
						ghttp.RespondWith(http.StatusOK, clusterInfo),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-public/configmaps/pinniped-info"),
						ghttp.RespondWith(http.StatusOK, pinnipedInfo),
					),
				)
				oldHomeDir := os.Getenv("HOME")
				tmpHomeDir, err := os.MkdirTemp(os.TempDir(), "home")
				Expect(err).To(BeNil(), "unable to create temporary home directory")
				os.Setenv("HOME", tmpHomeDir)

				// with Insecure skip TLS verify option
				tkf := NewTKGKubeconfigFetcher(ep, "", true)
				tkfo := tkf.(*TKGKubeconfigFetcherOptions)
				KCfgPath, KCfgCtx, err := tkfo.getKubeconfigForPriorTKGVersion()

				Expect(err).To(BeNil())
				Expect(KCfgCtx).To(Equal("tanzu-cli-" + clusterName + "@" + clusterName))
				Expect(KCfgPath).To(Equal(filepath.Join(tmpHomeDir, ".kube-tanzu", "config")))

				Expect(err).To(BeNil())
				Expect(KCfgCtx).To(Equal("tanzu-cli-" + clusterName + "@" + clusterName))
				Expect(KCfgPath).To(Equal(filepath.Join(tmpHomeDir, ".kube-tanzu", "config")))

				os.Setenv("HOME", oldHomeDir)

			})
		})
		Context("When the given endpoint(non vSphere with Tanzu) has the pinniped configured and  endpoint CA cert is provided", func() {
			It("should create the context successfully with kubeconfig file updated with pinniped auth info", func() {
				var clusterInfo, pinnipedInfo string
				endpoint = ep
				clusterInfo = helper.GetFakeClusterInfo(endpoint, servCert)
				pinnipedInfo = helper.GetFakePinnipedInfo(
					helper.PinnipedInfo{
						ClusterName:              clusterName,
						Issuer:                   issuer,
						IssuerCABundleData:       issuerCA,
						ConciergeIsClusterScoped: true,
					})
				tlsServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/wcp/loginbanner"),
						ghttp.RespondWith(http.StatusNotFound, "I'm a 404"),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-public/configmaps/cluster-info"),
						ghttp.RespondWith(http.StatusOK, clusterInfo),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v1/namespaces/kube-public/configmaps/pinniped-info"),
						ghttp.RespondWith(http.StatusOK, pinnipedInfo),
					),
				)
				oldHomeDir := os.Getenv("HOME")
				tmpHomeDir, err := os.MkdirTemp(os.TempDir(), "home")
				Expect(err).To(BeNil(), "unable to create temporary home directory")
				os.Setenv("HOME", tmpHomeDir)

				// with CA cert of endpoint
				pemCert := pem.EncodeToMemory(&pem.Block{
					Type:  "CERTIFICATE",
					Bytes: servCert.Raw,
				})
				epCACertFile, err := os.CreateTemp("", "cert")
				Expect(err).To(BeNil(), "unable to create temporary file for endpoint cert")
				err = os.WriteFile(epCACertFile.Name(), pemCert, 0600)
				Expect(err).To(BeNil(), "unable to write the endpoint certificate to file")

				tkf := NewTKGKubeconfigFetcher(ep, epCACertFile.Name(), false)
				tkfo := tkf.(*TKGKubeconfigFetcherOptions)
				KCfgPath, KCfgCtx, err := tkfo.getKubeconfigForPriorTKGVersion()

				Expect(err).To(BeNil())
				Expect(KCfgCtx).To(Equal("tanzu-cli-" + clusterName + "@" + clusterName))
				Expect(KCfgPath).To(Equal(filepath.Join(tmpHomeDir, ".kube-tanzu", "config")))

				os.Setenv("HOME", oldHomeDir)

			})
		})
	})
})
