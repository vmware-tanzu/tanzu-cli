// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package pluginlifecyclee2e provides plugin command specific E2E test cases
package pluginlifecyclee2e

import (
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// CheckAllLegacyPluginsExists checks all subList plugins exist in superList
func CheckAllLegacyPluginsExists(superList, subList []*framework.PluginInfo) bool {
	superSet := framework.LegacyPluginListToMap(superList)
	subSet := framework.LegacyPluginListToMap(subList)
	for key := range subSet {
		_, ok := superSet[key]
		// Plugin's Name, Target and Version are part of map Key, so no need to compare/validate again here if different then we can not find the plugin in superSet map
		if !ok {
			return false
		}
	}
	return true
}

// SearchAllPlugins runs the plugin search command and returns all the plugins from the search output
func SearchAllPlugins(tf *framework.Framework, opts ...framework.E2EOption) ([]*framework.PluginInfo, error) {
	pluginsSearchList, err := tf.PluginCmd.SearchPlugins("", opts...)
	return pluginsSearchList, err
}

// SearchAllPluginGroups runs the plugin group search command and returns all the plugin groups
func SearchAllPluginGroups(tf *framework.Framework, opts ...framework.E2EOption) ([]*framework.PluginGroup, error) {
	pluginGroups, err := tf.PluginCmd.SearchPluginGroups("", opts...)
	return pluginGroups, err
}
