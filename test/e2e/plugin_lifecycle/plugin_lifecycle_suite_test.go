// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// pluginlifecyclee2e provides plugin command specific E2E test cases
package pluginlifecyclee2e

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/util"
)

func TestPluginLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Lifecycle E2E Test Suite")
}

var (
	tf                         *framework.Framework
	e2eTestLocalCentralRepoURL string
	pluginsSearchList          []*framework.PluginInfo
	pluginGroups               []*framework.PluginGroup
	pluginGroupToPluginListMap map[string][]*framework.PluginInfo
)

const errorNoDiscoverySourcesFound = "there are no plugin discovery sources available. Please run 'tanzu plugin source init'"

// BeforeSuite initializes and set up the environment to execute the plugin life cycle and plugin group life cycle end-to-end test cases
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()

	err := framework.CleanConfigFiles(tf)
	Expect(err).To(BeNil())

	// Delete default plugin source, and perform negative test cases, then initialize the plugin source
	testWithoutPluginDiscoverySources(tf)

	// check E2E test central repo URL (TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL)
	e2eTestLocalCentralRepoURL = os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryURL)
	Expect(e2eTestLocalCentralRepoURL).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository URL", framework.TanzuCliE2ETestLocalCentralRepositoryURL))

	e2eTestLocalCentralRepoPluginHost := os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryHost)
	Expect(e2eTestLocalCentralRepoPluginHost).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository host", framework.TanzuCliE2ETestLocalCentralRepositoryHost))

	e2eTestLocalCentralRepoCACertPath := os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryCACertPath)
	Expect(e2eTestLocalCentralRepoCACertPath).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository CA cert path", framework.TanzuCliE2ETestLocalCentralRepositoryCACertPath))

	// set up the CA cert fort local central repository
	_ = tf.Config.ConfigCertDelete(e2eTestLocalCentralRepoPluginHost)
	_, err = tf.Config.ConfigCertAdd(&framework.CertAddOptions{Host: e2eTestLocalCentralRepoPluginHost, CACertificatePath: e2eTestLocalCentralRepoCACertPath, SkipCertVerify: "false", Insecure: "false"})
	Expect(err).To(BeNil())

	// set up the local central repository discovery image public key path
	e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath := os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryPluginDiscoveryImageSignaturePublicKeyPath)
	Expect(e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository discovery image signature public key path", framework.TanzuCliE2ETestLocalCentralRepositoryPluginDiscoveryImageSignaturePublicKeyPath))
	os.Setenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath, e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath)

	// set up the test central repo
	err = framework.UpdatePluginDiscoverySource(tf, e2eTestLocalCentralRepoURL)
	Expect(err).To(BeNil(), "should not get any error for plugin source update")

	// search plugin groups and make sure there plugin groups available
	pluginGroups, err = SearchAllPluginGroups(tf)
	Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)

	// check all required plugin groups (framework.PluginGroupsForLifeCycleTests) need for life cycle test are available in plugin group search output
	Expect(framework.IsAllPluginGroupsExists(pluginGroups, util.PluginGroupsForLifeCycleTests)).Should(BeTrue(), "all required plugin groups for life cycle tests should exists in plugin group search output")

	// search plugins and make sure there are plugins available
	pluginsSearchList, err = SearchAllPlugins(tf)
	Expect(err).To(BeNil(), framework.NoErrorForPluginSearch)
	Expect(len(pluginsSearchList)).Should(BeNumerically(">", 0))

	// check all required plugins (framework.PluginsForLifeCycleTests) for plugin life cycle e2e are available in plugin search output
	Expect(framework.CheckAllPluginsExists(pluginsSearchList, util.PluginsForLifeCycleTests)).To(BeTrue())

	pluginGroupToPluginListMap = make(map[string][]*framework.PluginInfo)
	for _, pg := range util.PluginGroupsForLifeCycleTests {
		plugins, err := GetAllPluginsFromGroup(tf, pg)
		Expect(err).To(BeNil(), framework.NoErrorForPluginGroupGet)

		key := pg.Group + ":" + pg.Latest
		pluginGroupToPluginListMap[key] = make([]*framework.PluginInfo, 0)
		for _, p := range plugins {
			pluginGroupToPluginListMap[key] = append(pluginGroupToPluginListMap[key], &framework.PluginInfo{
				Name:    p.PluginName,
				Target:  p.PluginTarget,
				Version: p.PluginVersion,
			})
		}
		// check for every plugin group (in framework.PluginGroupsForLifeCycleTests) there should be plugins available
		Expect(len(pluginGroupToPluginListMap[key])).Should(BeNumerically(">", 0), "there should be at least one plugin available for each plugin group in plugin group life cycle list")
	}
})

// testWithoutPluginDiscoverySources executes negative test cases when there are no plugin sources available
// it perfrom below use cases
// delete the default plugin discovery source
// tanzu plugin search
// tanzu plugin install plugin_name
// tanzu plugin install --group group_name
// tanzu plugin group search
// initialize plugin source
func testWithoutPluginDiscoverySources(tf *framework.Framework) {
	_, err := tf.PluginCmd.DeletePluginDiscoverySource("default")
	Expect(err).To(BeNil(), "there should not be any error to delete default discovery source")

	plugins, _, _, err := tf.PluginCmd.SearchPlugins("")
	Expect(err.Error()).To(ContainSubstring(errorNoDiscoverySourcesFound))
	Expect(len(plugins)).Should(BeNumerically("==", 0))

	_, _, err = tf.PluginCmd.InstallPlugin("unknowPlugin", "", "")
	Expect(err.Error()).To(ContainSubstring(errorNoDiscoverySourcesFound))

	_, _, err = tf.PluginCmd.InstallPluginsFromGroup("", "unknowGroup")
	Expect(err.Error()).To(ContainSubstring(errorNoDiscoverySourcesFound))

	_, err = tf.PluginCmd.SearchPluginGroups("")
	Expect(err.Error()).To(ContainSubstring(errorNoDiscoverySourcesFound))

	_, err = tf.PluginCmd.InitPluginDiscoverySource()
	Expect(err).To(BeNil(), "there should not be any error for plugin source init")
}
