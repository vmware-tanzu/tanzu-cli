// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package pluginsupplier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

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
		originalVarValue string
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
