// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// pluginsynce2etmc provides plugin sync command specific E2E test cases for tmc target
package pluginsynce2etmc

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	f "github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

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
			err := tf.Config.DeleteCLIConfigurationFiles()
			Expect(err).To(BeNil())
			// call init
			err = tf.Config.ConfigInit()
			Expect(err).To(BeNil())

			// update plugin discovery source
			err = f.UpdatePluginDiscoverySource(tf, e2eTestLocalCentralRepoURL)
			Expect(err).To(BeNil(), "should not get any error for plugin source update")

			// Add Cert
			_, err = tf.Config.ConfigCertAdd(&f.CertAddOptions{Host: e2eTestLocalCentralRepoPluginHost, CACertificatePath: e2eTestLocalCentralRepoCACertPath, SkipCertVerify: "false", Insecure: "false"})
			Expect(err).To(BeNil(), "should not be any error for cert add")
			list, err := tf.Config.ConfigCertList()
			Expect(err).To(BeNil(), "should not be any error for cert list")
			Expect(len(list)).To(Equal(1), "should not be any error for cert list")
		})
	})

	// Use case 1: create context when there are no plugins to sync
	// a. create empty mock response, and start mock http server
	// b. create context and validate current active context
	// c. list plugins and make sure no plugins installed
	// d. delete current context
	Context("Use case: create context when there are no plugins to sync", func() {
		var contextName string
		var err error
		// Test case: a. create empty mock response, and start mock http server
		It("mock tmc endpoint with expected plugins response and start REST API mock server", func() {
			pluginsToGenerateMockResponse := make([]*f.PluginInfo, 0)
			mockReqResMapping, err := f.ConvertPluginsInfoToTMCEndpointMockResponse(pluginsToGenerateMockResponse)
			Expect(err).To(BeNil(), noErrorForMockResponsePreparation)
			err = f.WriteToFileInJSONFormat(mockReqResMapping, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HttpMockServerName)
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
			_, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextName, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextName), activeContextShouldBeRecentlyAddedOne)
		})
		// Test case: c. list plugins and make sure no plugins installed
		It("list plugins and check number plugins should be same as installed in previous test", func() {
			pluginsList, err := tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsList)).Should(Equal(0), "there should not be any context specific plugins")
		})
		// Test case: d. delete current context
		It("delete current context and stop mock server", func() {
			err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), deleteContextWithoutError)
			err = f.StopContainer(tf, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use case 2: Created a context (make sure sync installs plugins), uninstall plugin, run plugins sync (should install uninstall plugins)
	// Steps:
	// a. mock tmc endpoint with plugins info, start the mock server
	// b. create context and make sure context has created
	// c. list plugins and validate plugins info, make sure all plugins are installed as per mock response
	// d. uninstall one of the installed plugin, make sure plugin is uninstalled, run plugin sync, make sure the uninstalled plugin has installed again.
	// e. delete current context, make sure all context specific plugins are uninstalled
	Context("Use case: create context, uninstall plugin, sync plugins", func() {
		//var pluginCRFilePaths []string
		var pluginsToGenerateMockResponse, installedPluginsList, pluginsList []*f.PluginInfo
		var contextName string
		var err error
		var ok bool

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
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HttpMockServerName)
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
			_, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextName, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextName), activeContextShouldBeRecentlyAddedOne)
		})

		// Test case: c. list plugins and validate plugins info, make sure all plugins are installed as per mock response
		It("list plugins and validate plugins being installed after context being created", func() {
			installedPluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(installedPluginsList)).Should(Equal(len(pluginsToGenerateMockResponse)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsList, pluginsToGenerateMockResponse)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: d. uninstall one of the installed plugin, make sure plugin is uninstalled,
		//				run plugin sync, make sure the uninstalled plugin has installed again.
		It("uninstall one of the installed plugin", func() {
			pluginToUninstall := pluginsToGenerateMockResponse[0]
			err := tf.PluginCmd.UninstallPlugin(pluginToUninstall.Name, pluginToUninstall.Target)
			Expect(err).To(BeNil(), "should not get any error for plugin uninstall")

			latestPluginsInstalledList := pluginsToGenerateMockResponse[1:]
			allPluginsList, err := tf.PluginCmd.ListPluginsForGivenContext(contextName, false)
			Expect(err).To(BeNil(), noErrorForPluginList)
			installedPluginsList = f.GetInstalledPlugins(allPluginsList)
			Expect(f.IsPluginExists(allPluginsList, f.GetGivenPluginFromTheGivenPluginList(allPluginsList, pluginToUninstall), f.NotInstalled)).To(BeTrue(), "uninstalled plugin should be listed as not installed")
			Expect(len(installedPluginsList)).Should(Equal(len(latestPluginsInstalledList)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsList, latestPluginsInstalledList)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)

			_, err = tf.PluginCmd.Sync()
			Expect(err).To(BeNil(), "should not get any error for plugin sync")

			installedPluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(installedPluginsList)).Should(Equal(len(pluginsToGenerateMockResponse)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsList, pluginsToGenerateMockResponse)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})
		// e. delete current context, make sure all context specific plugins are uninstalled
		It("delete current context", func() {
			err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), deleteContextWithoutError)

			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsList)).Should(Equal(0), "all context plugins should be uninstalled as context delete")
			err = f.StopContainer(tf, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use case 3: create a context (plugin sync) when test central repo does not have all plugins mocked in tmc endpoint response
	// Steps:
	// a. update tmc endpoint mock response (make sure at least a plugin does not exists in test central repo) and restart REST API mock server
	// b. create context and make sure context has created
	// c. list plugins and validate plugins info, make sure all plugins installed for which response mocked and available in central repo, the unavailable plugin (with incorrect version) should not be installed
	// d. run plugin sync and validate the plugin list
	// e. delete current context

	Context("Use case: Create context should not install unavailable plugins, plugin sync also should not install unavailable plugins", func() {
		var pluginsToGenerateMockResponse, pluginsGeneratedMockResponseWithCorrectInfo, pluginsList []*f.PluginInfo
		var pluginWithIncorrectVersion *f.PluginInfo
		var contextName, randomPluginVersion string
		var err error
		var ok bool
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
			err = f.WriteToFileInJSONFormat(mockReqResMapping, tmcPluginsMockFilePath)
			Expect(err).To(BeNil(), noErrorForMockResponseFileUpdate)

			// assign the original version back
			pluginsToGenerateMockResponse[numberOfPluginsToInstall-1].Version = actualVersion

			// skip last plugin in the slice as it has incorrect version info, which is not available in the mock response
			pluginsGeneratedMockResponseWithCorrectInfo = pluginsToGenerateMockResponse[:numberOfPluginsToInstall-1]
			// start http mock server
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HttpMockServerName)
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
			_, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextName, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextName), activeContextShouldBeRecentlyAddedOne)
		})

		// Test case: c. list plugins and validate plugins info, make sure all plugins installed for which response mocked and available in central repo, the unavailable plugin (with incorrect version) should not be installed
		It("list plugins and validate plugins being installed after context being created", func() {
			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsList)).Should(Equal(len(pluginsGeneratedMockResponseWithCorrectInfo)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(pluginsList, pluginsGeneratedMockResponseWithCorrectInfo)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: d. run plugin sync and validate the plugin list
		It("run plugin sync and validate err response in plugin sync, validate plugin list output", func() {
			_, err = tf.PluginCmd.Sync()
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(f.UnableToFindPluginWithVersionForTarget, pluginWithIncorrectVersion.Name, randomPluginVersion, pluginWithIncorrectVersion.Target)))

			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsList)).Should(Equal(len(pluginsGeneratedMockResponseWithCorrectInfo)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(pluginsList, pluginsGeneratedMockResponseWithCorrectInfo)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})
		// e. delete current context and stop mock server
		It("delete current context", func() {
			err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), deleteContextWithoutError)
			err = f.StopContainer(tf, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use case 4: Create two contexts, perform switch context use case, make installed plugins should be updated as per the context
	// Steps:
	// a. check the plugins availability
	// b. update mock response for first context, restart mock server, create context, and validate plugin list
	// c. update mock response for second context, restart mock server, create context, and validate plugin list
	// e. delete both contexts
	Context("Use case: create two contexts, and validate plugin list ", func() {
		var pluginsFromPluginGroup, pluginsToGenerateMockResponseOne, pluginsToGenerateMockResponseTwo, pluginsListOne, pluginsListTwo []*f.PluginInfo
		var contextNameOne, contextNameTwo string
		var TMCEndpointResponseOne, TMCEndpointResponseTwo *f.TMCPluginsMockRequestResponseMapping
		var err error
		var ok bool

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
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)

			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponseOne)), "the number of plugins in endpoint response and initially mocked should be same")

			contextNameOne = f.ContextPrefixTMC + f.RandomString(4)
			_, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextNameOne, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameOne), activeContextShouldBeRecentlyAddedOne)

			pluginsListOne, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameOne, true)
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
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)

			// check the tmc mocked endpoint is working as expected
			var mockResPluginsInfo f.TMCPluginsInfo
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponseTwo)), "the number of plugins in endpoint response and initially mocked should be same")

			contextNameTwo = f.ContextPrefixTMC + f.RandomString(4)
			_, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextNameTwo, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameTwo), activeContextShouldBeRecentlyAddedOne)

			pluginsListTwo, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameTwo, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsListTwo)).Should(Equal(len(pluginsToGenerateMockResponseTwo)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(pluginsListTwo, pluginsToGenerateMockResponseTwo)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// d. delete both contexts
		It("delete current context", func() {
			err = tf.ContextCmd.DeleteContext(contextNameOne)
			Expect(err).To(BeNil(), deleteContextWithoutError)
			err = tf.ContextCmd.DeleteContext(contextNameTwo)
			Expect(err).To(BeNil(), deleteContextWithoutError)
			err = f.StopContainer(tf, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use Case 5: Update the TMC endpoint with additional plugins, and ensure that the plugin sync updates the latest additional plugins.
	// Steps:
	// a. check the plugins availability
	// b. update mock response, restart mock server, create context, and validate plugin list
	// c. update mock response with additional plugins, restart mock server, run plugin sync, and validate plugin list
	// d. delete context
	Context("Use case: create two contexts, and validate plugin list ", func() {
		var pluginsFromPluginGroup, pluginsToGenerateMockResponseOne, pluginsToGenerateMockResponseTwo, pluginsListOne []*f.PluginInfo
		var contextNameOne string
		var TMCEndpointResponseOne, TMCEndpointResponseTwo *f.TMCPluginsMockRequestResponseMapping
		var err error
		var ok bool

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
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)

			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponseOne)), "the number of plugins in endpoint response and initially mocked should be same")

			contextNameOne = f.ContextPrefixTMC + f.RandomString(4)
			_, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextNameOne, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameOne), activeContextShouldBeRecentlyAddedOne)

			pluginsListOne, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameOne, true)
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
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)

			// check the tmc mocked endpoint is working as expected
			var mockResPluginsInfo f.TMCPluginsInfo
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponseTwo)), "the number of plugins in endpoint response and initially mocked should be same")

			_, err = tf.PluginCmd.Sync()
			Expect(err).To(BeNil(), "should not get any error for plugin sync")

			pluginsListOne, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameOne, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(pluginsListOne)).Should(Equal(len(pluginsToGenerateMockResponseTwo)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(pluginsListOne, pluginsToGenerateMockResponseTwo)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// d. delete both contexts
		It("delete current context", func() {
			err = tf.ContextCmd.DeleteContext(contextNameOne)
			Expect(err).To(BeNil(), deleteContextWithoutError)
			err = f.StopContainer(tf, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)
		})
	})

	// Use case 6: k8s and tmc contexts coexistnce, create k8s context and tmc context, validate plugins list and plugin sync
	// test cases:
	// Test case: a. k8s: create KIND cluster, apply CRD
	// Test case: b. k8s: apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
	// Test case: c. k8s: create context and make sure context has created
	// Test case: d. k8s: list plugins and validate plugins info, make sure all plugins are installed for which CRs were present on the cluster
	// Test case: e. TMC: mock tmc endpoint with plugins info, start the mock server
	// Test case: f. TMC: create context and make sure context has created
	// Test case: g. TMC: list plugins and validate plugins info, make sure all plugins are installed as per mock response
	// Test case: h. k8s: use first k8s context and check plugin list
	// Test case: i. k8s: uninstall one of the installed plugin, make sure plugin is uninstalled, run plugin sync, make sure the uninstalled plugin has installed again.
	// Test case: j. tmc: use tmc context and check plugin list
	// Test case: k. delete tmc/k8s contexts and the KIND cluster
	// f. delete current context and KIND cluster
	Context("Use case: create k8s and tmc specific contexts, validate plugins list and perform pluin sync, and perform context switch", func() {
		var clusterInfo *f.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsInfoForCRsApplied, installedPluginsListK8s []*f.PluginInfo
		var contextNameK8s string
		var err error
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
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = f.CreateTemporaryCRsForPluginsInGivenPluginGroup(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
			Expect(err).To(BeNil(), "should not get any error for config apply")
		})

		// Test case: c. k8s: create context and make sure context has created
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextNameK8s = f.ContextPrefixK8s + f.RandomString(4)
			err := tf.ContextCmd.CreateContextWithKubeconfig(contextNameK8s, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNameK8s), "the active context should be recently added context")
		})
		// Test case: d. k8s: list plugins and validate plugins info, make sure all plugins are installed for which CRs were present on the cluster
		It("list plugins and validate plugins being installed after context being created", func() {
			installedPluginsListK8s, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameK8s, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsListK8s)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(installedPluginsListK8s, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		var pluginsToGenerateMockResponse, installedPluginsListTMC []*f.PluginInfo
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
			err = f.StartMockServer(tf, tmcConfigFolderPath, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStartWithoutError)
			var mockResPluginsInfo f.TMCPluginsInfo
			// check the tmc mocked endpoint is working as expected
			err = f.GetHTTPCall(f.TMCPluginsMockServerEndpoint, &mockResPluginsInfo)
			Expect(err).To(BeNil(), "there should not be any error for GET http call on mockapi endpoint:"+f.TMCPluginsMockServerEndpoint)
			Expect(len(mockResPluginsInfo.Plugins)).Should(Equal(len(pluginsToGenerateMockResponse)), "the number of plugins in endpoint response and initially mocked should be same")
		})
		// Test case: f. TMC: create context and make sure context has created
		It("create context for TMC target with http mock server URL as endpoint", func() {
			contextNameTMC = f.ContextPrefixTMC + f.RandomString(4)
			_, err = tf.ContextCmd.CreateContextWithEndPointStaging(contextNameTMC, f.TMCMockServerEndpoint, f.AddAdditionalFlagAndValue(forceCSPFlag))
			Expect(err).To(BeNil(), noErrorWhileCreatingContext)
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameTMC), activeContextShouldBeRecentlyAddedOne)
		})

		// Test case: g. TMC: list plugins and validate plugins info, make sure all plugins are installed as per mock response
		It("list plugins and validate plugins being installed after context being created", func() {
			installedPluginsListTMC, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameTMC, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(installedPluginsListTMC)).Should(Equal(len(pluginsToGenerateMockResponse)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsListTMC, pluginsToGenerateMockResponse)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: h. k8s: use first k8s context and check plugin list
		It("use first context, check plugin list", func() {
			err = tf.ContextCmd.UseContext(contextNameK8s)
			Expect(err).To(BeNil(), "use context should not return any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameK8s), active, "the active context should be the recently switched one")

			installedPluginsListK8s, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameK8s, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsListK8s)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(installedPluginsListK8s, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: i. k8s: uninstall one of the installed plugin, make sure plugin is uninstalled, run plugin sync, make sure the uninstalled plugin has installed again.
		It("Uninstall one of the installed plugin", func() {
			pluginToUninstall := pluginsInfoForCRsApplied[0]
			err := tf.PluginCmd.UninstallPlugin(pluginToUninstall.Name, pluginToUninstall.Target)
			Expect(err).To(BeNil(), "should not get any error for plugin uninstall")

			latestPluginsInstalledList := pluginsInfoForCRsApplied[1:]
			allPluginsList, err := tf.PluginCmd.ListPluginsForGivenContext(contextNameK8s, false)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			installedPluginsListK8s = f.GetInstalledPlugins(allPluginsList)
			Expect(f.IsPluginExists(allPluginsList, f.GetGivenPluginFromTheGivenPluginList(allPluginsList, pluginToUninstall), f.NotInstalled)).To(BeTrue(), "uninstalled plugin should be listed as not installed")
			Expect(len(installedPluginsListK8s)).Should(Equal(len(latestPluginsInstalledList)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(installedPluginsListK8s, latestPluginsInstalledList)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")

			_, err = tf.PluginCmd.Sync()
			Expect(err).To(BeNil(), "should not get any error for plugin sync")

			installedPluginsListK8s, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameK8s, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsListK8s)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(installedPluginsListK8s, pluginsInfoForCRsApplied)).Should(BeTrue(), "plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: j. tmc: use tmc context and check plugin list
		It("use second context again, check plugin list", func() {

			err = tf.ContextCmd.UseContext(contextNameTMC)
			Expect(err).To(BeNil(), "use context should not return any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), activeContextShouldExists)
			Expect(active).To(Equal(contextNameTMC), activeContextShouldBeRecentlyAddedOne)

			// use context will install the plugins from the tmc endpoint, which is second set of plugins list
			installedPluginsListTMC, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameTMC, true)
			Expect(err).To(BeNil(), noErrorForPluginList)
			Expect(len(installedPluginsListTMC)).Should(Equal(len(pluginsToGenerateMockResponse)), numberOfPluginsSameAsNoOfPluginsInfoMocked)
			Expect(f.CheckAllPluginsExists(installedPluginsListTMC, pluginsToGenerateMockResponse)).Should(BeTrue(), pluginsInstalledAndMockedShouldBeSame)
		})

		// Test case: k. delete tmc/k8s contexts and the KIND cluster
		It("delete tmc/k8s contexts and the KIND cluster", func() {
			err = tf.ContextCmd.DeleteContext(contextNameTMC)
			Expect(err).To(BeNil(), "context should be deleted without error")
			err = f.StopContainer(tf, f.HttpMockServerName)
			Expect(err).To(BeNil(), mockServerShouldStopWithoutError)

			err = tf.ContextCmd.DeleteContext(contextNameK8s)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})
})
