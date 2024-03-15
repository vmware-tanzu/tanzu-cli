// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// pluginsynce2etmc provides plugin sync command specific E2E test cases for tmc target
package pluginsynce2etmc

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	f "github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	pl "github.com/vmware-tanzu/tanzu-cli/test/e2e/plugin_lifecycle"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var stdOut string
var stdErr string

const thereShouldNotBeError = "there should be an error"

// Below are the use cases executed in this suite
// Use case 1: create context when there are no plugins to sync
// Use case 2: created a context (make sure sync installs plugins), uninstall plugin, run plugins sync (should install uninstall plugins)
// Use case 3: create a context (plugin sync) when test central repo does not have all plugins mocked in tmc endpoint response
// Use case 4: create two contexts, perform switch context use case, make installed plugins should be updated as per the context
// Use Case 5: Update the TMC endpoint with additional plugins, and ensure that the plugin sync updates the latest additional plugins.
// Use case 6: create k8s context and tmc context, validate plugins list and plugin sync
var _ = f.CLICoreDescribe("[Tests:E2E][Feature:Plugin-Sync-TMC-lifecycle]", func() {
	// Delete the configuration files and initialize
	Context("Delete the configuration files and initialize", func() {
		It("Delete the configuration files and initialize", func() {
			err := f.CleanConfigFiles(tf)
			Expect(err).To(BeNil())

			// Add Cert
			_, err = tf.Config.ConfigCertAdd(&f.CertAddOptions{Host: e2eTestLocalCentralRepoPluginHost, CACertificatePath: e2eTestLocalCentralRepoCACertPath, SkipCertVerify: "false", Insecure: "false"})
			Expect(err).To(BeNil(), "should not be any error for cert add")
			list, err := tf.Config.ConfigCertList()
			Expect(err).To(BeNil(), "should not be any error for cert list")
			Expect(len(list)).To(Equal(1), "should not be any error for cert list")

			// update plugin discovery source
			err = f.UpdatePluginDiscoverySource(tf, e2eTestLocalCentralRepoURL)
			Expect(err).To(BeNil(), "should not get any error for plugin source update")
		})
	})

	// Use case 1: create context when there are no plugins to sync
	// a. create empty mock response, and start mock http server
	// b. create context and validate current active context
	// c. list plugins and make sure no plugins installed
	// d. delete current context
	Context("Use case 1: create context when there are no plugins to sync", func() {
		var contextName string
		var err error
		It("clean plugins", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")
		})
		// Test case: a. create empty mock response, and start mock http server
		It("mock tmc endpoint with expected plugins response and start REST API mock server", func() {
			pluginsToGenerateMockResponse := make([]*f.PluginInfo, 0)
			mockReqResMapping, err := f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponse)
			Expect(err).To(BeNil(), noErrorForMockResponsePreparation)
			err = f.WriteToFileInJSONFormat(mockReqResMapping, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)
			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponse)), "the number of plugins in endpoint response and initially mocked should be same")
		})
		// Test case: b. create context and validate current active context
		It("create context for TMC target with http mock server URL as endpoint", func() {
			contextName = f.ContextPrefixTMC + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextName, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextName), activeContextShouldBeRecentlyAddedOne)
		})
		// Test case: c. list plugins and make sure no plugins installed
		It("list plugins and check number plugins should be same as installed in previous test", func() {
			pluginsList, err := tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsList)).Should(Equal(0), "there should not be any context specific plugins")
		})
		// Test case: d. delete current context
		It("delete current context and stop mock server", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), deleteContextWithoutError)
			err = f.StopContainer(tf, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use case 2: Created a context (make sure sync installs plugins), uninstall plugin, run plugins sync (should install uninstall plugins)
	// Steps:
	// a. mock tmc endpoint with plugins info, start the mock server
	// b. create context and make sure context has created
	// c. list recommended installed plugins and validate plugins info, make sure all plugins are installed as per mock response
	// d. uninstall one of the installed plugin, make sure plugin is uninstalled, run plugin sync, make sure the uninstalled plugin has installed again.
	// e. delete current context, make sure all context specific plugins are uninstalled
	Context("Use case 2: create context, uninstall plugin, sync plugins", func() {
		//var pluginCRFilePaths []string
		var pluginsToGenerateMockResponse, installedPluginsList, pluginsList []*f.PluginInfo
		var contextName string
		var err error
		var ok bool

		It("clean plugins", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")
		})
		// Test case: a. mock tmc endpoint with plugins info, start the mock server
		It("mock tmc endpoint with expected plugins response and restart REST API mock server", func() {
			// get plugins from a group
			pluginsToGenerateMockResponse, ok = pluginGroupToPluginListMap[usePluginsFromTmcPluginGroup]
			Expect(ok).To(BeTrue(), pluginGroupShouldExists)
			Expect(len(pluginsToGenerateMockResponse) > numberOfPluginsToInstall).To(BeTrue(), testRepoDoesNotHaveEnoughPlugins)
			// mock tmc endpoint with only specific number of plugins info
			pluginsToGenerateMockResponse = pluginsToGenerateMockResponse[:numberOfPluginsToInstall]
			mockReqResMapping, err := f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponse[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), noErrorForMockResponsePreparation)
			err = f.WriteToFileInJSONFormat(mockReqResMapping, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)
			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponse)), "the number of plugins in endpoint response and initially mocked should be same")
		})
		// Test case: b. create context and make sure context has created
		It("create context for TMC target with http mock server URL as endpoint", func() {
			contextName = f.ContextPrefixTMC + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextName, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextName), activeContextShouldBeRecentlyAddedOne)
		})

		// Test case: c. list recommended installed plugins and validate plugins info, make sure all plugins are installed as per mock response
		It("list plugins and validate plugins being installed after context being created", func() {
			recommendedInstalledPluginsList, err := tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(recommendedInstalledPluginsList)).Should(Equal(len(pluginsToGenerateMockResponse)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(recommendedInstalledPluginsList, pluginsToGenerateMockResponse)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: d. uninstall one of the installed plugin, make sure plugin is uninstalled,
		//				run plugin sync, make sure the uninstalled plugin is installed again.
		It("uninstall one of the installed plugin", func() {

			var pluginToUninstall *f.PluginInfo
			var latestPluginsInstalledList []*f.PluginInfo
			for i := range pluginsToGenerateMockResponse {
				if i == 0 {
					pluginToUninstall = pluginsToGenerateMockResponse[i]
				}
				if i == 1 {
					latestPluginsInstalledList = pluginsToGenerateMockResponse[i:]
					break
				}
			}
			if pluginToUninstall != nil && len(latestPluginsInstalledList) > 0 {
				installedPluginsList, err = tf.PluginCmd.ListInstalledPlugins()
				Expect(err).To(BeNil(), "should not get any error for plugin list")

				err := tf.PluginCmd.UninstallPlugin(pluginToUninstall.Name, pluginToUninstall.Target)
				Expect(err).To(BeNil(), "should not get any error for plugin uninstall")

				installedPluginsListAfterUninstall, err := tf.PluginCmd.ListInstalledPlugins()
				Expect(err).To(BeNil(), "should not get any error for plugin list")

				Expect(len(installedPluginsList)).Should(Equal(len(installedPluginsListAfterUninstall)+1), "only one plugin should be uninstalled")

				recommendedPluginsList, err := tf.PluginCmd.ListRecommendedPluginsFromActiveContext(false)
				Expect(err).To(BeNil(), "should not get any error for plugin list")

				uninstalledPluginInfo := f.GetGivenPluginFromTheGivenPluginList(recommendedPluginsList, pluginToUninstall)
				Expect(uninstalledPluginInfo.Status).To(Equal(f.RecommendInstall), "uninstalled plugin should be listed as 'install recommended'")
				Expect(uninstalledPluginInfo.Recommended).To(Equal(pluginToUninstall.Version), "uninstalled plugin should also specify correct recommended column")
			} else {
				// fail the test case if at least two plugins are not available in the mock response
				Fail("there should be at least two plugins to uninstall")
			}

			_, _, err = tf.PluginCmd.Sync()
			Expect(err).To(BeNil(), "should not get any error for plugin sync")

			installedPluginsList, err = tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(f.CheckAllPluginsExists(installedPluginsList, pluginsToGenerateMockResponse)).Should(BeTrue(), "plugins being installed and plugins for the mocked plugins should be same")
		})
		// e. delete current context, make sure all context specific plugins are uninstalled
		It("delete current context", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), deleteContextWithoutError)

			pluginsList, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsList)).Should(Equal(0), "all context plugins should be uninstalled as context delete")
			err = f.StopContainer(tf, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use case 3: create a context (plugin sync) when test central repo does not have all plugins mocked in tmc endpoint response
	// Steps:
	// a. update tmc endpoint mock response (make sure at least a plugin does not exists in test central repo) and restart REST API mock server
	// b. create context and make sure context has created
	// c. list recommended plugins and validate plugins info, make sure all plugins installed for which response mocked and available in central repo, the unavailable plugin (with incorrect version) should not be installed
	// d. run plugin sync and validate the plugin list
	// e. unset the context and make sure plugins are still installed and set the context again
	// f. delete current context
	Context("Use case 3: Create context should not install unavailable plugins, plugin sync also should not install unavailable plugins", func() {
		var pluginsToGenerateMockResponse, pluginsGeneratedMockResponseWithCorrectInfo []*f.PluginInfo
		var pluginWithIncorrectVersion *f.PluginInfo
		var contextName, randomPluginVersion string
		var err error
		var ok bool
		It("clean plugins", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")
		})
		// Test case: a. update tmc endpoint mock response (make sure at least a plugin does not exists in test central repo) and restart REST API mock server
		It("mock tmc endpoint with expected plugins response and restart REST API mock server", func() {
			// get plugins from a group
			pluginsToGenerateMockResponse, ok = pluginGroupToPluginListMap[usePluginsFromTmcPluginGroup]
			Expect(ok).To(BeTrue(), pluginGroupShouldExists)
			Expect(len(pluginsToGenerateMockResponse) > numberOfPluginsToInstall).To(BeTrue(), testRepoDoesNotHaveEnoughPlugins)
			// mock tmc endpoint with only specific number of plugins info
			pluginsToGenerateMockResponse = pluginsToGenerateMockResponse[:numberOfPluginsToInstall]
			// assign incorrect version to last plugin in the slice, to make sure at least one plugin is not available
			// in the mock response
			actualVersion := pluginsToGenerateMockResponse[numberOfPluginsToInstall-1].Version
			randomPluginVersion = pluginsToGenerateMockResponse[numberOfPluginsToInstall-1].Version + f.RandomNumber(2)
			pluginsToGenerateMockResponse[numberOfPluginsToInstall-1].Version = randomPluginVersion
			pluginWithIncorrectVersion = pluginsToGenerateMockResponse[numberOfPluginsToInstall-1]
			// generate mock response for all plugins (including the incorrect version plugin)
			mockReqResMapping, err := f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponse)
			Expect(err).To(BeNil())
			err = f.WriteToFileInJSONFormat(mockReqResMapping, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// assign the original version back
			pluginsToGenerateMockResponse[numberOfPluginsToInstall-1].Version = actualVersion

			// skip last plugin in the slice as it has incorrect version info, which is not available in the mock response
			pluginsGeneratedMockResponseWithCorrectInfo = pluginsToGenerateMockResponse[:numberOfPluginsToInstall-1]
			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)
			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponse)), "the number of plugins in endpoint response and initially mocked should be same")
		})

		// Test case: b. create context and make sure context has created
		It("create context for TMC target with http mock server URL as endpoint", func() {
			contextName = f.ContextPrefixTMC + f.RandomString(4)
			_, stdErr, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextName, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			Expect(stdErr).NotTo(BeNil(), "there should be stderr")
			Expect(stdErr).To(ContainSubstring(f.UnableToSync), "there should be sync error as all plugins not available in repo")
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextName), activeContextShouldBeRecentlyAddedOne)
		})

		// Test case: c. list recommended plugins and validate plugins info, make sure all plugins installed for which response mocked and available in central repo, the unavailable plugin (with incorrect version) should not be installed
		It("list plugins and validate plugins being installed after context being created", func() {
			recommendedInstalledPlugins, err := tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(recommendedInstalledPlugins)).Should(Equal(len(pluginsGeneratedMockResponseWithCorrectInfo)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsGeneratedMockResponseWithCorrectInfo)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: d. run plugin sync and validate the plugin list
		It("run plugin sync and validate err response in plugin sync, validate plugin list output", func() {
			_, _, err = tf.PluginCmd.Sync()
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(f.UnableToFindPluginWithVersionForTarget, pluginWithIncorrectVersion.Name, randomPluginVersion, pluginWithIncorrectVersion.Target)))

			recommendedInstalledPlugins, err := tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(recommendedInstalledPlugins)).Should(Equal(len(pluginsGeneratedMockResponseWithCorrectInfo)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsGeneratedMockResponseWithCorrectInfo)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})
		// Test case: e. unset the context and make sure plugins are still installed and set the context again
		It("list plugins and validate plugins being installed after context being created", func() {
			installedPluginsList, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")

			stdOut, _, err = tf.ContextCmd.UnsetContext(contextName)
			Expect(err).To(BeNil(), "unset context should unset context without any error")

			installedPluginsListAfterUnset, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")

			Expect(len(installedPluginsList)).To(Equal(len(installedPluginsListAfterUnset)), "same number of installed plugins should be available after the context unset and plugins should not get deactivated")
			Expect(f.CheckAllPluginsExists(installedPluginsList, installedPluginsListAfterUnset)).Should(BeTrue(), "plugins being installed before and after the context unset should be same")

			_, _, err = tf.ContextCmd.UseContext(contextName)
			Expect(err).To(BeNil(), "use context should set context without any error")
		})
		// Test case: f. delete current context and stop mock server
		It("delete current context", func() {
			installedPluginsList, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")

			stdOut, _, err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), deleteContextWithoutError)

			installedPluginsListAfterCtxDelete, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsList)).To(Equal(len(installedPluginsListAfterCtxDelete)), "same number of installed plugins should be available after the context unset and plugins should not get deactivated")
			Expect(f.CheckAllPluginsExists(installedPluginsList, installedPluginsListAfterCtxDelete)).Should(BeTrue(), "plugins being installed before and after the context unset should be same")

			err = f.StopContainer(tf, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use case 4: Create two contexts, perform switch context use case, make installed plugins should be updated as per the context
	// Steps:
	// a. check the plugins availability
	// b. update mock response for first context, restart mock server, create context, and validate plugin list
	// c. update mock response for second context, restart mock server, create context, and validate plugin list
	// e. delete both contexts
	Context("Use case 4: create two contexts, and validate plugin list ", func() {
		var pluginsFromPluginGroup, pluginsToGenerateMockResponseOne, pluginsToGenerateMockResponseTwo, pluginsListOne, pluginsListTwo []*f.PluginInfo
		var contextNameOne, contextNameTwo string
		var TMCEndpointResponseOne, TMCEndpointResponseTwo *f.TMCPluginsMockRequestResponseMapping
		var err error
		var ok bool
		It("clean plugins", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")
		})
		// Test case: a. check the plugins availability
		It("check the plugins availability", func() {
			// get plugins from a group
			pluginsFromPluginGroup, ok = pluginGroupToPluginListMap[usePluginsFromTmcPluginGroup]
			Expect(ok).To(BeTrue(), pluginGroupShouldExists)
			Expect(len(pluginsFromPluginGroup) > numberOfPluginsToInstall*2).To(BeTrue(), testRepoDoesNotHaveEnoughPlugins)
			// mock tmc endpoint with only specific number of plugins info
			pluginsFromPluginGroup = pluginsFromPluginGroup[:numberOfPluginsToInstall*2]
		})

		// Test case: b. update mock response for first context, restart mock server, create context, and validate plugin list
		It("update mock response for first context, restart mock server, create context, and validate plugin list", func() {

			pluginsToGenerateMockResponseOne = pluginsFromPluginGroup[:numberOfPluginsToInstall]
			// generate mock response for all plugins (including the incorrect version plugin)
			TMCEndpointResponseOne, err = f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponseOne)
			err = f.WriteToFileInJSONFormat(TMCEndpointResponseOne, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)

			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponseOne)), "the number of plugins in endpoint response and initially mocked should be same")

			contextNameOne = f.ContextPrefixTMC + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextNameOne, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameOne), activeContextShouldBeRecentlyAddedOne)

			pluginsListOne, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsListOne)).Should(Equal(len(pluginsToGenerateMockResponseOne)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(pluginsListOne, pluginsToGenerateMockResponseOne)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})
		// Test case: c. update mock response for second context, restart mock server, create context, and validate plugin list
		It("update mock response for second context, restart mock server, create second context, and validate plugin list", func() {

			pluginsToGenerateMockResponseTwo = pluginsFromPluginGroup[numberOfPluginsToInstall:]
			TMCEndpointResponseTwo, err = f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponseTwo)

			err = f.WriteToFileInJSONFormat(TMCEndpointResponseTwo, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)

			// check the tmc mocked endpoint is working as expected
			var mockResPluginsInfo f.TMCPluginsInfo
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponseTwo)), "the number of plugins in endpoint response and initially mocked should be same")

			contextNameTwo = f.ContextPrefixTMC + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextNameTwo, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameTwo), activeContextShouldBeRecentlyAddedOne)

			pluginsListTwo, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsListTwo)).Should(Equal(len(pluginsToGenerateMockResponseTwo)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(pluginsListTwo, pluginsToGenerateMockResponseTwo)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// d. delete both contexts
		It("delete current context", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextNameOne)
			Expect(err).To(BeNil(), deleteContextWithoutError)
			_, _, err = tf.ContextCmd.DeleteContext(contextNameTwo)
			Expect(err).To(BeNil(), deleteContextWithoutError)
			err = f.StopContainer(tf, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use Case 5: Update the TMC endpoint with additional plugins, and ensure that the plugin sync updates the latest additional plugins.
	// Steps:
	// a. check the plugins availability
	// b. update mock response, restart mock server, create context, and validate plugin list
	// c. update mock response with additional plugins, restart mock server, run plugin sync, and validate plugin list
	// d. delete context

	Context("Use case 5: create two contexts, and validate plugin list ", func() {
		var pluginsFromPluginGroup, pluginsToGenerateMockResponseOne, pluginsToGenerateMockResponseTwo, pluginsListOne []*f.PluginInfo
		var contextNameOne string
		var TMCEndpointResponseOne, TMCEndpointResponseTwo *f.TMCPluginsMockRequestResponseMapping
		var err error
		var ok bool

		It("clean plugins", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")
		})
		// Test case: a. check the plugins availability
		It("check the plugins availability", func() {
			// get plugins from a group
			pluginsFromPluginGroup, ok = pluginGroupToPluginListMap[usePluginsFromTmcPluginGroup]
			Expect(ok).To(BeTrue(), pluginGroupShouldExists)
			Expect(len(pluginsFromPluginGroup) > numberOfPluginsToInstall*2).To(BeTrue(), testRepoDoesNotHaveEnoughPlugins)
			// mock tmc endpoint with only specific number of plugins info
			pluginsFromPluginGroup = pluginsFromPluginGroup[:numberOfPluginsToInstall*2]
		})

		// Test case: b. update mock response for first context, restart mock server, create context, and validate plugin list
		It("update mock response for first context, restart mock server, create context, and validate plugin list", func() {

			pluginsToGenerateMockResponseOne = pluginsFromPluginGroup[:numberOfPluginsToInstall]
			// generate mock response for all plugins (including the incorrect version plugin)
			TMCEndpointResponseOne, err = f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponseOne)
			err = f.WriteToFileInJSONFormat(TMCEndpointResponseOne, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)

			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponseOne)), "the number of plugins in endpoint response and initially mocked should be same")

			contextNameOne = f.ContextPrefixTMC + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextNameOne, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameOne), activeContextShouldBeRecentlyAddedOne)

			pluginsListOne, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsListOne)).Should(Equal(len(pluginsToGenerateMockResponseOne)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(pluginsListOne, pluginsToGenerateMockResponseOne)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})
		// Test case: c. update mock response with additional plugins, restart mock server, run plugin sync, and validate plugin list
		It("update mock response for second context, restart mock server, create second context, and validate plugin list", func() {

			pluginsToGenerateMockResponseTwo = pluginsFromPluginGroup[:numberOfPluginsToInstall*2]
			TMCEndpointResponseTwo, err = f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponseTwo)

			err = f.WriteToFileInJSONFormat(TMCEndpointResponseTwo, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)

			// check the tmc mocked endpoint is working as expected
			var mockResPluginsInfo f.TMCPluginsInfo
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponseTwo)), "the number of plugins in endpoint response and initially mocked should be same")

			_, _, err = tf.PluginCmd.Sync()
			Expect(err).To(BeNil(), "should not get any error for plugin sync")

			pluginsListOne, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsListOne)).Should(Equal(len(pluginsToGenerateMockResponseTwo)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(pluginsListOne, pluginsToGenerateMockResponseTwo)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// d. delete both contexts
		It("delete current context", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextNameOne)
			Expect(err).To(BeNil(), deleteContextWithoutError)
			err = f.StopContainer(tf, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use case 6: k8s and tmc contexts coexistence, create k8s context and tmc context, validate plugins list and plugin sync
	// test cases:
	// Test case: a. k8s: create KIND cluster, apply CRD
	// Test case: b. k8s: apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
	// Test case: c. k8s: create context and make sure context has created
	// Test case: d. k8s: list plugins and validate plugins info, make sure all plugins are installed for which CRs were present on the cluster
	// Test case: e. TMC: mock tmc endpoint with plugins info, start the mock server
	// Test case: f. TMC: create context and make sure context has created
	// Test case: g. TMC: list plugins and validate plugins info, make sure all plugins are installed as per mock response
	// Test case: h. validate plugin list consistency, it should sort by context name always
	// Test case: i. k8s: use first k8s context and check plugin list
	// Test case: j. k8s: uninstall one of the installed plugin, make sure plugin is uninstalled, run plugin sync, make sure the uninstalled plugin has installed again.
	// Test case: k. tmc: use tmc context and check plugin list
	// Test case: l. delete tmc/k8s contexts and the KIND cluster
	Context("Use case 6: create k8s and tmc specific contexts, validate plugins list and perform pluin sync, and perform context switch", func() {
		var clusterInfo *f.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsInfoForCRsApplied, recommendedInstalledPlugins []*f.PluginInfo
		var contextNameK8s string
		contexts := make([]string, 0)
		totalInstalledPlugins := 1 // telemetry plugin that is part of essentials plugin group will always be installed
		var err error

		It("clean plugins", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")
		})
		// Test case: a. k8s: create KIND cluster, apply CRD
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. k8s: apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
		It("apply CRD and CRs to KIND cluster", func() {
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), f.K8SCRDFilePath))
			Expect(err).To(BeNil(), "should not get any error for config apply")

			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[usePluginsFromK8sPluginGroup]
			Expect(ok).To(BeTrue(), "plugin group is not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
			Expect(err).To(BeNil(), "should not get any error for config apply")
			totalInstalledPlugins += numberOfPluginsToInstall
		})

		// Test case: c. k8s: create context and make sure context has created
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextNameK8s = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err := tf.ContextCmd.CreateContextWithKubeconfig(contextNameK8s, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNameK8s), "the active context should be recently added context")
			contexts = append(contexts, contextNameK8s)
		})
		// Test case: d. k8s: list plugins and validate plugins info, make sure all plugins are installed for which CRs were present on the cluster
		It("Test case: d; list plugins and validate plugins being installed after context being created", func() {
			recommendedInstalledPlugins, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		var pluginsToGenerateMockResponse []*f.PluginInfo
		var contextNameTMC string
		var ok bool

		// Test case: e. TMC: mock tmc endpoint with plugins info, start the mock server
		It("mock tmc endpoint with expected plugins response and restart REST API mock server", func() {
			// get plugins from a group
			pluginsToGenerateMockResponse, ok = pluginGroupToPluginListMap[usePluginsFromTmcPluginGroup]
			Expect(ok).To(BeTrue(), pluginGroupShouldExists)
			Expect(len(pluginsToGenerateMockResponse) > numberOfPluginsToInstall).To(BeTrue(), testRepoDoesNotHaveEnoughPlugins)
			// mock tmc endpoint with only specific number of plugins info
			pluginsToGenerateMockResponse = pluginsToGenerateMockResponse[:numberOfPluginsToInstall]
			mockReqResMapping, err := f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponse[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), noErrorForMockResponsePreparation)
			err = f.WriteToFileInJSONFormat(mockReqResMapping, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)
			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponse)), "the number of plugins in endpoint response and initially mocked should be same")
			totalInstalledPlugins += numberOfPluginsToInstall
		})
		// Test case: f. TMC: create context and make sure context has created
		It("create context for TMC target with http mock server URL as endpoint", func() {
			contextNameTMC = f.ContextPrefixTMC + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextNameTMC, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameTMC), activeContextShouldBeRecentlyAddedOne)
			contexts = append(contexts, contextNameTMC)
		})

		// Test case: g. TMC: list plugins and validate plugins info, make sure all plugins are installed as per mock response
		It("Test case: g: list plugins and validate plugins being installed after context being created", func() {
			recommendedInstalledPlugins, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsToGenerateMockResponse)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: h. validate plugin list consistency, it should sort by context name always
		It("Test case: h: list plugins and validate plugins being installed after context being created", func() {
			// IsSortedByName checks if the array of objects is sorted by name
			IsSortedByName := func(arr []*f.PluginInfo) bool {
				n := len(arr)

				// Iterate through the array to check if it is sorted
				for i := 1; i < n; i++ {

					// Compare the names of adjacent objects
					if arr[i-1].Name > arr[i].Name {
						return false
					}
				}

				// If the loop completes without returning false, the array is sorted
				return true
			}
			// check multiple times, the order should be consistent
			for j := 0; j < 5; j++ {
				installedPlugins, err := tf.PluginCmd.ListInstalledPlugins()
				Expect(err).To(BeNil(), noErrorForPluginList)
				Expect(totalInstalledPlugins).Should(Equal(len(installedPlugins)), "total installed plugins count should be equal to plugins installed for both contexts")

				// Filter plugins without context
				var installedPluginsWithContext []*f.PluginInfo
				for _, plugin := range installedPlugins {
					if plugin.Recommended != "" {
						installedPluginsWithContext = append(installedPluginsWithContext, plugin)
					}
				}
				Expect(IsSortedByName(installedPluginsWithContext)).To(BeTrue())
			}
		})

		// Test case: i. k8s: use first k8s context and check plugin list
		It("use first context, check plugin list", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")

			_, _, err = tf.ContextCmd.UseContext(contextNameK8s)
			Expect(err).To(BeNil(), "use context should not return any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeK8s))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameK8s), active, "the active context should be the recently switched one")

			recommendedInstalledPlugins, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(recommendedInstalledPlugins)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: k. tmc: use tmc context and check plugin list
		It("use second context again, check plugin list", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")

			_, _, err = tf.ContextCmd.UseContext(contextNameTMC)
			Expect(err).To(BeNil(), "use context should not return any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameTMC), activeContextShouldBeRecentlyAddedOne)

			// use context will install the plugins from the tmc endpoint, which is second set of plugins list
			recommendedInstalledPlugins, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(recommendedInstalledPlugins)).Should(Equal(len(pluginsToGenerateMockResponse)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsToGenerateMockResponse)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: l. delete tmc/k8s contexts and the KIND cluster
		It("delete tmc/k8s contexts and the KIND cluster", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextNameTMC)
			Expect(err).To(BeNil(), "context should be deleted without error")
			err = f.StopContainer(tf, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)

			_, _, err = tf.ContextCmd.DeleteContext(contextNameK8s)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, _, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Use case 7: Sync for single ContextType specific plugins, and validate the plugin list
	// run context create (make sure another ContextType context is active, but yet to install plugins), it should not perform the sync for all active contexts
	// run ContextType specific plugin sync (for k8s ContextType), make sync should not happen for tmc context even though its active
	// run ContextType specific plugin sync (for tmc ContextType), make sync should not happen for k8s context even though its active
	Context("Use case 7: create k8s and tmc specific contexts, validate plugins list and perform pluin sync, and perform context switch", func() {
		var clusterInfo *f.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsInfoForCRsApplied, installedPluginsListK8s []*f.PluginInfo
		var contextNameK8s string
		contexts := make([]string, 0)
		totalInstalledPlugins := 1 // telemetry plugin that is part of essentials plugin group will always be installed
		var err error
		It("clean plugins", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")
		})
		// Test case: a. k8s: create KIND cluster, apply CRD
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. k8s: apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
		It("apply CRD and CRs to KIND cluster", func() {
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), f.K8SCRDFilePath))
			Expect(err).To(BeNil(), "should not get any error for config apply")

			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[usePluginsFromK8sPluginGroup]
			Expect(ok).To(BeTrue(), "plugin group is not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
			Expect(err).To(BeNil(), "should not get any error for config apply")
			totalInstalledPlugins += numberOfPluginsToInstall
		})

		// Test case: c. k8s: create context and make sure context has created
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextNameK8s = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithKubeconfig(contextNameK8s, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeK8s))
			Expect(err).To(BeNil(), thereShouldNotBeError+" while getting active context")
			Expect(active).To(Equal(contextNameK8s), "the active context should be recently added context")
			contexts = append(contexts, contextNameK8s)
		})
		// Test case: d. k8s: list plugins and validate plugins info, make sure all plugins are installed for which CRs were present on the cluster
		It("Test case: d; list plugins and validate plugins being installed after context being created", func() {
			installedPluginsListK8s, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(f.CheckAllPluginsExists(installedPluginsListK8s, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		var pluginsToGenerateMockResponseTMC, installedPluginsListTMC []*f.PluginInfo
		var contextNameTMC string
		var ok bool

		// Test case: e. TMC: mock tmc endpoint with plugins info, start the mock server
		It("mock tmc endpoint with expected plugins response and restart REST API mock server", func() {
			// get plugins from a group
			pluginsToGenerateMockResponseTMC, ok = pluginGroupToPluginListMap[usePluginsFromTmcPluginGroup]
			Expect(ok).To(BeTrue(), pluginGroupShouldExists)
			Expect(len(pluginsToGenerateMockResponseTMC) > numberOfPluginsToInstall).To(BeTrue(), testRepoDoesNotHaveEnoughPlugins)
			// mock tmc endpoint with only specific number of plugins info
			pluginsToGenerateMockResponseTMC = pluginsToGenerateMockResponseTMC[:numberOfPluginsToInstall]
			mockReqResMapping, err := f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponseTMC[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), noErrorForMockResponsePreparation)
			err = f.WriteToFileInJSONFormat(mockReqResMapping, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)
			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponseTMC)), "the number of plugins in endpoint response and initially mocked should be same")
			totalInstalledPlugins += numberOfPluginsToInstall
		})
		// Test case: f. TMC: create context and make sure context has created
		It("create context for TMC target with http mock server URL as endpoint", func() {
			// Clean K8s context specific plugins
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), thereShouldNotBeError+" while cleaning plugins")

			contextNameTMC = f.ContextPrefixTMC + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextNameTMC, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameTMC), activeContextShouldBeRecentlyAddedOne)
			contexts = append(contexts, contextNameTMC)
		})

		// Test case: g. TMC: list plugins and validate plugins info, make sure all plugins are installed as per mock response
		// there should not be any k8s specific plugins installed/sync as part of tmc context creation
		It("Test case: g: list plugins and validate plugins being installed after context being created", func() {
			installedPluginsListTMC, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(installedPluginsListTMC)).Should(Equal(len(pluginsToGenerateMockResponseTMC)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsListTMC, pluginsToGenerateMockResponseTMC)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)

			// // Sync should not happen for the k8s context specific plugins
			// installedPluginsListK8S, err := tf.PluginCmd.ListRecommendedPluginsFromActiveContext(false)
			// Expect(len(installedPluginsListK8S)).Should(Equal(0))
			// Expect(err).To(BeNil(), noErrorForPluginList)
		})

		// Test case: context use (K8S) - should sync only specific context not all active context's
		// set both k8s and tmc context as active
		// clean plugins
		// unset k8s context
		// perform 'tanzu context use' for k8s context
		// sync should happen only for the specific context (k8s context), not for all active context's
		// perform 'tanzu plugin clean' and 'tanzu context use' for k8s context again, and check plugins list, it should be same as previous
		It("test context use for specific context-k8s", func() {
			_, _, err = tf.ContextCmd.UseContext(contextNameK8s)
			Expect(err).To(BeNil(), "use context should not return any error")

			_, _, err = tf.ContextCmd.UseContext(contextNameTMC)
			Expect(err).To(BeNil(), "use context should not return any error")

			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "plugin clean should not return any error")

			_, _, err = tf.ContextCmd.UnsetContext(contextNameK8s)
			Expect(err).To(BeNil(), "unset context should unset context without any error")

			_, _, err = tf.ContextCmd.UseContext(contextNameK8s)
			Expect(err).To(BeNil(), "use context should not return any error")

			// k8s contextType specific plugins only should be installed
			installedPluginsListK8s, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(f.CheckAllPluginsExists(installedPluginsListK8s, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")

			// // Sync should not happen for the tmc context specific plugins, as its contextType specific sync
			// installedPluginsListTMC, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			// Expect(len(installedPluginsListTMC)).Should(Equal(0))
			// Expect(err).To(BeNil(), noErrorForPluginList)

			// perform 'tanzu plugin clean' and 'tanzu context use' for k8s context again, and check plugins list, it should be same as previous
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "plugin clean should not return any error")

			_, _, err = tf.ContextCmd.UseContext(contextNameK8s)
			Expect(err).To(BeNil(), "use context should not return any error")

			// k8s contextType specific plugins only should be installed
			installedPluginsListK8s, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(f.CheckAllPluginsExists(installedPluginsListK8s, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")

			// // Sync should not happen for the tmc context specific plugins, as its contextType specific sync
			// installedPluginsListTMC, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			// Expect(len(installedPluginsListTMC)).Should(Equal(0))
			// Expect(err).To(BeNil(), noErrorForPluginList)
		})

		// Test case: context use (TMC) - should sync only specific context not all active context's
		// set both tmc and tmc context as active
		// clean plugins
		// unset tmc context
		// perform 'tanzu context use' for tmc context
		// sync should happen only for the specific context (tmc context), not for all active context's
		// perform 'tanzu plugin clean' and 'tanzu context use' for tmc context again, and check plugins list, it should be same as previous
		It("test context use for specific context-TMC", func() {
			_, _, err = tf.ContextCmd.UseContext(contextNameK8s)
			Expect(err).To(BeNil(), "use context should not return any error")

			_, _, err = tf.ContextCmd.UseContext(contextNameTMC)
			Expect(err).To(BeNil(), "use context should not return any error")

			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "plugin clean should not return any error")

			_, _, err = tf.ContextCmd.UnsetContext(contextNameTMC)
			Expect(err).To(BeNil(), "unset context should unset context without any error")

			_, _, err = tf.ContextCmd.UseContext(contextNameTMC)
			Expect(err).To(BeNil(), "use context should not return any error")

			// tmc contextType specific plugins only should be installed
			installedPluginsListTMC, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(installedPluginsListTMC)).Should(Equal(len(pluginsToGenerateMockResponseTMC)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsListTMC, pluginsToGenerateMockResponseTMC)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)

			// // Sync should not happen for the k8s context specific plugins, as its contextType specific sync
			// installedPluginsListK8s, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			// Expect(len(installedPluginsListK8s)).Should(Equal(0))
			// Expect(err).To(BeNil(), noErrorForPluginList)

			// perform 'tanzu plugin clean' and 'tanzu context use' for tmc context again, and check plugins list, it should be same as previous
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "plugin clean should not return any error")

			_, _, err = tf.ContextCmd.UseContext(contextNameTMC)
			Expect(err).To(BeNil(), "use context should not return any error")

			// tmc contextType specific plugins only should be installed
			installedPluginsListTMC, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(installedPluginsListTMC)).Should(Equal(len(pluginsToGenerateMockResponseTMC)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsListTMC, pluginsToGenerateMockResponseTMC)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)

			// // Sync should not happen for the k8s context specific plugins, as its contextType specific sync
			// installedPluginsListK8s, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			// Expect(len(installedPluginsListK8s)).Should(Equal(0))
			// Expect(err).To(BeNil(), noErrorForPluginList)

		})

		// Test case: context use for TMC context and K8s context
		// plugin should get sync for both tmc and k8s context
		It("context use for both TMC and k8s contexts", func() {
			_, _, err = tf.ContextCmd.UseContext(contextNameK8s)
			Expect(err).To(BeNil(), "use context should not return any error")

			_, _, err = tf.ContextCmd.UseContext(contextNameTMC)
			Expect(err).To(BeNil(), "use context should not return any error")

			// tmc and k8s contextType specific plugins should be installed
			recommendedInstalledPlugins, err := tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(recommendedInstalledPlugins)).Should(Equal(len(pluginsToGenerateMockResponseTMC)+len(pluginsInfoForCRsApplied)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsToGenerateMockResponseTMC)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsInfoForCRsApplied)).Should(BeTrue(), "plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: plugin sync use case, make sure plugins for both contexts are synced
		// plugin should get sync for both tmc and k8s context
		It("test plugin sync", func() {

			_, _, err = tf.ContextCmd.UseContext(contextNameK8s)
			Expect(err).To(BeNil(), "use context should not return any error")

			_, _, err = tf.ContextCmd.UseContext(contextNameTMC)
			Expect(err).To(BeNil(), "use context should not return any error")

			// perform plugin clean, then perform plugin sync
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "plugin clean should not return any error")

			_, _, err = tf.PluginCmd.Sync()
			Expect(err).To(BeNil(), "plugin sync should not return any error")

			// tmc and k8s contextType specific plugins should be installed
			recommendedInstalledPlugins, err := tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(recommendedInstalledPlugins)).Should(Equal(len(pluginsToGenerateMockResponseTMC)+len(pluginsInfoForCRsApplied)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsToGenerateMockResponseTMC)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
			Expect(f.CheckAllPluginsExists(recommendedInstalledPlugins, pluginsInfoForCRsApplied)).Should(BeTrue(), "plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: l. delete tmc/k8s contexts and the KIND cluster
		It("delete tmc/k8s contexts and the KIND cluster", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextNameTMC)
			Expect(err).To(BeNil(), "context should be deleted without error")
			err = f.StopContainer(tf, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)

			_, _, err = tf.ContextCmd.DeleteContext(contextNameK8s)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, _, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Use Case 8: Plugin List, sync, search and install functionalities with Context Issues
	// Use case details: In this use case, we will create one Tanzu Mission Control (TMC) context and one Kubernetes contexts.
	// The active K8s context will be associated with a kind cluster that has been deleted. As a result, there will be an issue
	// when attempting to discover plugins for this context. However, despite the issue, the plugin list and plugin sync commands
	// should continue to function properly, providing a list of plugins and performing synchronization with a warning message
	// indicating the issue.
	// The purpose of this use case is to test the resilience and functionality of the CLI in
	// scenarios where a specific context encounters issues, ensuring that the plugin list and
	// plugin sync commands can still be executed successfully, albeit with a warning message alerting the user to the issue.
	// test cases:
	// Test case: a. k8s: create KIND cluster, apply CRD
	// Test case: b. k8s: apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
	// Test case: c. k8s: create context and make sure context has created
	// Test case: d. k8s: list plugins and validate plugins info, make sure all plugins are installed for which CRs were present on the cluster
	// Test case: e. k8s: delete the kind cluster
	// Test case: f. TMC: mock tmc endpoint with plugins info, start the mock server
	// Test case: g. TMC: create context and make sure context has created
	// Test case: h. TMC: list plugins and validate plugins info, make sure all plugins are installed as per mock response
	// Test case: i. plugin search should work fine even though k8s context has issues
	// Test case: j. plugin install should work fine even though k8s context has issues
	// Test case: k. validate plugin has installed after above plugin uninstall and install operations
	// Test case: l. plugin sync should work fine even though k8s context has issues
	// Test case: m. validate the plugin list after above plugin sync operation
	// Test case: n. plugin group search and plugin group install when active context has issues
	// Test case: o. delete tmc/k8s contexts
	// Test case: i. delete tmc/k8s contexts and the KIND cluster
	Context("Use case 8: Plugin List, sync, search and install functionalities with Context Issues", func() {
		var clusterInfo *f.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsInfoForCRsApplied, installedPluginsListK8s []*f.PluginInfo
		var contextNameK8s string
		contexts := make([]string, 0)
		totalInstalledPlugins := 0
		var err error
		It("clean plugins", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")
		})
		// Test case: a. k8s: create KIND cluster, apply CRD
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. k8s: apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
		It("apply CRD and CRs to KIND cluster", func() {
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), f.K8SCRDFilePath))
			Expect(err).To(BeNil(), "should not get any error for config apply")

			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[usePluginsFromK8sPluginGroup]
			Expect(ok).To(BeTrue(), "plugin group is not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
			Expect(err).To(BeNil(), "should not get any error for config apply")
			totalInstalledPlugins += numberOfPluginsToInstall
		})

		// Test case: c. k8s: create context and make sure context has created
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextNameK8s = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err := tf.ContextCmd.CreateContextWithKubeconfig(contextNameK8s, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNameK8s), "the active context should be recently added context")
			contexts = append(contexts, contextNameK8s)
		})
		// Test case: d. k8s: list plugins and validate plugins info, make sure all plugins are installed for which CRs were present on the cluster
		It("Test case: d: list plugins and validate plugins being installed after context being created", func() {
			installedPluginsListK8s, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsListK8s)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(installedPluginsListK8s, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})
		// Test case: e. k8s: delete the kind cluster
		It("delete kind cluster associated with k8s context", func() {
			_, _, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
			list, out, se, err := tf.PluginCmd.ListPlugins()
			fmt.Println(list, out, se, err)
		})
		var pluginsToGenerateMockResponse, installedPluginsListTMC []*f.PluginInfo
		var contextNameTMC string
		var ok bool

		// Test case: f. TMC: mock tmc endpoint with plugins info, start the mock server
		It("Test case: f: mock tmc endpoint with expected plugins response and restart REST API mock server", func() {
			// get plugins from a group
			pluginsToGenerateMockResponse, ok = pluginGroupToPluginListMap[usePluginsFromTmcPluginGroup]
			Expect(ok).To(BeTrue(), pluginGroupShouldExists)
			Expect(len(pluginsToGenerateMockResponse) > numberOfPluginsToInstall).To(BeTrue(), testRepoDoesNotHaveEnoughPlugins)
			// mock tmc endpoint with only specific number of plugins info
			pluginsToGenerateMockResponse = pluginsToGenerateMockResponse[:numberOfPluginsToInstall]
			mockReqResMapping, err := f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponse[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), noErrorForMockResponsePreparation)
			err = f.WriteToFileInJSONFormat(mockReqResMapping, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)
			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponse)), "the number of plugins in endpoint response and initially mocked should be same")
			totalInstalledPlugins += numberOfPluginsToInstall
		})
		// Test case: g. TMC: create context and make sure context has created
		It("create context for TMC target with http mock server URL as endpoint", func() {
			contextNameTMC = f.ContextPrefixTMC + f.RandomString(4)
			_, _, err := tf.ContextCmd.CreateContextWithEndPointStaging(contextNameTMC, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.ContextTypeTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameTMC), activeContextShouldBeRecentlyAddedOne)
			contexts = append(contexts, contextNameTMC)
		})

		// Test case: h. TMC: list plugins and validate plugins info, make sure all plugins are installed as per mock response
		It("Test case: h: list plugins and validate plugins being installed after context being created", func() {
			installedPluginsListTMC, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).NotTo(BeNil(), "there should be an error for plugin list as one of the active context has issues")
			Expect(len(installedPluginsListTMC)).Should(Equal(len(pluginsToGenerateMockResponse)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsListTMC, pluginsToGenerateMockResponse)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: i. plugin search should work fine even though k8s context has issues
		It("plugin search when active context has issues", func() {
			var plugins []*f.PluginInfo
			plugins, _, _, err = tf.PluginCmd.SearchPlugins("")
			Expect(err).To(BeNil(), noErrorForPluginSearch)
			Expect(len(plugins) > 0).Should(BeTrue(), shouldBeSomePluginsForPluginSearch)
		})

		// Test case: j. plugin install should work fine even though k8s context has issues
		It("plugin install when active context has issues", func() {
			err := tf.PluginCmd.DeletePlugin(installedPluginsListTMC[0].Name, installedPluginsListTMC[0].Target)
			Expect(err).To(BeNil(), noErrorForPluginDelete)
			_, _, err = tf.PluginCmd.InstallPlugin(installedPluginsListTMC[0].Name, installedPluginsListTMC[0].Target, installedPluginsListTMC[0].Version)
			Expect(err).To(BeNil(), noErrorForPluginInstall)
		})

		// Test case: k. validate plugin has installed after above plugin uninstall and install operations
		It("plugin list validation after specific plugin uninstall and install", func() {
			pluginList, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).NotTo(BeNil(), "there should be an error for plugin list as one of the active context has issues")
			Expect(f.CheckAllPluginsExists(pluginList, []*f.PluginInfo{installedPluginsListTMC[0]})).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: l. plugin sync should work fine even though k8s context has issues
		It("plugin sync when active context has issues", func() {
			err := tf.PluginCmd.DeletePlugin(installedPluginsListTMC[0].Name, installedPluginsListTMC[0].Target)
			Expect(err).To(BeNil(), noErrorForPluginDelete)
			_, _, err = tf.PluginCmd.Sync()
			Expect(err).NotTo(BeNil(), "there should be an error for plugin sync as one of the active context has issues")
		})

		// Test case: m. validate the plugin list after above plugin sync operation
		It("plugin list validation after specific plugin delete, and sync", func() {
			installedPluginsListTMC, err = tf.PluginCmd.ListRecommendedPluginsFromActiveContext(true)
			Expect(err).NotTo(BeNil(), "there should be an error for plugin list as one of the active context has issues")
			Expect(len(installedPluginsListTMC)).Should(Equal(len(pluginsToGenerateMockResponse)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsListTMC, pluginsToGenerateMockResponse)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: n. plugin group search and plugin group install when active context has issues
		It("plugin group search and plugin install by group when active context has issues", func() {
			pgs, err := pl.SearchAllPluginGroups(tf)
			Expect(err).To(BeNil())
			_, _, err = tf.PluginCmd.InstallPluginsFromGroup("", pgs[0].Name)
			Expect(err).To(BeNil(), "should be no error for plugin install by group")
		})

		// Test case: o. delete tmc/k8s contexts
		It("delete tmc/k8s contexts", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextNameTMC)
			Expect(err).To(BeNil(), "context should be deleted without error")
			err = f.StopContainer(tf, f.HTTPMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)

			_, _, err = tf.ContextCmd.DeleteContext(contextNameK8s)
			Expect(err).To(BeNil(), "context should be deleted without error")
		})
	})
})
