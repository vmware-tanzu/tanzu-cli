// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// pluginsynce2e provides plugin sync command specific E2E test cases
package pluginsynce2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Plugin-Group-lifecycle]", func() {

	// Use case 1: create a KIND cluster, don't apply CRD and CRs, create context, make sure no plugins are installed
	// a. create k8s context for the KIND cluster
	// b. create context and validate current active context
	// c. list plugins and make sure no plugins installed
	// d. delete current context and KIND cluster
	Context("plugin install from group: install a plugin from a specific plugin group", func() {
		var clusterInfo *framework.ClusterInfo
		var contextName string
		var err error
		// Test case: a. create k8s context for the KIND cluster
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = framework.CreateKindCluster(tf, "sync-e2e-"+framework.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. create context and validate current active context
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextName = "sync-e2e-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextName), "the active context should be recently added context")
		})
		// Test case: c. list plugins and make sure no plugins installed
		It("list plugins and check number plugins should be same as installed in previous test", func() {
			pluginsList, err := tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "there should not be any context specific plugins")
		})
		// Test case: d. delete current context and KIND cluster
		It("delete current context and the KIND cluster", func() {
			err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Use case 2: Create kind cluster, apply CRD and CRs, create context, should install all plugins, uninstall the specific plugin, and perform plugin sync:
	// Steps:
	// a. create KIND cluster, apply CRD
	// b. apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
	// c. create context and make sure context has created
	// d. list plugins and validate plugins info, make sure all plugins installed for which CR's has applied to KIND cluster
	// e. uninstall one of the installed plugin, make sure plugin is uninstalled,
	//		run plugin sync, make sure the uninstalled plugin has installed again.
	// f. delete current context and KIND cluster
	Context("Use case: Install KIND Cluster, Apply CRD, Apply specific plugin CRs, create context and validate plugin sync", func() {
		var clusterInfo *framework.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsInfoForCRsApplied, installedPluginsList []*framework.PluginInfo
		var contextName string
		var err error
		// Test case: a. create KIND cluster, apply CRD
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = framework.CreateKindCluster(tf, "sync-e2e-"+framework.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
		It("apply CRD and CRs to KIND cluster", func() {
			ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), CRDFilePath))

			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[framework.PluginGroupsForLifeCycleTests[0].Group]
			Expect(ok).To(BeTrue(), "plugin group is not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = framework.CreateTemporaryCRsForPluginsInGivenPluginGroup(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
		})

		// Test case: c. create context and make sure context has created
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextName = "sync-e2e-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextName), "the active context should be recently added context")
		})
		// Test case: d. list plugins and validate plugins info, make sure all plugins are installed for which CRs were present on the cluster
		It("list plugins and validate plugins being installed after context being created", func() {
			installedPluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(installedPluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: e. uninstall one of the installed plugin, make sure plugin is uninstalled,
		//				run plugin sync, make sure the uninstalled plugin has installed again.
		It("Uninstall one of the installed plugin", func() {
			pluginToUninstall := pluginsInfoForCRsApplied[0]
			err := tf.PluginCmd.UninstallPlugin(pluginToUninstall.Name, pluginToUninstall.Target)
			Expect(err).To(BeNil(), "should not get any error for plugin uninstall")

			latestPluginsInstalledList := pluginsInfoForCRsApplied[1:]
			allPluginsList, err := tf.PluginCmd.ListPluginsForGivenContext(contextName, false)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			installedPluginsList = framework.GetInstalledPlugins(allPluginsList)
			Expect(framework.IsPluginExists(allPluginsList, framework.GetGivenPluginFromTheGivenPluginList(allPluginsList, pluginToUninstall), framework.NotInstalled)).To(BeTrue(), "uninstalled plugin should be listed as not installed")
			Expect(len(installedPluginsList)).Should(Equal(len(latestPluginsInstalledList)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(installedPluginsList, latestPluginsInstalledList)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")

			_, err = tf.PluginCmd.Sync()
			Expect(err).To(BeNil(), "should not get any error for plugin sync")

			installedPluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(installedPluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), "plugins being installed and plugins info for which CRs applied should be same")
		})
		// f. delete current context and the KIND cluster
		It("delete current context and the KIND cluster", func() {
			err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})
	// Use case 3: Test plugin sync when central repo does not have all plugin CRs being applied in KIND cluster
	// Steps:
	// a. create KIND cluster
	// b. apply CRD (cluster resource definition) and CRs (cluster resource) for a few plugins which are available in the central repo and CRs for plugins which are not available in the central repo
	// c. create context and make sure context has been created
	// d. list plugins and validate plugins info, make sure all plugins installed for which CRs have applied to the KIND cluster and are available in the central repo
	// e. run plugin sync and validate the plugin list
	// f. delete the KIND cluster
	Context("Use case: Install KIND Cluster, Apply CRD, Apply specific plugin CRs, create context and validate plugin sync", func() {
		var clusterInfo *framework.ClusterInfo
		var pluginCRFilePaths, pluginWithIncorrectVerCRFilePaths []string
		var pluginsInfoForCRsApplied, pluginsWithIncorrectVer, pluginsList []*framework.PluginInfo
		var contextName string
		var err error
		// Test case: a. create KIND cluster
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = framework.CreateKindCluster(tf, "sync-e2e-"+framework.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins which are available in centra repo
		// and CR's for plugins which are not available in central repo
		It("apply CRD and CRs to KIND cluster", func() {
			ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), CRDFilePath))
			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[framework.PluginGroupsForLifeCycleTests[0].Group]
			Expect(ok).To(BeTrue(), "plugin group is not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = framework.CreateTemporaryCRsForPluginsInGivenPluginGroup(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")

			pluginWithIncorrectVersion := *pluginsToGenerateCRs[numberOfPluginsToInstall]
			pluginWithIncorrectVersion.Version = pluginWithIncorrectVersion.Version + framework.RandomNumber(2)
			pluginsWithIncorrectVer, pluginWithIncorrectVerCRFilePaths, err = framework.CreateTemporaryCRsForPluginsInGivenPluginGroup(append(make([]*framework.PluginInfo, 0), &pluginWithIncorrectVersion))
			Expect(err).To(BeNil(), "should not get any error while generating CR files")

			ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
			ApplyConfigOnKindCluster(tf, clusterInfo, pluginWithIncorrectVerCRFilePaths)
		})

		// Test case: c. create context and make sure context has created
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextName = "sync-e2e-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextName), "the active context should be recently added context")
		})
		// Test case: d. list plugins and validate plugins info, make sure all plugins installed for which CR's has applied to KIND cluster and available in central repo
		It("list plugins and validate plugins being installed after context being created", func() {
			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(pluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: e. run plugin sync and validate the plugin list
		It("Uninstall one of the installed plugin", func() {
			_, err = tf.PluginCmd.Sync()
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.UnableToFindPluginForTarget, pluginsWithIncorrectVer[0].Name, pluginsWithIncorrectVer[0].Target)))

			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(pluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), "plugins being installed and plugins info for which CRs applied should be same")
		})
		// f. delete the KIND cluster
		It("delete the KIND cluster", func() {
			err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Use case 4: test delete context use case, it should uninstall plugins installed for the context
	// Steps:
	// a. create KIND cluster
	// b. apply CRD (cluster resource definition) and CRs (cluster resource) for few plugins
	// c. create context and make sure context gets created, list plugins, make sure all
	//    plugins installed for which CRs are applied in KIND cluster
	// d. delete the context, make sure all context specific plugins are uninstalled
	// e. delete the KIND cluster
	Context("Use case: Install KIND Cluster, Apply CRD, Apply specific plugin CRs, create context and validate plugin sync", func() {
		var clusterInfo *framework.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsInfoForCRsApplied, pluginsList []*framework.PluginInfo
		var contextName string
		var err error
		// Test case: a. create KIND cluster
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = framework.CreateKindCluster(tf, "sync-e2e-"+framework.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins which are available in centra repo
		// and CR's for plugins which are not available in central repo
		It("apply CRD and CRs to KIND cluster", func() {
			ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), CRDFilePath))
			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[framework.PluginGroupsForLifeCycleTests[0].Group]
			Expect(ok).To(BeTrue(), "plugin group is not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = framework.CreateTemporaryCRsForPluginsInGivenPluginGroup(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
		})

		// Test case: c. create context and make sure context has created, list plugins, make sure all plugins installed for which CR's are applied in KIND cluster
		It("create context and validate installed plugins list, should installed all plugins for which CRs has applied in KIND cluster", func() {
			By("create context with kubeconfig and context")
			contextName = "sync-e2e-" + framework.RandomString(4)
			err = tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(pluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})
		// Test case: d. delete the context, make sure all context specific plugins are uninstalled
		It("delete context, validate installed plugins list, should uninstalled all context plugins", func() {
			err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "there should be no error for delete context")

			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "all context plugins should be uninstalled as context delete")
		})

		// Test case: e. delete the KIND cluster
		It("delete the KIND cluster", func() {
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Use case 5: test switch context use case, make installed plugins should be updated as per the context
	// Steps:
	// a. create KIND clusters
	// b. for both clusters, apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
	// c. for cluster one, create random context and validate the plugin list should show all plugins for which CRs are applied
	// d. for cluster two, create random context and validate the plugin list should show all plugins for which CRs are applied
	// e. switch context's, make sure installed plugins also updated
	// f. delete the KIND clusters
	Context("Use case: Install KIND Cluster, Apply CRD, Apply specific plugin CRs, create context and validate plugin sync", func() {
		var clusterOne, clusterTwo *framework.ClusterInfo
		var pluginCRFilePathsClusterOne, pluginCRFilePathsClusterTwo []string
		var pluginsInfoForCRsAppliedClusterOne, pluginsListClusterOne []*framework.PluginInfo
		var pluginsInfoForCRsAppliedClusterTwo, pluginsListClusterTwo []*framework.PluginInfo
		var contextNameClusterOne, contextNameClusterTwo string
		var err error

		// Test case: a. create KIND clusters
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterOne, err = framework.CreateKindCluster(tf, "sync-e2e-"+framework.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
			clusterTwo, err = framework.CreateKindCluster(tf, "sync-e2e-"+framework.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. for both clusters, apply CRD (cluster resource definition) and CR's (cluster resource) for few plugins
		It("apply CRD and CRs to KIND cluster", func() {
			ApplyConfigOnKindCluster(tf, clusterOne, append(make([]string, 0), CRDFilePath))
			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[framework.PluginGroupsForLifeCycleTests[0].Group]
			Expect(ok).To(BeTrue(), "plugin group is not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall*2).To(BeTrue(), "we don't have enough plugins in local test central repo")

			pluginsInfoForCRsAppliedClusterOne, pluginCRFilePathsClusterOne, err = framework.CreateTemporaryCRsForPluginsInGivenPluginGroup(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			ApplyConfigOnKindCluster(tf, clusterOne, pluginCRFilePathsClusterOne)

			ApplyConfigOnKindCluster(tf, clusterTwo, append(make([]string, 0), CRDFilePath))
			pluginsInfoForCRsAppliedClusterTwo, pluginCRFilePathsClusterTwo, err = framework.CreateTemporaryCRsForPluginsInGivenPluginGroup(pluginsToGenerateCRs[numberOfPluginsToInstall : numberOfPluginsToInstall*2])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			ApplyConfigOnKindCluster(tf, clusterTwo, pluginCRFilePathsClusterTwo)
		})

		// Test case: c. for cluster one, create random context and validate the plugin list should show all plugins for which CRs are applied
		It("create context and validate installed plugins list, should installed all plugins for which CRs has applied in KIND cluster", func() {
			By("create context with kubeconfig and context")
			contextNameClusterOne = "sync-e2e-" + framework.RandomString(4)
			err = tf.ContextCmd.CreateContextWithKubeconfig(contextNameClusterOne, clusterOne.KubeConfigPath, clusterOne.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			pluginsListClusterOne, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameClusterOne, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsListClusterOne)).Should(Equal(len(pluginsInfoForCRsAppliedClusterOne)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(pluginsListClusterOne, pluginsInfoForCRsAppliedClusterOne)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: d. for cluster two, create random context and validate the plugin list should show all plugins for which CRs are applied
		It("create context and validate installed plugins list, should installed all plugins for which CRs has applied in KIND cluster", func() {
			By("create context with kubeconfig and context")
			contextNameClusterTwo = "sync-e2e-" + framework.RandomString(4)
			err = tf.ContextCmd.CreateContextWithKubeconfig(contextNameClusterTwo, clusterTwo.KubeConfigPath, clusterTwo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			pluginsListClusterTwo, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameClusterTwo, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsListClusterTwo)).Should(Equal(len(pluginsInfoForCRsAppliedClusterTwo)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(pluginsListClusterTwo, pluginsInfoForCRsAppliedClusterTwo)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: e. switch context's, make sure installed plugins also updated
		It("switch context, make sure installed plugins also updated", func() {
			err = tf.ContextCmd.UseContext(contextNameClusterTwo)
			Expect(err).To(BeNil(), "there should not be any error for use context")
			pluginsListClusterTwo, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameClusterTwo, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsListClusterTwo)).Should(Equal(len(pluginsInfoForCRsAppliedClusterTwo)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(pluginsListClusterTwo, pluginsInfoForCRsAppliedClusterTwo)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")

			err = tf.ContextCmd.UseContext(contextNameClusterOne)
			Expect(err).To(BeNil(), "there should not be any error for use context")
			pluginsListClusterOne, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameClusterOne, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsListClusterOne)).Should(Equal(len(pluginsInfoForCRsAppliedClusterOne)), "number of plugins should be same as number of plugins CRs applied")
			Expect(framework.CheckAllPluginsExists(pluginsListClusterOne, pluginsInfoForCRsAppliedClusterOne)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: f. delete the KIND clusters
		It("delete the KIND cluster", func() {
			err = tf.ContextCmd.DeleteContext(contextNameClusterOne)
			Expect(err).To(BeNil(), "context should be deleted without error")
			err = tf.ContextCmd.DeleteContext(contextNameClusterTwo)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, err = tf.KindCluster.DeleteCluster(clusterOne.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
			_, err = tf.KindCluster.DeleteCluster(clusterTwo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})
})
