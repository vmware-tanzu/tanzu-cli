// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package pluginsupplier

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/otiai10/copy"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPluginSupplierSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "plugin supplier test suite")
}

var _ = Describe("GetInstalledPlugins", func() {
	var (
		cdir string
		err  error
		pd1  *cli.PluginInfo
	)
	BeforeEach(func() {
		cdir, err = os.MkdirTemp("", "test-catalog-cache")
		Expect(err).ToNot(HaveOccurred())
		common.DefaultCacheDir = cdir
	})
	AfterEach(func() {
		os.RemoveAll(cdir)
	})

	Context("when no standalone plugins installed", func() {
		BeforeEach(func() {
		})

		It("should return empty plugin list", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(0))
		})
	})
	Context("when a standalone plugins installed", func() {
		BeforeEach(func() {
			pd1, err = fakeInstallPlugin("", "fake-server-plugin1", types.TargetK8s, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed standalone plugin ", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(1))
			Expect(installedPlugins).Should(ContainElement(*pd1))
		})
	})
})

var _ = Describe("GetInstalledPlugins (standalone plugins)", func() {
	var (
		cdir             string
		err              error
		configFile       *os.File
		configFileNG     *os.File
		pd1              *cli.PluginInfo
		pd2              *cli.PluginInfo
		pd3              *cli.PluginInfo
		pd4              *cli.PluginInfo
		pd5              *cli.PluginInfo
		pd6              *cli.PluginInfo
		originalVarValue string
	)
	const (
		tmcContextName = "test-tmc-context"
		k8sContextName = "test-mc"
	)
	BeforeEach(func() {
		cdir, err = os.MkdirTemp("", "test-catalog-cache")
		Expect(err).ToNot(HaveOccurred())
		common.DefaultCacheDir = cdir

		configFile, err = os.CreateTemp("", "config")
		Expect(err).To(BeNil())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), configFile.Name())
		Expect(err).To(BeNil(), "Error while copying tanzu config file for testing")
		os.Setenv("TANZU_CONFIG", configFile.Name())

		configFileNG, err = os.CreateTemp("", "config_ng")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng.yaml"), configFileNG.Name())
		Expect(err).To(BeNil(), "Error while coping tanzu config-ng file for testing")

		originalVarValue = os.Getenv(constants.ConfigVariableStandaloneOverContextPlugins)
	})
	AfterEach(func() {
		os.RemoveAll(cdir)
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())

		os.Setenv(constants.ConfigVariableStandaloneOverContextPlugins, originalVarValue)
	})

	Context("when no standalone or server plugins installed", func() {

		It("should return empty plugin list", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(0))
		})
	})
	Context("when a standalone plugins installed", func() {
		BeforeEach(func() {
			pd1, err = fakeInstallPlugin("", "fake-server-plugin1", types.TargetK8s, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed standalone plugin ", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(1))
			Expect(installedPlugins).Should(ContainElement(*pd1))
		})

		It("should return correct result for IsPluginInstalled", func() {
			isInstalled := IsPluginInstalled("fake-server-plugin1", types.TargetK8s, "v1.0.0")
			Expect(isInstalled).To(BeTrue())
			isInstalled = IsPluginInstalled("random-plugin", types.TargetK8s, "v1.0.0")
			Expect(isInstalled).To(BeFalse())
		})
	})

	Context("when a standalone and server plugin for k8s target installed", func() {
		BeforeEach(func() {
			pd1, err = fakeInstallPlugin("", "fake-server-plugin1", types.TargetK8s, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			contextNameFromConfig := k8sContextName
			pd2, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin2", types.TargetK8s, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed plugins ", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(2)) // server plugins will be migrated as standalone
			Expect(installedPlugins).Should(ContainElement(*pd1))
			Expect(installedPlugins).Should(ContainElement(*pd2))
		})
	})

	Context("when a standalone plugin and server plugin for both tmc and k8s targets installed", func() {
		BeforeEach(func() {
			pd1, err = fakeInstallPlugin("", "fake-server-plugin1", types.TargetK8s, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			contextNameFromConfig := k8sContextName
			pd2, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin2", types.TargetK8s, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			contextNameFromConfig = tmcContextName
			pd3, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin3", types.TargetTMC, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed plugins ", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(3)) // server plugins will be migrated as standalone
			Expect(installedPlugins).Should(ContainElement(*pd1))
			Expect(installedPlugins).Should(ContainElement(*pd2))
			Expect(installedPlugins).Should(ContainElement(*pd3))
		})
	})

	Context("when a standalone plugin and server plugin of the same name and target are installed", func() {
		BeforeEach(func() {
			sharedPluginName := "fake-plugin"
			sharedPluginTarget := types.TargetK8s
			pd1, err = fakeInstallPlugin("", sharedPluginName, sharedPluginTarget, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			contextNameFromConfig := k8sContextName
			pd2, err = fakeInstallPlugin(contextNameFromConfig, sharedPluginName, sharedPluginTarget, "v2.0.0")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return the server plugin only", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(1))
			Expect(installedPlugins).Should(ContainElement(*pd2))
		})
	})

	Context("when multiple standalone plugins and server plugins are installed with some overlap", func() {
		BeforeEach(func() {
			pd1, err = fakeInstallPlugin("", "fake-server-plugin1", types.TargetK8s, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			contextNameFromConfig := k8sContextName
			pd2, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin2", types.TargetK8s, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())

			sharedPluginName := "fake-plugin1"
			sharedPluginTarget := types.TargetK8s
			pd3, err = fakeInstallPlugin("", sharedPluginName, sharedPluginTarget, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())
			pd4, err = fakeInstallPlugin(contextNameFromConfig, sharedPluginName, sharedPluginTarget, "v2.0.0")
			Expect(err).ToNot(HaveOccurred())

			sharedPluginName = "fake-plugin2"
			sharedPluginTarget = types.TargetTMC
			pd5, err = fakeInstallPlugin("", sharedPluginName, sharedPluginTarget, "v1.0.0")
			Expect(err).ToNot(HaveOccurred())
			pd6, err = fakeInstallPlugin(contextNameFromConfig, sharedPluginName, sharedPluginTarget, "v2.0.0")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not return any server plugins and only standalone plugins should be listed", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(4))
			Expect(installedPlugins).Should(ContainElement(*pd1))
			Expect(installedPlugins).Should(ContainElement(*pd2))
			Expect(installedPlugins).ShouldNot(ContainElement(*pd3))
			Expect(installedPlugins).Should(ContainElement(*pd4))
			Expect(installedPlugins).ShouldNot(ContainElement(*pd5))
			Expect(installedPlugins).Should(ContainElement(*pd6))
		})
	})

	Context("with a catalog cache from an older CLI version", func() {
		BeforeEach(func() {
			cdir, err = os.MkdirTemp("", "test-catalog-cache")
			Expect(err).ToNot(HaveOccurred())
			common.DefaultCacheDir = cdir

			err = copy.Copy(
				filepath.Join("..", "fakes", "cache", "catalog_v0.29.yaml"),
				// filepath.Join("..", "fakes", "cache", "catalog.yaml"),
				filepath.Join(common.DefaultCacheDir, "catalog.yaml"))
			Expect(err).To(BeNil(), "Error while copying tanzu catalog file for testing")

			configFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), configFile.Name())
			Expect(err).To(BeNil(), "Error while copying tanzu config file for testing")
			os.Setenv("TANZU_CONFIG", configFile.Name())

			configFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
			err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng.yaml"), configFileNG.Name())
			Expect(err).To(BeNil(), "Error while coping tanzu config-ng file for testing")
		})
		AfterEach(func() {
			os.RemoveAll(cdir)
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(configFile.Name())
			os.RemoveAll(configFileNG.Name())
		})

		It("should find the installed standalone plugin along with server plugins with server plugins migrated in catalog", func() {
			installedStandalonePlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			sort.Sort(cli.PluginInfoSorter(installedStandalonePlugins))
			Expect(len(installedStandalonePlugins)).To(Equal(3))
			Expect(installedStandalonePlugins[0]).Should(Equal(
				cli.PluginInfo{
					Name:                         "cluster",
					Description:                  "cluster functionality",
					Version:                      "v0.0.1",
					BuildSHA:                     "01234567",
					Digest:                       "",
					Group:                        plugin.SystemCmdGroup,
					DocURL:                       "",
					Hidden:                       false,
					CompletionType:               0,
					CompletionArgs:               nil,
					CompletionCommand:            "",
					Aliases:                      nil,
					InstallationPath:             "/Users/test/Library/Application Support/tanzu-cli/cluster/v0.0.1_2ddee7c0a8ecbef610a651bc8d83657fd3438f1038e817b4a7d44f2d0b3bac72_kubernetes",
					Discovery:                    "test-mc",
					Scope:                        "",
					Status:                       "",
					DiscoveredRecommendedVersion: "v0.0.1",
					Target:                       types.TargetK8s,
					PostInstallHook:              nil,
					DefaultFeatureFlags:          map[string]bool{},
					InvokedAs:                    nil,
					SupportedContextType:         nil,
				},
			))
			Expect(installedStandalonePlugins[1]).Should(Equal(
				cli.PluginInfo{
					Name:                         "iam",
					Description:                  "IAM Policies for tmc resources",
					Version:                      "v0.0.1",
					BuildSHA:                     "01234567",
					Digest:                       "",
					Group:                        plugin.ManageCmdGroup,
					DocURL:                       "",
					Hidden:                       false,
					CompletionType:               0,
					CompletionArgs:               nil,
					CompletionCommand:            "",
					Aliases:                      nil,
					InstallationPath:             "/Users/test/Library/Application Support/tanzu-cli/iam/v0.0.1_2de17ef20dfb00dd8bcf5cb61cbce3cbddcd0a71fba858817343188c093cef7c_mission-control",
					Discovery:                    "test-tmc-context",
					Scope:                        "",
					Status:                       "",
					DiscoveredRecommendedVersion: "v0.0.1",
					Target:                       types.TargetTMC,
					PostInstallHook:              nil,
					DefaultFeatureFlags:          map[string]bool{},
					InvokedAs:                    nil,
					SupportedContextType:         nil,
				},
			))
			Expect(installedStandalonePlugins[2]).Should(Equal(
				cli.PluginInfo{
					Name:                         "isolated-cluster",
					Description:                  "Prepopulating images/bundle for internet-restricted environments",
					Version:                      "v0.29.0",
					BuildSHA:                     "e403941f7",
					Digest:                       "",
					Group:                        plugin.RunCmdGroup,
					DocURL:                       "",
					Hidden:                       false,
					CompletionType:               0,
					CompletionArgs:               nil,
					CompletionCommand:            "",
					Aliases:                      nil,
					InstallationPath:             "/Users/test/Library/Application Support/tanzu-cli/isolated-cluster/v0.29.0_78d8b432ca369a161fca39e777aeb81fe63c2ba8b8dd25b1b8270eeab485a2ca_",
					Discovery:                    "",
					Scope:                        "",
					Status:                       "",
					DiscoveredRecommendedVersion: "v0.29.0",
					Target:                       types.TargetUnknown,
					DefaultFeatureFlags:          map[string]bool{},
					PostInstallHook:              nil,
				},
			))
		})
	})
})

func fakeInstallPlugin(contextName, pluginName string, target types.Target, version string) (*cli.PluginInfo, error) {
	cc, err := catalog.NewContextCatalogUpdater(contextName)
	if err != nil {
		return nil, err
	}
	defer cc.Unlock()
	pi := &cli.PluginInfo{
		Name:             pluginName,
		InstallationPath: "/path/to/plugin/" + pluginName + "/" + version,
		Version:          version,
		Hidden:           true,
		Target:           target,
		DefaultFeatureFlags: map[string]bool{
			"test-feature": true,
		},
	}
	err = cc.Upsert(pi)
	if err != nil {
		return nil, err
	}
	return pi, nil
}
