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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPluginSupplierSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "plugin supplier test suite")
}

var _ = Describe("GetInstalledStandalonePlugins", func() {
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
			installedPlugins, err := GetInstalledStandalonePlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(0))
		})
	})
	Context("when a standalone plugins installed", func() {
		BeforeEach(func() {
			pd1, err = fakeInstallPlugin("", "fake-server-plugin1")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed standalone plugin ", func() {
			installedPlugins, err := GetInstalledStandalonePlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(1))
			Expect(installedPlugins).Should(ContainElement(*pd1))
		})
	})
})

var _ = Describe("GetInstalledServerPlugins", func() {
	var (
		cdir            string
		err             error
		tkgConfigFile   *os.File
		tkgConfigFileNG *os.File
		pd1             *cli.PluginInfo
		pd2             *cli.PluginInfo
	)
	const (
		tmcContextName = "test-tmc-context"
		k8sContextName = "test-mc"
	)
	BeforeEach(func() {
		cdir, err = os.MkdirTemp("", "test-catalog-cache")
		Expect(err).ToNot(HaveOccurred())
		common.DefaultCacheDir = cdir

		tkgConfigFile, err = os.CreateTemp("", "config")
		Expect(err).To(BeNil())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), tkgConfigFile.Name())
		Expect(err).To(BeNil(), "Error while copying tanzu config file for testing")
		os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

		tkgConfigFileNG, err = os.CreateTemp("", "config_ng")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng.yaml"), tkgConfigFileNG.Name())
		Expect(err).To(BeNil(), "Error while coping tanzu config-ng file for testing")
	})
	AfterEach(func() {
		os.RemoveAll(cdir)
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
	})

	Context("when no server/context plugins installed", func() {
		It("should return empty plugin list", func() {
			installedPlugins, err := GetInstalledServerPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(0))
		})
	})
	Context("when a server plugin for k8s target installed", func() {
		BeforeEach(func() {
			contextNameFromConfig := k8sContextName
			pd1, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin1")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed server plugin ", func() {
			installedPlugins, err := GetInstalledServerPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(1))
			Expect(installedPlugins).Should(ContainElement(*pd1))
		})
	})
	Context("when a server plugin for tmc target installed", func() {
		BeforeEach(func() {
			contextNameFromConfig := tmcContextName
			pd1, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin2")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed server plugin ", func() {
			installedPlugins, err := GetInstalledServerPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(1))
			Expect(installedPlugins).Should(ContainElement(*pd1))
		})
	})
	Context("when a server plugin for both tmc and k8s targets installed", func() {
		BeforeEach(func() {
			contextNameFromConfig := k8sContextName
			pd1, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin1")
			Expect(err).ToNot(HaveOccurred())

			contextNameFromConfig = tmcContextName
			pd2, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin2")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed server plugins ", func() {
			installedPlugins, err := GetInstalledServerPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(2))
			Expect(installedPlugins).Should(ContainElement(*pd1))
			Expect(installedPlugins).Should(ContainElement(*pd2))
		})
	})
})

var _ = Describe("GetInstalledPlugins (both standalone and context plugins)", func() {
	var (
		cdir            string
		err             error
		tkgConfigFile   *os.File
		tkgConfigFileNG *os.File
		pd1             *cli.PluginInfo
		pd2             *cli.PluginInfo
		pd3             *cli.PluginInfo
	)
	const (
		tmcContextName = "test-tmc-context"
		k8sContextName = "test-mc"
	)
	BeforeEach(func() {
		cdir, err = os.MkdirTemp("", "test-catalog-cache")
		Expect(err).ToNot(HaveOccurred())
		common.DefaultCacheDir = cdir

		tkgConfigFile, err = os.CreateTemp("", "config")
		Expect(err).To(BeNil())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), tkgConfigFile.Name())
		Expect(err).To(BeNil(), "Error while copying tanzu config file for testing")
		os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

		tkgConfigFileNG, err = os.CreateTemp("", "config_ng")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng.yaml"), tkgConfigFileNG.Name())
		Expect(err).To(BeNil(), "Error while coping tanzu config-ng file for testing")
	})
	AfterEach(func() {
		os.RemoveAll(cdir)
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
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
			pd1, err = fakeInstallPlugin("", "fake-server-plugin1")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed standalone plugin ", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(1))
			Expect(installedPlugins).Should(ContainElement(*pd1))
		})
	})
	Context("when a standalone and server plugin for k8s target installed", func() {
		BeforeEach(func() {
			pd1, err = fakeInstallPlugin("", "fake-server-plugin1")
			Expect(err).ToNot(HaveOccurred())

			contextNameFromConfig := k8sContextName
			pd2, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin2")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed plugins ", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(2))
			Expect(installedPlugins).Should(ContainElement(*pd1))
			Expect(installedPlugins).Should(ContainElement(*pd2))
		})
	})

	Context("when a standalone plugin and server plugin for both tmc and k8s targets installed", func() {
		BeforeEach(func() {
			pd1, err = fakeInstallPlugin("", "fake-server-plugin1")
			Expect(err).ToNot(HaveOccurred())

			contextNameFromConfig := k8sContextName
			pd2, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin2")
			Expect(err).ToNot(HaveOccurred())

			contextNameFromConfig = tmcContextName
			pd3, err = fakeInstallPlugin(contextNameFromConfig, "fake-server-plugin3")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return installed plugins ", func() {
			installedPlugins, err := GetInstalledPlugins()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(installedPlugins)).To(Equal(3))
			Expect(installedPlugins).Should(ContainElement(*pd1))
			Expect(installedPlugins).Should(ContainElement(*pd2))
			Expect(installedPlugins).Should(ContainElement(*pd3))
		})
	})
})

func fakeInstallPlugin(contextName, pluginName string) (*cli.PluginInfo, error) {
	cc, err := catalog.NewContextCatalog(contextName)
	if err != nil {
		return nil, err
	}
	pi := &cli.PluginInfo{
		Name:             pluginName,
		InstallationPath: "/path/to/plugin/" + pluginName,
		Version:          "1.0.0",
		Hidden:           true,
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
