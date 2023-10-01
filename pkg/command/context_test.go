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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	ucprt "github.com/vmware-tanzu/tanzu-plugin-runtime/ucp"
)

func TestCliCmdSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cli core command suite")
}

const (
	existingContext                      = "test-mc"
	testUseContext                       = "test-use-context"
	testUseContextWithValidKubeContext   = "test-use-context-with-valid-kubecontext"
	testUseContextWithInvalidKubeContext = "test-use-context-with-invalid-kubecontext"
	jsonStr                              = "json"
	testmc                               = "test-mc"
	targetK8s                            = "k8s"
	targetMissionControl                 = "mission-control"
	targetUCP                            = "ucp"
	testContextName                      = "test-context"
	testEndpoint                         = "test.ucp.cloud.vmware.com"
	testProject                          = "test-project"
	testSpace                            = "test-space"
)

var _ = Describe("Test tanzu context command", func() {
	var (
		tkgConfigFile   *os.File
		tkgConfigFileNG *os.File
		err             error
		buf             bytes.Buffer
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
		Expect(err).To(BeNil(), "Error while copying tanzu-ng config file for testing")
	})
	AfterEach(func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
	})

	Describe("tanzu context list", func() {
		cmd := &cobra.Command{}
		BeforeEach(func() {
			cmd.SetOut(&buf)
		})
		AfterEach(func() {
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
			cmd.SetOut(&buf)
		})
		AfterEach(func() {
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
			cmd.SetOut(&buf)
		})
		AfterEach(func() {
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
		var kubeconfigFile *os.File

		cmd := &cobra.Command{}

		BeforeEach(func() {
			cmd.SetOut(&buf)

			kubeconfigFile, err = os.CreateTemp("", "kubeconfig")
			kubeconfigPath := kubeconfigFile.Name()
			Expect(err).To(BeNil())
			err = copy.Copy(filepath.Join("..", "fakes", "config", "kubeconfig1.yaml"), kubeconfigPath)
			Expect(err).To(BeNil())

			// repopulate the temp "config/config-ng" with some contexts referencing actual
			// kubeconfig path/contexts
			fmtBytes, err := os.ReadFile(filepath.Join("..", "fakes", "config", "tanzu_config_ng_yaml.tmpl"))
			Expect(err).To(BeNil())
			fileContent := fmt.Sprintf(string(fmtBytes), kubeconfigPath, kubeconfigPath, "foo-context")
			bytesWritten, err := tkgConfigFileNG.WriteAt([]byte(fileContent), 0)
			Expect(err).To(BeNil())
			Expect(bytesWritten).To(Equal(len(fileContent)))

			fmtBytes, err = os.ReadFile(filepath.Join("..", "fakes", "config", "tanzu_config_yaml.tmpl"))
			Expect(err).To(BeNil())
			fileContent = fmt.Sprintf(string(fmtBytes), kubeconfigPath, kubeconfigPath, "foo-context")
			bytesWritten, err = tkgConfigFile.WriteAt([]byte(fileContent), 0)
			Expect(err).To(BeNil())
			Expect(bytesWritten).To(Equal(len(fileContent)))
		})

		AfterEach(func() {
			os.RemoveAll(kubeconfigFile.Name())
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
			targetStr = targetMissionControl
			err = useCtx(cmd, []string{testUseContext})
			Expect(err).To(BeNil())

			cctx, err := config.GetCurrentContext(configtypes.Target(targetStr))
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring(testUseContext))
		})
		It("should return error without setting context if it has invalid kubeconfig/kubecontext reference", func() {
			err = useCtx(cmd, []string{testUseContextWithInvalidKubeContext})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unable to update current kube context:"))

			cctx, err := config.GetCurrentContext(configtypes.TargetK8s)
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring(existingContext))
		})
		It("should set the kubernetes type context if its kubeconfig/kubecontext reference is valid", func() {
			err = useCtx(cmd, []string{testUseContextWithValidKubeContext})
			Expect(err).To(BeNil())

			cctx, err := config.GetCurrentContext(configtypes.TargetK8s)
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring(testUseContextWithValidKubeContext))
		})
		It("should set the kubernetes type context even if it has invalid kubeconfig/kubecontext reference if skip flag is set ", func() {
			os.Setenv(constants.SkipUpdateKubeconfigOnContextUse, "true")
			err = useCtx(cmd, []string{testUseContextWithInvalidKubeContext})
			Expect(err).To(BeNil())

			cctx, err := config.GetCurrentContext(configtypes.TargetK8s)
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring(testUseContextWithInvalidKubeContext))
			os.Unsetenv(constants.SkipUpdateKubeconfigOnContextUse)
		})
	})
	Describe("tanzu context get-token", func() {
		const (
			fakeContextName = "fake-context"
			fakeAccessToken = "fake-access-token"
			fakeEndpoint    = "fake.ucp.cloud.vmware.com"
			fakeIssuer      = "https://fake.issuer.come/auth"
		)
		var err error
		cmd := &cobra.Command{}
		ucpContext := &configtypes.Context{}

		BeforeEach(func() {
			cmd.SetOut(&buf)

			ucpContext = &configtypes.Context{
				Name:   fakeContextName,
				Target: configtypes.TargetUCP,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: fakeEndpoint,
					Auth: configtypes.GlobalServerAuth{
						AccessToken: fakeAccessToken,
						Issuer:      fakeIssuer,
					},
				},
			}
		})
		AfterEach(func() {
			resetContextCommandFlags()
			buf.Reset()
		})
		It("should return error if the context to be used doesn't exist", func() {
			err = getToken(cmd, []string{"non-existing-context"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context non-existing-context not found"))

		})
		It("should return error if the context type is not UCP", func() {
			ucpContext.Target = configtypes.TargetK8s
			err = config.SetContext(ucpContext, false)
			Expect(err).To(BeNil())

			err = getToken(cmd, []string{fakeContextName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(`context "fake-context" is not of type UCP`))

		})
		It("should return error if the access token refresh fails", func() {
			ucpContext.GlobalOpts.Auth.Expiration = time.Now().Add(-time.Hour)

			err = config.SetContext(ucpContext, false)
			Expect(err).To(BeNil())
			err = getToken(cmd, []string{fakeContextName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to refresh the token"))
		})
		It("should print the exec credentials if the access token is valid(not expired)", func() {
			ucpContext.GlobalOpts.Auth.Expiration = time.Now().Add(time.Hour)

			err = config.SetContext(ucpContext, false)
			Expect(err).To(BeNil())
			err = getToken(cmd, []string{fakeContextName})
			Expect(err).To(BeNil())

			execCredential := &clientauthv1.ExecCredential{}
			err = json.NewDecoder(&buf).Decode(execCredential)
			Expect(err).To(BeNil())
			Expect(execCredential.Kind).To(Equal("ExecCredential"))
			Expect(execCredential.APIVersion).To(Equal("client.authentication.k8s.io/v1"))
			Expect(execCredential.Status.Token).To(Equal(fakeAccessToken))
			expectedTime := metav1.NewTime(ucpContext.GlobalOpts.Auth.Expiration).Rfc3339Copy()
			Expect(execCredential.Status.ExpirationTimestamp.Equal(&expectedTime)).To(BeTrue())
		})
	})
	Describe("tanzu context update ucp-active-resource", func() {
		var (
			kubeconfigFilePath *os.File
			err                error
		)
		ucpContext := &configtypes.Context{}
		cmd := &cobra.Command{}

		BeforeEach(func() {
			testKubeconfigFilePath := "../fakes/config/kubeconfig1.yaml"
			kubeconfigFilePath, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			err = copy.Copy(testKubeconfigFilePath, kubeconfigFilePath.Name())
			Expect(err).To(BeNil())

			ucpContext = &configtypes.Context{
				Name:   testContextName,
				Target: configtypes.TargetUCP,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: testEndpoint,
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.tanzu.cloud.vmware.com:443/org/test-org-id",
					Path:     kubeconfigFilePath.Name(),
					Context:  "tanzu-cli-myucp",
				},
				AdditionalMetadata: map[string]interface{}{
					ucprt.OrgIDKey: "test-org-id",
				},
			}
		})
		AfterEach(func() {
			resetContextCommandFlags()
			os.Unsetenv("KUBECONFIG")
			os.RemoveAll(kubeconfigFilePath.Name())
		})
		It("should return error if the context to be updated doesn't exist", func() {
			// set the context in the config
			err = config.SetContext(ucpContext, false)
			Expect(err).To(BeNil())

			err = setUCPCtxActiveResource(cmd, []string{"non-existing-context"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context non-existing-context not found"))

		})
		It("should return error if the context type is not UCP", func() {
			ucpContext.Target = configtypes.TargetK8s
			err = config.SetContext(ucpContext, false)
			Expect(err).To(BeNil())

			err = setUCPCtxActiveResource(cmd, []string{testContextName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(`context "test-context" is not of type UCP`))

		})
		It("should return error if user tries to set space as active resource without providing project name", func() {
			err = config.SetContext(ucpContext, false)
			Expect(err).To(BeNil())

			projectStr = ""
			spaceStr = testSpace
			err = setUCPCtxActiveResource(cmd, []string{testContextName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("space cannot be set without project name. Please provide project name also using --project option"))
		})
		It("should update the UCP context active resource to project given project name and also update the kubeconfig cluster URL accordingly", func() {
			ucpContext.GlobalOpts.Auth.Expiration = time.Now().Add(-time.Hour)

			err = config.SetContext(ucpContext, false)
			Expect(err).To(BeNil())

			projectStr = testProject
			err = setUCPCtxActiveResource(cmd, []string{testContextName})
			Expect(err).To(BeNil())

			ctx, err := config.GetContext(testContextName)
			Expect(err).To(BeNil())
			Expect(ctx.AdditionalMetadata[ucprt.ProjectNameKey]).To(Equal(testProject))
			Expect(ctx.AdditionalMetadata[ucprt.SpaceNameKey]).To(BeEmpty())
			kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
			Expect(err).To(BeNil())
			Expect(kubeconfig.Clusters["tanzu-cli-myucp/current"].Server).To(Equal(ucpContext.ClusterOpts.Endpoint + "/project/" + testProject))
		})
		It("should update the UCP context active resource to space given project and space names and also update the kubeconfig cluster URL accordingly", func() {
			err = config.SetContext(ucpContext, false)
			Expect(err).To(BeNil())

			projectStr = testProject
			spaceStr = testSpace
			err = setUCPCtxActiveResource(cmd, []string{testContextName})
			Expect(err).To(BeNil())

			ctx, err := config.GetContext(testContextName)
			Expect(err).To(BeNil())
			Expect(ctx.AdditionalMetadata[ucprt.ProjectNameKey]).To(Equal(testProject))
			Expect(ctx.AdditionalMetadata[ucprt.SpaceNameKey]).To(Equal(testSpace))
			kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
			Expect(err).To(BeNil())
			Expect(kubeconfig.Clusters["tanzu-cli-myucp/current"].Server).To(Equal(ucpContext.ClusterOpts.Endpoint + "/project/" + testProject + "/space/" + testSpace))
		})
	})

	Describe("tanzu context unset", func() {
		cmd := &cobra.Command{}
		BeforeEach(func() {
			targetStr = ""
			cmd.SetOut(&buf)
		})
		AfterEach(func() {
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
			targetStr = targetMissionControl
			err = unsetCtx(cmd, []string{name})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(contextNotExistsForTarget, name, targetStr)))
		})
		// correct context name and target, for tmc target, make sure context set inactive
		It("should not return error and context should set inactive", func() {
			name = "test-tmc-context"
			targetStr = targetMissionControl
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
		fakeUCPEndpoint    = "https://fake.api.tanzu.cloud.vmware.com"
	)
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
	})

	Describe("create context with kubeconfig", func() {
		AfterEach(func() {
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
		Context("with both kubeconfig and kubecontext provided", func() {
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
				ctxName = existingContext
				ctx, err = createNewContext()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(`context "test-mc" already exists`))
			})
		})
	})
	Describe("create context with tmc endpoint", func() {
		AfterEach(func() {
			resetContextCommandFlags()
		})
		Context("with only endpoint and context name provided", func() {
			It("should create context with given endpoint and context name", func() {
				endpoint = fakeTMCEndpoint
				ctxName = testContextName
				ctx, err = createNewContext()
				Expect(err).To(BeNil())
				Expect(ctx.Name).To(ContainSubstring("fake-context-name"))
				Expect(string(ctx.Target)).To(ContainSubstring(targetMissionControl))
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

	Describe("create context with ucp endpoint", func() {
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

		Describe("create context with ucp endpoint", func() {
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
					endpoint = fakeUCPEndpoint
					ctxName = testContextName
					ctx, err = createNewContext()
					Expect(err).To(BeNil())
					Expect(ctx.Name).To(ContainSubstring("fake-context-name"))
					Expect(string(ctx.Target)).To(ContainSubstring(targetUCP))
					Expect(ctx.GlobalOpts.Endpoint).To(ContainSubstring(endpoint))
				})
			})
			Context("context name already exists", func() {
				It("should return error", func() {
					endpoint = fakeUCPEndpoint
					ctxName = existingContext
					ctx, err = createNewContext()
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring(`context "test-mc" already exists`))
				})
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
	skipTLSVerify = false
	endpointCACertPath = ""
	projectStr = ""
	spaceStr = ""
}
