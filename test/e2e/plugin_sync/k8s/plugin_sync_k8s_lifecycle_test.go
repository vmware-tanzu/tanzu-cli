// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// pluginsynce2ek8s provides plugin sync command specific E2E test cases
package pluginsynce2ek8s

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	f "github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var stdOut string

var backOff = wait.Backoff{
	Steps:    10,
	Duration: 5 * time.Second,
	Factor:   1.0,
	Jitter:   0.1,
}

// Below Use cases executed in this test suite
// cleanup and initialize the config files
// Use case 1: create a KIND cluster, don't apply CRD and CRs, create context, make sure no plugins are installed
// Use case 2: Create kind cluster, apply CRD and CRs, create context, should install all plugins, uninstall the specific plugin, and perform plugin sync
// Use case 3: Test plugin sync when central repo does not have all plugin CRs being applied in KIND cluster
// Use case 4: test delete context use case, it should uninstall plugins installed for the context
// Use case 5: test switch context use case, make installed plugins should be updated as per the context

var _ = f.CLICoreDescribe("[Tests:E2E][Feature:Plugin-sync-lifecycle]", func() {

	// cleanup and initialize the config files
	Context("Delete config files and initialize", func() {
		It("Delete config files and initialize", func() {
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

	// Use case 1: create a KIND cluster, don't apply CRD and CRs, create context, make sure no plugins are installed
	// a. create k8s context for the KIND cluster
	// b. create context and validate current active context
	// c. list plugins and make sure no plugins installed
	// d. delete current context and KIND cluster
	Context("Use case 1: Install KIND Cluster, create context and validate plugin sync", func() {
		var clusterInfo *f.ClusterInfo
		var contextName string
		var err error
		// Test case: a. create k8s context for the KIND cluster
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. create context and validate current active context
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextName = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err := tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
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
			_, _, err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, _, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Use case 2: Create kind cluster, apply CRD and CRs, create context, should install all plugins, uninstall the specific plugin, and perform plugin sync:
	// Steps:
	// a. create KIND cluster, apply CRD
	// b. apply CRD and CRs for few plugins
	// c. create context and make sure context has created
	// d. list plugins and validate plugins info, make sure all plugins installed for which CR's has applied to KIND cluster
	// e. simulate context-scoped plugin upgrade by applying updated CRs again (with updated plugin versions) to KIND cluster to validate sync (BugFix: https://github.com/vmware-tanzu/tanzu-cli/issues/358)
	// f. run plugin sync and validate the plugin list (it should upgrade plugins to latest version based on the updated CRs on the cluster) (BugFix: https://github.com/vmware-tanzu/tanzu-cli/issues/358)
	// g. uninstall one of the installed plugin, make sure plugin is uninstalled,
	//		run plugin sync, make sure the uninstalled plugin has installed again.
	// h. delete current context and KIND cluster
	Context("Use case 2: Install KIND Cluster, Apply CRD, Apply specific plugin CRs, create context and validate plugin sync, simulate plugin upgrade by applyging different CRs, sync and validate plugins", func() {
		var clusterInfo *f.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsInfoForCRsApplied, installedPluginsList []*f.PluginInfo
		var contextName string
		var err error
		// Test case: a. create KIND cluster, apply CRD
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. apply CRD and CRs for few plugins
		It("apply CRD and CRs to KIND cluster", func() {
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), f.K8SCRDFilePath))
			Expect(err).To(BeNil(), "should not get any error for config apply")

			// pluginGroupToPluginListMap contains following plugin-group as keys [vmware-tkg/default:v9.9.9],[vmware-tkg/default:v0.0.1],[vmware-tmc/default:v9.9.9],[vmware-tmc/default:v0.0.1]
			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap["vmware-tkg/default:v9.9.9"]
			Expect(ok).To(BeTrue(), "plugin group does not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
			Expect(err).To(BeNil(), "should not get any error for config apply")
		})

		// Test case: c. create context and make sure context has created
		It("create context with kubeconfig and context", func() {
			contextName = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
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
			Expect(f.CheckAllPluginsExists(installedPluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})
		// Test case: e. simulate context-scoped plugin upgrade by applying updated CRs again (with updated plugin versions) to KIND cluster to validate sync
		It("apply updated CRs (with updated plugin versions) to KIND cluster to validate sync", func() {
			// pluginGroupToPluginListMap: [vmware-tkg/default:v9.9.9][]pluginInfo, [vmware-tkg/default:v0.0.1][]pluginInfo, [vmware-tmc/default:v9.9.9][]pluginInfo, [vmware-tmc/default:v0.0.1][]pluginInfo
			// usePluginsFromPluginGroup: [vmware-tkg/default:v9.9.9][]pluginInfo
			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap["vmware-tkg/default:v0.0.1"]
			Expect(ok).To(BeTrue(), "plugin group does not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")

			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
			Expect(err).To(BeNil(), "should not get any error for config apply")
		})
		// Test case: f. list plugins and validate plugins info, make sure all plugins are installed for which CRs were present on the cluster
		It("list plugins and validate plugins being installed after context being created", func() {
			installedPluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, false)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of installed status plugins should be same as number of plugins CRs applied")
			for i := range installedPluginsList {
				Expect(installedPluginsList[i].Status).To(Equal(f.UpdateAvailable), "all installed context-scoped plugin status should show 'update available'")
			}

			_, _, err = tf.PluginCmd.Sync()
			Expect(err).To(BeNil(), "should not get any error for plugin sync")
			installedPluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of installed status plugins should be same as number of plugins CRs applied")

			Expect(f.CheckAllPluginsExists(installedPluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})
		// Test case: g. uninstall one of the installed plugin, make sure plugin is uninstalled,
		//				run plugin sync, make sure the uninstalled plugin has installed again.
		It("Uninstall one of the installed plugin", func() {
			pluginToUninstall := pluginsInfoForCRsApplied[0]
			err = tf.PluginCmd.UninstallPlugin(pluginToUninstall.Name, pluginToUninstall.Target)
			Expect(err).To(BeNil(), "should not get any error for plugin uninstall")

			latestPluginsInstalledList := pluginsInfoForCRsApplied[1:]
			allPluginsList, err := tf.PluginCmd.ListPluginsForGivenContext(contextName, false)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			installedPluginsList = f.GetInstalledPlugins(allPluginsList)
			Expect(f.IsPluginExists(allPluginsList, f.GetGivenPluginFromTheGivenPluginList(allPluginsList, pluginToUninstall), f.NotInstalled)).To(BeTrue(), "uninstalled plugin should be listed as not installed")
			Expect(len(installedPluginsList)).Should(Equal(len(latestPluginsInstalledList)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(installedPluginsList, latestPluginsInstalledList)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")

			_, _, err = tf.PluginCmd.Sync()
			Expect(err).To(BeNil(), "should not get any error for plugin sync")

			installedPluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(installedPluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(installedPluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), "plugins being installed and plugins info for which CRs applied should be same")
		})
		// Test case: h.1 Unset the context and make sure plugin are deactivated, and set the context again
		It("list plugins and validate plugins being installed after context being created", func() {
			installedPluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			stdOut, _, err = tf.ContextCmd.UnsetContext(contextName)
			Expect(err).To(BeNil(), "unset context should unset context without any error")
			for i := range installedPluginsList {
				Expect(stdOut).To(ContainSubstring(fmt.Sprintf(f.DeactivatingPlugin, installedPluginsList[i].Name, installedPluginsList[i].Version, contextName)))
			}
			_, _, err = tf.ContextCmd.UseContext(contextName)
			Expect(err).To(BeNil(), "use context should set context without any error")
		})
		// Test case: h.2 delete current context and the KIND cluster
		It("delete current context and the KIND cluster", func() {
			installedPluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			stdOut, _, err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "context should be deleted without error")
			// delete context should deactivate all installed plugins
			for i := range installedPluginsList {
				Expect(stdOut).To(ContainSubstring(fmt.Sprintf(f.DeactivatingPlugin, installedPluginsList[i].Name, installedPluginsList[i].Version, contextName)))
			}
			_, _, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Use case 3: Test plugin sync when central repo does not have all plugin CRs being applied in KIND cluster
	// Steps:
	// a. create KIND cluster
	// b. apply CRD and CRs for a few plugins which are available in the central repo and CRs for plugins which are not available in the central repo
	// c. create context and make sure context has been created
	// d. list plugins and validate plugins info, make sure all plugins installed for which CRs have applied to the KIND cluster and are available in the central repo
	// e. run plugin sync and validate the plugin list
	// f. delete the KIND cluster
	Context("Use case 3: Install KIND Cluster, Apply CRD, Apply specific plugin CRs, create context and validate plugin sync", func() {
		var clusterInfo *f.ClusterInfo
		var pluginCRFilePaths, pluginWithIncorrectVerCRFilePaths []string
		var pluginsInfoForCRsApplied, pluginsWithIncorrectVer, pluginsList []*f.PluginInfo
		var contextName string
		var err error
		// Test case: a. create KIND cluster
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. apply CRD and CRs for few plugins which are available in centra repo
		// and CR's for plugins which are not available in central repo
		It("apply CRD and CRs to KIND cluster", func() {
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), f.K8SCRDFilePath))
			Expect(err).To(BeNil(), "should not get any error for config apply")
			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[usePluginsFromPluginGroup]
			Expect(ok).To(BeTrue(), "plugin group does not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")

			pluginWithIncorrectVersion := *pluginsToGenerateCRs[numberOfPluginsToInstall]
			pluginWithIncorrectVersion.Version = pluginWithIncorrectVersion.Version + f.RandomNumber(2)
			pluginsWithIncorrectVer, pluginWithIncorrectVerCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos(append(make([]*f.PluginInfo, 0), &pluginWithIncorrectVersion))
			Expect(err).To(BeNil(), "should not get any error while generating CR files")

			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
			Expect(err).To(BeNil(), "should not get any error for config apply")
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginWithIncorrectVerCRFilePaths)
			Expect(err).To(BeNil(), "should not get any error for config apply")
		})

		// Test case: c. create context and make sure context has created
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextName = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
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
			Expect(f.CheckAllPluginsExists(pluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		const (
			ContextActivated         = "Activated context '%s' (Type: %s)"
			PluginWillBeInstalled    = "The following plugins will be installed for context '%s' of contextType '%s':"
			PluginsTableHeaderRegExp = "NAME\\s+TARGET\\s+VERSION"
			PluginsRow               = "%s\\s+%s\\s+%s"
			PluginInstalledRegExp    = "Installed plugin '%s:.+' with target '%s'|Reinitialized plugin '%s:.+' with target '%s'"
		)
		// validate the 'context use' output UX
		// clean plugins, unset context, set context, validate UX
		It("perform plugin cleanup", func() {
			err = tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin cleanup")
			// unset the context
			_, _, err = tf.ContextCmd.UnsetContext(contextName)
			Expect(err).To(BeNil(), "unset context should unset context without any error")
			var stdErr string
			stdOut, stdErr, err = tf.ContextCmd.UseContext(contextName)
			Expect(len(stdOut)).Should(Equal(0), "should not get any output for context use")
			Expect(err).To(BeNil(), "use context should set context without any error")
			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(stdErr).To(ContainSubstring(fmt.Sprintf(ContextActivated, contextName, types.TargetK8s)))
			Expect(stdErr).To(ContainSubstring(fmt.Sprintf(PluginWillBeInstalled, contextName, types.TargetK8s)))
			Expect(stdErr).To(MatchRegexp(PluginsTableHeaderRegExp))
			for i := range pluginsList {
				// Validate plugin list output
				Expect(stdErr).To(MatchRegexp(fmt.Sprintf(PluginsRow, pluginsList[i].Name, pluginsList[i].Target, pluginsList[i].Version)))
			}
		})

		// Test case: e. run plugin sync and validate the plugin list
		It("run plugin sync and validate err response in plugin sync, validate plugin list output", func() {
			// sync should fail with error as there is a plugin which does not exists in repository with the given random version
			_, _, err = tf.PluginCmd.Sync()
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(f.UnableToFindPluginWithVersionForTarget, pluginsWithIncorrectVer[0].Name, pluginsWithIncorrectVer[0].Version, pluginsWithIncorrectVer[0].Target)))

			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(pluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), "plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: f. delete the KIND cluster
		It("delete the KIND cluster", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, _, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})
	// Use case 4: test delete context use case, it should uninstall plugins installed for the context
	// Steps:
	// a. create KIND cluster
	// b. apply CRD and CRs for few plugins
	// c. create context and make sure context gets created, list plugins, make sure all
	//    plugins installed for which CRs are applied in KIND cluster
	// d. delete the context, make sure all context specific plugins are uninstalled
	// e. delete the KIND cluster
	Context("Use case 4: Install KIND Cluster, Apply CRD, Apply specific plugin CRs, create context and validate plugin sync", func() {
		var clusterInfo *f.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsInfoForCRsApplied, pluginsList []*f.PluginInfo
		var contextName string
		var err error
		// Test case: a. create KIND cluster
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. apply CRD and CRs for few plugins which are available in centra repo
		// and CR's for plugins which are not available in central repo
		It("apply CRD and CRs to KIND cluster", func() {
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), f.K8SCRDFilePath))
			Expect(err).To(BeNil(), "should not get any error for config apply")
			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[usePluginsFromPluginGroup]
			Expect(ok).To(BeTrue(), "plugin group does not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")
			pluginsInfoForCRsApplied, pluginCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
			Expect(err).To(BeNil(), "should not get any error for config apply")
		})

		// Test case: c. create context and make sure context has created, list plugins, make sure all plugins installed for which CR's are applied in KIND cluster
		It("create context and validate installed plugins list, should installed all plugins for which CRs has applied in KIND cluster", func() {
			By("create context with kubeconfig and context")
			contextName = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(len(pluginsInfoForCRsApplied)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(pluginsList, pluginsInfoForCRsApplied)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})
		// Test case: d. delete the context, make sure all context specific plugins are uninstalled
		It("delete context, validate installed plugins list, should uninstalled all context plugins", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "there should be no error for delete context")

			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "all context plugins should be uninstalled as context delete")
		})

		// Test case: e. delete the KIND cluster
		It("delete the KIND cluster", func() {
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, _, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Use case 5: test switch context use case, make installed plugins should be updated as per the context
	// Steps:
	// a. create KIND clusters
	// b. for both clusters, apply CRD and CRs for few plugins
	// c. for cluster one, create random context and validate the plugin list should show all plugins for which CRs are applied
	// d. for cluster two, create random context and validate the plugin list should show all plugins for which CRs are applied
	// e. switch context's, make sure installed plugins also updated
	// f. delete the KIND clusters
	Context("Use case 5: Install KIND Cluster, Apply CRD, Apply specific plugin CRs, create context and validate plugin sync", func() {
		var clusterOne, clusterTwo *f.ClusterInfo
		var pluginCRFilePathsClusterOne, pluginCRFilePathsClusterTwo []string
		var pluginsInfoForCRsAppliedClusterOne, pluginsListClusterOne []*f.PluginInfo
		var pluginsInfoForCRsAppliedClusterTwo, pluginsListClusterTwo []*f.PluginInfo
		var contextNameClusterOne, contextNameClusterTwo string
		var err error

		// Test case: a. create KIND clusters
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterOne, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
			clusterTwo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})

		// Test case: b. for both clusters, apply CRD and CRs for few plugins
		It("apply CRD and CRs to KIND cluster", func() {
			err = f.ApplyConfigOnKindCluster(tf, clusterOne, append(make([]string, 0), f.K8SCRDFilePath))
			pluginsToGenerateCRs, ok := pluginGroupToPluginListMap[usePluginsFromPluginGroup]
			Expect(ok).To(BeTrue(), "plugin group does not exist in the map")
			Expect(len(pluginsToGenerateCRs) > numberOfPluginsToInstall).To(BeTrue(), "we don't have enough plugins in local test central repo")

			pluginsInfoForCRsAppliedClusterOne, pluginCRFilePathsClusterOne, err = f.CreateTemporaryCRsFromPluginInfos(pluginsToGenerateCRs[:numberOfPluginsToInstall])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			err = f.ApplyConfigOnKindCluster(tf, clusterOne, pluginCRFilePathsClusterOne)
			Expect(err).To(BeNil(), "should not get any error for config apply")

			err = f.ApplyConfigOnKindCluster(tf, clusterTwo, append(make([]string, 0), f.K8SCRDFilePath))
			Expect(err).To(BeNil(), "should not get any error for config apply")
			pluginsInfoForCRsAppliedClusterTwo, pluginCRFilePathsClusterTwo, err = f.CreateTemporaryCRsFromPluginInfos(pluginsToGenerateCRs[numberOfPluginsToInstall : numberOfPluginsToInstall*2])
			Expect(err).To(BeNil(), "should not get any error while generating CR files")
			err = f.ApplyConfigOnKindCluster(tf, clusterTwo, pluginCRFilePathsClusterTwo)
			Expect(err).To(BeNil(), "should not get any error for config apply")
		})

		// Test case: c. for cluster one, create random context and validate the plugin list should show all plugins for which CRs are applied
		It("create context and validate installed plugins list, should installed all plugins for which CRs has applied in KIND cluster", func() {
			By("create context with kubeconfig and context")
			contextNameClusterOne = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithKubeconfig(contextNameClusterOne, clusterOne.KubeConfigPath, clusterOne.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			pluginsListClusterOne, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameClusterOne, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsListClusterOne)).Should(Equal(len(pluginsInfoForCRsAppliedClusterOne)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(pluginsListClusterOne, pluginsInfoForCRsAppliedClusterOne)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: d. for cluster two, create random context and validate the plugin list should show all plugins for which CRs are applied
		It("create context and validate installed plugins list, should installed all plugins for which CRs has applied in KIND cluster", func() {
			By("create context with kubeconfig and context")
			contextNameClusterTwo = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithKubeconfig(contextNameClusterTwo, clusterTwo.KubeConfigPath, clusterTwo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			pluginsListClusterTwo, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameClusterTwo, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsListClusterTwo)).Should(Equal(len(pluginsInfoForCRsAppliedClusterTwo)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(pluginsListClusterTwo, pluginsInfoForCRsAppliedClusterTwo)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: e. switch context's, make sure installed plugins also updated
		It("switch context, make sure installed plugins also updated", func() {
			_, _, err = tf.ContextCmd.UseContext(contextNameClusterTwo)
			Expect(err).To(BeNil(), "there should not be any error for use context")
			pluginsListClusterTwo, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameClusterTwo, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsListClusterTwo)).Should(Equal(len(pluginsInfoForCRsAppliedClusterTwo)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(pluginsListClusterTwo, pluginsInfoForCRsAppliedClusterTwo)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")

			_, _, err = tf.ContextCmd.UseContext(contextNameClusterOne)
			Expect(err).To(BeNil(), "there should not be any error for use context")
			pluginsListClusterOne, err = tf.PluginCmd.ListPluginsForGivenContext(contextNameClusterOne, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsListClusterOne)).Should(Equal(len(pluginsInfoForCRsAppliedClusterOne)), "number of plugins should be same as number of plugins CRs applied")
			Expect(f.CheckAllPluginsExists(pluginsListClusterOne, pluginsInfoForCRsAppliedClusterOne)).Should(BeTrue(), " plugins being installed and plugins info for which CRs applied should be same")
		})

		// Test case: f. delete the KIND clusters
		It("delete the KIND cluster", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextNameClusterOne)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, _, err = tf.ContextCmd.DeleteContext(contextNameClusterTwo)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, _, err = tf.KindCluster.DeleteCluster(clusterOne.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
			_, _, err = tf.KindCluster.DeleteCluster(clusterTwo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Use case 6: Test plugin sync when discovered plugin versions are of format vMAJOR or vMAJOR.MINOR
	// Steps:
	// a. create KIND cluster
	// b. apply CRD
	// c. create context and make sure context has created
	// d. apply CRs with different plugin versions and validate plugins being installed after context being created
	// e. delete the KIND cluster
	Context("Use case 6: Install KIND Cluster, Apply CRD, Apply specific plugin CRs with vMAJOR.MINOR and/or vMAJOR combinations, create context and validate plugin sync, validate correct plugins gets installed", func() {
		var clusterInfo *f.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsList []*f.PluginInfo
		var contextName string
		var err error
		// Test case: a. create KIND cluster
		It("create KIND cluster", func() {
			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: b. apply CRD
		It("apply CRD to KIND cluster", func() {
			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, append(make([]string, 0), f.K8SCRDFilePath))
			Expect(err).To(BeNil(), "should not get any error for config apply")
		})

		// Test case: c. create context and make sure context has created
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextName = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextName), "the active context should be recently added context")
		})

		// Test case: d. apply CRs with different plugin versions and validate plugins being installed after context being created
		It("apply CRs with different plugin versions and validate plugins being installed after context being created", func() {
			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "no plugins should be available at this time")

			for _, testcase := range PluginsMultiVersionInstallTests {
				pluginInfo := testcase.pluginInfo
				_, pluginCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos([]*f.PluginInfo{&pluginInfo})
				Expect(err).To(BeNil(), "should not get any error while generating CR files")
				err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
				Expect(err).To(BeNil(), "should not get any error for config apply")

				_, _, err = tf.PluginCmd.Sync()
				if testcase.err != "" {
					Expect(err.Error()).To(ContainSubstring(testcase.err))
				} else {
					Expect(err).To(BeNil(), "should not get any error for syncing plugins")
					var pd []*f.PluginDescribe
					pd, err = tf.PluginCmd.DescribePlugin(testcase.pluginInfo.Name, testcase.pluginInfo.Target, f.GetJsonOutputFormatAdditionalFlagFunction())
					Expect(err).To(BeNil(), f.PluginDescribeShouldNotThrowErr)
					Expect(len(pd)).To(Equal(1), f.PluginDescShouldExist)
					Expect(pd[0].Name).To(Equal(testcase.pluginInfo.Name), f.PluginNameShouldMatch)
					Expect(pd[0].Version).To(Equal(testcase.expectedInstalledVersion), f.PluginNameShouldMatch)
				}
			}
		})

		// Test case: e. delete the KIND cluster
		It("delete the KIND cluster", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, _, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})

	// Perform plugin sync tests by deploying the cliplugins CRD through a package via kapp-controller
	// Requirements:
	// - CRD_PACKAGE_IMAGE is set (otherwise the test is skipped)
	// - the test has permissions to publish the crd package to set CRD_PACKAGE_IMAGE location
	Context("Deploy CRD via package", func() {
		var clusterInfo *f.ClusterInfo
		var pluginCRFilePaths []string
		var pluginsList []*f.PluginInfo
		var contextName string
		var err error

		BeforeEach(func() {
			if os.Getenv("CRD_PACKAGE_IMAGE") == "" {
				Skip("Skipping test because CRD_PACKAGE_IMAGE is not set")
			}
		})

		It("create KIND cluster, deploying kapp-controller and crd package", func() {
			const (
				kappYAML           = "../../../../package/cliplugin.cli.tanzu.vmware.com/test/kapp-controller.yaml"
				packageYAML        = "../../../../package/cliplugin.cli.tanzu.vmware.com/carvel-artifacts/packages/cliplugin.cli.tanzu.vmware.com/package.yml"
				packageinstallYAML = "../../../../package/cliplugin.cli.tanzu.vmware.com/test/package-pi.yaml"
			)

			// Create KIND cluster, which is used in test cases to create context's
			clusterInfo, err = f.CreateKindCluster(tf, f.ContextPrefixK8s+f.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")

			yamlPaths := []string{kappYAML}
			yamlPaths2 := []string{packageYAML, packageinstallYAML}

			err = f.ApplyConfigOnKindCluster(tf, clusterInfo, yamlPaths)
			Expect(err).To(BeNil(), "should not get any error for apply kapp-controller config")

			var waitArgs []string
			waitArgs = []string{"--for=condition=Ready", "pod", "-l", "app=kapp-controller", "-A"}
			err = retry.OnError(
				backOff,
				func(e error) bool { return e != nil },
				func() error {
					return tf.KindCluster.WaitForCondition(clusterInfo.ClusterKubeContext, waitArgs)
				},
			)
			Expect(err).To(BeNil(), "kapp controller should be available")

			err = retry.OnError(
				backOff,
				func(e error) bool { return e != nil },
				func() error {
					return f.ApplyConfigOnKindCluster(tf, clusterInfo, yamlPaths2)
				},
			)
			Expect(err).To(BeNil(), "should not get any error for config apply")

			waitArgs = []string{"--for=condition=established", "crd", "cliplugins.cli.tanzu.vmware.com"}
			err = retry.OnError(
				backOff,
				func(e error) bool { return e != nil },
				func() error {
					return tf.KindCluster.WaitForCondition(clusterInfo.ClusterKubeContext, waitArgs)
				},
			)
			Expect(err).To(BeNil(), "should not get any error waiting for cli plugins crd")
		})

		It("create context with kubeconfig and context", func() {
			contextName = f.ContextPrefixK8s + f.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextName), "the active context should be recently added context")
		})

		// apply CRs with different plugin versions and validate plugins being installed after context being created
		It("apply CRs with different plugin versions and validate plugins being installed after context being created", func() {
			pluginsList, err = tf.PluginCmd.ListPluginsForGivenContext(contextName, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(Equal(0), "no plugins should be available at this time")

			for _, testcase := range PluginsMultiVersionInstallTests {
				pluginInfo := testcase.pluginInfo
				_, pluginCRFilePaths, err = f.CreateTemporaryCRsFromPluginInfos([]*f.PluginInfo{&pluginInfo})
				Expect(err).To(BeNil(), "should not get any error while generating CR files")
				err = f.ApplyConfigOnKindCluster(tf, clusterInfo, pluginCRFilePaths)
				Expect(err).To(BeNil(), "should not get any error for config apply")

				_, _, err = tf.PluginCmd.Sync()
				if testcase.err != "" {
					Expect(err.Error()).To(ContainSubstring(testcase.err))
				} else {
					Expect(err).To(BeNil(), "should not get any error for syncing plugins")
					var pd []*f.PluginDescribe
					pd, err = tf.PluginCmd.DescribePlugin(testcase.pluginInfo.Name, testcase.pluginInfo.Target, f.GetJsonOutputFormatAdditionalFlagFunction())
					Expect(err).To(BeNil(), f.PluginDescribeShouldNotThrowErr)
					Expect(len(pd)).To(Equal(1), f.PluginDescShouldExist)
					Expect(pd[0].Name).To(Equal(testcase.pluginInfo.Name), f.PluginNameShouldMatch)
					Expect(pd[0].Version).To(Equal(testcase.expectedInstalledVersion), f.PluginNameShouldMatch)
				}
			}
		})

		It("delete the KIND cluster", func() {
			_, _, err = tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "context should be deleted without error")
			_, _, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})
})
