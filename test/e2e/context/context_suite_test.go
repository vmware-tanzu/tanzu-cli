// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package context

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestContext(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Context-K8S Suite")
}

var (
	tf           *framework.Framework
	clusterInfo  *framework.ClusterInfo
	contextNames []string
	err          error
)

const ContextCreateLimit = 100

// BeforeSuite created KIND cluster
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()
	// Create KIND cluster, which is used in test cases to create context's
	clusterInfo, err = framework.CreateKindCluster(tf, "context-e2e-"+framework.RandomNumber(4))
	Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
	contextNames = make([]string, 0)
})

// AfterSuite deletes the KIND cluster created in BeforeSuite
var _ = AfterSuite(func() {
	// delete the KIND cluster which was created in the suite setup
	_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
	Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
})
