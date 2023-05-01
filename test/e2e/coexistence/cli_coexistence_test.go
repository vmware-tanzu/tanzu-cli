// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package coexistence_test

import (
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	pluginlifecyclee2e "github.com/vmware-tanzu/tanzu-cli/test/e2e/plugin_lifecycle"
)

var _ = ginkgo.Describe("CLI Coexistence Tests", func() {

	ginkgo.BeforeEach(func() {
		ginkgo.By("Cleaning up Tanzu CLI and all related files before each test")
		ginkgo.By("Uninstall Tanzu CLI")
		err := tf.UninstallTanzuCLI()
		gomega.Expect(err).To(gomega.BeNil())
		ginkgo.By("Cleanup completed")
	})

	ginkgo.Context("Install legacy Tanzu CLI and new Tanzu CLI", func() {
		ginkgo.It("Both Tanzu CLIs coexist", func() {
			ginkgo.By("Install the legacy Tanzu CLI")
			err := tf.InstallLegacyTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err := tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Install the new Tanzu CLI to coexist along with legacy Tanzu CLI")
			err = tf.InstallNewTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of new Tanzu CLI")
			version, err = tf.CLIVersion(framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(version).To(gomega.ContainSubstring(newTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err = tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())
		})
	})

	ginkgo.Context("Plugins lifecycle: list, install, search, describe", func() {
		ginkgo.It("Use case 1: When new Tanzu CLI is installed to coexist along with legacy Tanzu CLI, Plugins installed using legacy Tanzu CLI and new Tanzu CLI should be available and installed when accessed using legacy Tanzu CLI and new Tanzu CLI", func() {
			ginkgo.By("Install the legacy Tanzu CLI")
			err := tf.InstallLegacyTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err := tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Installing few plugins using legacy Tanzu CLI")
			for _, plugin := range PluginsForLegacyTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install for legacy tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe for legacy tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe for legacy tanzu cli")
			}

			ginkgo.By("Verify the installed plugins using legacy Tanzu CLI")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list for legacy tanzu cli")
			gomega.Expect(pluginlifecyclee2e.CheckAllLegacyPluginsExists(pluginsList, PluginsForLegacyTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using legacy tanzu cli is not same as the plugins being installed using legacy tanzu cli")

			ginkgo.By("Install the new Tanzu CLI which coexist with the legacy Tanzu CLI")
			err = tf.InstallNewTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of new Tanzu CLI")
			version, err = tf.CLIVersion(framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(version).To(gomega.ContainSubstring(newTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			// set up the test local central repository host CA cert in the config file
			setTestLocalCentralRepoCertConfig([]framework.E2EOption{framework.WithTanzuCommandPrefix(framework.TzPrefix)})

			// set up the local central repository discovery image signature public key path
			os.Setenv("TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH", e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath)

			ginkgo.By("Update plugin discovery source with test central repo")
			_, err = tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: "default", SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL}, framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin source update")

			ginkgo.By("Search plugins and make sure there are plugins available using new Tanzu CLI")
			pluginsSearchList := pluginlifecyclee2e.SearchAllPlugins(tf, framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(len(pluginsSearchList)).Should(gomega.BeNumerically(">", 0))

			ginkgo.By("Installing few plugins using new Tanzu CLI")
			for _, plugin := range PluginsForNewTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version, framework.WithTanzuCommandPrefix(framework.TzPrefix))
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install for new tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target, framework.WithTanzuCommandPrefix(framework.TzPrefix))
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe for new tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe for new tanzu cli")
			}

			ginkgo.By("Installing few plugins using new Tanzu CLI")
			pluginsList, err = tf.PluginCmd.ListPlugins(framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list for new tanzu cli")
			gomega.Expect(framework.CheckAllPluginsExists(pluginsList, PluginsForNewTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using new tanzu cli is not same as the plugins being installed using new tanzu cli")

			ginkgo.By("Verify the installed plugins using legacy Tanzu CLI")
			pluginsList, err = tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list using legacy tanzu cli")
			gomega.Expect(pluginlifecyclee2e.CheckAllLegacyPluginsExists(pluginsList, PluginsForLegacyTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using legacy tanzu cli is not same as the plugins being installed using legacy tanzu cli")
		})

		ginkgo.It("Use case 2: When new Tanzu CLI is installed to override the installation of legacy Tanzu CLI, Plugins installed using legacy Tanzu CLI and new Tanzu CLI should be available and installed when accessed using new Tanzu CLI", func() {
			ginkgo.By("Install the legacy Tanzu CLI")
			err := tf.InstallLegacyTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err := tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Installing few plugins using legacy Tanzu CLI")
			for _, plugin := range PluginsForLegacyTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install using legacy tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe using legacy tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe using legcy tanzu cli")
			}

			ginkgo.By("Verify the installed plugins using legacy Tanzu CLI")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list")
			gomega.Expect(pluginlifecyclee2e.CheckAllLegacyPluginsExists(pluginsList, PluginsForLegacyTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using legacy tanzu cli is not same as the plugins being installed using legacy tanzu cli")

			ginkgo.By("Install the new Tanzu CLI overriding the legacy Tanzu CLI")
			err = tf.InstallNewTanzuCLI(framework.WithOverride(true))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of new Tanzu CLI")
			version, err = tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(newTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			// set up the test local central repository host CA cert in the config file
			setTestLocalCentralRepoCertConfig([]framework.E2EOption{})

			// set up the local central repository discovery image signature public key path
			os.Setenv("TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH", e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath)

			ginkgo.By("Update plugin discovery source with test central repo")
			_, err = tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: "default", SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL})
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin source update")

			ginkgo.By("Search plugins and make sure there are plugins available using new Tanzu CLI")
			pluginsSearchList := pluginlifecyclee2e.SearchAllPlugins(tf)
			gomega.Expect(len(pluginsSearchList)).Should(gomega.BeNumerically(">", 0))

			ginkgo.By("Installing few plugins using new Tanzu CLI")
			for _, plugin := range PluginsForNewTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install using new tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe using new tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe using new tanzu cli")
			}

			ginkgo.By("Verify the installed plugins using new Tanzu CLI")
			pluginsList, err = tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list using new tanzu cli")
			gomega.Expect(framework.CheckAllPluginsExists(pluginsList, PluginsForNewTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using new tanzu cli is not same as the plugins being installed using new tanzu cli")
		})

		ginkgo.It("Use case 3: After Reinstalling new Tanzu CLI to override the installation of legacy Tanzu CLI, Plugins installed using legacy Tanzu CLI and new Tanzu CLI should be available and installed using new tanzu cli", func() {
			ginkgo.By("Install the legacy Tanzu CLI")
			err := tf.InstallLegacyTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err := tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Install few plugins using legacy Tanzu CLI")
			for _, plugin := range PluginsForLegacyTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install using legacy tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe using legacy tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe using legcy tanzu cli")
			}

			ginkgo.By("Verify the installed plugins using legacy Tanzu CLI")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list")
			gomega.Expect(pluginlifecyclee2e.CheckAllLegacyPluginsExists(pluginsList, PluginsForLegacyTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using legacy tanzu cli is not same as the plugins being installed using legacy tanzu cli")

			ginkgo.By("Install the new Tanzu CLI that overrides the legacy Tanzu CLI")
			err = tf.InstallNewTanzuCLI(framework.WithOverride(true))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of new Tanzu CLI")
			version, err = tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(newTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			// set up the test local central repository host CA cert in the config file
			setTestLocalCentralRepoCertConfig([]framework.E2EOption{})

			// set up the local central repository discovery image signature public key path
			os.Setenv("TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH", e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath)

			ginkgo.By("Update plugin discovery source with test central repo")
			_, err = tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: "default", SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL})
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin source update")

			ginkgo.By("Search plugins and make sure there are plugins available using new Tanzu CLI")
			pluginsSearchList := pluginlifecyclee2e.SearchAllPlugins(tf)
			gomega.Expect(len(pluginsSearchList)).Should(gomega.BeNumerically(">", 0))

			ginkgo.By("Install few plugins using new Tanzu CLI")
			for _, plugin := range PluginsForNewTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install using new tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe using new tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe using new tanzu cli")
			}

			ginkgo.By("Verify the installed plugins using new Tanzu CLI")
			pluginsList, err = tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list using new tanzu cli")
			gomega.Expect(framework.CheckAllPluginsExists(pluginsList, PluginsForNewTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using new tanzu cli is not same as the plugins being installed using new tanzu cli")

			ginkgo.By("Reinstall new Tanzu CLI overriding the legacy Tanzu CLI")
			err = tf.ReinstallNewTanzuCLI(framework.WithOverride(true))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed plugins using reinstalled new Tanzu CLI")
			pluginsList, err = tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list using new tanzu cli")
			gomega.Expect(framework.CheckAllPluginsExists(pluginsList, PluginsForNewTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using new tanzu cli is not same as the plugins being installed using new tanzu cli")
			gomega.Expect(pluginlifecyclee2e.CheckAllLegacyPluginsExists(pluginsList, PluginsForLegacyTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using new tanzu cli is not same as the plugins being installed using legacy tanzu cli")
		})

		ginkgo.It("Use case 4: After Reinstalling new Tanzu CLI to coexist along with the legacy Tanzu CLI, Plugins installed using legacy Tanzu CLI and new Tanzu CLI should be available and installed using new tanzu cli", func() {
			ginkgo.By("Install the legacy Tanzu CLI")
			err := tf.InstallLegacyTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err := tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Install few plugins using legacy Tanzu CLI")
			for _, plugin := range PluginsForLegacyTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install using legacy tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe using legacy tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe using legcy tanzu cli")
			}

			ginkgo.By("Verify the installed plugins using legacy Tanzu CLI")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list")
			gomega.Expect(pluginlifecyclee2e.CheckAllLegacyPluginsExists(pluginsList, PluginsForLegacyTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using legacy tanzu cli is not same as the plugins being installed using legacy tanzu cli")

			ginkgo.By("Install the new Tanzu CLI which coexist with the legacy Tanzu CLI")
			err = tf.InstallNewTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of new Tanzu CLI")
			version, err = tf.CLIVersion(framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(version).To(gomega.ContainSubstring(newTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			// set up the test local central repository host CA cert in the config file
			setTestLocalCentralRepoCertConfig([]framework.E2EOption{framework.WithTanzuCommandPrefix(framework.TzPrefix)})

			// set up the local central repository discovery image signature public key path
			os.Setenv("TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH", e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath)

			ginkgo.By("Update plugin discovery source with test central repo")
			_, err = tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: "default", SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL}, framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin source update")

			ginkgo.By("search plugins and make sure there are plugins available")
			pluginsSearchList := pluginlifecyclee2e.SearchAllPlugins(tf, framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(len(pluginsSearchList)).Should(gomega.BeNumerically(">", 0))

			ginkgo.By("Install few plugins using new Tanzu CLI")
			for _, plugin := range PluginsForNewTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version, framework.WithTanzuCommandPrefix(framework.TzPrefix))
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install using new tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target, framework.WithTanzuCommandPrefix(framework.TzPrefix))
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe using new tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe using new tanzu cli")
			}

			ginkgo.By("Verify the installed plugins using new Tanzu CLI")
			pluginsList, err = tf.PluginCmd.ListPlugins(framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list using new tanzu cli")
			gomega.Expect(framework.CheckAllPluginsExists(pluginsList, PluginsForNewTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using new tanzu cli is not same as the plugins being installed using new tanzu cli")

			ginkgo.By("Reinstall new Tanzu CLI overriding the legacy Tanzu CLI")
			err = tf.ReinstallNewTanzuCLI(framework.WithOverride(true))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed plugins using reinstalled new Tanzu CLI")
			pluginsList, err = tf.PluginCmd.ListPlugins(framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list using new tanzu cli")
			gomega.Expect(framework.CheckAllPluginsExists(pluginsList, PluginsForNewTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using new tanzu cli is not same as the plugins being installed using new tanzu cli")
			gomega.Expect(pluginlifecyclee2e.CheckAllLegacyPluginsExists(pluginsList, PluginsForLegacyTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using new tanzu cli is not same as the plugins being installed using legacy tanzu cli")
		})

		ginkgo.It("Use case 5: After Rollback to legacy Tanzu CLI, Plugins installed using legacy Tanzu CLI should be available and installed using legacy Tanzu CLI", func() {
			ginkgo.By("Install the legacy Tanzu CLI")
			err := tf.InstallLegacyTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err := tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Install few plugins using legacy Tanzu CLI")
			for _, plugin := range PluginsForLegacyTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install using legacy tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe using legacy tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe using legacy tanzu cli")
			}

			ginkgo.By("Verify the installed plugins using legacy Tanzu CLI")
			pluginsList, err := tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list")
			gomega.Expect(pluginlifecyclee2e.CheckAllLegacyPluginsExists(pluginsList, PluginsForLegacyTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using legacy tanzu cli is not same as the plugins being installed using legacy tanzu cli")

			ginkgo.By("Install the new Tanzu CLI that overrides the legacy Tanzu CLI")
			err = tf.InstallNewTanzuCLI(framework.WithOverride(true))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of new Tanzu CLI")
			version, err = tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(newTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			// set up the test local central repository host CA cert in the config file
			setTestLocalCentralRepoCertConfig([]framework.E2EOption{})

			// set up the local central repository discovery image signature public key path
			os.Setenv("TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH", e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath)

			ginkgo.By("Update plugin discovery source with test central repo")
			_, err = tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: "default", SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL})
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin source update")

			ginkgo.By("search plugins and make sure there are plugins available using new Tanzu CLI")
			pluginsSearchList := pluginlifecyclee2e.SearchAllPlugins(tf)
			gomega.Expect(len(pluginsSearchList)).Should(gomega.BeNumerically(">", 0))

			ginkgo.By("Install few plugins using new Tanzu CLI")
			for _, plugin := range PluginsForNewTanzuCLICoexistenceTests {
				target := plugin.Target
				if plugin.Target == framework.GlobalTarget { // currently target "global" is not supported as target for install command
					target = ""
				}
				err := tf.PluginCmd.InstallPlugin(plugin.Name, target, plugin.Version)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin install using new tanzu cli")
				str, err := tf.PluginCmd.DescribePlugin(plugin.Name, plugin.Target)
				gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin describe using new tanzu cli")
				gomega.Expect(str).NotTo(gomega.BeNil(), "there should be output for plugin describe using new tanzu cli")
			}

			ginkgo.By("Verify the installed plugins using new Tanzu CLI")
			pluginsList, err = tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list using new tanzu cli")
			gomega.Expect(framework.CheckAllPluginsExists(pluginsList, PluginsForNewTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using new tanzu cli is not same as the plugins being installed using new tanzu cli")

			ginkgo.By("Rollback to legacy Tanzu CLI")
			err = tf.RollbackToLegacyTanzuCLI(tf)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed plugins using legacy Tanzu CLI")
			pluginsList, err = tf.PluginCmd.ListPlugins()
			gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin list using legacy tanzu cli")
			gomega.Expect(pluginlifecyclee2e.CheckAllLegacyPluginsExists(pluginsList, PluginsForLegacyTanzuCLICoexistenceTests)).Should(gomega.BeTrue(), "the plugin list output using legacy tanzu cli is not same as the plugins being installed using legacy tanzu cli")
			gomega.Expect(framework.CheckAllPluginsExists(pluginsList, PluginsForNewTanzuCLICoexistenceTests)).Should(gomega.BeFalse(), "the plugin list output using new tanzu cli is not same as the plugins being installed using new tanzu cli")
		})
	})

	ginkgo.Context("Tanzu Core CLI commands config set, get should work as expected using legacy Tanzu CLI and new Tanzu CLI", func() {
		ginkgo.It("Use case 1: when new Tanzu CLI is installed to coexist along with legacy Tanzu CLI", func() {
			ginkgo.By("Install the legacy Tanzu CLI")
			err := tf.InstallLegacyTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err := tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Install the new Tanzu CLI the coexists with the legacy Tanzu CLI")
			err = tf.InstallNewTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of new Tanzu CLI")
			version, err = tf.CLIVersion(framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(version).To(gomega.ContainSubstring(newTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Set Feature flag using legacy Tanzu CLI")
			flagName := "cli-coexistence-test-" + framework.RandomString(4)
			randomFeatureFlagPath := "features.global." + flagName
			flagVal := framework.True
			err = tf.Config.ConfigSetFeatureFlag(randomFeatureFlagPath, flagVal)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Validate the value of random feature flag set in previous step using legacy Tanzu CLI")
			val, err := tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(val).Should(gomega.Equal(framework.True))

			ginkgo.By("Validate the value of random feature flag set in previous step using new Tanzu CLI")
			val, err = tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath, framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(val).Should(gomega.Equal(framework.True))

			ginkgo.By("Unset random feature flag which was set in previous step using new Tanzu CLI")
			err = tf.Config.ConfigUnsetFeature(randomFeatureFlagPath, framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Validate the unset random feature flag in previous step using legacy Tanzu CLI")
			val, err = tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(val).Should(gomega.Equal(""))

			ginkgo.By("Validate the unset random feature flag in previous step using new Tanzu CLI")
			val, err = tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath, framework.WithTanzuCommandPrefix(framework.TzPrefix))
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(val).Should(gomega.Equal(""))
		})

		ginkgo.It("Use case 2: when new Tanzu CLI overrides the installation of legacy Tanzu CLI", func() {
			ginkgo.By("Install the legacy Tanzu CLI")
			err := tf.InstallLegacyTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err := tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Set Feature flag using legacy Tanzu CLI")
			flagName := "cli-coexistence-test-" + framework.RandomString(4)
			randomFeatureFlagPath := "features.global." + flagName
			flagVal := framework.True

			ginkgo.By(" Set random feature flag using legacy Tanzu CLI")
			err = tf.Config.ConfigSetFeatureFlag(randomFeatureFlagPath, flagVal)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Validate the value of random feature flag set in previous step using legacy Tanzu CLI")
			val, err := tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(val).Should(gomega.Equal(framework.True))

			ginkgo.By("Install the new Tanzu CLI the overrides the legacy Tanzu CLI")
			err = tf.InstallNewTanzuCLI(framework.WithOverride(true))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of new Tanzu CLI")
			version, err = tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(newTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Validate the value of random feature flag set in previous step using new Tanzu CLI")
			val, err = tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(val).Should(gomega.Equal(framework.True))

			ginkgo.By("Unset random feature flag which was set in previous step using new Tanzu CLI")
			err = tf.Config.ConfigUnsetFeature(randomFeatureFlagPath)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Validate the unset random feature flag in previous step using new Tanzu CLI")
			val, err = tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(val).Should(gomega.Equal(""))

			ginkgo.By("Install the legacy Tanzu CLI")
			err = tf.InstallLegacyTanzuCLI()
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Verify the installed version of legacy Tanzu CLI")
			version, err = tf.CLIVersion()
			gomega.Expect(version).To(gomega.ContainSubstring(legacyTanzuCLIVersion))
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Validate the unset random feature flag in previous step using legacy Tanzu CLI")
			val, err = tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(val).Should(gomega.Equal(""))
		})
	})

})

func setTestLocalCentralRepoCertConfig(options []framework.E2EOption) {
	e2eTestLocalCentralRepoPluginHost := os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryHost)
	gomega.Expect(e2eTestLocalCentralRepoPluginHost).NotTo(gomega.BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository host", framework.TanzuCliE2ETestLocalCentralRepositoryHost))

	// set up the CA cert fort local central repository
	_ = tf.Config.ConfigCertDelete(e2eTestLocalCentralRepoPluginHost, options...)
	_, err := tf.Config.ConfigCertAdd(&framework.CertAddOptions{Host: e2eTestLocalCentralRepoPluginHost, CACertificatePath: e2eTestLocalCentralRepoCACertPath, SkipCertVerify: "false", Insecure: "false"}, options...)
	gomega.Expect(err).To(gomega.BeNil())
}
