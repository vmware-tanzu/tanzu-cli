// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugin provides plugin command specific E2E test cases
package plugin

import (
	"fmt"

	"github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// IsPluginSourceExists checks the sourceName is exists in the given list of PluginSourceInfo's
func IsPluginSourceExists(list []framework.PluginSourceInfo, sourceName string) bool {
	for _, val := range list {
		if val.Name == sourceName {
			return true
		}
	}
	return false
}

// CheckAllPluginsAvailable checks requiredPlugins are exists in the allPlugins
func CheckAllPluginsAvailable(allPlugins *[]framework.PluginInfo, requiredPlugins []framework.PluginInfo) bool {
	set := framework.PluginListToSet(&requiredPlugins)
	for _, plugin := range *allPlugins {
		key := fmt.Sprintf(framework.PluginKey, plugin.Name, plugin.Target, plugin.Version)
		_, ok := set[key]
		if ok {
			delete(set, key)
		}
	}
	return len(set) == 0
}

// CheckAllPluginsExists checks all PluginInfo's in subList are available in superList
// superList is the super set, subList is sub set
func CheckAllPluginsExists(superList, subList *[]framework.PluginInfo) bool {
	superSet := framework.PluginListToMap(superList)
	subSet := framework.PluginListToMap(subList)
	for key, val := range subSet {
		val2, ok := superSet[key]
		// Plugin's Name, Target and Version are part of map Key, so no need to compare/validate again here if different then we can not find the plugin in superSet map
		if !ok || val.Description != val2.Description {
			return false
		}
	}
	return true
}

// SearchAllPlugins runs the plugin search command and returns all the plugins from the search output
func SearchAllPlugins(tf *framework.Framework) *[]framework.PluginInfo {
	pluginsSearchList, err := tf.PluginCmd.SearchPlugins("")
	gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin search")
	return &pluginsSearchList
}
