// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package pluginlifecyclee2e provides plugin command specific E2E test cases
package pluginlifecyclee2e

import (
	"github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

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
