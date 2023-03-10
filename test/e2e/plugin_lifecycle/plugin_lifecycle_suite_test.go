// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugin provides plugin command specific E2E test cases
package plugin

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestPluginLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PluginLifecycle Suite")
}

var (
	tf                         *framework.Framework
	e2eTestLocalCentralRepoURL string
	pluginsSearchList          []*framework.PluginInfo
	pluginGroups               []*framework.PluginGroup
	pluginGroupToPluginListMap map[string][]*framework.PluginInfo
	pluginSourceName           string
)

// BeforeSuite initializes and set up the environment to execute the plugin life cycle and plugin group life cycle end-to-end test cases
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()
	// check E2E test central repo URL (TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL)
	e2eTestLocalCentralRepoURL = os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryURL)
	Expect(e2eTestLocalCentralRepoURL).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository URL", framework.TanzuCliE2ETestLocalCentralRepositoryURL))
	// set E2E test central repo URL to TANZU_CLI_PRE_RELEASE_REPO_IMAGE
	os.Setenv(framework.CentralRepositoryPreReleaseRepoImage, e2eTestLocalCentralRepoURL)

	// search plugin groups and make sure there plugin groups available
	pluginGroups = SearchAllPluginGroups(tf)

	// check all required plugin groups (framework.PluginGroupsForLifeCycleTests) need for life cycle test are available in plugin group search output
	Expect(IsAllPluginGroupsExists(pluginGroups, framework.PluginGroupsForLifeCycleTests)).Should(BeTrue(), "all required plugin groups for life cycle tests should exists in plugin group search output")

	// search plugins and make sure there are plugins available
	pluginsSearchList = SearchAllPlugins(tf)
	Expect(len(pluginsSearchList)).Should(BeNumerically(">", 0))

	// check all required plugins (framework.PluginsForLifeCycleTests) for plugin life cycle e2e are available in plugin search output
	CheckAllPluginsAvailable(pluginsSearchList, framework.PluginsForLifeCycleTests)

	pluginGroupToPluginListMap = MapPluginsToPluginGroups(pluginsSearchList, framework.PluginGroupsForLifeCycleTests)

	// check for every plugin group (in framework.PluginGroupsForLifeCycleTests) there should be plugin's available
	for pg := range pluginGroupToPluginListMap {
		Expect(len(pluginGroupToPluginListMap[pg])).Should(BeNumerically(">", 0), "there should be at least one plugin available for each plugin group in plugin group life cycle list")
	}
})
