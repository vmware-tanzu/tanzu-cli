// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package coexistence_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestCliCoexistence(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "CLI Coexistence E2E Test Suite")
}

var (
	tf *framework.Framework

	newTanzuCLIVersion    string
	legacyTanzuCLIVersion string

	e2eTestLocalCentralRepoURL string

	e2eTestLocalCentralRepoCACertPath                                 string
	e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath string

	// PluginsForLegacyTanzuCLICoexistenceTests is list of plugins (which are published in local central repo) used in plugin life cycle test cases
	PluginsForLegacyTanzuCLICoexistenceTests []*framework.PluginInfo

	// PluginsForNewTanzuCLICoexistenceTests is list of plugins (which are published in local central repo) used in plugin life cycle test cases
	PluginsForNewTanzuCLICoexistenceTests []*framework.PluginInfo

	// PluginGroupsForNewTanzuCLICoexistenceTests is list of plugin groups (which are published in local central repo) used in plugin group life cycle test cases
	PluginGroupsForNewTanzuCLICoexistenceTests []*framework.PluginGroup
)

// BeforeSuite initializes and set up the environment to execute the plugin life cycle and plugin group life cycle end-to-end test cases
var _ = ginkgo.BeforeSuite(func() {
	tf = framework.NewFramework()

	ginkgo.By("Setting up the Environment variables required to run CLI Coexistence Tests")
	// check E2E test central repo URL (TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL)
	e2eTestLocalCentralRepoURL = os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryURL)
	gomega.Expect(e2eTestLocalCentralRepoURL).NotTo(gomega.BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository URL", framework.TanzuCliE2ETestLocalCentralRepositoryURL))
	// set E2E test central repo URL to TANZU_CLI_PRE_RELEASE_REPO_IMAGE
	err := os.Setenv(framework.TanzuCliE2ETestCentralRepositoryURL, e2eTestLocalCentralRepoURL)
	gomega.Expect(err).To(gomega.BeNil())

	// verify the legacy Tanzu CLI version is set
	legacyTanzuCLIVersion = os.Getenv(framework.CLICoexistenceLegacyTanzuCLIVersion)
	gomega.Expect(legacyTanzuCLIVersion).NotTo(gomega.BeEmpty(), fmt.Sprintf("legacy tanzu CLI %s should be set", framework.CLICoexistenceLegacyTanzuCLIVersion))

	// verify the new Tanzu CLI version is set
	newTanzuCLIVersion = os.Getenv(framework.CLICoexistenceNewTanzuCLIVersion)
	gomega.Expect(newTanzuCLIVersion).NotTo(gomega.BeEmpty(), fmt.Sprintf("new tanzu CLI %s should be set", framework.CLICoexistenceNewTanzuCLIVersion))

	// test local central repository CA certs are mounted at path /cosign-key-pair in docker
	e2eTestLocalCentralRepoCACertPath = "/localhost_certs/localhost.crt"
	e2eTestLocalCentralRepoPluginDiscoveryImageSignaturePublicKeyPath = "/cosign-key-pair/cosign.pub"

	// verify the legacy Tanzu CLI installation path is set
	legacyTanzuCLIInstallationPath := os.Getenv(framework.CLICoexistenceLegacyTanzuCLIInstallationPath)
	gomega.Expect(legacyTanzuCLIInstallationPath).NotTo(gomega.BeEmpty(), fmt.Sprintf("legacy tanzu CLI installation path %s should be set", framework.CLICoexistenceLegacyTanzuCLIInstallationPath))

	// verify the legacy Tanzu CLI installation path is set
	newTanzuCLIInstallationPath := os.Getenv(framework.CLICoexistenceNewTanzuCLIInstallationPath)
	gomega.Expect(newTanzuCLIInstallationPath).NotTo(gomega.BeEmpty(), fmt.Sprintf("new tanzu CLI installation path %s should be set", framework.CLICoexistenceNewTanzuCLIInstallationPath))

	ceipParticipation := os.Getenv(framework.CLICoexistenceTanzuCEIPParticipation)
	gomega.Expect(ceipParticipation).NotTo(gomega.BeEmpty(), fmt.Sprintf("ceip %s should be set", framework.CLICoexistenceTanzuCEIPParticipation))

	PluginsForLegacyTanzuCLICoexistenceTests = []*framework.PluginInfo{
		{
			Name:    "telemetry",
			Target:  framework.KubernetesTarget,
			Version: legacyTanzuCLIVersion,
		},
	}

	PluginsForNewTanzuCLICoexistenceTests = []*framework.PluginInfo{
		{
			Name:        "isolated-cluster",
			Description: "Desc for isolated-cluster",
			Target:      framework.GlobalTarget,
			Version:     "v9.9.9",
		},
	}

	PluginGroupsForNewTanzuCLICoexistenceTests = []*framework.PluginGroup{{Group: "vmware-tmc/tmc-user", Latest: "v2.1.0", Description: "Desc for vmware-tmc/tmc-user:v2.1.0"}, {Group: "vmware-tkg/default", Latest: "v2.1.0", Description: "Desc for vmware-tkg/default:v2.1.0"}}
})
