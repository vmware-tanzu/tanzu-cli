// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// context provides context command specific E2E test cases
package context

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// Test suite tests the context life cycle use cases for the k8s target
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-k8s]", func() {
	var (
		tf           *framework.Framework
		clusterInfo  *framework.ClusterInfo
		contextNames []string
	)

	BeforeSuite(func() {
		tf = framework.NewFramework()
		// Create KIND cluster, which is used in test cases to create context's
		clusterInfo = CreateKindCluster(tf, "context-e2e-"+framework.RandomNumber(4))
		contextNames = make([]string, 0)
	})
	Context("Context lifecycle tests for k8s target", func() {
		// Test case: Create context for k8s target with kubeconfig and its context as input
		It("create context with kubeconfig and context", func() {
			ctxName := "context-config-context-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			contextNames = append(contextNames, ctxName)
		})
		// Test case: Create context for k8s target with default kubeconfig and its context only as input value
		It("create context with kubeconfig and context", func() {
			ctxName := "context-defaultConfig-context-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithDefaultKubeconfig(ctxName, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			contextNames = append(contextNames, ctxName)
			active, err := tf.ContextCmd.GetActiveContext()
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(ctxName), "the active context should be recently added context")
		})
		// Test case: test 'tanzu context use' command with the specific context name (not the recently created one)
		It("use context command", func() {
			err := tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err := tf.ContextCmd.GetActiveContext()
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")
		})
		// Test case: test 'tanzu context list' command, should list all contexts created
		It("list context should have all added contexts", func() {
			list := GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(len(contextNames)), "list context should have all contexts added in previous tests")
		})
		// Test case: test 'tanzu context delete' command, make sure to delete all context's created in previous test cases
		It("delete context command", func() {
			for _, ctx := range contextNames {
				err := tf.ContextCmd.DeleteContext(ctx)
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			list := GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(0), "delete context should have deleted all given contexts")
		})
	})
	AfterSuite(func() {
		// delete the KIND cluster which was created in the suite setup
		_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
		Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
	})

})
