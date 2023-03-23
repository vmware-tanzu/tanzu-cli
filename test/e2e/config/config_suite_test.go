// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config_e2e_test provides config command specific E2E test cases
package config_e2e_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/context"
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
)

// BeforeSuite creates KIND cluster needed to test 'tanzu config server' use cases
// initializes the tf
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()
	// Create KIND cluster, which is used in test cases to create server's/context's
	clusterInfo = context.CreateKindCluster(tf, "config-e2e-"+framework.RandomNumber(4))
	contextNames = make([]string, 0)
})

// AfterSuite deletes the KIND which is created in BeforeSuite
var _ = AfterSuite(func() {
	// delete the KIND cluster which was created in the suite setup
	_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
	Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
})
