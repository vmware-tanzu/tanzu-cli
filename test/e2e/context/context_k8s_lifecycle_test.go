// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// context provides context command specific E2E test cases
package context

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

const ContextNameConfigPrefix = "context-config-k8s-"

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
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			contextNames = append(contextNames, ctxName)
		})
		// Test case: (negative test) Create context for k8s target with incorrect kubeconfig file path and its context as input
		It("create context with incorrect kubeconfig and context", func() {
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithKubeconfig(ctxName, framework.RandomString(4), clusterInfo.ClusterContext)
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
		})
		// Test case: (negative test) Create context for k8s target with kubeconfig file path and incorrect context as input
		It("create context with kubeconfig and incorrect context", func() {
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, framework.RandomString(4))
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
		})
		// Test case: Create context for k8s target with "default" kubeconfig and its context only as input value
		It("create context with kubeconfig and context", func() {
			ctxName := "context-defaultConfig-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithDefaultKubeconfig(ctxName, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			contextNames = append(contextNames, ctxName)
			active, err := tf.ContextCmd.GetActiveContext(framework.TargetTypeK8s)
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(ctxName), "the active context should be recently added context")
		})
		// Test case: test 'tanzu context use' command with the specific context name (not the recently created one)
		It("use context command", func() {
			err := tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err := tf.ContextCmd.GetActiveContext(framework.TargetTypeK8s)
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")
		})
		// Test case: (negative test) test 'tanzu context use' command with the specific context name (incorrect, which is not exists)
		It("use context command with incorrect context as input", func() {
			err := tf.ContextCmd.UseContext(framework.RandomString(4))
			Expect(err).ToNot(BeNil())
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
		// Test case: (negative test) test 'tanzu context delete' command for context name which is not exists
		It("delete context command", func() {
			err := tf.ContextCmd.DeleteContext(framework.RandomString(4))
			Expect(err).ToNot(BeNil())
		})
	})
	AfterSuite(func() {
		// delete the KIND cluster which was created in the suite setup
		_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
		Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
	})

})
