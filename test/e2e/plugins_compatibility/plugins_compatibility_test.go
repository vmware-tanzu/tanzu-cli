// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugincompatibility provides plugins compatibility E2E test cases
package plugincompatibility

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// This test suite has test case for plugin compatibility use cases
// Goal of these test suites is to validate the plugins built with different Tanzu CLI Runtime Library can co-exists and operate at the same time.
// Below test suite, searches for test plugins (plugin name prefix with "test-plugin-") in the test central repository and
// installs all test plugins, executes basic commands on all installed test plugins, and finally uninstalls all test plugins.
// Each test plugin built using specific Tanzu CLI Runtime library versions.
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Plugin-Compatibility]", func() {
	var (
		tf      *framework.Framework
		plugins []*framework.PluginInfo
	)
	// In the BeforeSuite search for the test-plugin-'s from the TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL test central repository
	BeforeSuite(func() {
		tf = framework.NewFramework()
		// get all plugins with name prefix "test-plugin-"
		plugins = PluginsForCompatibilityTesting(tf)
		Expect(len(plugins)).NotTo(BeZero(), fmt.Sprintf("there are no test-plugin-'s in test central repo:%s , make sure its valid test central repo with test-plugins", os.Getenv(framework.TanzuCliE2ETestCentralRepositoryURL)))
	})
	Context("Uninstall test plugins and verify status using plugin list", func() {
		// Test case: Before installing test plugins, uninstall test plugins (if any installed already) and verify status using plugin list
		It("Uninstall test plugins (if any test plugin installed already) and verify status using plugin list", func() {
			UninstallPlugins(tf, plugins)
			ok := IsAllPluginsUnInstalled(tf, plugins)
			Expect(ok).To(BeTrue(), "All test plugins should be uninstalled")
		})
	})
	Context("Install plugins for plugins compatibility", func() {
		// Test case: install all test plugins
		It("Install all test plugins", func() {
			for _, plugin := range plugins {
				log.Infof("Installing test plugin:%s", plugin)
				err := tf.PluginCmd.InstallPlugin(plugin.Name, "", "")
				Expect(err).To(BeNil(), fmt.Sprintf("should not occur any error while installing the test plugin: %s", plugin))
			}
		})
		// Test case: list all plugins and make sure all above installed test plugins are listed with status "installed"
		It("List plugins and make sure installed plugins are listed", func() {
			ok := IsAllPluginsInstalled(tf, plugins)
			Expect(ok).To(BeTrue(), "All test plugins should be installed and listed in plugin list output as installed")
		})
	})
	Context("Test installed compatibility test-plugins", func() {
		// Test case: run basic commands on installed test plugins, to make sure works/co-exists with other plugins build with different runtime version
		It("run basic commands on the installed test-plugins", func() {
			for _, plugin := range plugins {
				info, err := tf.PluginCmd.ExecuteSubCommand(plugin.Name + " info")
				Expect(err).To(BeNil(), "should not occur any error when plugin info command executed")
				Expect(info).NotTo(BeNil(), "there should be some out for plugin info command executed")
			}
		})
	})
	Context("Test installed compatibility test-plugins with hello-world command", func() {
		// Test case: run hello-world commands on installed test plugins, to make sure works/co-exists with other plugins build with different runtime version
		It("run hello-world commands on the installed test-plugins", func() {
			for _, plugin := range plugins {
				output, err := tf.PluginCmd.ExecuteSubCommand(plugin.Name + " hello-world")
				Expect(err).To(BeNil(), "should not occur any error when plugin hello-world command executed")
				Expect(output).To(ContainSubstring("the command hello-world executed successfully"))
			}
		})
	})
	Context("Uninstall all installed compatibility test-plugins", func() {
		// Test case: uninstall all installed compatibility test-plugins
		It("uninstall all test-plugins", func() {
			for _, plugin := range plugins {
				log.Infof("Uninstalling test plugin: %s", plugin)
				err := tf.PluginCmd.UninstallPlugin(plugin.Name, plugin.Target)
				Expect(err).To(BeNil(), fmt.Sprintf("should not occur any error while uninstalling the test plugin: %s", plugin))
			}
		})
		// Test case: list all plugins and make sure all above uninstalled test plugins should not be listed in the output
		It("List plugins and check uninstalled plugins exists", func() {
			ok := IsAllPluginsUnInstalled(tf, plugins)
			Expect(ok).To(BeTrue(), "All test plugins should be uninstalled")
		})
	})
})
