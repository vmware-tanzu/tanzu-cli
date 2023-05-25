// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// pluginsynce2etmc provides plugin sync command specific E2E test cases
package pluginsynce2etmc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	helper "github.com/vmware-tanzu/tanzu-cli/test/e2e/plugin_lifecycle"
)

func TestPluginSyncLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin-Sync TMC Lifecycle E2E Test Suite")
}

var (
	tf                                *framework.Framework
	e2eTestLocalCentralRepoURL        string
	pluginsSearchList                 []*framework.PluginInfo
	pluginGroups                      []*framework.PluginGroup
	pluginGroupToPluginListMap        map[string][]*framework.PluginInfo
	usePluginsFromTmcPluginGroup      string
	usePluginsFromK8sPluginGroup      string
	K8SCRDFilePath                    string
	tmcMappingsDir                    string
	tmcPluginsMockFilePath            string
	tmcConfigFolderPath               string
	e2eTestLocalCentralRepoPluginHost string
	e2eTestLocalCentralRepoCACertPath string
)

const mockAPIEndpointForPlugins = "/pluginsInfo"
const routesFileName = "routes.json"
const tmcPluginsMockFile = "tmcPluginsMock.json"
const numberOfPluginsToInstall = 3

const forceCSPFlag = " --force-csp true"
const tmcConfigFolderName = "tmc"
const numberOfPluginsSameAsNoOfPluginsInfoMocked = "number of plugins should be same as number of plugins mocked in tmc endpoint response"
const pluginsInstalledAndMockedShouldBeSame = "plugins being installed and plugins being mocked in tmc endpoint response should be same"
const noErrorForMockResponseFileUpdate = "there should not be any error while updating the tmc endpoint mock response"
const noErrorForMockResponsePreparation = "there should not be any error while preparing the tmc endpoint mock response"
const deleteContextWithoutError = "context should be deleted without error"
const mockServerShouldStartWithoutError = "mock server should start without error"
const mockServerShouldStopWithoutError = "mock server should stop without error"
const noErrorWhileCreatingContext = "context should create without any error"
const activeContextShouldExists = "there should be a active context"
const pluginGroupShouldExists = "plugin group should exist in the map"
const noErrorForPluginList = "should not get any error for plugin list"
const activeContextShouldBeRecentlyAddedOne = "the active context should be recently added context"
const testRepoDoesNotHaveEnoughPlugins = "test central repo does not have enough plugins to continue e2e tests"

// BeforeSuite initializes and set up the environment to execute the plugin sync test cases for tmc target
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()

	err := framework.CleanConfigFiles(tf)
	Expect(err).To(BeNil())

	// check E2E test central repo URL (TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) and update the same for plugin discovery
	e2eTestLocalCentralRepoURL = os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryURL)
	Expect(e2eTestLocalCentralRepoURL).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository URL", framework.TanzuCliE2ETestLocalCentralRepositoryURL))

	// Check whether the TMC token is set and whether TANZU_CLI_E2E_TEST_ENVIRONMENT is set to skip HTTPS hardcoding when mocking TMC response.
	Expect(os.Getenv(framework.TanzuAPIToken)).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with TMC API Token", framework.TanzuAPIToken))
	Expect(os.Getenv(framework.CLIE2ETestEnvironment)).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set as true otherwise e2e tests will fails", framework.CLIE2ETestEnvironment))

	// create tmc/mappings config folder, in $HOME/.tanzu-cli-e2e/temp directory
	tmcConfigFolderPath = filepath.Join(framework.FullPathForTempDir, tmcConfigFolderName)
	tmcMappingsDir = filepath.Join(tmcConfigFolderPath, "mappings")
	_ = framework.CreateDir(tmcMappingsDir)

	// create a file to update http request/response mocking for every test case
	tmcPluginsMockFilePath = filepath.Join(tmcMappingsDir, tmcPluginsMockFile)

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

	// set up the test central repo
	err = framework.UpdatePluginDiscoverySource(tf, e2eTestLocalCentralRepoURL)
	Expect(err).To(BeNil(), "should not get any error for plugin source update")

	// Search for plugin groups and ensure that there are available plugin groups.
	pluginGroups, err = helper.SearchAllPluginGroups(tf)
	Expect(err).To(BeNil(), framework.NoErrorForPluginGroupSearch)

	// Check that all required plugin groups for life cycle tests (listed in framework.PluginGroupsForLifeCycleTests) are available in the plugin group search output.
	Expect(framework.IsAllPluginGroupsExists(pluginGroups, framework.PluginGroupsForLifeCycleTests)).Should(BeTrue(), "all required plugin groups for life cycle tests should exists in plugin group search output")

	// Retrieve the TMC specific plugin group from which plugins will be used to perform E2E tests.
	usePluginsFromTmcPluginGroup = framework.GetPluginGroupWhichStartsWithGivenPrefix(framework.PluginGroupsForLifeCycleTests, framework.TMCPluginGroupPrefix)
	Expect(usePluginsFromTmcPluginGroup).NotTo(BeEmpty(), "there should be a tmc specific plugin group")
	// Retrieve the k8s specific plugin group from which plugins will be used to perform E2E tests.
	usePluginsFromK8sPluginGroup = framework.GetPluginGroupWhichStartsWithGivenPrefix(framework.PluginGroupsForLifeCycleTests, framework.K8SPluginGroupPrefix)
	Expect(usePluginsFromTmcPluginGroup).NotTo(BeEmpty(), "there should be a k8s specific plugin group")

	// search plugins and make sure there are plugins available
	pluginsSearchList, err = helper.SearchAllPlugins(tf)
	Expect(err).To(BeNil(), framework.NoErrorForPluginSearch)
	Expect(len(pluginsSearchList)).Should(BeNumerically(">", 0))

	pluginGroupToPluginListMap = make(map[string][]*framework.PluginInfo)
	for _, pg := range framework.PluginGroupsForLifeCycleTests {
		// Setup plugin list for both versions of the group
		for _, v := range []string{"v9.9.9", "v0.0.1"} {
			pg.Latest = v
			plugins, err := helper.GetAllPluginsFromGroup(tf, pg)
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
	}
})

// After the Suite, delete the temporary directory (including the TMC config directory within the temporary directory) that was created during test case execution
var _ = AfterSuite(func() {
	err := os.RemoveAll(framework.FullPathForTempDir) // delete an entire directory
	Expect(err).To(BeNil(), "should not get any error while deleting temp directory")
})
