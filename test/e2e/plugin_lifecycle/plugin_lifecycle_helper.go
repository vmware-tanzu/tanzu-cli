// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugin provides plugin command specific E2E test cases
package plugin

import (
	"fmt"

	"github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// IsPluginSourceExists checks the sourceName is exists in the given list of PluginSourceInfo's
func IsPluginSourceExists(list []*framework.PluginSourceInfo, sourceName string) bool {
	for _, val := range list {
		if val.Name == sourceName {
			return true
		}
	}
	return false
}

// CheckAllPluginsAvailable checks requiredPlugins are exists in the allPlugins
func CheckAllPluginsAvailable(allPlugins, requiredPlugins []*framework.PluginInfo) bool {
	set := framework.PluginListToSet(requiredPlugins)
	for _, plugin := range allPlugins {
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
func CheckAllPluginsExists(superList, subList []*framework.PluginInfo) bool {
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
func SearchAllPlugins(tf *framework.Framework) []*framework.PluginInfo {
	pluginsSearchList, err := tf.PluginCmd.SearchPlugins("")
	gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin search")
	return pluginsSearchList
}

// SearchAllPluginGroups runs the plugin group search command and returns all the plugin groups
func SearchAllPluginGroups(tf *framework.Framework) []*framework.PluginGroup {
	pluginGroups, err := tf.PluginCmd.SearchPluginGroups("")
	gomega.Expect(err).To(gomega.BeNil(), "should not get any error for plugin search")
	return pluginGroups
}

// IsAllPluginGroupsExists takes the two list of PluginGroups (super list and sub list), check if all sub list PluginGroup are exists in super list PluginGroup
func IsAllPluginGroupsExists(superList, subList []*framework.PluginGroup) bool {
	superMap := framework.PluginGroupToMap(superList)
	subMap := framework.PluginGroupToMap(subList)
	for ele := range subMap {
		_, exists := superMap[ele]
		if !exists {
			return false
		}
	}
	return true
}

func MapPluginsToPluginGroups(list []*framework.PluginInfo, pg []*framework.PluginGroup) map[string][]*framework.PluginInfo {
	m := make(map[string][]*framework.PluginInfo)
	for _, pluginGroup := range pg {
		m[pluginGroup.Group] = make([]*framework.PluginInfo, 0)
	}
	for i := range list {
		plugin := list[i]
		key := "vmware-"
		if plugin.Target == string(types.TargetK8s) {
			key += framework.TKG + "/"
		} else if plugin.Target == string(types.TargetTMC) {
			key += framework.TMC + "/"
		}
		key += plugin.Version
		pluginList, ok := m[key]
		if ok {
			pluginList = append(pluginList, plugin)
			m[key] = pluginList
		}
	}
	return m
}

// GetPluginFromFirstListButNotExistsInSecondList returns a plugin which is exists in first plugin list but not in second plugin list
func GetPluginFromFirstListButNotExistsInSecondList(first, second []*framework.PluginInfo) (*framework.PluginInfo, error) {
	m1 := framework.PluginListToMap(first)
	m2 := framework.PluginListToMap(second)
	for plugin := range m1 {
		if _, ok := m2[plugin]; !ok {
			return m1[plugin], nil
		}
	}
	return nil, fmt.Errorf("there is no plugin which is not common in the given pluginInfo's")
}
