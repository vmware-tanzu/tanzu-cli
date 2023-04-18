// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugincompatibility_test provides plugins compatibility E2E test cases
package plugincompatibility_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	plugincompatibility "github.com/vmware-tanzu/tanzu-cli/test/e2e/plugins_compatibility"

	plugincompatibility "github.com/vmware-tanzu/tanzu-cli/test/e2e/plugins_compatibility"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// This test suite has test case for plugin compatibility use cases
// Goal of these test suites is to validate the plugins built with different Tanzu CLI Runtime Library can co-exists and operate at the same time.
// Below test suite, searches for test plugins (plugin name prefix with "test-plugin-") in the test central repository and
// installs all test plugins, executes basic commands on all installed test plugins, and finally uninstalls all test plugins.
// Each test plugin built using specific Tanzu CLI Runtime library versions.
// Here are sequence of test cases in below suite:
// a. Before installing test plugins, uninstall test plugins (if any installed already) and verify status using plugin list
// b. install all test plugins from repo gcr.io/eminent-nation-87317/tanzu-cli/test/v1/plugins/plugin-inventory:latest
// c. list all plugins and make sure all above installed test plugins are listed with status "installed"
// d. run basic commands on installed test plugins, to make sure works/co-exists with other plugins build with different runtime version
// e. run hello-world commands on installed test plugins, to make sure works/co-exists with other plugins build with different runtime version
// f. uninstall all installed compatibility test-plugins
// g. list all plugins and make sure all above uninstalled test plugins should not be listed in the output
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Plugin-Compatibility]", func() {

	Context("Uninstall test plugins and verify status using plugin list", func() {
		// Test case: a. Before installing test plugins, uninstall test plugins (if any installed already) and verify status using plugin list
		It("Uninstall test plugins (if any test plugin installed already) and verify status using plugin list", func() {
			plugincompatibility.UninstallPlugins(tf, plugins)
			ok := plugincompatibility.IsAllPluginsUnInstalled(tf, plugins)
			Expect(ok).To(BeTrue(), "All test plugins should be uninstalled")
		})
	})
	Context("Install plugins for plugins compatibility", func() {
		// Test case: b. install all test plugins from repo TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL
		It("Install all test plugins", func() {
			for _, plugin := range plugins {
				log.Infof("Installing test plugin:%s", plugin)
				err := tf.PluginCmd.InstallPlugin(plugin.Name, "", "")
				Expect(err).To(BeNil(), fmt.Sprintf("should not occur any error while installing the test plugin: %s", plugin))
			}
		})
		// Test case: c. list all plugins and make sure all above installed test plugins are listed with status "installed"
		It("List plugins and make sure installed plugins are listed", func() {
			ok := plugincompatibility.IsAllPluginsInstalled(tf, plugins)
			Expect(ok).To(BeTrue(), "All test plugins should be installed and listed in plugin list output as installed")
		})
	})
	Context("Test installed compatibility test-plugins", func() {
		// Test case: d. run basic commands on installed test plugins, to make sure works/co-exists with other plugins build with different runtime version
		It("run basic commands on the installed test-plugins", func() {
			for _, plugin := range plugins {
				info, err := tf.PluginCmd.ExecuteSubCommand(plugin.Name + " info")
				Expect(err).To(BeNil(), "should not occur any error when plugin info command executed")
				Expect(info).NotTo(BeNil(), "there should be some out for plugin info command executed")
			}
		})
	})
	Context("Test installed compatibility test-plugins with hello-world command", func() {
		// Test case: e. run hello-world commands on installed test plugins, to make sure works/co-exists with other plugins build with different runtime version
		It("run hello-world commands on the installed test-plugins", func() {
			for _, plugin := range plugins {
				output, err := tf.PluginCmd.ExecuteSubCommand(plugin.Name + " hello-world")
				Expect(err).To(BeNil(), "should not occur any error when plugin hello-world command executed")
				Expect(output).To(ContainSubstring("the command hello-world executed successfully"))
			}
		})
	})
	Context("Uninstall all installed compatibility test-plugins", func() {
		// Test case: f. uninstall all installed compatibility test-plugins
		It("uninstall all test-plugins", func() {
			for _, plugin := range plugins {
				log.Infof("Uninstalling test plugin: %s", plugin)
				err := tf.PluginCmd.UninstallPlugin(plugin.Name, plugin.Target)
				Expect(err).To(BeNil(), fmt.Sprintf("should not occur any error while uninstalling the test plugin: %s", plugin))
			}
		})
		// Test case: g. list all plugins and make sure all above uninstalled test plugins should not be listed in the output
		It("List plugins and check uninstalled plugins exists", func() {
			ok := plugincompatibility.IsAllPluginsUnInstalled(tf, plugins)
			Expect(ok).To(BeTrue(), "All test plugins should be uninstalled")
		})
	})
})
