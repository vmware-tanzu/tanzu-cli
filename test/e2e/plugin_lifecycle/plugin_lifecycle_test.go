// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// pluginlifecyclee2e provides plugin command specific E2E test cases
package pluginlifecyclee2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/util"
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

	// use case: tanzu plugin source list, update, delete, init
	// a. list plugin sources and validate plugin source created in previous step
	// b. update plugin source URL
	// c. (negative test) delete plugin source which is not exists
	// d. delete plugin source which was created in previous test case
	// e. initialize the default plugin source
	Context("plugin source use cases: tanzu plugin source list, update, delete, init", func() {
		const pluginSourceName = "default"
		// Test case: list plugin sources and validate plugin source created in previous step
		It("list plugin source and validate previously created plugin source available", func() {
			list, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not get any error for plugin source list")
			Expect(framework.IsPluginSourceExists(list, pluginSourceName)).To(BeTrue())
			Expect(list[0].Image).To(Equal(e2eTestLocalCentralRepoURL))
		})
		// Test case: update plugin source URL
		It("update previously created plugin source URL", func() {
			newImage := framework.RandomString(5)
			_, err := tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: pluginSourceName, SourceType: framework.SourceType, URI: newImage})
			Expect(err).ToNot(BeNil(), "should get an error for an invalid image for plugin source update")
			list, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not get any error for plugin source list")
			Expect(framework.IsPluginSourceExists(list, pluginSourceName)).To(BeTrue())
			// The plugin source should note have changed
			Expect(list[0].Image).To(Equal(e2eTestLocalCentralRepoURL))
		})
		// Test case: (negative test) delete plugin source which is not exists
		It("negative test case: delete plugin source which is not exists", func() {
			wrongName := framework.RandomString(5)
			_, err := tf.PluginCmd.DeletePluginDiscoverySource(wrongName)
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.DiscoverySourceNotFound, wrongName)))
		})
		// Test case: delete plugin source which was created in previous test case
		It("delete previously created plugin source and validate with plugin source list", func() {
			_, err := tf.PluginCmd.DeletePluginDiscoverySource(pluginSourceName)
			Expect(err).To(BeNil(), "should not get any error for plugin source delete")
			list, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not get any error for plugin source list")
			Expect(framework.IsPluginSourceExists(list, pluginSourceName)).To(BeFalse())
		})
		// Test case: delete plugin source which was created in previous test case
		It("initialize the default plugin source and validate with plugin source list", func() {
			_, err := tf.PluginCmd.InitPluginDiscoverySource()
			Expect(err).To(BeNil(), "should not get any error for plugin source init")
			list, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not get any error for plugin source list")
			Expect(framework.IsPluginSourceExists(list, pluginSourceName)).To(BeTrue())
			Expect(list[0].Image).To(Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage))
		})
		It("put back the E2E plugin repository", func() {
			_, err := tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: pluginSourceName, SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL})
			Expect(err).To(BeNil(), "should not get any error for plugin source update")
		})
	})
	// use case: tanzu plugin clean, install and describe, list, delete
	// a. clean plugins if any installed already
	// b. install all plugins from framework.PluginsForLifeCycleTests, and validate the installation by running describe command on each plugin
	// c. (negative) describe plugin with incorrect target type
	// d. (negative) describe plugin with incorrect plugin name
	// e. list plugins and validate the list plugins output has all plugins which are installed in previous steps
	// f. delete all plugins which are installed, and validate by running list plugin command
	Context("plugin use cases: tanzu plugin clean, install and describe, list, delete", func() {
		// Test case: clean plugins if any installed already
		It("clean plugins if any installed already", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")
		})
		// Test case: install all plugins from framework.PluginsForLifeCycleTests, and validate the installation by running describe command on each plugin
		It("install plugins and describe each installed plugin", func() {
			for _, plugin := range util.PluginsForLifeCycleTests {
				target := plugin.Target
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				Expect(err).To(BeNil(), "should not get any error for plugin install")

				pd, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target, framework.GetJsonOutputFormatAdditionalFlagFunction())
				Expect(err).To(BeNil(), framework.PluginDescribeShouldNotThrowErr)
				Expect(len(pd)).To(Equal(1), framework.PluginDescShouldExist)
				Expect(pd[0].Name).To(Equal(plugin.Name), framework.PluginNameShouldMatch)
			}
		})
		It("explicitly verify that the plugin using a sha reference was installed", func() {
			pd, err := tf.PluginCmd.DescribePlugin("plugin-with-sha", "global", framework.GetJsonOutputFormatAdditionalFlagFunction())
			Expect(err).To(BeNil(), framework.PluginDescribeShouldNotThrowErr)
			Expect(len(pd)).To(Equal(1), framework.PluginDescShouldExist)
			Expect(pd[0].Name).To(Equal("plugin-with-sha"), framework.PluginNameShouldMatch)
		})
		// Test case: (negative) describe plugin with incorrect target type
		It("plugin describe: describe installed plugin with incorrect target type", func() {
			_, err := tf.PluginCmd.DescribePlugin(util.PluginsForLifeCycleTests[0].Name, framework.RandomString(5), framework.GetJsonOutputFormatAdditionalFlagFunction())
			Expect(err.Error()).To(ContainSubstring(framework.InvalidTargetSpecified))
		})
		// Test case: (negative) describe plugin with incorrect plugin name
		It("plugin describe: describe installed plugin with incorrect plugin name as input", func() {
			name := framework.RandomString(5)
			_, err := tf.PluginCmd.DescribePlugin(name, util.PluginsForLifeCycleTests[0].Target, framework.GetJsonOutputFormatAdditionalFlagFunction())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.UnableToFindPlugin, name)))
		})
		// Test case: list plugins and validate the list plugins output has all plugins which are installed in previous steps
		It("list plugins and check number plugins should be same as installed in previous test", func() {
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(len(util.PluginsForLifeCycleTests)), "plugins list should return all installed plugins")
			Expect(framework.CheckAllPluginsExists(pluginsList, util.PluginsForLifeCycleTests)).Should(BeTrue(), "the plugin list output is not same as the plugins being installed")
		})
		// Test case: delete all plugins which are installed for a specific target, and validate by running list plugin command
		It("delete all plugins for target kubernetes and verify with plugin list", func() {
			// count how many plugins are installed that are not for the k8s target
			count := 0
			for _, plugin := range util.PluginsForLifeCycleTests {
				if plugin.Target != framework.KubernetesTarget {
					count++
				}
			}

			err := tf.PluginCmd.DeletePlugin(cli.AllPlugins, framework.KubernetesTarget)
			Expect(err).To(BeNil(), "should not get any error for plugin delete all")

			// validate above plugin delete with plugin list command, plugin list should return 0 plugins of target kubernetes
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(count), "incorrect number of installed plugins after deleting")
			for _, plugin := range pluginsList {
				Expect(plugin.Target).To(Not(Equal(framework.KubernetesTarget)))
			}
		})
		// Test case: delete all plugins which are installed, and validate by running list plugin command
		It("delete all remaining plugins and verify with plugin list", func() {
			for _, plugin := range util.PluginsForLifeCycleTests {
				if plugin.Target != framework.KubernetesTarget {
					// We don't delete kubernetes plugins since they were all deleted in the previous step
					err := tf.PluginCmd.DeletePlugin(plugin.Name, plugin.Target)
					Expect(err).To(BeNil(), "should not get any error for plugin delete")
				}
			}
			// validate above plugin delete with plugin list command, plugin list should return 1 plugin (the essential plugin)
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			// This is because essential plugins will always be installed
			Expect(len(pluginsList)).Should(Equal(1), "there should be only one plugin available after uninstall all this is because essential plugins will always be installed")
		})
	})
	// use case: tanzu plugin clean, install, clean and list
	// a. clean plugins if any installed already
	// b. install all plugin
	// c. run clean plugin command and validate with list plugin command
	Context("plugin use cases: tanzu plugin clean, install, clean and list", func() {
		// Test case: clean plugins if any installed already
		It("clean plugins", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")
		})
		// Test case: install all plugins from framework.PluginsForLifeCycleTests
		It("install plugins and describe installed plugins", func() {
			for _, plugin := range util.PluginsForLifeCycleTests {
				target := plugin.Target
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				Expect(err).To(BeNil(), "should not get any error for plugin install")

				pd, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target, framework.GetJsonOutputFormatAdditionalFlagFunction())
				Expect(err).To(BeNil(), framework.PluginDescribeShouldNotThrowErr)
				Expect(len(pd)).To(Equal(1), framework.PluginDescShouldExist)
				Expect(pd[0].Name).To(Equal(plugin.Name), framework.PluginNameShouldMatch)
			}
			// validate installed plugins count same as number of plugins installed
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(len(util.PluginsForLifeCycleTests)), "plugins list should return all installed plugins")
		})
		// Test case: run clean plugin command and validate with list plugin command
		It("clean plugins and verify with plugin list", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "there should not be any plugins available after uninstall all")
		})
	})

	// use case: negative test cases for plugin install and plugin delete commands
	// a. install plugin with incorrect value target flag
	// b. install plugin with incorrect plugin name
	// c. install plugin with incorrect value for flag --version
	Context("plugin use cases: negative test cases for plugin install and plugin delete commands", func() {
		// Test case: a. install plugin with incorrect value target flag
		It("install plugin with random string for target flag", func() {
			err := tf.PluginCmd.InstallPlugin(util.PluginsForLifeCycleTests[0].Name, framework.RandomString(5), util.PluginsForLifeCycleTests[0].Version)
			Expect(err.Error()).To(ContainSubstring(framework.InvalidTargetSpecified))
		})
		// Test case: b. install plugin with incorrect plugin name
		It("install plugin with random string for target flag", func() {
			name := framework.RandomString(5)
			err := tf.PluginCmd.InstallPlugin(name, "", "")
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.UnableToFindPlugin, name)))
		})
		// Test case: c. install plugin with incorrect value for flag --version
		It("install plugin with incorrect version", func() {
			for _, plugin := range util.PluginsForLifeCycleTests {
				if !(plugin.Target == framework.GlobalTarget) {
					version := plugin.Version + framework.RandomNumber(3)
					err := tf.PluginCmd.InstallPlugin(plugin.Name, plugin.Target, version)
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.UnableToFindPluginWithVersionForTarget, plugin.Name, version, plugin.Target)))
					break
				}
			}
		})
	})

	// use case: test plugin commands help message
	// a. plugin install help message
	// b. plugin describe help message
	// c. plugin upgrade help message
	// d. plugin delete help message
	Context("plugin use cases: negative test cases for plugin install and plugin delete commands", func() {
		// Test case: a. plugin install help message
		It("tanzu plugin install help message", func() {
			out, _, err := tf.PluginCmd.RunPluginCmd("install -h")
			Expect(err).To(BeNil())
			Expect(out).To(ContainSubstring("tanzu plugin install [PLUGIN_NAME] [flags]"))
		})
		// Test case: b. plugin describe help message
		It("tanzu plugin describe help message", func() {
			out, _, err := tf.PluginCmd.RunPluginCmd("describe -h")
			Expect(err).To(BeNil())
			Expect(out).To(ContainSubstring("tanzu plugin describe PLUGIN_NAME [flags]"))
		})
		// Test case: c. plugin upgrade help message
		It("tanzu plugin upgrade help message", func() {
			out, _, err := tf.PluginCmd.RunPluginCmd("upgrade -h")
			Expect(err).To(BeNil())
			Expect(out).To(ContainSubstring("tanzu plugin upgrade PLUGIN_NAME [flags]"))
		})
		// Test case: d. plugin uninstall help message
		It("tanzu plugin uninstall help message", func() {
			out, _, err := tf.PluginCmd.RunPluginCmd("uninstall -h")
			Expect(err).To(BeNil())
			Expect(out).To(ContainSubstring("tanzu plugin uninstall PLUGIN_NAME [flags]"))
		})
	})

	// use case: tanzu plugin install (with different shorthand version like vMAJOR, vMAJOR.MINOR), verify with describe command
	// a. clean plugins if any installed already
	// b. install plugins with different shorthand version like vMAJOR, vMAJOR.MINOR and validate with describe command
	// c. run clean plugin command and validate with list plugin command
	Context("plugin use cases: tanzu plugin clean, install (with different shorthand version like vMAJOR, vMAJOR.MINOR), verify, clean", func() {
		// Test case: clean plugins if any installed already
		It("clean plugins", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")
		})
		// Test case: install plugins with different shorthand version like vMAJOR, vMAJOR.MINOR
		It("install plugins and describe installed plugins", func() {
			for _, testcase := range PluginsMultiVersionInstallTests {
				err := tf.PluginCmd.InstallPlugin(testcase.plugInfo.Name, testcase.plugInfo.Target, testcase.plugInfo.Version)
				if testcase.err != "" {
					Expect(err.Error()).To(ContainSubstring(testcase.err))
				} else {
					Expect(err).To(BeNil(), "should not get any error for plugin install")
					pd, err := tf.PluginCmd.DescribePlugin(testcase.plugInfo.Name, testcase.plugInfo.Target, framework.GetJsonOutputFormatAdditionalFlagFunction())
					Expect(err).To(BeNil(), framework.PluginDescribeShouldNotThrowErr)
					Expect(len(pd)).To(Equal(1), framework.PluginDescShouldExist)
					Expect(pd[0].Name).To(Equal(testcase.plugInfo.Name), framework.PluginNameShouldMatch)
					Expect(pd[0].Version).To(Equal(testcase.expectedInstalledVersion), framework.PluginNameShouldMatch)
				}
			}
		})
		// Test case: run clean plugin command and validate with list plugin command
		It("clean plugins and verify with plugin list", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "there should not be any plugins available after uninstall all")
		})
	})
})
