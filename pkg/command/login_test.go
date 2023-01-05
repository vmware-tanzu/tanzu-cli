// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"crypto/x509"
	"net/http"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	tkgauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tkg"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes/helper"

	"github.com/otiai10/copy"
	"k8s.io/client-go/tools/clientcmd"

	configapi "github.com/vmware-tanzu/tanzu-plugin-runtime/apis/config/v1alpha1"
)

var _ = Describe("create new server", func() {
	const (
		existingContext    = "test-mc"
		testKubeContext    = "test-k8s-context"
		testKubeConfigPath = "/fake/path/kubeconfig"
		testServerName     = "fake-server-name"
		fakeTMCEndpoint    = "https://cloud.vmware.com/auth"
	)
	Describe("create server with kubeconfig", func() {
		var (
			tkgConfigFile   *os.File
			tkgConfigFileNG *os.File
			err             error
			svr             *configapi.Server
		)

		BeforeEach(func() {
			tkgConfigFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), tkgConfigFile.Name())
			Expect(err).To(BeNil(), "Error while copying tanzu config file for testing")
			os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

			tkgConfigFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
			err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng.yaml"), tkgConfigFileNG.Name())
			Expect(err).To(BeNil(), "Error while coping tanzu config file for testing")
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tkgConfigFile.Name())
			os.RemoveAll(tkgConfigFileNG.Name())
			resetLoginCommandFlags()
		})
		Context("with only kubecontext provided", func() {
			It("should create server with given kubecontext and default kubeconfig path", func() {
				kubeContext = testKubeContext
				name = testServerName
				svr, err = createNewServer()
				Expect(err).To(BeNil())
				Expect(svr.Name).To(ContainSubstring(name))
				Expect(svr.Type).To(BeEquivalentTo(configapi.ManagementClusterServerType))
				Expect(svr.ManagementClusterOpts.Context).To(ContainSubstring("test-k8s-context"))
				Expect(svr.ManagementClusterOpts.Path).To(ContainSubstring(clientcmd.RecommendedHomeFile))
			})
		})
		Context("with both kubeconfig and  kubecontext provided", func() {
			It("should create server with given kubecontext and kubeconfig path", func() {
				kubeContext = testKubeContext
				kubeConfig = testKubeConfigPath
				name = testServerName
				svr, err = createNewServer()
				Expect(err).To(BeNil())
				Expect(svr.Name).To(ContainSubstring(name))
				Expect(svr.Type).To(BeEquivalentTo(configapi.ManagementClusterServerType))
				Expect(svr.ManagementClusterOpts.Context).To(ContainSubstring("test-k8s-context"))
				Expect(svr.ManagementClusterOpts.Path).To(ContainSubstring(kubeConfig))
			})
		})
		Context("server name already exists", func() {
			It("should return error", func() {
				kubeContext = testKubeContext
				kubeConfig = testKubeConfigPath
				name = existingContext
				svr, err = createNewServer()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(`server "test-mc" already exists`))
			})
		})
		Describe("create server with tmc server endpoint", func() {
			var (
				tkgConfigFile   *os.File
				tkgConfigFileNG *os.File
				err             error
				svr             *configapi.Server
			)

			BeforeEach(func() {
				tkgConfigFile, err = os.CreateTemp("", "config")
				Expect(err).To(BeNil())
				err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), tkgConfigFile.Name())
				Expect(err).To(BeNil(), "Error while copying tanzu config file for testing")
				os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

				tkgConfigFileNG, err = os.CreateTemp("", "config_ng")
				Expect(err).To(BeNil())
				os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
				err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng.yaml"), tkgConfigFileNG.Name())
				Expect(err).To(BeNil(), "Error while coping tanzu config file for testing")
			})
			AfterEach(func() {
				os.Unsetenv("TANZU_CONFIG")
				os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
				os.RemoveAll(tkgConfigFile.Name())
				os.RemoveAll(tkgConfigFileNG.Name())
				resetLoginCommandFlags()
			})
			Context("with only endpoint and context name provided", func() {
				It("should create server with given endpoint and context name", func() {
					endpoint = fakeTMCEndpoint
					name = "fake-server-name"
					svr, err = createNewServer()
					Expect(err).To(BeNil())
					Expect(svr.Name).To(ContainSubstring(name))
					Expect(svr.Type).To(BeEquivalentTo(configapi.GlobalServerType))
					Expect(svr.GlobalOpts.Endpoint).To(ContainSubstring(endpoint))
				})
			})
			Context("server name already exists", func() {
				It("should return error", func() {
					endpoint = fakeTMCEndpoint
					name = existingContext
					svr, err = createNewServer()
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring(`server "test-mc" already exists`))
				})
			})
		})

		Describe("create server with non-tmc endpoint", func() {
			const (
				clustername = "fake-cluster"
				issuer      = "https://fakeissuer.com"
				issuerCA    = "fakeCAData"
			)
			var (
				svr       *configapi.Server
				ep        string
				tlsServer *ghttp.Server
				servCert  *x509.Certificate
				err       error
			)
			BeforeEach(func() {
				tlsServer = ghttp.NewTLSServer()
				ep = tlsServer.URL()
				servCert = tlsServer.HTTPTestServer.Certificate()
			})
			AfterEach(func() {
				resetLoginCommandFlags()
				tlsServer.Close()
			})
			Context("When the given endpoint(non vSphere with Tanzu) fails to provide the pinniped info", func() {
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
					name = testServerName

					svr, err = createNewServer()
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("error creating kubeconfig with tanzu pinniped-auth login plugin"))
				})
			})
			Context("When the given endpoint(non vSphere with Tanzu) has the pinniped configured", func() {
				It("should create the server successfully with kubeconfig file updated with pinniped auth info", func() {
					var clusterInfo, pinnipedInfo string
					endpoint = ep
					clusterInfo = helper.GetFakeClusterInfo(endpoint, servCert)
					pinnipedInfo = helper.GetFakePinnipedInfo(
						helper.PinnipedInfo{
							ClusterName:              clustername,
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
					name = testServerName
					oldHomeDir := os.Getenv("HOME")
					tmpHomeDir, err := os.MkdirTemp(os.TempDir(), "home")
					Expect(err).To(BeNil(), "unable to create temporary home directory")
					os.Setenv("HOME", tmpHomeDir)

					svr, err = createNewServer()
					os.Setenv("HOME", oldHomeDir)
					Expect(err).To(BeNil())
					Expect(svr.Name).To(ContainSubstring(name))
					Expect(svr.Type).To(BeEquivalentTo(configapi.ManagementClusterServerType))
					Expect(svr.ManagementClusterOpts.Endpoint).To(ContainSubstring(endpoint))
					Expect(svr.ManagementClusterOpts.Path).To(ContainSubstring(filepath.Join(tmpHomeDir, tkgauth.TanzuLocalKubeDir, tkgauth.TanzuKubeconfigFile)))
				})
			})
		})
	})
})

func resetLoginCommandFlags() {
	name = ""
	endpoint = ""
	apiToken = ""
	kubeConfig = ""
	kubeContext = ""
	server = ""
}
