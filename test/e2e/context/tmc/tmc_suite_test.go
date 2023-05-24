// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// tmc provides context command e2e test cases for tmc target
package tmc

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestTmc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Context TMC E2E Test Suite")
}

var (
	tf                 *framework.Framework
	tmcClusterInfo     *framework.ClusterInfo
	k8sClusterInfo     *framework.ClusterInfo
	contextNames       []string
	contextNamesStress []string
	ctxsStress         []string
	err                error
)

const prefix = "ctx-tmc-"
const prefixK8s = "ctx-k8s-"
const maxCtx = 25
const ContextShouldNotExists = "the context %s should not exists"
const ContextShouldExistsAsCreated = "the context %s should exists as its been created"

// This suite has below e2e tests:
// 1. context (for tmc target) life cycle tests
// 2. context life cycle tests with k8s and tmc contexts co-existing
// 3. context stress tests for tmc target
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()

	// Check whether the TMC token TANZU_API_TOKEN and tmc url TANZU_CLI_TMC_UNSTABLE_URL are set or not
	Expect(os.Getenv(framework.TanzuAPIToken)).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with TMC API Token", framework.TanzuAPIToken))
	Expect(os.Getenv(framework.TanzuCliTmcUnstableURL)).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with TMC endpoint URL", framework.TanzuCliTmcUnstableURL))

	// get TMC TANZU_CLI_TMC_UNSTABLE_URL and TANZU_API_TOKEN from environment variables
	tmcClusterInfo = framework.GetTMCClusterInfo()
	Expect(tmcClusterInfo.EndPoint).NotTo(Equal(""), "TMC cluster URL is must needed to create TMC context")
	Expect(tmcClusterInfo.APIKey).NotTo(Equal(""), "TMC API Key is must needed to create TMC context")
	contextNames = make([]string, 0)
	ctxsStress = make([]string, 0)

	// Create KIND cluster, which is used in test cases to create context's
	k8sClusterInfo, err = framework.CreateKindCluster(tf, "context-e2e-"+framework.RandomNumber(4))
	Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")

	// delete config files
	err = tf.Config.DeleteCLIConfigurationFiles()
	Expect(err).To(BeNil())
	// call init
	err = tf.Config.ConfigInit()
	Expect(err).To(BeNil())
	// should create config files
	Expect(tf.Config.IsCLIConfigurationFilesExists()).To(BeTrue())
})

// AfterSuite deletes the KIND cluster created in BeforeSuite
var _ = AfterSuite(func() {
	// delete the KIND cluster which was created in the suite setup
	_, err := tf.KindCluster.DeleteCluster(k8sClusterInfo.Name)
	Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
})
