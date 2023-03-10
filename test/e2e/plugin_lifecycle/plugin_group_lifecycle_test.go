// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugin provides plugin command specific E2E test cases
package plugin

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// This test suite covers plugin group life cycle use cases for central repository
// it uses local central repo to discovery plugins and plugin groups, for which we need to make sure that
// docker is running and also local central repo is running, start with 'make start-test-central-repo'
// we need to update PATH with tanzu binary
// run the tests with make target 'make start-test-central-repo',
// this make target by default updates the local central repository URL to environment variable TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL
// the use case being covered in this suite are:
//  1. 'plugin group search' there should be some groups always
//  2. 'tanzu plugin install <plugin|all> --group <group>'
//     'tanzu plugin install all --group <group>' will be validated by getting list of plugins belongs to specific plugin group by mapping
//     plugin group version ( vmware-tmc/v9.9.9 ) with plugin version from the plugin search output, tanzu plugin group get <group> is
//     not supported yet.
//
// Use cases not covered:
// 1. 'tanzu plugin group get <group>'
// 2. 'tanzu plugin group describe <group>'

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Plugin-Group-lifecycle]", func() {
	// Use cases:
	// a. clean, install one plugin from a plugin group and validate the installation by running plugin describe.
	// b. install all plugins in a group (to make sure we should be able to install all plugins in a group even when some plugins in group already installed) and validate the installation by running plugin describe for all plugins in a plugin group.
	Context("plugin install from group: install a plugin from a specific plugin group", func() {
		// Test case: clean plugins if any installed already
		It("clean plugins if any installed already", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")
		})
		// Test case: install a plugin from each plugin group and validate the installation
		It("install a plugin from each plugin group and validate the plugin installation by 'plugin describe'", func() {
			for pg := range pluginGroupToPluginListMap {
				plugins := pluginGroupToPluginListMap[pg]
				err := tf.PluginCmd.InstallPluginsFromGroup(plugins[0].Name, pg)
				Expect(err).To(BeNil(), "should not get any error for plugin install from plugin group")

				str, err := tf.PluginCmd.DescribePlugin(plugins[0].Name, plugins[0].Target)
				Expect(err).To(BeNil(), "should not get any error for plugin describe")
				Expect(str).NotTo(BeNil(), "there should be output for plugin describe")
			}
		})
		// Test case: install a plugin from each plugin group and validate the installation with plugin describe
		It("install all plugins in each group", func() {
			for pg := range pluginGroupToPluginListMap {
				err := tf.PluginCmd.InstallPluginsFromGroup("all", pg)
				Expect(err).To(BeNil(), "should not get any error for plugin install from plugin group")
				plugins := pluginGroupToPluginListMap[pg]
				for i := range plugins {
					str, err := tf.PluginCmd.DescribePlugin(plugins[i].Name, plugins[i].Target)
					Expect(err).To(BeNil(), "should not get any error for plugin describe")
					Expect(str).NotTo(BeNil(), "there should be output for plugin describe")
				}
			}
		})
	})
	// Use cases:
	// a. clean installation with "all": clean, install all plugin from a plugin group and validate the installation by running plugin describe.
	// b. clean installation without "all": clean, install all plugin from a plugin group (pass empty string instead of "all") and validate the installation by running plugin describe.
	Context("plugin install from group: perform all plugin installation in a group", func() {
		// Test case: a. clean plugins if any installed already, before running 'tanzu plugin install all --group group_name'
		It("clean plugins if any installed already", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")
		})
		// Test case: a. with command 'tanzu plugin install all --group group_name': when no plugins in a group has installed already: install a plugin from each plugin group and validate the installation with plugin describe
		It("install all plugins in each group", func() {
			for pg := range pluginGroupToPluginListMap {
				err := tf.PluginCmd.InstallPluginsFromGroup("all", pg)
				Expect(err).To(BeNil(), "should not get any error for plugin install from plugin group")
				plugins := pluginGroupToPluginListMap[pg]
				for i := range plugins {
					str, err := tf.PluginCmd.DescribePlugin(plugins[i].Name, plugins[i].Target)
					Expect(err).To(BeNil(), "should not get any error for plugin describe")
					Expect(str).NotTo(BeNil(), "there should be output for plugin describe")
				}
			}
		})
		// Test case: b. clean plugins if any installed already, before running 'tanzu plugin install --group group_name'
		It("clean plugins if any installed already", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")
		})
		// Test case: b. with command 'tanzu plugin install --group group_name': when no plugins in a group has installed already: install a plugin from each plugin group and validate the installation with plugin describe
		It("install all plugins in each group", func() {
			for pg := range pluginGroupToPluginListMap {
				err := tf.PluginCmd.InstallPluginsFromGroup("", pg)
				Expect(err).To(BeNil(), "should not get any error for plugin install from plugin group")
				plugins := pluginGroupToPluginListMap[pg]
				for i := range plugins {
					str, err := tf.PluginCmd.DescribePlugin(plugins[i].Name, plugins[i].Target)
					Expect(err).To(BeNil(), "should not get any error for plugin describe")
					Expect(str).NotTo(BeNil(), "there should be output for plugin describe")
				}
			}
		})
	})
	// Use cases: This context covers NEGATIVE USE CASES:
	// a. incorrect plugin group: clean, install a plugin with incorrect plugin group name
	// b. incorrect plugin name: install a plugin with incorrect name and correct plugin group name.
	Context("plugin install from group: perform all plugin installation in a group", func() {
		// Test case: clean plugins if any installed already
		It("clean plugins if any installed already", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")
		})
		// Test case: a. install a plugin with incorrect plugin group name
		It("install a plugin with incorrect plugin group name", func() {
			for pg := range pluginGroupToPluginListMap {
				plugins := pluginGroupToPluginListMap[pg]
				err := tf.PluginCmd.InstallPluginsFromGroup(plugins[0].Name, framework.RandomString(5))
				Expect(err).NotTo(BeNil(), "plugin installation should fail as plugin group name is incorrect")
				break
			}
		})
		// Test case: b. install a plugin with incorrect plugin name (which is not exists in given plugin group) and correct group name
		It("install a plugin with incorrect plugin name (which does not exist in given group) and correct group name", func() {
			for pg := range pluginGroupToPluginListMap {
				err := tf.PluginCmd.InstallPluginsFromGroup(framework.RandomString(5), pg)
				Expect(err).NotTo(BeNil(), "plugin installation should fail as plugin name is not exists in plugin group")
				break
			}
		})
	})
	// TODO:
	//	1) Use case: 	Install a plugin from a specific group eg: 'vmware-tkg/v0.0.1', plugin: secret target: kubernetes, Version: v0.0.1
	//					Install same plugin different version but same target from another group eg: 'vmware-tkg/v9.9.9', plugin: secret target: kubernetes, Version: v9.9.9
	//			expected results: it should install successfully
	//	2) Use case: 	Install a same plugin (with same version) from a same group, but different targets, eg: group: 'vmware-tkg/v9.9.9' plugin: secret target: kubernetes, and another plugin:  plugin: secret target: mission-control
	//					Install all plugins in the group: tanzu plugin install --group 'vmware-tkg/v9.9.9'
	//		expected results: it should install all plugins, and there should be two secret plugins installed one for kubernetes and another for mission-control with same version.

})
