// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/onsi/gomega/ghttp"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig/fakes"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
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
	yamlStr                              = "yaml"
	testmc                               = "test-mc"
	contextTypeK8s                       = "k8s"
	contextTypeMissionControl            = "mission-control"
	contextTypeTanzu                     = "tanzu"
	testContextName                      = "test-context"
	testEndpoint                         = "test.tanzu.cloud.vmware.com"
	testProject                          = "test-project"
	testProjectID                        = "test-project-id"
	testSpace                            = "test-space"
	testClustergroup                     = "test-clustergroup"
)

const kubeconfigContent1 = `apiVersion: v1
kind: Config
preferences: {}
clusters:
- cluster:
    server: https://example.com/1:6443
  name: cluster-name1
- cluster:
    server: https://example.com/2:6443
  name: cluster-name2
contexts:
- context:
    cluster: cluster-name1
    namespace: default
    user: user-name1
  name: context-name1
- context:
    cluster: cluster-name2
    namespace: default
    user: user-name2
  name: context-name2
current-context: context-name1
users:
- name: user-name1
  user:
    token: token1
- name: user-name2
  user:
    token: token2
  `

const kubeconfigContent2 = `apiVersion: v1
kind: Config
preferences: {}
clusters:
- cluster:
    server: https://example.com/1:6443
  name: cluster-name8
- cluster:
    server: https://example.com/2:6443
  name: cluster-name9
contexts:
- context:
    cluster: cluster-name8
    namespace: default
    user: user-name8
  name: context-name8
- context:
    cluster: cluster-name9
    namespace: default
    user: user-name9
  name: context-name9
current-context: context-name8
users:
- name: user-name8
  user:
    token: token8
- name: user-name9
  user:
    token: token9
`

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
			contextTypeStr = contextTypeK8s
			os.RemoveAll(tkgConfigFileNG.Name())
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			Expect(buf.String()).To(Equal("  NAME  ISACTIVE  TYPE  \n"))

			buf.Reset()
			contextTypeStr = "tmc"
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			Expect(buf.String()).To(Equal("  NAME  ISACTIVE  TYPE  \n"))

		})
		It("should return contexts if tanzu config file has contexts available", func() {
			contextTypeStr = contextTypeK8s
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring("NAME     ISACTIVE  TYPE"))
			Expect(buf.String()).To(ContainSubstring("test-mc  true      kubernetes"))
			Expect(buf.String()).ToNot(ContainSubstring("test-tmc-context"))
			Expect(buf.String()).ToNot(ContainSubstring(testUseContext))

			buf.Reset()
			contextTypeStr = contextTypeK8s
			showAllColumns = true
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			Expect(buf.String()).To(ContainSubstring("NAME     ISACTIVE  TYPE        ENDPOINT       KUBECONFIGPATH  KUBECONTEXT"))
			Expect(buf.String()).To(ContainSubstring("test-mc  true      kubernetes  test-endpoint  test-path       test-mc-context "))

		})
		It("should return contexts in yaml format if tanzu config file has contexts available", func() {
			contextTypeStr = contextTypeK8s
			outputFormat = yamlStr
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			expectedYaml := `
- additionalmetadata:
    isPinnipedEndpoint: true
  endpoint: test-endpoint
  iscurrent: "true"
  ismanagementcluster: "true"
  kubeconfigpath: test-path
  kubecontext: test-mc-context
  name: test-mc
  type: kubernetes`
			Expect(buf.String()).To(ContainSubstring(expectedYaml[1:]))
			Expect(buf.String()).ToNot(ContainSubstring("test-tmc-context"))
			Expect(buf.String()).ToNot(ContainSubstring(testUseContext))
		})

		It("should return with tanzu related columns without --wide", func() {
			buf.Reset()
			contextTypeStr = contextTypeTanzu
			err = listCtx(cmd, nil)
			lines := strings.Split(buf.String(), "\n")
			columnsString := strings.Join(strings.Fields(lines[0]), " ")

			Expect(err).To(BeNil())
			Expect(columnsString).To(Equal("NAME ISACTIVE TYPE PROJECT SPACE"))
		})

		It("should return with tanzu related columns when listing all contexts without --wide", func() {
			buf.Reset()
			contextTypeStr = ""
			err = listCtx(cmd, nil)
			lines := strings.Split(buf.String(), "\n")
			columnsString := strings.Join(strings.Fields(lines[0]), " ")

			Expect(err).To(BeNil())
			Expect(columnsString).To(Equal("NAME ISACTIVE TYPE PROJECT SPACE"))
		})
		It("should return with tanzu related columns when listing all contexts with --wide", func() {
			buf.Reset()
			contextTypeStr = ""
			showAllColumns = true
			err = listCtx(cmd, nil)
			lines := strings.Split(buf.String(), "\n")
			columnsString := strings.Join(strings.Fields(lines[0]), " ")

			Expect(err).To(BeNil())
			Expect(columnsString).To(Equal("NAME ISACTIVE TYPE PROJECT PROJECTID SPACE CLUSTERGROUP ENDPOINT KUBECONFIGPATH KUBECONTEXT"))
		})

		It("should not return tanzu related columns when not listing tanzu contexts without --wide", func() {
			buf.Reset()
			contextTypeStr = contextTypeK8s
			err = listCtx(cmd, nil)
			lines := strings.Split(buf.String(), "\n")
			columnsString := strings.Join(strings.Fields(lines[0]), " ")

			Expect(err).To(BeNil())
			Expect(columnsString).To(Equal("NAME ISACTIVE TYPE"))
		})
		It("should not return tanzu related columns when not listing tanzu contexts with --wide", func() {
			buf.Reset()
			contextTypeStr = contextTypeK8s
			showAllColumns = true
			err = listCtx(cmd, nil)
			lines := strings.Split(buf.String(), "\n")
			columnsString := strings.Join(strings.Fields(lines[0]), " ")

			Expect(err).To(BeNil())
			Expect(columnsString).To(Equal("NAME ISACTIVE TYPE ENDPOINT KUBECONFIGPATH KUBECONTEXT"))
		})

		It("should return tanzu contexts in yaml format if tanzu config file has tanzu contexts", func() {
			contextTypeStr = contextTypeTanzu
			outputFormat = yamlStr
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			// leading space, added for readability, to be trimmed on compare
			expectedYaml := `
- additionalmetadata:
    tanzuOrgID: dummyO
    tanzuProjectName: dummyP
  endpoint: kube-endpoint
  iscurrent: "false"
  ismanagementcluster: "false"
  kubeconfigpath: dummy/path
  kubecontext: dummy-context
  name: test-tanzu-context
  type: tanzu`
			Expect(buf.String()).To(ContainSubstring(expectedYaml[1:]))
			Expect(buf.String()).ToNot(ContainSubstring("test-tmc-context"))
			Expect(buf.String()).ToNot(ContainSubstring(testUseContext))
		})

		It("should return tanzu contexts in JSON format if tanzu config file has tanzu contexts", func() {
			contextTypeStr = contextTypeTanzu
			outputFormat = jsonStr
			err = listCtx(cmd, nil)
			Expect(err).To(BeNil())
			// leading space, added for readability, to be trimmed on compare
			expectedYaml := `
[
  {
    "additionalmetadata": {
      "tanzuOrgID": "dummyO",
      "tanzuProjectName": "dummyP"
    },
    "endpoint": "kube-endpoint",
    "iscurrent": "false",
    "ismanagementcluster": "false",
    "kubeconfigpath": "dummy/path",
    "kubecontext": "dummy-context",
    "name": "test-tanzu-context",
    "type": "tanzu"
  }
]`
			Expect(buf.String()).To(ContainSubstring(expectedYaml[1:]))
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
			// leading space, added for readability, to be trimmed on compare
			expectedYaml := `
name: test-mc
target: kubernetes
contextType: kubernetes
clusterOpts:
    endpoint: test-endpoint
    path: test-path
    context: test-mc-context
    isManagementCluster: true`
			Expect(buf.String()).To(ContainSubstring(expectedYaml[1:]))
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
			Expect(err.Error()).ToNot(ContainSubstring("Deleting the context entry from the config will remove it from the list of tracked contexts. You will need to use `tanzu context create` to re-create this context. Are you sure you want to continue?"))
			Expect(err.Error()).To(ContainSubstring("context fake-mc not found"))
		})
		It("should delete context successfully if the config file has contexts available", func() {
			err = deleteCtx(cmd, []string{existingContext})
			Expect(err).To(BeNil())

			err = getCtx(cmd, []string{existingContext})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context test-mc not found"))
		})
		It("should delete context successfully and also delete(best-effort) the kubecontext in the kubeconfig for tanzu context", func() {
			kubeconfigFilePath, err := os.CreateTemp("", "kubeconfig")
			Expect(err).To(BeNil())

			err = copy.Copy(filepath.Join("..", "fakes", "config", "kubeconfig1.yaml"), kubeconfigFilePath.Name())
			Expect(err).To(BeNil(), "Error while copying kubeconfig config file for testing")

			c, err := config.GetContext("test-tanzu-context")
			Expect(err).To(BeNil())
			c.ClusterOpts.Path = kubeconfigFilePath.Name()
			c.ClusterOpts.Context = "tanzu-cli-mytanzu"

			err = config.SetContext(c, false)
			Expect(err).To(BeNil())

			err = deleteCtx(cmd, []string{"test-tanzu-context"})
			Expect(err).To(BeNil())

			err = getCtx(cmd, []string{"test-tanzu-context"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context test-tanzu-context not found"))

			kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
			Expect(err).To(BeNil())
			Expect(kubeconfig.Clusters["tanzu-cli-mytanzu/current"]).To(BeNil())
			Expect(kubeconfig.Contexts["tanzu-cli-mytanzu"]).To(BeNil())
			Expect(kubeconfig.AuthInfos["tanzu-cli-mytanzu-user"]).To(BeNil())
		})
		It("should delete context successfully and also delete(best-effort) the kubecontext in the kubeconfig for k8s context with pinniped endpoint(specified as context's additionalMetadata)", func() {
			kubeconfigFilePath, err := os.CreateTemp("", "kubeconfig")
			Expect(err).To(BeNil())
			Expect(tkgConfigFileNG.Name()).ToNot(BeEmpty())
			err = copy.Copy(filepath.Join("..", "fakes", "config", "kubeconfig1.yaml"), kubeconfigFilePath.Name())
			Expect(err).To(BeNil(), "Error while copying kubeconfig config file for testing")

			// update the CLI k8s context to point to the existing kubeconfig context to validate the kubeconfig is deleted while deleting the CLI context
			c, err := config.GetContext("test-mc")
			Expect(err).To(BeNil())
			c.ClusterOpts.Path = kubeconfigFilePath.Name()
			c.ClusterOpts.Context = "foo-context"

			err = config.SetContext(c, false)
			Expect(err).To(BeNil())

			err = deleteCtx(cmd, []string{"test-mc"})
			Expect(err).To(BeNil())

			err = getCtx(cmd, []string{"test-mc"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context test-mc not found"))

			kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
			Expect(err).To(BeNil())
			Expect(kubeconfig.Clusters["foo-cluster"]).To(BeNil())
			Expect(kubeconfig.Contexts["foo-context"]).To(BeNil())
			Expect(kubeconfig.AuthInfos["blue-user"]).To(BeNil())
		})
		It("should delete context successfully and should not return error if deleting(best-effort) the kubecontext in the kubeconfig fails", func() {
			kubeconfigFilePath, err := os.CreateTemp("", "kubeconfig")
			Expect(err).To(BeNil())

			err = copy.Copy(filepath.Join("..", "fakes", "config", "kubeconfig1.yaml"), kubeconfigFilePath.Name())
			Expect(err).To(BeNil(), "Error while copying kubeconfig config file for testing")

			c, err := config.GetContext("test-tanzu-context")
			Expect(err).To(BeNil())
			c.ClusterOpts.Path = "non-existent-kubeconfigFile"
			c.ClusterOpts.Context = "non-existing-context"

			err = config.SetContext(c, false)
			Expect(err).To(BeNil())

			err = deleteCtx(cmd, []string{"test-tanzu-context"})
			Expect(err).To(BeNil())

			err = getCtx(cmd, []string{"test-tanzu-context"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context test-tanzu-context not found"))

		})
		It("should delete context successfully and should not return error if kubeconfig details are missing in the context", func() {
			c, err := config.GetContext("test-tanzu-context")
			Expect(err).To(BeNil())
			c.ClusterOpts = nil

			err = config.DeleteContext(c.Name)
			Expect(err).To(BeNil())

			err = config.AddContext(c, false)
			Expect(err).To(BeNil())

			err = deleteCtx(cmd, []string{"test-tanzu-context"})
			Expect(err).To(BeNil())

			err = getCtx(cmd, []string{"test-tanzu-context"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context test-tanzu-context not found"))

		})
	})

	Describe("tanzu context get-token", func() {
		const (
			fakeContextName = "fake-context"
			fakeAccessToken = "fake-access-token"
			fakeEndpoint    = "fake.tanzu.cloud.vmware.com"
			fakeIssuer      = "https://fake.issuer.come/auth"
		)
		var err error
		cmd := &cobra.Command{}
		tanzuContext := &configtypes.Context{}

		BeforeEach(func() {
			cmd.SetOut(&buf)

			tanzuContext = &configtypes.Context{
				Name:        fakeContextName,
				ContextType: configtypes.ContextTypeTanzu,
				AdditionalMetadata: map[string]interface{}{
					config.OrgIDKey: "fakeOrgID",
				},
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: fakeEndpoint,
					Auth: configtypes.GlobalServerAuth{
						AccessToken: fakeAccessToken,
						Issuer:      fakeIssuer,
						Type:        common.APITokenType,
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
		It("should return error if the context type is not tanzu", func() {
			tanzuContext.ContextType = configtypes.ContextTypeK8s
			err = config.SetContext(tanzuContext, false)
			Expect(err).To(BeNil())

			err = getToken(cmd, []string{fakeContextName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(`context "fake-context" is not of type tanzu`))

		})
		It("should return error if the authorization was done using CSP API Token and the access token refresh fails", func() {
			tanzuContext.GlobalOpts.Auth.Expiration = time.Now().Add(-time.Hour)

			err = config.SetContext(tanzuContext, false)
			Expect(err).To(BeNil())
			err = getToken(cmd, []string{fakeContextName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to refresh the token"))
		})
		It("should return error if the authorization was done using id-token(CSP interactive login) and the access token refresh fails", func() {
			tanzuContext.GlobalOpts.Auth.Expiration = time.Now().Add(-time.Hour)
			tanzuContext.GlobalOpts.Auth.Type = common.IDTokenType

			err = config.SetContext(tanzuContext, false)
			Expect(err).To(BeNil())
			err = getToken(cmd, []string{fakeContextName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to refresh the token"))
		})
		It("should print the exec credentials if the access token is valid(not expired)", func() {
			tanzuContext.GlobalOpts.Auth.Expiration = time.Now().Add(time.Hour)

			err = config.SetContext(tanzuContext, false)
			Expect(err).To(BeNil())
			err = getToken(cmd, []string{fakeContextName})
			Expect(err).To(BeNil())

			execCredential := &clientauthv1.ExecCredential{}
			err = json.NewDecoder(&buf).Decode(execCredential)
			Expect(err).To(BeNil())
			Expect(execCredential.Kind).To(Equal("ExecCredential"))
			Expect(execCredential.APIVersion).To(Equal("client.authentication.k8s.io/v1"))
			Expect(execCredential.Status.Token).To(Equal(fakeAccessToken))
			expectedTime := metav1.NewTime(tanzuContext.GlobalOpts.Auth.Expiration).Rfc3339Copy()
			Expect(execCredential.Status.ExpirationTimestamp.Equal(&expectedTime)).To(BeTrue())
		})
	})
	Describe("tanzu context update tanzu-active-resource", func() {
		var (
			kubeconfigFilePath *os.File
			err                error
		)
		tanzuContext := &configtypes.Context{}
		cmd := &cobra.Command{}

		BeforeEach(func() {
			testKubeconfigFilePath := "../fakes/config/kubeconfig1.yaml"
			kubeconfigFilePath, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			err = copy.Copy(testKubeconfigFilePath, kubeconfigFilePath.Name())
			Expect(err).To(BeNil())

			tanzuContext = &configtypes.Context{
				Name:        testContextName,
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: testEndpoint,
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.tanzu.cloud.vmware.com:443/org/test-org-id",
					Path:     kubeconfigFilePath.Name(),
					Context:  "tanzu-cli-mytanzu",
				},
				AdditionalMetadata: map[string]interface{}{
					config.OrgIDKey: "test-org-id",
				},
			}
		})
		AfterEach(func() {
			resetContextCommandFlags()
			os.Unsetenv("KUBECONFIG")
			os.RemoveAll(kubeconfigFilePath.Name())
			//os.Unsetenv(constants.UseStableKubeContextNameForTanzuContext)
		})
		It("should return error if the context to be updated doesn't exist", func() {
			// set the context in the config
			err = config.SetContext(tanzuContext, false)
			Expect(err).To(BeNil())

			err = setTanzuCtxActiveResource(cmd, []string{"non-existing-context"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context non-existing-context not found"))

		})
		It("should return error if the context type is not tanzu", func() {
			tanzuContext.ContextType = configtypes.ContextTypeK8s
			err = config.SetContext(tanzuContext, false)
			Expect(err).To(BeNil())

			err = setTanzuCtxActiveResource(cmd, []string{testContextName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(`context "test-context" is not of type tanzu`))

		})
		It("should return error if user tries to set space as active resource without providing project name", func() {
			err = config.SetContext(tanzuContext, false)
			Expect(err).To(BeNil())

			projectStr = ""
			spaceStr = testSpace
			err = setTanzuCtxActiveResource(cmd, []string{testContextName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("space cannot be set without project. Please set the project"))
		})
		It("should throw an error if the clustergroup and space both are provided by the user when setting active resource", func() {
			err = config.SetContext(tanzuContext, false)
			Expect(err).To(BeNil())

			projectStr = testProject
			spaceStr = testSpace
			clustergroupStr = testClustergroup
			err = setTanzuCtxActiveResource(cmd, []string{testContextName})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("either space or clustergroup can be set as active resource. Please provide either --space or --clustergroup option"))
		})
		Context("when the stable context name for tanzu context option is enabled", func() {
			BeforeEach(func() {
				err = os.Setenv(constants.UseStableKubeContextNameForTanzuContext, "true")
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() {
				err = os.Unsetenv(constants.UseStableKubeContextNameForTanzuContext)
			})
			It("should update the tanzu context active resource to project given project name only and also update the kubeconfig cluster URL accordingly", func() {
				tanzuContext.GlobalOpts.Auth.Expiration = time.Now().Add(-time.Hour)
				err = config.SetContext(tanzuContext, false)
				Expect(err).To(BeNil())

				projectStr = testProject
				projectIDStr = ""
				spaceStr = ""
				clustergroupStr = ""
				err = setTanzuCtxActiveResource(cmd, []string{testContextName})
				Expect(err).To(BeNil())

				ctx, err := config.GetContext(testContextName)
				Expect(err).To(BeNil())
				Expect(ctx.AdditionalMetadata[config.ProjectNameKey]).To(Equal(testProject))
				Expect(ctx.AdditionalMetadata[config.SpaceNameKey]).To(BeEmpty())
				kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
				Expect(err).To(BeNil())
				Expect(kubeconfig.Clusters["tanzu-cli-mytanzu/current"].Server).To(Equal(tanzuContext.ClusterOpts.Endpoint + "/project/" + testProject))
			})
			It("should update the tanzu context active resource to project given project(name and ID) and also update the kubeconfig cluster URL accordingly", func() {
				tanzuContext.GlobalOpts.Auth.Expiration = time.Now().Add(-time.Hour)

				err = config.SetContext(tanzuContext, false)
				Expect(err).To(BeNil())

				projectStr = testProject
				projectIDStr = testProjectID
				err = setTanzuCtxActiveResource(cmd, []string{testContextName})
				Expect(err).To(BeNil())

				ctx, err := config.GetContext(testContextName)
				Expect(err).To(BeNil())
				Expect(ctx.AdditionalMetadata[config.ProjectNameKey]).To(Equal(testProject))
				Expect(ctx.AdditionalMetadata[config.SpaceNameKey]).To(BeEmpty())
				kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
				Expect(err).To(BeNil())
				Expect(kubeconfig.Clusters["tanzu-cli-mytanzu/current"].Server).To(Equal(tanzuContext.ClusterOpts.Endpoint + "/project/" + testProjectID))
			})
			It("should update the tanzu context active resource to space given project(name and ID) and space names and also update the kubeconfig cluster URL accordingly", func() {
				err = config.SetContext(tanzuContext, false)
				Expect(err).To(BeNil())

				projectStr = testProject
				projectIDStr = testProjectID
				spaceStr = testSpace
				err = setTanzuCtxActiveResource(cmd, []string{testContextName})
				Expect(err).To(BeNil())

				ctx, err := config.GetContext(testContextName)
				Expect(err).To(BeNil())
				Expect(ctx.AdditionalMetadata[config.ProjectNameKey]).To(Equal(testProject))
				Expect(ctx.AdditionalMetadata[config.SpaceNameKey]).To(Equal(testSpace))
				kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
				Expect(err).To(BeNil())
				Expect(kubeconfig.Clusters["tanzu-cli-mytanzu/current"].Server).To(Equal(tanzuContext.ClusterOpts.Endpoint + "/project/" + testProjectID + "/space/" + testSpace))
			})
			It("should update the tanzu context active resource to clustergroup given project and clustergroup names and also update the kubeconfig cluster URL accordingly", func() {
				err = config.SetContext(tanzuContext, false)
				Expect(err).To(BeNil())

				projectStr = testProject
				projectIDStr = testProjectID
				spaceStr = ""
				clustergroupStr = testClustergroup
				err = setTanzuCtxActiveResource(cmd, []string{testContextName})
				Expect(err).To(BeNil())

				ctx, err := config.GetContext(testContextName)
				Expect(err).To(BeNil())
				Expect(ctx.AdditionalMetadata[config.ProjectNameKey]).To(Equal(testProject))
				Expect(ctx.AdditionalMetadata[config.SpaceNameKey]).To(Equal(""))
				Expect(ctx.AdditionalMetadata[config.ClusterGroupNameKey]).To(Equal(testClustergroup))
				kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
				Expect(err).To(BeNil())
				Expect(kubeconfig.Clusters["tanzu-cli-mytanzu/current"].Server).To(Equal(tanzuContext.ClusterOpts.Endpoint + "/project/" + testProjectID + "/clustergroup/" + testClustergroup))
			})
		})
		Context("when the stable context name for tanzu context option is not enabled", func() {
			BeforeEach(func() {
				err = os.Unsetenv(constants.UseStableKubeContextNameForTanzuContext)
			})
			It("should update the tanzu context active resource to project given project name only and also update the kubeconfig cluster URL and kubecontext name accordingly", func() {
				tanzuContext.GlobalOpts.Auth.Expiration = time.Now().Add(-time.Hour)
				err = config.SetContext(tanzuContext, false)
				Expect(err).To(BeNil())

				projectStr = testProject
				projectIDStr = ""
				spaceStr = ""
				clustergroupStr = ""
				err = setTanzuCtxActiveResource(cmd, []string{testContextName})
				Expect(err).To(BeNil())

				ctx, err := config.GetContext(testContextName)
				Expect(err).To(BeNil())
				Expect(ctx.AdditionalMetadata[config.ProjectNameKey]).To(Equal(testProject))
				Expect(ctx.AdditionalMetadata[config.SpaceNameKey]).To(BeEmpty())
				kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
				Expect(err).To(BeNil())
				Expect(ctx.ClusterOpts.Context).To(Equal("tanzu-cli-" + testContextName + ":" + testProject))
				Expect(kubeconfig.Contexts[ctx.ClusterOpts.Context]).ToNot(BeNil())
				Expect(kubeconfig.Clusters["tanzu-cli-"+testContextName+":"+testProject].Server).To(Equal(tanzuContext.ClusterOpts.Endpoint + "/project/" + testProject))
			})
			It("should update the tanzu context active resource to project given project(name and ID) and also update the kubeconfig cluster URL and kubecontext name accordingly", func() {
				tanzuContext.GlobalOpts.Auth.Expiration = time.Now().Add(-time.Hour)

				err = config.SetContext(tanzuContext, false)
				Expect(err).To(BeNil())

				projectStr = testProject
				projectIDStr = testProjectID
				err = setTanzuCtxActiveResource(cmd, []string{testContextName})
				Expect(err).To(BeNil())

				ctx, err := config.GetContext(testContextName)
				Expect(err).To(BeNil())
				Expect(ctx.AdditionalMetadata[config.ProjectNameKey]).To(Equal(testProject))
				Expect(ctx.AdditionalMetadata[config.SpaceNameKey]).To(BeEmpty())
				kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
				Expect(err).To(BeNil())
				Expect(ctx.ClusterOpts.Context).To(Equal("tanzu-cli-" + testContextName + ":" + testProject))
				Expect(kubeconfig.Contexts[ctx.ClusterOpts.Context]).ToNot(BeNil())
				Expect(kubeconfig.Clusters["tanzu-cli-"+testContextName+":"+testProject].Server).To(Equal(tanzuContext.ClusterOpts.Endpoint + "/project/" + testProjectID))
			})
			It("should update the tanzu context active resource to space given project(name and ID) and space names and also update the kubeconfig cluster URL and kubecontext name accordingly", func() {
				err = config.SetContext(tanzuContext, false)
				Expect(err).To(BeNil())

				projectStr = testProject
				projectIDStr = testProjectID
				spaceStr = testSpace
				err = setTanzuCtxActiveResource(cmd, []string{testContextName})
				Expect(err).To(BeNil())

				ctx, err := config.GetContext(testContextName)
				Expect(err).To(BeNil())
				Expect(ctx.AdditionalMetadata[config.ProjectNameKey]).To(Equal(testProject))
				Expect(ctx.AdditionalMetadata[config.SpaceNameKey]).To(Equal(testSpace))
				kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
				Expect(err).To(BeNil())
				Expect(ctx.ClusterOpts.Context).To(Equal("tanzu-cli-" + testContextName + ":" + testProject + ":" + testSpace))
				Expect(kubeconfig.Contexts[ctx.ClusterOpts.Context]).ToNot(BeNil())
				Expect(kubeconfig.Clusters["tanzu-cli-"+testContextName+":"+testProject+":"+testSpace].Server).To(Equal(tanzuContext.ClusterOpts.Endpoint + "/project/" + testProjectID + "/space/" + testSpace))
			})
			It("should update the tanzu context active resource to clustergroup given project and clustergroup names and also update the kubeconfig cluster URL and kubecontext name accordingly", func() {
				err = config.SetContext(tanzuContext, false)
				Expect(err).To(BeNil())

				projectStr = testProject
				projectIDStr = testProjectID
				spaceStr = ""
				clustergroupStr = testClustergroup
				err = setTanzuCtxActiveResource(cmd, []string{testContextName})
				Expect(err).To(BeNil())

				ctx, err := config.GetContext(testContextName)
				Expect(err).To(BeNil())
				Expect(ctx.AdditionalMetadata[config.ProjectNameKey]).To(Equal(testProject))
				Expect(ctx.AdditionalMetadata[config.SpaceNameKey]).To(Equal(""))
				Expect(ctx.AdditionalMetadata[config.ClusterGroupNameKey]).To(Equal(testClustergroup))
				kubeconfig, err := clientcmd.LoadFromFile(kubeconfigFilePath.Name())
				Expect(err).To(BeNil())
				Expect(ctx.ClusterOpts.Context).To(Equal("tanzu-cli-" + testContextName + ":" + testProject + ":" + testClustergroup))
				Expect(kubeconfig.Contexts[ctx.ClusterOpts.Context]).ToNot(BeNil())
				Expect(kubeconfig.Clusters["tanzu-cli-"+testContextName+":"+testProject+":"+testClustergroup].Server).To(Equal(tanzuContext.ClusterOpts.Endpoint + "/project/" + testProjectID + "/clustergroup/" + testClustergroup))
			})
		})

	})

	Describe("tanzu context unset", func() {
		var name string
		cmd := &cobra.Command{}
		BeforeEach(func() {
			targetStr = ""
			contextTypeStr = ""
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
			Expect(err.Error()).To(ContainSubstring(invalidTargetErrorForContextCommands))
		})
		// correct context name and but incorrect target
		It("should return error when context type is invalid", func() {
			name = testmc
			contextTypeStr = "incorrect2"
			err = unsetCtx(cmd, []string{name})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(invalidContextType))
		})
		// correct context name and target, but context not active
		It("should return error when given context not active", func() {
			name = testUseContext
			contextTypeStr = contextTypeMissionControl
			err = unsetCtx(cmd, []string{name})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(contextNotExistsForContextType, name, contextTypeStr)))
		})
		// correct context name and target, for tmc target, make sure context set inactive
		It("should not return error and context should set inactive", func() {
			name = "test-tmc-context"
			contextTypeStr = contextTypeMissionControl
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
			contextTypeStr = contextTypeK8s
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
		existingContext                   = "test-mc"
		testKubeContext                   = "test-k8s-context"
		testKubeConfigPath                = "/fake/path/kubeconfig"
		testContextName                   = "fake-context-name"
		fakeTMCEndpoint                   = "tmc.cloud.vmware.com:443"
		fakeTanzuEndpoint                 = "https://fake.tanzu.cloud.vmware.com"
		fakeAlternateTanzuEndpoint        = "https://fake.acme.com"
		expectedAlternateTanzuHubEndpoint = "https://fake.acme.com/hub"
		expectedAlternateTanzuUCPEndpoint = "https://fake.acme.com/ucp"
		expectedAlternateTanzuTMCEndpoint = "https://fake.acme.com"

		expectedTanzuHubEndpoint = "https://api.fake.tanzu.cloud.vmware.com/hub"
		expectedTanzuUCPEndpoint = "https://ucp.fake.tanzu.cloud.vmware.com"
		expectedTanzuTMCEndpoint = "https://ops.fake.tanzu.cloud.vmware.com"
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
				Expect(string(ctx.ContextType)).To(ContainSubstring("kubernetes"))
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
				Expect(string(ctx.ContextType)).To(ContainSubstring("kubernetes"))
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
				Expect(string(ctx.ContextType)).To(ContainSubstring(contextTypeMissionControl))
				Expect(ctx.GlobalOpts.Endpoint).To(ContainSubstring(endpoint))
			})
		})
		Context("with endpoint URL having https/http scheme", func() {
			const httpsURL = "https://cloud.vmware.com"
			const httpURL = "http://cloud.vmware.com"
			It("should return error", func() {
				endpoint = httpsURL
				ctxName = testContextName
				ctx, err = createNewContext()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("TMC endpoint URL https://cloud.vmware.com should not contain http or https scheme. It should be of the format host[:port]"))

				endpoint = httpURL
				ctxName = testContextName
				ctx, err = createNewContext()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("TMC endpoint URL http://cloud.vmware.com should not contain http or https scheme. It should be of the format host[:port]"))
			})
			It("should not return error when E2E test environment variable is set true", func() {
				endpoint = httpsURL
				os.Setenv(constants.E2ETestEnvironment, "true")
				defer os.Unsetenv(constants.E2ETestEnvironment)
				ctxName = testContextName
				ctx, err = createNewContext()
				Expect(err).ToNot(HaveOccurred())

				endpoint = httpURL
				ctxName = testContextName
				ctx, err = createNewContext()
				Expect(err).ToNot(HaveOccurred())
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

	Describe("create context with tanzu endpoint", func() {
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

		Describe("create tanzu context", func() {
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

				// Reset the variables before running tests
				tanzuHubEndpoint, tanzuUCPEndpoint, tanzuTMCEndpoint = "", "", ""

				// Mock the default central configuration reader
				fakeDefaultCentralConfigReader := fakes.CentralConfig{}
				fakeDefaultCentralConfigReader.GetTanzuPlatformSaaSEndpointListReturns([]string{fakeTanzuEndpoint})
				fakeDefaultCentralConfigReader.GetTanzuPlatformEndpointToServiceEndpointMapReturns(centralconfig.TanzuPlatformEndpointToServiceEndpointMap{}, nil)
				centralconfig.DefaultCentralConfigReader = &fakeDefaultCentralConfigReader
			})
			AfterEach(func() {
				os.Unsetenv("TANZU_CONFIG")
				os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
				os.RemoveAll(tkgConfigFile.Name())
				os.RemoveAll(tkgConfigFileNG.Name())
				resetContextCommandFlags()
			})
			Context("with tanzu endpoint and context name provided", func() {
				It("should create context with given endpoint and context name", func() {
					endpoint = fakeTanzuEndpoint
					ctxName = testContextName
					ctx, err = createNewContext()
					Expect(err).To(BeNil())
					Expect(ctx.Name).To(ContainSubstring("fake-context-name"))
					Expect(string(ctx.ContextType)).To(Equal(contextTypeTanzu))
					Expect(ctx.GlobalOpts.Endpoint).To(Equal(expectedTanzuUCPEndpoint))
					Expect(ctx.AdditionalMetadata[config.TanzuHubEndpointKey].(string)).To(Equal(expectedTanzuHubEndpoint))
					Expect(ctx.AdditionalMetadata[config.TanzuMissionControlEndpointKey].(string)).To(Equal(expectedTanzuTMCEndpoint))
					idpType := ctx.AdditionalMetadata[config.TanzuIdpTypeKey].(config.IdpType)
					Expect(string(idpType)).To(ContainSubstring("csp"))
				})
			})
			Context("with alternate tanzu endpoint and context name provided", func() {
				It("should create uaa-based context with given endpoint and context name", func() {
					endpoint = fakeAlternateTanzuEndpoint
					ctxName = testContextName
					contextTypeStr = "tanzu"
					ctx, err = createNewContext()
					Expect(err).To(BeNil())
					Expect(ctx.Name).To(ContainSubstring("fake-context-name"))
					Expect(string(ctx.ContextType)).To(Equal(contextTypeTanzu))
					Expect(ctx.GlobalOpts.Endpoint).To(Equal(expectedAlternateTanzuUCPEndpoint))
					Expect(ctx.AdditionalMetadata[config.TanzuHubEndpointKey].(string)).To(Equal(expectedAlternateTanzuHubEndpoint))
					Expect(ctx.AdditionalMetadata[config.TanzuMissionControlEndpointKey].(string)).To(Equal(expectedAlternateTanzuTMCEndpoint))
					idpType := ctx.AdditionalMetadata[config.TanzuIdpTypeKey].(config.IdpType)
					Expect(string(idpType)).To(ContainSubstring("uaa"))
				})
			})

			Context("context name already exists", func() {
				It("should return error", func() {
					endpoint = fakeTanzuEndpoint
					ctxName = existingContext
					ctx, err = createNewContext()
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring(`context "test-mc" already exists`))
				})
			})
		})

	})
})

var _ = Describe("testing context use", func() {
	const (
		existingContext = "test-mc"
	)
	var (
		tkgConfigFile   *os.File
		tkgConfigFileNG *os.File
		kubeconfigFile  *os.File
		err             error
	)

	BeforeEach(func() {
		tkgConfigFile, err = os.CreateTemp("", "config")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

		tkgConfigFileNG, err = os.CreateTemp("", "config_ng")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())

		kubeconfigFile, err = os.CreateTemp("", "kubeconfig")
		kubeconfigPath := kubeconfigFile.Name()
		Expect(err).To(BeNil())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "kubeconfig1.yaml"), kubeconfigPath)
		Expect(err).To(BeNil())

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
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
		os.RemoveAll(kubeconfigFile.Name())
		resetContextCommandFlags()
	})

	Describe("tanzu context use", func() {
		cmd := &cobra.Command{}

		It("should return error if the context to be used doesn't exist", func() {
			unattended = true
			err = useCtx(cmd, []string{"fake-mc"})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("context fake-mc not found"))

		})
		It("should set the context as the current-context if the config file has context available", func() {
			err = useCtx(cmd, []string{testUseContext})
			Expect(err).To(BeNil())

			cctx, err := config.GetActiveContext(configtypes.ContextTypeTMC)
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring(testUseContext))
		})
		It("should return error without setting context if it has invalid kubeconfig/kubecontext reference", func() {
			err = useCtx(cmd, []string{testUseContextWithInvalidKubeContext})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unable to update current kube context:"))

			cctx, err := config.GetActiveContext(configtypes.ContextTypeK8s)
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring(existingContext))
		})
		It("should set the kubernetes type context if its kubeconfig/kubecontext reference is valid", func() {
			err = useCtx(cmd, []string{testUseContextWithValidKubeContext})
			Expect(err).To(BeNil())

			cctx, err := config.GetActiveContext(configtypes.ContextTypeK8s)
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring(testUseContextWithValidKubeContext))
		})
		It("should set the kubernetes type context even if it has invalid kubeconfig/kubecontext reference if skip flag is set ", func() {
			os.Setenv(constants.SkipUpdateKubeconfigOnContextUse, "true")
			err = useCtx(cmd, []string{testUseContextWithInvalidKubeContext})
			Expect(err).To(BeNil())

			cctx, err := config.GetActiveContext(configtypes.ContextTypeK8s)
			Expect(err).To(BeNil())
			Expect(cctx.Name).To(ContainSubstring(testUseContextWithInvalidKubeContext))
			os.Unsetenv(constants.SkipUpdateKubeconfigOnContextUse)
		})
	})
})

func TestCompletionContext(t *testing.T) {
	ctxK8s1 := &configtypes.Context{
		Name:        "tkg1",
		ContextType: configtypes.ContextTypeK8s,
		ClusterOpts: &configtypes.ClusterServer{Endpoint: "https://example.com/myendpoint/k8s/1"},
	}
	ctxK8s2 := &configtypes.Context{
		Name:        "tkg2",
		ContextType: configtypes.ContextTypeK8s,
		ClusterOpts: &configtypes.ClusterServer{Path: "/example.com/mypath/k8s/2", Context: "ctxTkg2"},
	}
	ctxTMC1 := &configtypes.Context{
		Name:        "tmc1",
		ContextType: configtypes.ContextTypeTMC,
		GlobalOpts:  &configtypes.GlobalServer{Endpoint: "https://example.com/myendpoint/tmc/1"},
	}
	ctxTMC2 := &configtypes.Context{
		Name:        "tmc2",
		ContextType: configtypes.ContextTypeTMC,
		GlobalOpts:  &configtypes.GlobalServer{Endpoint: "https://example.com/myendpoint/tmc/2"},
	}
	ctxTanzu1 := &configtypes.Context{
		Name:        "tanzu1",
		ContextType: configtypes.ContextTypeTanzu,
		ClusterOpts: &configtypes.ClusterServer{Endpoint: "https://example.com/myendpoint/tanzu/1"},
	}
	ctxTanzu2 := &configtypes.Context{
		Name:        "tanzu2",
		ContextType: configtypes.ContextTypeTanzu,
		ClusterOpts: &configtypes.ClusterServer{Endpoint: "https://example.com/myendpoint/tanzu/2"},
	}

	expectedOutForAllCtxs := ctxTanzu1.Name + "\t" + ctxTanzu1.ClusterOpts.Endpoint + "\n"
	expectedOutForAllCtxs += ctxTanzu2.Name + "\t" + ctxTanzu2.ClusterOpts.Endpoint + "\n"
	expectedOutForAllCtxs += ctxK8s1.Name + "\t" + ctxK8s1.ClusterOpts.Endpoint + "\n"
	expectedOutForAllCtxs += ctxK8s2.Name + "\t" + ctxK8s2.ClusterOpts.Path + ":" + ctxK8s2.ClusterOpts.Context + "\n"
	expectedOutForAllCtxs += ctxTMC1.Name + "\t" + ctxTMC1.GlobalOpts.Endpoint + "\n"
	expectedOutForAllCtxs += ctxTMC2.Name + "\t" + ctxTMC2.GlobalOpts.Endpoint + "\n"

	expectedOutForActiveCtxs := ctxK8s1.Name + "\t" + ctxK8s1.ClusterOpts.Endpoint + "\n"
	expectedOutForActiveCtxs += ctxTMC1.Name + "\t" + ctxTMC1.GlobalOpts.Endpoint + "\n"

	expectedOutForTMCActiveCtx := ctxTMC1.Name + "\t" + ctxTMC1.GlobalOpts.Endpoint + "\n"

	expectedOutForTanzuCtxs := ctxTanzu1.Name + "\t" + ctxTanzu1.ClusterOpts.Endpoint + "\n"
	expectedOutForTanzuCtxs += ctxTanzu2.Name + "\t" + ctxTanzu2.ClusterOpts.Endpoint + "\n"

	expectedOutforTypeFlag := compK8sContextType + "\n" + compTanzuContextType + "\n" + compTMCContextType + "\n"

	kubeconfigFile1, err := os.CreateTemp("", "kubeconfig")
	assert.Nil(t, err)
	n, err := kubeconfigFile1.WriteString(kubeconfigContent1)
	assert.Nil(t, err)
	assert.Equal(t, len(kubeconfigContent1), n)

	kubeconfigFile2, err := os.CreateTemp("", "kubeconfig")
	assert.Nil(t, err)
	n, err = kubeconfigFile2.WriteString(kubeconfigContent2)
	assert.Nil(t, err)
	assert.Equal(t, len(kubeconfigContent2), n)

	// Set the default config file to the second file
	os.Setenv("KUBECONFIG", kubeconfigFile2.Name())

	// This is global logic and needs not be tested for each
	// command.  Let's deactivate it.
	os.Setenv("TANZU_ACTIVE_HELP", "no_short_help")

	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		// =====================
		// tanzu context list
		// =====================
		{
			test: "no completion after the list command",
			args: []string{"__complete", "context", "list", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "completion for the --type flag value of the list command",
			args: []string{"__complete", "context", "list", "--type", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutforTypeFlag + ":4\n",
		},
		{
			test: "completion for the --output flag value of the list command",
			args: []string{"__complete", "context", "list", "--output", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForOutputFlag + ":4\n",
		},
		// =====================
		// tanzu context current
		// =====================
		{
			test: "no completion after the current command",
			args: []string{"__complete", "context", "current", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// =====================
		// tanzu context delete
		// =====================
		{
			test: "complete all contexts after the delete command",
			args: []string{"__complete", "context", "delete", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForAllCtxs + ":4\n",
		},
		{
			test: "no completion after the first argument of the delete command",
			args: []string{"__complete", "context", "delete", "tkg1", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// =====================
		// tanzu context get
		// =====================
		{
			test: "complete all contexts after the get command",
			args: []string{"__complete", "context", "get", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForAllCtxs + ":4\n",
		},
		{
			test: "no completion after the first argument of the get command",
			args: []string{"__complete", "context", "get", "tkg1", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "completion for the --output flag value of the get command",
			args: []string{"__complete", "context", "get", "--output", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForOutputFlag + ":4\n",
		},
		// =====================
		// tanzu context use
		// =====================
		{
			test: "complete all contexts after the use command",
			args: []string{"__complete", "context", "use", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForAllCtxs + ":4\n",
		},
		{
			test: "no completion after the first argument of the use command",
			args: []string{"__complete", "context", "use", "tkg1", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// =====================
		// tanzu context unset
		// =====================
		{
			test: "complete active contexts after the unset command",
			args: []string{"__complete", "context", "unset", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForActiveCtxs + ":4\n",
		},
		{
			test: "no completion after the first argument of the unset command",
			args: []string{"__complete", "context", "unset", "tkg1", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "complete active context matching the --type flag for the unset command",
			args: []string{"__complete", "context", "unset", "--type", "tmc", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForTMCActiveCtx + ":4\n",
		},
		{
			test: "completion for the --type flag value of the unset command",
			args: []string{"__complete", "context", "unset", "--type", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutforTypeFlag + ":4\n",
		},
		// =====================
		// tanzu context create
		// =====================
		{
			test: "completion for the arg of the create command",
			args: []string{"__complete", "context", "create", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ Please specify a name for the context\n:4\n",
		},
		{
			test: "completion after one arg of the create command",
			args: []string{"__complete", "context", "create", "tkg1", ""},
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "--\n:6\n",
		},
		{
			test: "completion after one arg of the create command with --endpoint",
			args: []string{"__complete", "context", "create", "tkg1", "--endpoint", "uri", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "completion after one arg of the create command with --kubecontext",
			args: []string{"__complete", "context", "create", "tkg1", "--kubecontext", "ctx", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "completion after one arg of the create command with --kubeconfig",
			args: []string{"__complete", "context", "create", "tkg1", "--kubeconfig", "path", ""},
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "--\n:6\n",
		},
		{
			test: "completion for the --endpoint flag value of the create command",
			args: []string{"__complete", "context", "create", "--endpoint", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ Please enter the endpoint for which to create the context\n:4\n",
		},
		{
			test: "completion for the --api-token flag value of the create command",
			args: []string{"__complete", "context", "create", "--api-token", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ Please enter your api-token (you can instead set the variable TANZU_API_TOKEN)\n:4\n",
		},
		{
			test: "completion for the --kubeconfig flag value of the create command",
			args: []string{"__complete", "context", "create", "--kubeconfig", ""},
			// ":0" is the value of the ShellCompDirectiveDefault which indicates
			// that file completion will be performed
			expected: ":0\n",
		},
		{
			test: "completion for the --kubecontext flag with --kubeconfig",
			args: []string{"__complete", "context", "create", "--kubeconfig", kubeconfigFile1.Name(), "--kubecontext", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "context-name1\tuser-name1@cluster-name1\n" +
				"context-name2\tuser-name2@cluster-name2\n" + ":4\n",
		},
		{
			test: "completion for the --kubecontext flag without --kubeconfig",
			args: []string{"__complete", "context", "create", "--kubecontext", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "context-name8\tuser-name8@cluster-name8\n" +
				"context-name9\tuser-name9@cluster-name9\n" + ":4\n",
		},
		{
			test: "completion for the --type flag",
			args: []string{"__complete", "context", "create", "--type", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutforTypeFlag + ":4\n",
		},
		// =====================
		// tanzu context get-token
		// =====================
		{
			test: "completion for the context get-token tanzu command",
			args: []string{"__complete", "context", "get-token", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForTanzuCtxs + ":4\n",
		},
		{
			test: "no completion after the first argument of the context get-token command",
			args: []string{"__complete", "context", "get-token", "tanzu", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// =====================
		// tanzu context update
		// =====================
		{
			test: "completion for the context update tanzu command",
			args: []string{"__complete", "context", "update", "tanzu-active-resource", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForTanzuCtxs + ":4\n",
		},
		{
			test: "no completion after the first argument of the context update tanzu command",
			args: []string{"__complete", "context", "update", "tanzu-active-resource", "tanzu", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
	}

	// Setup a temporary configuration
	configFile, err := os.CreateTemp("", "config")
	assert.Nil(t, err)
	os.Setenv("TANZU_CONFIG", configFile.Name())
	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(t, err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

	// Add some context, two per target
	_ = config.SetContext(ctxK8s1, true)
	_ = config.SetContext(ctxK8s2, false)
	_ = config.SetContext(ctxTMC1, true)
	_ = config.SetContext(ctxTMC2, false)
	_ = config.SetContext(ctxTanzu1, false)
	_ = config.SetContext(ctxTanzu2, false)

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Nil(err)

			assert.Equal(spec.expected, out.String())

			resetContextCommandFlags()
		})
	}

	os.Unsetenv("TANZU_CONFIG")
	os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
	os.RemoveAll(configFile.Name())
	os.RemoveAll(configFileNG.Name())

	os.Unsetenv("KUBECONFIG")
	os.RemoveAll(kubeconfigFile1.Name())
	os.RemoveAll(kubeconfigFile2.Name())

	os.Unsetenv("TANZU_ACTIVE_HELP")
}

func TestContextCurrentCmd(t *testing.T) {
	ctxK8s := &configtypes.Context{
		Name:        "tkg",
		ContextType: configtypes.ContextTypeK8s,
		ClusterOpts: &configtypes.ClusterServer{
			Endpoint: "https://example.com/myendpoint/k8s/1",
			Context:  "kube-context-name",
			Path:     "/home/user/.kube/config",
		},
	}
	ctxTMC := &configtypes.Context{
		Name:        "tmc",
		ContextType: configtypes.ContextTypeTMC,
		GlobalOpts:  &configtypes.GlobalServer{Endpoint: "https://example.com/myendpoint/tmc/1"},
	}
	ctxTanzuNoOrg := &configtypes.Context{
		Name:        "tanzu",
		ContextType: configtypes.ContextTypeTanzu,
		ClusterOpts: &configtypes.ClusterServer{
			Endpoint: "https://example.com/myendpoint/tanzu/1",
			Context:  "kube-context-name",
			Path:     "/home/user/.kube/config",
		},
	}
	ctxTanzuNoProject := &configtypes.Context{
		Name:        "tanzu",
		ContextType: configtypes.ContextTypeTanzu,
		ClusterOpts: &configtypes.ClusterServer{
			Endpoint: "https://example.com/myendpoint/tanzu/1",
			Context:  "kube-context-name",
			Path:     "/home/user/.kube/config",
		},
		AdditionalMetadata: map[string]interface{}{
			config.OrgIDKey:   "org-id",
			config.OrgNameKey: "org-name",
		},
	}
	ctxTanzuNoSpace := &configtypes.Context{
		Name:        "tanzu",
		ContextType: configtypes.ContextTypeTanzu,
		ClusterOpts: &configtypes.ClusterServer{
			Endpoint: "https://example.com/myendpoint/tanzu/1",
			Context:  "kube-context-name",
			Path:     "/home/user/.kube/config",
		},
		AdditionalMetadata: map[string]interface{}{
			config.OrgIDKey:       "org-id",
			config.OrgNameKey:     "org-name",
			config.ProjectNameKey: "project-name",
			config.ProjectIDKey:   "project-id",
		},
	}
	ctxTanzuSpace := &configtypes.Context{
		Name:        "tanzu",
		ContextType: configtypes.ContextTypeTanzu,
		ClusterOpts: &configtypes.ClusterServer{
			Endpoint: "https://example.com/myendpoint/tanzu/1",
			Context:  "kube-context-name",
			Path:     "/home/user/.kube/config",
		},
		AdditionalMetadata: map[string]interface{}{
			config.OrgIDKey:       "org-id",
			config.OrgNameKey:     "org-name",
			config.ProjectNameKey: "project-name",
			config.ProjectIDKey:   "project-id",
			config.SpaceNameKey:   "space-name",
		},
	}
	ctxTanzuClustergroup := &configtypes.Context{
		Name:        "tanzu",
		ContextType: configtypes.ContextTypeTanzu,
		ClusterOpts: &configtypes.ClusterServer{
			Endpoint: "https://example.com/myendpoint/tanzu/1",
			Context:  "kube-context-name",
			Path:     "/home/user/.kube/config",
		},
		AdditionalMetadata: map[string]interface{}{
			config.OrgIDKey:            "org-id",
			config.OrgNameKey:          "org-name",
			config.ProjectNameKey:      "project-name",
			config.ProjectIDKey:        "project-id",
			config.ClusterGroupNameKey: "clustergroup-name",
		},
	}

	tests := []struct {
		test           string
		activeContexts []*configtypes.Context
		short          bool
		expected       string
	}{
		{
			test:     "no active context",
			expected: "There is no active context\n",
		},
		{
			test:     "no active context short",
			short:    true,
			expected: "There is no active context\n",
		},
		{
			test:           "single k8s active context",
			activeContexts: []*configtypes.Context{ctxK8s},
			expected: `  Name:            tkg
  Type:            kubernetes
  Kube Config:     /home/user/.kube/config
  Kube Context:    kube-context-name
`,
		},
		{
			test:           "single k8s active context short",
			short:          true,
			activeContexts: []*configtypes.Context{ctxK8s},
			expected:       "tkg\n",
		},
		{
			test:           "single tmc active context",
			activeContexts: []*configtypes.Context{ctxTMC},
			expected: `  Name:        tmc
  Type:        mission-control
`,
		},
		{
			test:           "single tmc active context short",
			short:          true,
			activeContexts: []*configtypes.Context{ctxTMC},
			expected:       "tmc\n",
		},
		{
			test:           "both k8s and tmc active contexts",
			activeContexts: []*configtypes.Context{ctxK8s, ctxTMC},
			expected: `  Name:            tkg
  Type:            kubernetes
  Kube Config:     /home/user/.kube/config
  Kube Context:    kube-context-name
`,
		},
		{
			test:           "both k8s and tmc active contexts short",
			short:          true,
			activeContexts: []*configtypes.Context{ctxK8s, ctxTMC},
			expected:       "tkg\n",
		},
		{
			test:           "tanzu no org",
			activeContexts: []*configtypes.Context{ctxTanzuNoOrg, ctxTMC},
			expected: `  Name:            tanzu
  Type:            tanzu
  Kube Config:     /home/user/.kube/config
  Kube Context:    kube-context-name
`,
		},
		{
			test:           "tanzu no org short",
			short:          true,
			activeContexts: []*configtypes.Context{ctxTanzuNoOrg, ctxTMC},
			expected:       "tanzu\n",
		},
		{
			test:           "tanzu just org",
			activeContexts: []*configtypes.Context{ctxTanzuNoProject, ctxTMC},
			expected: `  Name:            tanzu
  Type:            tanzu
  Organization:    org-name (org-id)
  Project:         none set
  Kube Config:     /home/user/.kube/config
  Kube Context:    kube-context-name
`,
		},
		{
			test:           "tanzu just org short",
			short:          true,
			activeContexts: []*configtypes.Context{ctxTanzuNoProject, ctxTMC},
			expected:       "tanzu\n",
		},
		{
			test:           "tanzu just project",
			activeContexts: []*configtypes.Context{ctxTanzuNoSpace},
			expected: `  Name:            tanzu
  Type:            tanzu
  Organization:    org-name (org-id)
  Project:         project-name (project-id)
  Kube Config:     /home/user/.kube/config
  Kube Context:    kube-context-name
`,
		},
		{
			test:           "tanzu just project short",
			short:          true,
			activeContexts: []*configtypes.Context{ctxTanzuNoSpace},
			expected:       "tanzu:project-name\n",
		},
		{
			test:           "tanzu with space",
			activeContexts: []*configtypes.Context{ctxTanzuSpace},
			expected: `  Name:            tanzu
  Type:            tanzu
  Organization:    org-name (org-id)
  Project:         project-name (project-id)
  Space:           space-name
  Kube Config:     /home/user/.kube/config
  Kube Context:    kube-context-name
`,
		},
		{
			test:           "tanzu with space short",
			short:          true,
			activeContexts: []*configtypes.Context{ctxTanzuSpace},
			expected:       "tanzu:project-name:space-name\n",
		},
		{
			test:           "tanzu with clustergroup",
			activeContexts: []*configtypes.Context{ctxTanzuClustergroup, ctxTMC},
			expected: `  Name:             tanzu
  Type:             tanzu
  Organization:     org-name (org-id)
  Project:          project-name (project-id)
  Cluster Group:    clustergroup-name
  Kube Config:      /home/user/.kube/config
  Kube Context:     kube-context-name
`,
		},
		{
			test:           "tanzu with clustergroup short",
			short:          true,
			activeContexts: []*configtypes.Context{ctxTanzuClustergroup, ctxTMC},
			expected:       "tanzu:project-name:clustergroup-name\n",
		},
	}

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			// Setup a temporary configuration
			configFile, err := os.CreateTemp("", "config")
			assert.Nil(err)
			os.Setenv("TANZU_CONFIG", configFile.Name())
			configFileNG, err := os.CreateTemp("", "config_ng")
			assert.Nil(err)
			os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

			// Add some active contexts
			for i := range spec.activeContexts {
				_ = config.SetContext(spec.activeContexts[i], true)
			}

			rootCmd, err := NewRootCmd()
			assert.Nil(err)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			args := []string{"context", "current"}
			if spec.short {
				args = append(args, "--short")
			}
			rootCmd.SetArgs(args)

			err = rootCmd.Execute()
			assert.Nil(err)

			assert.Equal(spec.expected, out.String())

			resetContextCommandFlags()

			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(configFile.Name())
			os.RemoveAll(configFileNG.Name())
		})
	}
}

func resetContextCommandFlags() {
	ctxName = ""
	endpoint = ""
	apiToken = ""
	kubeConfig = ""
	kubeContext = ""
	skipTLSVerify = false
	showAllColumns = false
	endpointCACertPath = ""
	projectStr = ""
	spaceStr = ""
	targetStr = ""
	contextTypeStr = ""
	outputFormat = ""
	shortCtx = false
}

func TestCreateContextWithTanzuTypeAndKubeconfigFlags(t *testing.T) {
	contextTypeStr = contextTypeTanzu
	kubeConfig = "fake-kubeconfig"
	err := createCtx(&cobra.Command{}, []string{})
	assert.NotNil(t, err)
	assert.EqualError(t, err, `the '–-kubeconfig' flag is not applicable when creating a context of type 'tanzu'`)
}
