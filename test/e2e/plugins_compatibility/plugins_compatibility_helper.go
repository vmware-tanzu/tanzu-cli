// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugincompatibility provides plugins compatibility E2E test cases
package plugincompatibility

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// PluginsForCompatibilityTesting search for test-plugin-'s from the test central repository and returns all test-plugin-'s
func PluginsForCompatibilityTesting(tf *framework.Framework) []*framework.PluginInfo {
	list, err := tf.PluginCmd.SearchPlugins("")
	gomega.Expect(err).To(gomega.BeNil(), "should not occur any error while searching for plugins")
	testPlugins := make([]*framework.PluginInfo, 0)
	for _, plugin := range list {
		if strings.HasPrefix(plugin.Name, framework.TestPluginsPrefix) {
			testPlugins = append(testPlugins, plugin)
		}
	}
	return testPlugins
}

// IsAllPluginsInstalled takes list of plugins and checks if all plugins are installed
func IsAllPluginsInstalled(tf *framework.Framework, plugins []*framework.PluginInfo) bool {
	pluginListOutput, err := tf.PluginCmd.ListPlugins()
	gomega.Expect(err).To(gomega.BeNil(), "should not occur any error while listing plugins")
	pluginMap := framework.PluginListToSet(plugins)
	for _, pluginInfo := range pluginListOutput {
		key := fmt.Sprintf(framework.PluginKey, pluginInfo.Name, pluginInfo.Target, pluginInfo.Version)
		_, ok := pluginMap[key]
		if ok && pluginInfo.Status == framework.Installed {
			delete(pluginMap, key)
		}
	}
	return len(pluginMap) == 0
}

// IsAllPluginsUnInstalled takes list of plugins and checks if all plugins are uninstalled
func IsAllPluginsUnInstalled(tf *framework.Framework, plugins []*framework.PluginInfo) bool {
	pluginListOutput, err := tf.PluginCmd.ListPlugins()
	gomega.Expect(err).To(gomega.BeNil(), "should not occur any error while listing plugins")
	pluginMap := framework.PluginListToSet(plugins)
	for _, pluginInfo := range pluginListOutput {
		key := fmt.Sprintf(framework.PluginKey, pluginInfo.Name, pluginInfo.Target, pluginInfo.Version)
		_, ok := pluginMap[key]
		if ok && pluginInfo.Status == framework.Installed {
			delete(pluginMap, key)
			log.Errorf(" %s plugin is installed", pluginInfo.Name)
		}
	}
	return len(pluginMap) == len(plugins)
}

// UninstallPlugins lists plugins and uninstalls provided plugins if any plugins are installed
func UninstallPlugins(tf *framework.Framework, plugins []*framework.PluginInfo) {
	pluginListOutput, err := tf.PluginCmd.ListPlugins()
	gomega.Expect(err).To(gomega.BeNil(), "should not occur any error while listing plugins")
	pluginMap := framework.PluginListToSet(plugins)
	for _, pluginInfo := range pluginListOutput {
		key := fmt.Sprintf(framework.PluginKey, pluginInfo.Name, pluginInfo.Target, pluginInfo.Version)
		_, ok := pluginMap[key]
		if ok && pluginInfo.Status == framework.Installed {
			err := tf.PluginCmd.UninstallPlugin(pluginInfo.Name, pluginInfo.Target)
			gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("error while uninstalling plugin: %s", pluginInfo.Name))
		}
	}
}
