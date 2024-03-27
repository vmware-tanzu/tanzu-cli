// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package central_config_e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestCentralConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Central config E2E Test Suite")
}

var (
	tf *framework.Framework
)

var _ = BeforeSuite(func() {
	tf = framework.NewFramework()

	err := framework.CleanConfigFiles(tf)
	Expect(err).To(BeNil())

	// check E2E test central repo URL (TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL)
	e2eTestLocalCentralRepoURL := os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryURL)
	Expect(e2eTestLocalCentralRepoURL).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository URL", framework.TanzuCliE2ETestLocalCentralRepositoryURL))

	e2eTestLocalCentralRepoPluginHost := os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryHost)
	Expect(e2eTestLocalCentralRepoPluginHost).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository host", framework.TanzuCliE2ETestLocalCentralRepositoryHost))

	e2eTestLocalCentralRepoCACertPath := os.Getenv(framework.TanzuCliE2ETestLocalCentralRepositoryCACertPath)
	Expect(e2eTestLocalCentralRepoCACertPath).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with local central repository CA cert path", framework.TanzuCliE2ETestLocalCentralRepositoryCACertPath))

	// set up the CA cert for local central repository
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

	// check that the central config file is present
	centralCfg := filepath.Join(framework.TestHomeDir, ".cache", "tanzu", common.PluginInventoryDirName, config.DefaultStandaloneDiscoveryName, "central_config.yaml")
	_, err = os.Stat(centralCfg)
	Expect(err).To(BeNil(), "central config file should exist")
})
