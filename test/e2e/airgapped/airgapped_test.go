// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package airgapped

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	pluginlifecyclee2e "github.com/vmware-tanzu/tanzu-cli/test/e2e/plugin_lifecycle"
)

const InvalidPath = "invalid path for \"%s\""

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Airgapped-Plugin-DownloadBundle-UploadBundle-Lifecycle]", func() {

	Context("Download plugin bundle, Upload plugin bundle and plugin lifecycle tests with plugin group 'vmware-tkg/default:v0.0.1'", func() {
		// Test case: download plugin bundle for plugin-group vmware-tkg/default:v0.0.1
		It("download plugin bundle with specific plugin-group vmware-tkg/default:v0.0.1", func() {
			err := tf.PluginCmd.DownloadPluginBundle(e2eTestLocalCentralRepoImage, []string{"vmware-tkg/default:v0.0.1"}, filepath.Join(tempDir, "plugin_bundle_vmware-tkg-default-v0.0.1.tar.gz"))
			Expect(err).To(BeNil(), "should not get any error while downloading plugin bundle with specific group")
		})

		// Test case: upload plugin bundle downloaded using vmware-tkg/default:v0.0.1 plugin-group to the airgapped repository
		It("upload plugin bundle that was downloaded using vmware-tkg/default:v0.0.1 plugin-group to the airgapped repository", func() {
			err := tf.PluginCmd.UploadPluginBundle(e2eAirgappedCentralRepo, filepath.Join(tempDir, "plugin_bundle_vmware-tkg-default-v0.0.1.tar.gz"))
			Expect(err).To(BeNil(), "should not get any error while uploading plugin bundle")
		})

		// Test case: validate that the updating the discovery source to point to new airgapped repository works
		It("update discovery source to point to new airgapped repository discovery image", func() {
			err := framework.UpdatePluginDiscoverySource(tf, e2eAirgappedCentralRepoImage)
			Expect(err).To(BeNil(), "should not get any error for plugin source update")
			// Also set the skip signature verification for the airgapped repository discovery image
			os.Setenv(constants.PluginDiscoveryImageSignatureVerificationSkipList, e2eAirgappedCentralRepoImage)
		})

		// Test case: Validate that the correct plugins and plugin group exists with `tanzu plugin search` and `tanzu plugin group search` output
		It("validate the plugins from group 'vmware-tkg/default:v0.0.1' exists", func() {
			// search plugin groups
			pluginGroups, err = pluginlifecyclee2e.SearchAllPluginGroups(tf)
			Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)
			// check all expected plugin groups are available in the `plugin group search` output from the airgapped repository
			expectedPluginGroups := []*framework.PluginGroup{{Group: "vmware-tkg/default", Latest: "v0.0.1", Description: "Desc for vmware-tkg/default:v0.0.1"}}
			Expect(framework.IsAllPluginGroupsExists(pluginGroups, expectedPluginGroups)).Should(BeTrue(), "all required plugin groups for life cycle tests should exists in plugin group search output")

			// search plugins and make sure correct number of plugins available
			// check expected plugins are available in the `plugin search` output from the airgapped repository
			expectedPlugins := pluginsForPG_TKG_001
			pluginsSearchList, err = pluginlifecyclee2e.SearchAllPlugins(tf)
			Expect(err).To(BeNil(), framework.NoErrorForPluginSearch)
			Expect(len(pluginsSearchList)).To(Equal(len(expectedPlugins)))
			Expect(framework.CheckAllPluginsExists(pluginsSearchList, expectedPlugins)).To(BeTrue())
		})

		// Test case: Validate that the plugins can be installed from the plugin-group
		It("validate that plugins can be installed from group 'vmware-tkg/default:v0.0.1'", func() {
			// All plugins should get installed from the group
			err := tf.PluginCmd.InstallPluginsFromGroup("", "vmware-tkg/default:v0.0.1")
			Expect(err).To(BeNil())

			// Verify all plugins got installed with `tanzu plugin list`
			installedPlugins, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil())
			Expect(framework.CheckAllPluginsExists(installedPlugins, pluginsForPG_TKG_001)).To(BeTrue())
		})

		// Test case: (Negative) try to install plugins that are not migrated to the airgapped repository
		It("installing plugins that are not migrated to the airgapped repository should throw an error", func() {
			// All plugins should get installed from the group
			err := tf.PluginCmd.InstallPlugin("isolated-cluster", "", "")
			Expect(err).NotTo(BeNil())
			err = tf.PluginCmd.InstallPlugin("pinniped-auth", "", "")
			Expect(err).NotTo(BeNil())
		})
	})

	Context("Download plugin bundle, Upload plugin bundle and plugin lifecycle tests with plugin group 'vmware-tmc/tmc-user:v9.9.9'", func() {
		// Test case: download plugin bundle for plugin-group vmware-tmc/tmc-user:v9.9.9
		It("download plugin bundle for plugin-group vmware-tmc/tmc-user:v9.9.9", func() {
			err := tf.PluginCmd.DownloadPluginBundle(e2eTestLocalCentralRepoImage, []string{"vmware-tmc/tmc-user:v9.9.9"}, filepath.Join(tempDir, "plugin_bundle_vmware-tmc-default-v9.9.9.tar.gz"))
			Expect(err).To(BeNil(), "should not get any error while downloading plugin bundle with specific group")
		})

		// Test case: upload plugin bundle downloaded using vmware-tmc/tmc-user:v9.9.9 plugin-group to the airgapped repository
		It("upload plugin bundle downloaded using vmware-tmc/tmc-user:v9.9.9 plugin-group to the airgapped repository", func() {
			err := tf.PluginCmd.UploadPluginBundle(e2eAirgappedCentralRepo, filepath.Join(tempDir, "plugin_bundle_vmware-tmc-default-v9.9.9.tar.gz"))
			Expect(err).To(BeNil(), "should not get any error while uploading plugin bundle with specific group")
		})

		It("validate the plugins from group 'vmware-tmc/tmc-user:v9.9.9' exists", func() {
			// search plugin groups and make sure there plugin groups available
			pluginGroups, err = pluginlifecyclee2e.SearchAllPluginGroups(tf)
			Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)
			// check all expected plugin groups are available in plugin group search output
			expectedPluginGroups := []*framework.PluginGroup{
				{Group: "vmware-tkg/default", Latest: "v0.0.1", Description: "Desc for vmware-tkg/default:v0.0.1"},
				{Group: "vmware-tmc/tmc-user", Latest: "v9.9.9", Description: "Desc for vmware-tmc/tmc-user:v9.9.9"},
			}
			Expect(framework.IsAllPluginGroupsExists(pluginGroups, expectedPluginGroups)).Should(BeTrue(), "all required plugin groups for life cycle tests should exists in plugin group search output")

			// search plugins and make sure correct number of plugins available
			// check expected plugins are available in the `plugin search` output from the airgapped repository
			pluginsSearchList, err = pluginlifecyclee2e.SearchAllPlugins(tf)
			Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)
			expectedPlugins := append(pluginsForPG_TKG_001, pluginsForPG_TMC_999...)
			Expect(len(pluginsSearchList)).To(Equal(len(expectedPlugins)))
			Expect(framework.CheckAllPluginsExists(pluginsSearchList, expectedPlugins)).To(BeTrue())
		})

		It("validate that plugins can be installed from group 'vmware-tmc/tmc-user:v9.9.9'", func() {
			// All plugins should get installed from the group
			err := tf.PluginCmd.InstallPluginsFromGroup("", "vmware-tmc/tmc-user:v9.9.9")
			Expect(err).To(BeNil())

			// Verify all plugins got installed with `tanzu plugin list`
			installedPlugins, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil())
			Expect(framework.CheckAllPluginsExists(installedPlugins, pluginsForPG_TMC_999)).To(BeTrue())
		})

	})

	Context("Download plugin bundle, Upload plugin bundle and plugin lifecycle tests with plugin group 'vmware-tmc/tmc-user:v0.0.1'", func() {
		// Test case: download plugin bundle for plugin-group vmware-tmc/tmc-user:v0.0.1
		It("download plugin bundle for plugin-group vmware-tmc/tmc-user:v0.0.1", func() {
			err := tf.PluginCmd.DownloadPluginBundle(e2eTestLocalCentralRepoImage, []string{"vmware-tmc/tmc-user:v0.0.1"}, filepath.Join(tempDir, "plugin_bundle_vmware-tmc-v0.0.1.tar.gz"))
			Expect(err).To(BeNil(), "should not get any error while downloading plugin bundle with specific group")
		})

		// Test case: upload plugin bundle downloaded using vmware-tmc/tmc-user:v0.0.1 plugin-group to the airgapped repository
		It("upload plugin bundle downloaded using vmware-tmc/tmc-user:v0.0.1 plugin-group to the airgapped repository", func() {
			err := tf.PluginCmd.UploadPluginBundle(e2eAirgappedCentralRepo, filepath.Join(tempDir, "plugin_bundle_vmware-tmc-v0.0.1.tar.gz"))
			Expect(err).To(BeNil(), "should not get any error while downloading plugin bundle with specific group")
		})

		It("validate the plugins from group 'vmware-tmc/tmc-user:v0.0.1' exists", func() {
			// search plugin groups
			pluginGroups, err = pluginlifecyclee2e.SearchAllPluginGroups(tf)
			Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)
			// check all expected plugin groups are available in the `plugin group search` output from the airgapped repository
			expectedPluginGroups := []*framework.PluginGroup{
				{Group: "vmware-tkg/default", Latest: "v0.0.1", Description: "Desc for vmware-tkg/default:v0.0.1"},
				{Group: "vmware-tmc/tmc-user", Latest: "v9.9.9", Description: "Desc for vmware-tmc/tmc-user:v9.9.9"},
			}
			Expect(framework.IsAllPluginGroupsExists(pluginGroups, expectedPluginGroups)).Should(BeTrue(), "all required plugin groups for life cycle tests should exists in plugin group search output")

			// search plugins and make sure correct number of plugins available
			// check expected plugins are available in the `plugin search` output from the airgapped repository
			expectedPlugins := append(pluginsForPG_TKG_001, pluginsForPG_TMC_999...)
			pluginsSearchList, err = pluginlifecyclee2e.SearchAllPlugins(tf)
			Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)
			Expect(len(pluginsSearchList)).To(Equal(len(expectedPlugins)))
			Expect(framework.CheckAllPluginsExists(pluginsSearchList, expectedPlugins)).To(BeTrue())
		})

		It("validate that plugins can be installed from group 'vmware-tmc/tmc-user:v0.0.1'", func() {
			// All plugins should get installed from the group
			err := tf.PluginCmd.InstallPluginsFromGroup("", "vmware-tmc/tmc-user:v0.0.1")
			Expect(err).To(BeNil())

			// Verify all plugins got installed with `tanzu plugin list`
			installedPlugins, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil())
			Expect(framework.CheckAllPluginsExists(installedPlugins, pluginsForPG_TMC_001)).To(BeTrue())
		})

	})

	Context("Download plugin bundle, Upload plugin bundle and plugin lifecycle tests without specifying any plugin group", func() {
		// Test case: download the entire plugin bundle without specifying plugin group
		It("download the entire plugin bundle without specifying plugin group", func() {
			err := tf.PluginCmd.DownloadPluginBundle(e2eTestLocalCentralRepoImage, []string{}, filepath.Join(tempDir, "plugin_bundle_complete.tar.gz"))
			Expect(err).To(BeNil(), "should not get any error while downloading plugin bundle without specifying group")
		})

		// Test case: upload plugin bundle downloaded without specifying plugin-group to the airgapped repository
		It("upload plugin bundle downloaded without specifying plugin-group to the airgapped repository", func() {
			err := tf.PluginCmd.UploadPluginBundle(e2eAirgappedCentralRepo, filepath.Join(tempDir, "plugin_bundle_complete.tar.gz"))
			Expect(err).To(BeNil(), "should not get any error while uploading plugin bundle without specifying group")
		})

		// Test case: validate that all plugins and plugin groups exists
		It("validate that all plugins and plugin groups exists", func() {
			// search plugin groups and make sure there plugin groups available
			pluginGroups, err = pluginlifecyclee2e.SearchAllPluginGroups(tf)
			Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)
			// check all expected plugin groups are available in the `plugin group search` output from the airgapped repository
			expectedPluginGroups := []*framework.PluginGroup{
				{Group: "vmware-tkg/default", Latest: "v9.9.9", Description: "Desc for vmware-tkg/default:v9.9.9"},
				{Group: "vmware-tmc/tmc-user", Latest: "v9.9.9", Description: "Desc for vmware-tmc/tmc-user:v9.9.9"},
			}
			Expect(framework.IsAllPluginGroupsExists(pluginGroups, expectedPluginGroups)).Should(BeTrue(), "all required plugin groups for life cycle tests should exists in plugin group search output")

			// search plugins and make sure correct number of plugins available
			// check expected plugins are available in the `plugin search` output from the airgapped repository
			expectedPlugins := append(pluginsForPG_TKG_999, pluginsForPG_TMC_999...)
			expectedPlugins = append(expectedPlugins, pluginsNotInAnyPG_999...)
			pluginsSearchList, err = pluginlifecyclee2e.SearchAllPlugins(tf)
			Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)
			Expect(len(pluginsSearchList)).To(Equal(len(expectedPlugins)))
			Expect(framework.CheckAllPluginsExists(pluginsSearchList, expectedPlugins)).To(BeTrue())
		})

		// Test case: validate that plugins can be installed from group newly added plugin-group 'vmware-tkg/default:v9.9.9'
		It("validate that plugins can be installed from group newly added plugin-group 'vmware-tkg/default:v9.9.9'", func() {
			// All plugins should get installed from the group
			err := tf.PluginCmd.InstallPluginsFromGroup("", "vmware-tkg/default:v9.9.9")
			Expect(err).To(BeNil())

			// Verify all plugins got installed with `tanzu plugin list`
			installedPlugins, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil())
			Expect(framework.CheckAllPluginsExists(installedPlugins, pluginsForPG_TKG_999)).To(BeTrue())
		})

		// Test case: validate that all plugins that are not part of any plugin-groups can be installed as well
		It("validate that all plugins not part of any plugin-groups can be installed as well", func() {
			// All plugins should get installed from the group
			err := tf.PluginCmd.InstallPlugin("isolated-cluster", "", "")
			Expect(err).To(BeNil())
			err = tf.PluginCmd.InstallPlugin("pinniped-auth", "", "")
			Expect(err).To(BeNil())

			// Verify above plugins got installed with `tanzu plugin list`
			installedPlugins, err := tf.PluginCmd.ListInstalledPlugins()
			Expect(err).To(BeNil())
			Expect(framework.CheckAllPluginsExists(installedPlugins, pluginsNotInAnyPG_999)).To(BeTrue())
		})

		// Test case: (negative use case) empty path for --to-tar
		It("plugin download-bundle when to-tar path is empty", func() {
			err := tf.PluginCmd.DownloadPluginBundle(e2eTestLocalCentralRepoImage, []string{}, "")
			Expect(err).NotTo(BeNil(), "should throw error for incorrect input path")
			Expect(strings.Contains(err.Error(), fmt.Sprintf(InvalidPath, ""))).To(BeTrue())
		})
		// Test case: (negative use case) directory name only for for --to-tar
		It("plugin download-bundle when to-tar path is a directory", func() {
			err := tf.PluginCmd.DownloadPluginBundle(e2eTestLocalCentralRepoImage, []string{}, tempDir)
			Expect(err).NotTo(BeNil(), "should throw error for incorrect input path")
			Expect(strings.Contains(err.Error(), fmt.Sprintf(InvalidPath, tempDir))).To(BeTrue())
		})
		// Test case: (negative use case) current directory only for --to-tar
		It("plugin download-bundle when to-tar path is current directory", func() {
			err := tf.PluginCmd.DownloadPluginBundle(e2eTestLocalCentralRepoImage, []string{}, ".")
			Expect(err).NotTo(BeNil(), "should throw error for incorrect input path")
			Expect(strings.Contains(err.Error(), fmt.Sprintf(InvalidPath, "."))).To(BeTrue())
		})
	})
})
