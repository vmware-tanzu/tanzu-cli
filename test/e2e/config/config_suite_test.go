// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config_e2e_test provides config command specific E2E test cases
package config_e2e_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var (
	tf           *framework.Framework
	clusterInfo  *framework.ClusterInfo
	contextNames []string
	err          error
	flags        []string
)

const (
	featureFlagPrefix          = "features.global.e2e-test-"
	numberOfFlagsForStressTest = 100
	noErrorForFeatureFlagSet   = "there should not be any error for global feature flag set operation"
	noErrorForFeatureFlagGet   = "there should not be any error for global feature flag set operation"
	noErrorForConfigInit       = "there should not be any error for config init operation"
)

// BeforeSuite creates KIND cluster needed to test 'tanzu config server' use cases
// initializes the tf
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()
	// Create KIND cluster, which is used in test cases to create server's/context's
	clusterInfo, err = framework.CreateKindCluster(tf, "config-e2e-"+framework.RandomNumber(4))
	Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")

	contextNames = make([]string, 0)
	flags = make([]string, 0, 0)
})

// AfterSuite deletes the KIND which is created in BeforeSuite
var _ = AfterSuite(func() {
	// delete the KIND cluster which was created in the suite setup
	_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
	Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
})
