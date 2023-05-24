// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
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
			svr             *configtypes.Server
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

			err = os.Setenv(constants.EULAPromptAnswer, "yes")
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tkgConfigFile.Name())
			os.RemoveAll(tkgConfigFileNG.Name())
			os.Unsetenv(constants.EULAPromptAnswer)
			resetLoginCommandFlags()
		})
		Context("with only kubecontext provided", func() {
			It("should create server with given kubecontext and default kubeconfig path", func() {
				kubeContext = testKubeContext
				name = testServerName
				svr, err = createNewServer()
				Expect(err).To(BeNil())
				Expect(svr.Name).To(ContainSubstring(name))
				Expect(svr.Type).To(BeEquivalentTo(configtypes.ManagementClusterServerType))
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
				Expect(svr.Type).To(BeEquivalentTo(configtypes.ManagementClusterServerType))
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
				svr             *configtypes.Server
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
					Expect(svr.Type).To(BeEquivalentTo(configtypes.GlobalServerType))
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
	})
})

func resetLoginCommandFlags() {
	name = ""
	endpoint = ""
	apiToken = ""
	kubeConfig = ""
	kubeContext = ""
	server = ""
	skipTLSVerify = false
	endpointCACertPath = ""
}
