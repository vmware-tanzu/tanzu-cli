// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

func TestCliCmdSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cli core command suite")
}

const (
	targetK8s       = "k8s"
	existingContext = "test-mc"
	testUseContext  = "test-use-context"
	jsonStr         = "json"
	testmc          = "test-mc"
	missionControl  = "mission-control"
)

var _ = Describe("Test tanzu context command", func() {
	var (
		tkgConfigFile   *os.File
		tkgConfigFileNG *os.File
		err             error
		buf             bytes.Buffer
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
			Expect(err).To(BeNil(), "Error while copying tanzu config_ng file for testing")

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
			Expect(buf.String()).ToNot(ContainSubstring(testUseContext))

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
			Expect(buf.String()).ToNot(ContainSubstring(testUseContext))
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
			Expect(err).To(BeNil(), "Error while copying tanzu config_ng file for testing")

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
			Expect(err).To(BeNil(), "Error while copying tanzu-ng config file for testing")

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
			Expect(err).To(BeNil(), "Error while copying tanzu config_ng file for testing")

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
			targetStr = missionControl
			err = useCtx(cmd, []string{testUseContext})
			Expect(err).To(BeNil())

			cctx, err := config.GetCurrentContext(targetStr)
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring(testUseContext))
		})
	})

	Describe("tanzu context unset", func() {
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
			Expect(err).To(BeNil(), "Error while copying tanzu config_ng file for testing")
			targetStr = ""
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
		// incorrect context name (not exists)
		It("should return error when context is not exists", func() {
			name = "not-exists"
			err = unsetCtx(cmd, []string{name})
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(contextNotActiveOrNotExists, name)))
		})
		// correct context name but not active
		It("should return error when context is not active", func() {
			name = testUseContext
			err = unsetCtx(cmd, []string{name})
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(contextNotActiveOrNotExists, name)))
		})
		// correct context name and active
		It("should not return error when given context is active", func() {
			name = testmc
			err = unsetCtx(cmd, []string{name})
			Expect(err).To(BeNil())
		})
		// correct context name and but incorrect target
		It("should return error when target is invalid", func() {
			name = testmc
			targetStr = "incorrect"
			err = unsetCtx(cmd, []string{name})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(invalidTarget))
		})
		// correct context name and target, but context not active
		It("should return error when given context not active", func() {
			name = testUseContext
			targetStr = missionControl
			err = unsetCtx(cmd, []string{name})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(contextNotExistsForTarget, name, targetStr)))
		})
		// correct context name and target, for tmc target, make sure context set inactive
		It("should not return error and context should set inactive", func() {
			name = "test-tmc-context"
			targetStr = missionControl
			err = unsetCtx(cmd, []string{name})
			Expect(err).To(BeNil())

			outputFormat = jsonStr
			buf.Reset()
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			list, err := StringToJSON[ContextListInfo](buf.String())
			Expect(err).To(BeNil())
			exists := false
			for i := range list {
				if list[i].Name == name {
					exists = true
					Expect(list[i].Iscurrent).To(Equal("false"))
				}
			}
			Expect(exists).To(BeTrue(), "context should exist")
		})

		// correct context name and target, for k8s target, make sure context set inactive
		It("should not return error and context should set inactive", func() {
			name = "test-mc"
			targetStr = "k8s"
			err = unsetCtx(cmd, []string{name})
			Expect(err).To(BeNil())
			outputFormat = jsonStr

			buf.Reset()
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			list, err := StringToJSON[ContextListInfo](buf.String())
			Expect(err).To(BeNil())
			exists := false
			for i := range list {
				if list[i].Name == name {
					exists = true
					Expect(list[i].Iscurrent).To(Equal("false"))
				}
			}
			Expect(exists).To(BeTrue(), "context should exist")
		})
	})
})

// StringToJSON is a generic function to convert given json string to struct type
func StringToJSON[T ContextListInfo](jsonStr string) ([]*T, error) {
	var list []*T
	err := json.Unmarshal([]byte(jsonStr), &list)
	return list, err
}

type ContextListInfo struct {
	Endpoint            string `json:"endpoint"`
	Iscurrent           string `json:"iscurrent"`
	Ismanagementcluster string `json:"ismanagementcluster"`
	Kubeconfigpath      string `json:"kubeconfigpath"`
	Kubecontext         string `json:"kubecontext"`
	Name                string `json:"name"`
	Type                string `json:"type"`
}

var _ = Describe("create new context", func() {
	const (
		existingContext    = "test-mc"
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
			ctx             *configtypes.Context
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
			Expect(err).To(BeNil(), "Error while copying tanzu config_ng file for testing")
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
				Expect(ctx.Target).To(ContainSubstring("kubernetes"))
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
				Expect(ctx.Target).To(ContainSubstring("kubernetes"))
				Expect(ctx.ClusterOpts.Context).To(ContainSubstring("test-k8s-context"))
				Expect(ctx.ClusterOpts.Path).To(ContainSubstring(kubeConfig))
			})
		})
		Context("context name already exists", func() {
			It("should return error", func() {
				kubeContext = testKubeContext
				kubeConfig = testKubeConfigPath
				ctxName = existingContext
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
			ctx             *configtypes.Context
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
			Expect(err).To(BeNil(), "Error while copying tanzu config_ng file for testing")
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
				Expect(ctx.Target).To(ContainSubstring(missionControl))
				Expect(ctx.GlobalOpts.Endpoint).To(ContainSubstring(endpoint))
			})
		})
		Context("context name already exists", func() {
			It("should return error", func() {
				endpoint = fakeTMCEndpoint
				ctxName = existingContext
				ctx, err = createNewContext()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(`context "test-mc" already exists`))
			})
		})
	})

	Describe("create context with non-tmc endpoint", func() {
		var (
			tlsServer *ghttp.Server
			err       error
		)
		BeforeEach(func() {
			tlsServer = ghttp.NewTLSServer()
			err = os.Setenv(constants.EULAPromptAnswer, "yes")
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			resetContextCommandFlags()
			os.Unsetenv(constants.EULAPromptAnswer)
			tlsServer.Close()
		})

		Describe("create context with self-managed tmc endpoint", func() {
			var (
				tkgConfigFile   *os.File
				tkgConfigFileNG *os.File
				err             error
				ctx             *configtypes.Context
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
				Expect(err).To(BeNil(), "Error while copying tanzu config_ng file for testing")
			})
			AfterEach(func() {
				os.Unsetenv("TANZU_CONFIG")
				os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
				os.RemoveAll(tkgConfigFile.Name())
				os.RemoveAll(tkgConfigFileNG.Name())
				resetContextCommandFlags()
			})
			Context("with endpoint and context name provided", func() {
				It("should create context with given endpoint and context name", func() {
					selfManaged = true
					endpoint = fakeTMCEndpoint
					ctxName = testContextName
					ctx, err = createNewContext()
					Expect(err).To(BeNil())
					Expect(ctx.Name).To(ContainSubstring("fake-context-name"))
					Expect(ctx.Target).To(ContainSubstring(missionControl))
					Expect(ctx.GlobalOpts.Endpoint).To(ContainSubstring(endpoint))
				})
			})
			Context("context name already exists", func() {
				It("should return error", func() {
					selfManaged = true
					endpoint = fakeTMCEndpoint
					ctxName = existingContext
					ctx, err = createNewContext()
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring(`context "test-mc" already exists`))
				})
			})
		})

	})
	Describe("get Issuer URL from self-managed tmc endpoint", func() {
		Context("endpoint url invalid format", func() {
			It("should return error", func() {

				emptyEP := ""
				_, err := getIssuerURLForTMCEndPoint(emptyEP)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("cannot get issuer URL for empty TMC endpoint"))

				invalidFmtEP := "invalidformat"
				_, err = getIssuerURLForTMCEndPoint(invalidFmtEP)
				Expect(err).ToNot(BeNil())

				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("TMC endpoint URL %s should be of the format host:port", invalidFmtEP)))

				onlyPortEP := ":8888"
				_, err = getIssuerURLForTMCEndPoint(onlyPortEP)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(fmt.Sprintf("TMC endpoint URL %s should be of the format host:port", onlyPortEP)))

			})
		})
		Context("valid endpoint url format", func() {
			It("should return the issuer URL successfully", func() {

				validEP := "test.endpoint.com:554"
				wantIssuerURL := "https://pinniped-supervisor.test.endpoint.com/provider/pinniped"
				issuerURL, err := getIssuerURLForTMCEndPoint(validEP)
				Expect(err).To(BeNil())
				Expect(issuerURL).To(Equal(wantIssuerURL))

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
	selfManaged = false
	skipTLSVerify = false
	endpointCACertPath = ""
}
