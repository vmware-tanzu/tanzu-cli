// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"crypto/x509"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	cliapi "github.com/vmware-tanzu/tanzu-framework/apis/cli/v1alpha1"
	configapi "github.com/vmware-tanzu/tanzu-plugin-runtime/apis/config/v1alpha1"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	tkgauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tkg"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes/helper"
)

func TestCliCmdSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cli core command suite")
}

var _ = Describe("Test tanzu context command", func() {
	var (
		tkgConfigFile   *os.File
		tkgConfigFileNG *os.File
		err             error
		buf             bytes.Buffer
	)
	const (
		targetK8s       = "k8s"
		existingContext = "test-mc"
	)

	Describe("tanzu context list", func() {
		cmd := &cobra.Command{}
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

			cmd.SetOut(&buf)
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tkgConfigFile.Name())
			os.RemoveAll(tkgConfigFileNG.Name())
			resetContextCommandFlags()
			buf.Reset()
		})
		It("should return empty rows if there are no contexts", func() {
			targetStr = targetK8s
			os.RemoveAll(tkgConfigFileNG.Name())
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			Expect(buf.String()).To(Equal("  NAME  ISACTIVE  ENDPOINT  KUBECONFIGPATH  KUBECONTEXT  \n"))

			buf.Reset()
			targetStr = "tmc"
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			Expect(buf.String()).To(Equal("  NAME  ISACTIVE  ENDPOINT  \n"))

		})
		It("should return contexts if tanzu config file has contexts available", func() {
			targetStr = targetK8s
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring("test-mc  true      test-endpoint  test-path       test-mc-context"))
			Expect(buf.String()).ToNot(ContainSubstring("test-tmc-context"))
			Expect(buf.String()).ToNot(ContainSubstring("test-use-context"))

		})
		It("should return contexts in yaml format if tanzu config file has contexts available", func() {
			targetStr = targetK8s
			outputFormat = "yaml"
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			expectedYaml := `- endpoint: test-endpoint
  iscurrent: "true"
  ismanagementcluster: "true"
  kubeconfigpath: test-path
  kubecontext: test-mc-context
  name: test-mc
  type: kubernetes`
			Expect(buf.String()).To(ContainSubstring(expectedYaml))
			Expect(buf.String()).ToNot(ContainSubstring("test-tmc-context"))
			Expect(buf.String()).ToNot(ContainSubstring("test-use-context"))
		})

	})
	Describe("tanzu context get", func() {
		cmd := &cobra.Command{}
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

			cmd.SetOut(&buf)
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tkgConfigFile.Name())
			os.RemoveAll(tkgConfigFileNG.Name())
			resetContextCommandFlags()
			buf.Reset()
		})
		It("should return error if there are no contexts", func() {
			os.RemoveAll(tkgConfigFileNG.Name())
			err = getCtx(cmd, nil)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("no contexts found"))

		})
		It("should return contexts if tanzu config file has contexts available", func() {
			err = getCtx(cmd, []string{existingContext})
			Expect(err).To(BeNil())
			expectedYaml := `name: test-mc
target: kubernetes
clusterOpts:
    endpoint: test-endpoint
    path: test-path
    context: test-mc-context
    isManagementCluster: true`
			Expect(buf.String()).To(ContainSubstring(expectedYaml))
		})

	})

	Describe("tanzu context delete", func() {
		cmd := &cobra.Command{}
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

			cmd.SetOut(&buf)
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tkgConfigFile.Name())
			os.RemoveAll(tkgConfigFileNG.Name())
			resetContextCommandFlags()
			buf.Reset()
		})
		It("should return error if the context to be deleted doesn't exist", func() {
			unattended = true
			err = deleteCtx(cmd, []string{"fake-mc"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context fake-mc not found"))

		})
		It("should delete context successfully if the config file has contexts available", func() {

			err = deleteCtx(cmd, []string{existingContext})
			Expect(err).To(BeNil())

			err = getCtx(cmd, []string{existingContext})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context test-mc not found"))
		})
	})

	Describe("tanzu context use", func() {
		cmd := &cobra.Command{}
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

			cmd.SetOut(&buf)
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tkgConfigFile.Name())
			os.RemoveAll(tkgConfigFileNG.Name())
			resetContextCommandFlags()
			buf.Reset()
		})
		It("should return error if the context to be used doesn't exist", func() {
			unattended = true
			err = useCtx(cmd, []string{"fake-mc"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context fake-mc not found"))

		})
		It("should set the context as the current-context if the config file has context available", func() {
			targetStr = "mission-control"
			err = useCtx(cmd, []string{"test-use-context"})
			Expect(err).To(BeNil())

			cctx, err := config.GetCurrentContext(cliapi.Target(targetStr))
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring("test-use-context"))
		})
	})
})

var _ = Describe("create new context", func() {
	const (
		exisitingContext   = "test-mc"
		testKubeContext    = "test-k8s-context"
		testKubeConfigPath = "/fake/path/kubeconfig"
		testContextName    = "fake-context-name"
		fakeTMCEndpoint    = "https://cloud.vmware.com/auth"
	)

	Describe("create context with kubeconfig", func() {
		var (
			tkgConfigFile   *os.File
			tkgConfigFileNG *os.File
			err             error
			ctx             *configapi.Context
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
			resetContextCommandFlags()
		})
		Context("with only kubecontext provided", func() {
			It("should create context with given kubecontext and default kubeconfig path", func() {
				kubeContext = testKubeContext
				ctxName = testContextName
				ctx, err = createNewContext()
				Expect(err).To(BeNil())
				Expect(ctx.Name).To(ContainSubstring("fake-context-name"))
				Expect(string(ctx.Target)).To(ContainSubstring("kubernetes"))
				Expect(ctx.ClusterOpts.Context).To(ContainSubstring("test-k8s-context"))
				Expect(ctx.ClusterOpts.Path).To(ContainSubstring(clientcmd.RecommendedHomeFile))
			})
		})
		Context("with both kubeconfig and  kubecontext provided", func() {
			It("should create context with given kubecontext and kubeconfig path", func() {
				kubeContext = testKubeContext
				kubeConfig = testKubeConfigPath
				ctxName = testContextName
				ctx, err = createNewContext()
				Expect(err).To(BeNil())
				Expect(ctx.Name).To(ContainSubstring("fake-context-name"))
				Expect(string(ctx.Target)).To(ContainSubstring("kubernetes"))
				Expect(ctx.ClusterOpts.Context).To(ContainSubstring("test-k8s-context"))
				Expect(ctx.ClusterOpts.Path).To(ContainSubstring(kubeConfig))
			})
		})
		Context("context name already exists", func() {
			It("should return error", func() {
				kubeContext = testKubeContext
				kubeConfig = testKubeConfigPath
				ctxName = exisitingContext
				ctx, err = createNewContext()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(`context "test-mc" already exists`))
			})
		})
	})
	Describe("create context with tmc endpoint", func() {
		var (
			tkgConfigFile   *os.File
			tkgConfigFileNG *os.File
			err             error
			ctx             *configapi.Context
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
			resetContextCommandFlags()
		})
		Context("with only endpoint and context name provided", func() {
			It("should create context with given endpoint and context name", func() {
				endpoint = fakeTMCEndpoint
				ctxName = testContextName
				ctx, err = createNewContext()
				Expect(err).To(BeNil())
				Expect(ctx.Name).To(ContainSubstring("fake-context-name"))
				Expect(string(ctx.Target)).To(ContainSubstring("mission-control"))
				Expect(ctx.GlobalOpts.Endpoint).To(ContainSubstring(endpoint))
			})
		})
		Context("context name already exists", func() {
			It("should return error", func() {
				endpoint = fakeTMCEndpoint
				ctxName = exisitingContext
				ctx, err = createNewContext()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(`context "test-mc" already exists`))
			})
		})
	})

	Describe("create context with non-tmc endpoint", func() {
		const (
			clustername = "fake-cluster"
			issuer      = "https://fakeissuer.com"
			issuerCA    = "fakeCAData"
		)
		var (
			ctx       *configapi.Context
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
			resetContextCommandFlags()
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
				ctxName = testContextName

				ctx, err = createNewContext()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("error creating kubeconfig with tanzu pinniped-auth login plugin"))
			})
		})
		Context("When the given endpoint(non vSphere with Tanzu) has the pinniped configured", func() {
			It("should create the context successfully with kubeconfig file updated with pinniped auth info", func() {
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
				ctxName = testContextName

				oldHomeDir := os.Getenv("HOME")
				tmpHomeDir, err := os.MkdirTemp(os.TempDir(), "home")
				Expect(err).To(BeNil(), "unable to create temporary home directory")
				os.Setenv("HOME", tmpHomeDir)

				ctx, err = createNewContext()
				os.Setenv("HOME", oldHomeDir)
				Expect(err).To(BeNil())
				Expect(ctx.Name).To(ContainSubstring("fake-context-name"))
				Expect(string(ctx.Target)).To(ContainSubstring("kubernetes"))
				Expect(ctx.ClusterOpts.Endpoint).To(ContainSubstring(endpoint))
				Expect(ctx.ClusterOpts.Path).To(ContainSubstring(filepath.Join(tmpHomeDir, tkgauth.TanzuLocalKubeDir, tkgauth.TanzuKubeconfigFile)))
			})
		})

	})
})

func resetContextCommandFlags() {
	ctxName = ""
	endpoint = ""
	apiToken = ""
	kubeConfig = ""
	kubeContext = ""
}
