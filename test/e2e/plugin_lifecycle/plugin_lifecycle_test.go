// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugin provides plugin command specific E2E test cases
package plugin

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// This test suite covers plugin life cycle use cases for central repository
// it uses local central repo to discovery plugins, for which we need to make sure that
// docker is running and also local central repo is running, start with 'make start-test-central-repo'
// we need to update PATH with tanzu binary
// run the tests with make target 'make start-test-central-repo',
// this make target by default updates the local central repository URL to environment variable TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL
// the use case being covered in this suite are:
// 1. plugin search, install, delete, describe, list (with negative use cases)
// 2. plugin source add/update/list/delete (with negative use cases)
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Plugin-lifecycle]", func() {
	Context("plugin source use cases: tanzu plugin source add, list, update, delete", func() {
		// Test case: add plugin source
		It("add plugin source", func() {
			pluginSourceName = framework.RandomString(5)
			_, err := tf.PluginCmd.AddPluginDiscoverySource(&framework.DiscoveryOptions{Name: pluginSourceName, SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL})
			Expect(err).To(BeNil(), "should not get any error for plugin source add")
		})
		// Test case: list plugin sources and validate plugin source created in previous step
		It("list plugin source and validate previously created plugin source available", func() {
			list, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not get any error for plugin source list")
			Expect(IsPluginSourceExists(list, pluginSourceName)).To(BeTrue())
		})
		// Test case: update plugin source URL
		It("update previously created plugin source URL", func() {
			_, err := tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: pluginSourceName, SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL})
			Expect(err).To(BeNil(), "should not get any error for plugin source update")
		})
		// Test case: (negative test) update plugin source URL with incorrect type (--type)
		It("update previously created plugin source with incorrect type", func() {
			_, err := tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: pluginSourceName, SourceType: framework.SourceType + framework.RandomString(3), URI: e2eTestLocalCentralRepoURL})
			Expect(err.Error()).To(ContainSubstring(framework.UnknownDiscoverySourceType))
		})
		// Test case: (negative test) delete plugin source which is not exists
		It("negative test case: delete plugin source which is not exists", func() {
			_, err := tf.PluginCmd.DeletePluginDiscoverySource(framework.RandomString(5))
			Expect(err.Error()).To(ContainSubstring(framework.DiscoverySourceNotFound))
		})
		// Test case: delete plugin source which was created in previous test case
		It("delete previously created plugin source and validate with plugin source list", func() {
			_, err := tf.PluginCmd.DeletePluginDiscoverySource(pluginSourceName)
			Expect(err).To(BeNil(), "should not get any error for plugin source delete")
			list, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not get any error for plugin source list")
			Expect(IsPluginSourceExists(list, pluginSourceName)).To(BeFalse())
		})
	})
	Context("plugin use cases: tanzu plugin clean, install and describe, list, delete", func() {
		// Test case: clean plugins if any installed already
		It("clean plugins if any installed already", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")
		})
		// Test case: install all plugins from framework.PluginsForLifeCycleTests, and validate the installation by running describe command on each plugin
		It("install plugins and describe each installed plugin", func() {
			for _, plugin := range framework.PluginsForLifeCycleTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				Expect(err).To(BeNil(), "should not get any error for plugin install")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				Expect(err).To(BeNil(), "should not get any error for plugin describe")
				Expect(str).NotTo(BeNil(), "there should be output for plugin describe")
			}
		})
		// Test case: (negative) describe plugin with incorrect target type
		It("plugin describe: describe installed plugin with incorrect target type", func() {
			str, err := tf.PluginCmd.DescribePlugin(framework.PluginsForLifeCycleTests[0].Name, framework.RandomString(5))
			Expect(str).To(BeEmpty(), "stdout should be empty when target type is incorrect for plugin describe")
			Expect(err.Error()).To(ContainSubstring(framework.InvalidTargetSpecified))
		})
		// Test case: (negative) describe plugin with incorrect plugin name
		It("plugin describe: describe installed plugin with incorrect plugin name as input", func() {
			name := framework.RandomString(5)
			str, err := tf.PluginCmd.DescribePlugin(name, framework.PluginsForLifeCycleTests[0].Target)
			Expect(str).To(BeEmpty(), "stdout should be empty when plugin name is incorrect for plugin describe command")
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.UnableToFindPlugin, name)))
		})
		// Test case: list plugins and validate the list plugins output has all plugins which are installed in previous steps
		It("list plugins and check number plugins should be same as installed in previous test", func() {
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(len(framework.PluginsForLifeCycleTests)), "plugins list should return all installed plugins")
			Expect(CheckAllPluginsExists(pluginsList, framework.PluginsForLifeCycleTests)).Should(BeTrue(), "the plugin list output is not same as the plugins being installed")
		})
		// Test case: delete all plugins which are installed, and validate by running list plugin command
		It("delete all plugins and verify with plugin list", func() {
			for _, plugin := range framework.PluginsForLifeCycleTests {
				err := tf.PluginCmd.DeletePlugin(plugin.Name, plugin.Target)
				Expect(err).To(BeNil(), "should not get any error for plugin delete")
			}
			// validate above plugin delete with plugin list command, plugin list should return 0 plugins
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "there should not be any plugins available after uninstall all")
		})
	})
	Context("plugin use cases: tanzu plugin clean, install, clean and list", func() {
		// Test case: clean plugins if any installed already
		It("clean plugins", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")
		})
		// Test case: install all plugins from framework.PluginsForLifeCycleTests
		It("install plugins and describe installed plugins", func() {
			for _, plugin := range framework.PluginsForLifeCycleTests {
				target := plugin.Target
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				Expect(err).To(BeNil(), "should not get any error for plugin install")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				Expect(err).To(BeNil(), "should not get any error for plugin describe")
				Expect(str).NotTo(BeNil(), "there should be output for plugin describe")
			}
			// validate installed plugins count same as number of plugins installed
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(len(framework.PluginsForLifeCycleTests)), "plugins list should return all installed plugins")
		})
		// Test case: run clean plugin command and validate with list plugin command
		It("clean plugins and verify with plugin list", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "there should not be any plugins available after uninstall all")
		})
	})
	Context("plugin use cases: negative test cases for plugin install and plugin delete commands", func() {
		// Test case: install plugin with incorrect value target flag
		It("install plugin with random string for target flag", func() {
			err := tf.PluginCmd.InstallPlugin(framework.PluginsForLifeCycleTests[0].Name, framework.RandomString(5), framework.PluginsForLifeCycleTests[0].Version)
			Expect(err.Error()).To(ContainSubstring(framework.InvalidTargetSpecified))
		})
		// Test case: install plugin with incorrect plugin name
		It("install plugin with random string for target flag", func() {
			name := framework.RandomString(5)
			err := tf.PluginCmd.InstallPlugin(name, "", "")
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.UnableToFindPlugin, name)))
		})
		// Test case: install plugin with incorrect value for flag --version
		It("install plugin with incorrect version", func() {
			for _, plugin := range framework.PluginsForLifeCycleTests {
				if !(plugin.Target == framework.GlobalTarget) {
					err := tf.PluginCmd.InstallPlugin(plugin.Name, plugin.Target, plugin.Version+framework.RandomNumber(3))
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.UnableToFindPluginForTarget, plugin.Name, plugin.Target)))
					break
				}
			}
		})
	})
})
