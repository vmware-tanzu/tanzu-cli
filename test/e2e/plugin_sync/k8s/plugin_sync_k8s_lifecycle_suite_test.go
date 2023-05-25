// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// pluginsynce2ek8s provides plugin sync command specific E2E test cases
package pluginsynce2ek8s

import (
	"fmt"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	helper "github.com/vmware-tanzu/tanzu-cli/test/e2e/plugin_lifecycle"
)

func TestPluginSyncLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin-Sync K8S Lifecycle E2E Test Suite")
}

var (
	tf                                *framework.Framework
	e2eTestLocalCentralRepoURL        string
	pluginsSearchList                 []*framework.PluginInfo
	pluginGroups                      []*framework.PluginGroup
	pluginGroupToPluginListMap        map[string][]*framework.PluginInfo
	usePluginsFromPluginGroup         string
	e2eTestLocalCentralRepoPluginHost string
	e2eTestLocalCentralRepoCACertPath string
)

const numberOfPluginsToInstall = 3

// BeforeSuite initializes and set up the environment to execute the plugin life cycle and plugin group life cycle end-to-end test cases
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()

	err := framework.CleanConfigFiles(tf)
	Expect(err).To(BeNil())

	// check E2E test central repo URL (TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL)
	e2eTestLocalCentralRepoURL = os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryURL)
	Expect(e2eTestLocalCentralRepoURL).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository URL", framework.TanzuCliE2ETestLocalCentralRepositoryURL))

	e2eTestLocalCentralRepoPluginHost = os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryHost)
	Expect(e2eTestLocalCentralRepoPluginHost).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository host", framework.TanzuCliE2ETestLocalCentralRepositoryHost))

	e2eTestLocalCentralRepoCACertPath = os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryCACertPath)
	Expect(e2eTestLocalCentralRepoCACertPath).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository CA cert path", framework.TanzuCliE2ETestLocalCentralRepositoryCACertPath))

	// set up the CA cert for local central repository
	_ = tf.Config.ConfigCertDelete(e2eTestLocalCentralRepoPluginHost)
	_, err = tf.Config.ConfigCertAdd(&framework.CertAddOptions{Host: e2eTestLocalCentralRepoPluginHost, CACertificatePath: e2eTestLocalCentralRepoCACertPath, SkipCertVerify: "false", Insecure: "false"})
	Expect(err).To(BeNil(), "should not be any error for cert add")
	// list and validate the cert added
	list, err := tf.Config.ConfigCertList()
	Expect(err).To(BeNil(), "should not be any error for cert list")
	Expect(len(list)).To(Equal(1), "should not be any error for cert list")

	// set up the local central repository discovery image public key path
	e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath := os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryPluginDiscoveryImageSignaturePublicKeyPath)
	Expect(e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository discovery image signature public key path", framework.TanzuCliE2ETestLocalCentralRepositoryPluginDiscoveryImageSignaturePublicKeyPath))
	os.Setenv(framework.TanzuCliPluginDiscoveryImageSignaturePublicKeyPath, e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath)

	// set up the test central repo
	err = framework.UpdatePluginDiscoverySource(tf, e2eTestLocalCentralRepoURL)
	Expect(err).To(BeNil(), "should not get any error for plugin source update")

	// search plugin groups and make sure there plugin groups available
	pluginGroups, err = helper.SearchAllPluginGroups(tf)
	Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)

	// check all required plugin groups (framework.PluginGroupsForLifeCycleTests) need for life cycle test are available in plugin group search output
	Expect(framework.IsAllPluginGroupsExists(pluginGroups, framework.PluginGroupsForLifeCycleTests)).Should(BeTrue(), "all required plugin groups for life cycle tests should exists in plugin group search output")

	// get k8s plugin group
	usePluginsFromPluginGroup = framework.GetPluginGroupWhichStartsWithGivenPrefix(framework.PluginGroupsForLifeCycleTests, framework.K8SPluginGroupPrefix)
	Expect(usePluginsFromPluginGroup).NotTo(BeEmpty(), "there should be a k8s specific plugin group")

	// search plugins and make sure there are plugins available
	pluginsSearchList, err = helper.SearchAllPlugins(tf)
	Expect(err).To(BeNil(), framework.NoErrorForPluginSearch)
	Expect(len(pluginsSearchList)).Should(BeNumerically(">", 0))

	// check all required plugins (framework.PluginsForLifeCycleTests) for plugin life cycle e2e are available in plugin search output
	Expect(framework.CheckAllPluginsExists(pluginsSearchList, framework.PluginsForLifeCycleTests)).To(BeTrue())

	pluginGroupToPluginListMap = framework.MapPluginsToPluginGroups(pluginsSearchList, framework.PluginGroupsForLifeCycleTests)
	for pluginGroupLatest := range framework.PluginGroupsLatestToOldVersions {
		framework.CopyPluginsBetweenPluginGroupsAndUpdatePluginsVersion(pluginGroupToPluginListMap, pluginGroupLatest, framework.PluginGroupsLatestToOldVersions[pluginGroupLatest], strings.Split(strings.Split(framework.PluginGroupsLatestToOldVersions[pluginGroupLatest], "/")[1], ":")[1])
	}

	// check for every plugin group (in framework.PluginGroupsForLifeCycleTests) there should be plugins available
	for pg := range pluginGroupToPluginListMap {
		Expect(len(pluginGroupToPluginListMap[pg])).Should(BeNumerically(">", 0), "there should be at least one plugin available for each plugin group in plugin group life cycle list")
	}
})

// AfterSuite deletes the temp directory created during test cases execution
var _ = AfterSuite(func() {
	err := os.RemoveAll(framework.FullPathForTempDir) // delete an entire directory
	Expect(err).To(BeNil(), "should not get any error while deleting temp directory")
})
