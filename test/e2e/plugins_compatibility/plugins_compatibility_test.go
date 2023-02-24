// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugincompatibility provides plugins compatibility E2E test cases
package plugincompatibility

import (
	"fmt"

	"github.com/aunum/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
		plugins []string
	)
	// In the BeforeSuite sets the "features.global.central-repository" flag
	// and searches for the test-plugin-'s from the TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL test central repository
	BeforeSuite(func() {
		tf = framework.NewFramework()
		err := tf.Config.ConfigSetFeatureFlag(framework.CentralRepositoryFeatureFlag, framework.True)
		Expect(err).To(BeNil())
		plugins = PluginsForCompatibilityTesting(tf)
	})
	Context("Install plugins for plugins compatibility", func() {
		// Test case: search for all plugins with name prefix "test-plugin-", install all the test plugins
		It("Install all test plugins", func() {
			for _, plugin := range plugins {
				log.Infof("Installing test plugin:%s", plugin)
				err := tf.PluginCmd.InstallPlugin(plugin)
				Expect(err).To(BeNil(), fmt.Sprintf("should not occur any error while installing the test plugin: %s", plugin))
			}
		})
	})
	Context("Test installed compatibility test-plugins", func() {
		// Test case: search for all plugins with name prefix "test-plugin-", install all the test plugins
		It("run basic commands on the installed test-plugins", func() {
			for _, plugin := range plugins {
				info, err := tf.PluginCmd.ExecuteSubCommand(plugin + " info")
				Expect(err).To(BeNil(), "should not occur any error when plugin info command executed")
				Expect(info).NotTo(BeNil(), "there should be some out for plugin info command executed")
			}
		})
	})
	Context("Uninstall all installed compatibility test-plugins", func() {
		// Test case: uninstall all installed compatibility test-plugins
		It("uninstall all test-plugins", func() {
			for _, plugin := range plugins {
				log.Infof("Uninstalling test plugin: %s", plugin)
				err := tf.PluginCmd.UninstallPlugin(plugin)
				Expect(err).To(BeNil(), fmt.Sprintf("should not occur any error while uninstalling the test plugin: %s", plugin))
			}
		})
	})
})
