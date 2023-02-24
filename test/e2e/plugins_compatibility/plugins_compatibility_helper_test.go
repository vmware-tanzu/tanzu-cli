// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugincompatibility provides plugins compatibility E2E test cases
package plugincompatibility

import (
	"os"
	"strings"

	"github.com/aunum/log"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// PluginsForCompatibilityTesting search for test-plugin-'s from the test central repository and returns all test-plugin-'s
func PluginsForCompatibilityTesting(tf *framework.Framework) []string {
	list, err := tf.PluginCmd.SearchPlugins("")
	Expect(err).To(BeNil(), "should not occur any error while searching for plugins")
	testPlugins := make([]string, 0)
	for _, plugin := range list {
		if strings.HasPrefix(plugin.Name, framework.TestPluginsPrefix) {
			testPlugins = append(testPlugins, plugin.Name)
		}
	}
	if len(testPlugins) == 0 {
		url := os.Getenv(framework.TanzuCliE2ETestCentralRepositoryURL)
		log.Errorf("there are no test-plugin-'s in test central repo:%s make sure, its valid test central repo with test-plugins", url)
	}
	return testPlugins
}
