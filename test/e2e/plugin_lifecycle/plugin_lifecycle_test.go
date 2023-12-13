// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// pluginlifecyclee2e provides plugin command specific E2E test cases
package pluginlifecyclee2e

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
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
		var e2eDigestFileName string

		pluginDataDir := filepath.Join(framework.TestHomeDir, ".cache", "tanzu", common.PluginInventoryDirName, pluginSourceName)

		// Test case: list plugin sources and validate plugin source created in previous step
		It("list plugin source and validate previously created plugin source available", func() {
			list, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not get any error for plugin source list")
			Expect(framework.IsPluginSourceExists(list, pluginSourceName)).To(BeTrue())
			Expect(list[0].Image).To(Equal(e2eTestLocalCentralRepoURL))

			// Get the digest file name to compare it later
			matches, _ := filepath.Glob(filepath.Join(pluginDataDir, "digest.*"))
			Expect(len(matches)).To(Equal(1), "should have found exactly one digest file")
			e2eDigestFileName = matches[0]
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
			// Digest should not have changed
			matches, _ := filepath.Glob(filepath.Join(pluginDataDir, "digest.*"))
			Expect(len(matches)).To(Equal(1), "should have found exactly one digest file")
			Expect(matches[0]).To(Equal(e2eDigestFileName), "digest file should not have changed")
		})
		// Test case: (negative test) delete plugin source which is not exists
		It("negative test case: delete plugin source which does not exists", func() {
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
			// Save the original signature public key path
			originalSignaturePublicKeyPath := os.Getenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath)
			// Unset the signature public key path to get the default one
			os.Unsetenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath)

			_, err := tf.PluginCmd.InitPluginDiscoverySource()
			Expect(err).To(BeNil(), "should not get any error for plugin source init")
			list, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not get any error for plugin source list")
			Expect(framework.IsPluginSourceExists(list, pluginSourceName)).To(BeTrue())
			Expect(list[0].Image).To(Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage))

			// Digest SHOULD have changed
			matches, _ := filepath.Glob(filepath.Join(pluginDataDir, "digest.*"))
			Expect(len(matches)).To(Equal(1), "should have found exactly one digest file")
			Expect(matches[0]).NotTo(Equal(e2eDigestFileName), "digest file should have changed")

			// Set the original signature public key path back
			os.Setenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath, originalSignaturePublicKeyPath)
		})
		It("try to set the plugin source to an unsigned one and make sure it does not get changed", func() {
			// To make the plugin source unsigned, we remove the signature public key path
			originalSignaturePublicKeyPath := os.Getenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath)
			// Unset the signature public key path to get the default one
			os.Unsetenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath)

			_, err := tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: pluginSourceName, SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL})
			Expect(err).ToNot(BeNil(), "should get an error for plugin source update to unsigned image")

			// Check the plugin source was not updated
			list, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not get any error for plugin source list")
			Expect(framework.IsPluginSourceExists(list, pluginSourceName)).To(BeTrue())
			Expect(list[0].Image).To(Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage))

			// Set the original signature public key path back
			os.Setenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath, originalSignaturePublicKeyPath)
		})
		It("put back the E2E plugin repository", func() {
			_, err := tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: pluginSourceName, SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL})
			Expect(err).To(BeNil(), "should not get any error for plugin source update")

			// Digest should be back to its original value
			matches, _ := filepath.Glob(filepath.Join(pluginDataDir, "digest.*"))
			Expect(len(matches)).To(Equal(1), "should have found exactly one digest file")
			Expect(matches[0]).To(Equal(e2eDigestFileName), "digest file should have changed back")
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
			Expect(err).To(BeNil(), "should not get any error for plugin uninstall all")

			// validate above plugin uninstall with plugin list command, plugin list should return 0 plugins of target kubernetes
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
			// validate above plugin uninstall with plugin list command, plugin list should return 1 plugin (the essential plugin)
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

	// use case: negative test cases for plugin install
	// a. install plugin with incorrect value target flag
	// b. install plugin with incorrect plugin name
	// c. install plugin with incorrect value for flag --version
	Context("plugin use cases: negative test cases for plugin install", func() {
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
	// d. plugin uninstall help message
	Context("plugin use cases: negative test cases for plugin install and plugin uninstall commands", func() {
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

	// use case: Plugin inventory DB digest check is only done once its TTL has expired
	// a. run a "tanzu plugin source update" to do a digest check
	// b. remove the metadata.digest.none file. This will cause a printout when the digest is checked after the cache TTL has expired
	// c. set the TTL to 12 seconds and sleep for 14 seconds and check that a "tanzu plugin search" does a digest check
	// d. repeatedly sleep a few seconds, then run a "tanzu plugin search" and make sure no digest check is done (no printout)
	// e. sleep a few seconds passed the TTL, then run a "tanzu plugin search" and make sure the digest check is done (printout)
	// f. unset the TTL override
	// g. clean plugins (which will also remove the DB file) and make sure a "tanzu plugin search" immediately does a digest check
	// h. cleanup
	Context("plugin inventory DB digest check is only done once its TTL has expired", func() {
		const (
			pluginSourceName     = "default"
			refreshingDBPrintout = "Reading plugin inventory for"
		)
		pluginDataDir := filepath.Join(framework.TestHomeDir, ".cache", "tanzu", common.PluginInventoryDirName, pluginSourceName)
		metadataDigest := filepath.Join(pluginDataDir, "metadata.digest.none")

		It("update plugin source to force a refresh of the digest", func() {
			err := framework.UpdatePluginDiscoverySource(tf, e2eTestLocalCentralRepoURL)
			Expect(err).To(BeNil(), "should not get any error for plugin source update")

			// Do a plugin list to get the essential plugins installed, so that
			// it does not happen when we are running the digest test below
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(1), "the essential plugin should be installed")
		})
		// Test case: verify that a plugin search does not check the digest until the TTL has expired
		It("plugin search uses the existing DB until TTL expires", func() {
			// Use this function to remove the metadata digest file so that we can expect the
			// refreshingDBPrintout printout defined above when the digest is checked
			// after its TTL has expired
			removeDigestFile := func() {
				// Remove the metadata digest file
				err := os.Remove(metadataDigest)
				Expect(err).To(BeNil(), "unable to remove metadata digest file")
			}

			// Set the TTL to something small: 12 seconds
			os.Setenv(constants.ConfigVariablePluginDBDigestTTL, "12")

			// Now wait for the TTL to expire so we can start fresh
			removeDigestFile()
			time.Sleep(time.Second * 14) // Sleep for 14 seconds
			_, errStream, err := tf.PluginCmd.RunPluginCmd("search --name plugin_first")
			Expect(err).To(BeNil())
			Expect(errStream).To(ContainSubstring(refreshingDBPrintout))

			// For the first 9 seconds, we should not see any printouts about refreshing the DB
			removeDigestFile()
			for i := 0; i < 3; i++ {
				time.Sleep(time.Second * 3) // Sleep for 3 seconds

				_, errStream, err = tf.PluginCmd.RunPluginCmd(fmt.Sprintf("search --name plugin_%d", i))
				Expect(err).To(BeNil())
				// No printouts on the error stream about refreshing the DB
				Expect(errStream).ToNot(ContainSubstring(refreshingDBPrintout))

				// No digest file created
				_, err := os.Stat(metadataDigest)
				Expect(err).ToNot(BeNil(), "should not have found a digest file")
			}

			// Once the TTL of 12 seconds has expired, we should see a printout about refreshing the DB
			time.Sleep(time.Second * 5) // Sleep for a final 5 seconds

			_, errStream, err = tf.PluginCmd.RunPluginCmd("search --name plugin_last")
			Expect(err).To(BeNil())
			// Now we expect printouts on the error stream about refreshing the DB
			Expect(errStream).To(ContainSubstring(refreshingDBPrintout))

			// Also, the digest file should have been created
			_, err = os.Stat(metadataDigest)
			Expect(err).To(BeNil(), "metadata digest file should have been created")

			// Unset the TTL override
			os.Unsetenv(constants.ConfigVariablePluginDBDigestTTL)
		})
		// Run "plugin clean" which also removes the plugin DB and make sure a "plugin search" immediately does a digest check
		It("clean DB and do a plugin search", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")

			// Make sure a "plugin search" immediately does a digest check
			_, errPrintout, err := tf.PluginCmd.RunPluginCmd("search --name plugin_after_clean")
			Expect(err).To(BeNil())
			// Now we expect printouts on the error stream about refreshing the DB
			Expect(errPrintout).To(ContainSubstring(refreshingDBPrintout))
		})
		// Clean up at the end.
		It("clean plugins and verify with plugin list", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "there should not be any plugins available after uninstall all")
		})
	})

	// use case: Plugin inventory DB digest check is only done once its TTL has expired for
	// discoveries added through TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY
	// a. remove the default discovery and add a discovery image to TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY
	// b. remove the metadata.digest.none file. This will cause a printout when the digest is checked after the cache TTL has expired
	// c. set the TTL to 12 seconds and sleep for 14 seconds and check that a "tanzu plugin search" does a digest check
	// d. repeatedly sleep a few seconds, then run a "tanzu plugin search" and make sure no digest check is done (no printout)
	// e. sleep a few seconds passed the TTL, then run a "tanzu plugin search" and make sure the digest check is done (printout)
	// f. unset the TTL override
	// g. clean plugins (which will also remove the DB file) and make sure a "tanzu plugin search" immediately does a digest check
	// h. cleanup
	Context("plugin inventory DB digest check is only done once its TTL has expired", func() {
		const (
			defaultPluginSourceName = "default"
			testPluginSourceName    = "disc_0"
			refreshingDBPrintout    = "Reading plugin inventory for"
		)
		pluginDataDir := filepath.Join(framework.TestHomeDir, ".cache", "tanzu", common.PluginInventoryDirName, testPluginSourceName)
		metadataDigest := filepath.Join(pluginDataDir, "metadata.digest.none")

		It("delete default plugin source and add a test one", func() {
			_, err := tf.PluginCmd.DeletePluginDiscoverySource("default")
			Expect(err).To(BeNil(), "should not get any error for plugin source delete")

			os.Setenv("TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY", e2eTestLocalCentralRepoURL)

			// Do a plugin group search to fill the cache with the plugin inventory
			_, _, err = tf.PluginCmd.RunPluginCmd("group search")
			Expect(err).To(BeNil())

			// Do a plugin list to get the essential plugins installed, so that
			// it does not happen when we are running the digest test below
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(1), "the essential plugin should be installed")
		})
		// Test case: verify that a plugin search does not check the digest until the TTL has expired
		It("plugin search uses the existing DB until TTL expires", func() {
			// Use this function to remove the metadata digest file so that we can expect the
			// refreshingDBPrintout printout defined above when the digest is checked
			// after its TTL has expired
			removeDigestFile := func() {
				// Remove the metadata digest file
				err := os.Remove(metadataDigest)
				Expect(err).To(BeNil(), "unable to remove metadata digest file")
			}

			// Set the TTL to something small: 12 seconds
			os.Setenv(constants.ConfigVariablePluginDBDigestTTL, "12")

			// Now wait for the TTL to expire so we can start fresh
			removeDigestFile()
			time.Sleep(time.Second * 14) // Sleep for 14 seconds
			_, errStream, err := tf.PluginCmd.RunPluginCmd("search --name plugin_first")
			Expect(err).To(BeNil())
			Expect(errStream).To(ContainSubstring(refreshingDBPrintout))

			// For the first 9 seconds, we should not see any printouts about refreshing the DB
			removeDigestFile()
			for i := 0; i < 3; i++ {
				time.Sleep(time.Second * 3) // Sleep for 3 seconds

				_, errStream, err = tf.PluginCmd.RunPluginCmd(fmt.Sprintf("search --name plugin_%d", i))
				Expect(err).To(BeNil())
				// No printouts on the error stream about refreshing the DB
				Expect(errStream).ToNot(ContainSubstring(refreshingDBPrintout))

				// No digest file created
				_, err := os.Stat(metadataDigest)
				Expect(err).ToNot(BeNil(), "should not have found a digest file")
			}

			// Once the TTL of 12 seconds has expired, we should see a printout about refreshing the DB
			time.Sleep(time.Second * 5) // Sleep for a final 5 seconds

			_, errStream, err = tf.PluginCmd.RunPluginCmd("search --name plugin_last")
			Expect(err).To(BeNil())
			// Now we expect printouts on the error stream about refreshing the DB
			Expect(errStream).To(ContainSubstring(refreshingDBPrintout))

			// Also, the digest file should have been created
			_, err = os.Stat(metadataDigest)
			Expect(err).To(BeNil(), "metadata digest file should have been created")

			// Unset the TTL override
			os.Unsetenv(constants.ConfigVariablePluginDBDigestTTL)
		})
		// Change the TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY value and make sure
		// the digest check is done immediately
		It("set a different test discovery", func() {
			os.Setenv("TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY", "localhost:9876/tanzu-cli/plugins/central:large")

			// Make sure a "plugin search" immediately does a digest check
			_, errPrintout, err := tf.PluginCmd.RunPluginCmd("search --name plugin_after_new_discovery")
			Expect(err).To(BeNil())
			// We expect printouts on the error stream about refreshing the DB
			Expect(errPrintout).To(ContainSubstring(refreshingDBPrintout))
		})
		// Clean up at the end.
		It("clean plugins and verify with plugin list", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")
			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "there should not be any plugins available after uninstall all")
		})
		It("put back the E2E plugin repository", func() {
			os.Unsetenv("TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY")

			// Save the original signature public key path
			originalSignaturePublicKeyPath := os.Getenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath)
			// Unset the signature public key path to get the default one
			os.Unsetenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath)

			// First put back the "default" plugin source
			_, err := tf.PluginCmd.InitPluginDiscoverySource()
			Expect(err).To(BeNil(), "should not get any error for plugin source init")

			// Set the original signature public key path back
			os.Setenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath, originalSignaturePublicKeyPath)

			// Now reset it to the e2e test url
			_, err = tf.PluginCmd.UpdatePluginDiscoverySource(&framework.DiscoveryOptions{Name: defaultPluginSourceName, SourceType: framework.SourceType, URI: e2eTestLocalCentralRepoURL})
			Expect(err).To(BeNil(), "should not get any error for plugin source update")
		})
	})
})
