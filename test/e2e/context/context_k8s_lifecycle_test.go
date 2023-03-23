// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// context provides context command specific E2E test cases
package context

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

const ContextNameConfigPrefix = "context-config-k8s-"

// Test suite tests the context life cycle use cases for the k8s target
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-k8s]", func() {
	Context("Context lifecycle tests for k8s target", func() {
		// Test case: list and delete context's if any exists already, before running test cases.
		It("list and delete contexts if any exists already", func() {
			By("list and delete contexts if any exists already before running test cases")
			list, err := tf.ContextCmd.ListContext()
			Expect(err).To(BeNil(), "list context should not return any error")
			for _, ctx := range list {
				err := tf.ContextCmd.DeleteContext(ctx.Name)
				Expect(err).To(BeNil(), "delete context should delete context without any error")
				Expect(IsContextExists(tf, ctx.Name)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists, ctx.Name))
			}
			list, err = tf.ContextCmd.ListContext()
			Expect(err).To(BeNil(), "list context should not return any error")
			Expect(len(list)).To(Equal(0), "all contexts should be deleted")
		})
		// Test case: Create context for k8s target with kubeconfig and its context as input
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			Expect(IsContextExists(tf, ctxName)).To(BeTrue(), fmt.Sprintf(ContextShouldExistsAsCreated, ctxName))
			contextNames = append(contextNames, ctxName)
		})
		// Test case: (negative test) Create context for k8s target with incorrect kubeconfig file path and its context as input
		It("create context with incorrect kubeconfig and context", func() {
			By("create context with incorrect kubeconfig and context")
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithKubeconfig(ctxName, framework.RandomString(4), clusterInfo.ClusterContext)
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
			Expect(IsContextExists(tf, ctxName)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists, ctxName))
		})
		// Test case: (negative test) Create context for k8s target with kubeconfig file path and incorrect context as input
		It("create context with kubeconfig and incorrect context", func() {
			By("create context with kubeconfig and incorrect context")
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, framework.RandomString(4))
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
			Expect(IsContextExists(tf, ctxName)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists, ctxName))
		})
		// Test case: Create context for k8s target with "default" kubeconfig and its context only as input value
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			ctxName := "context-defaultConfig-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithDefaultKubeconfig(ctxName, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			Expect(IsContextExists(tf, ctxName)).To(BeTrue(), fmt.Sprintf(ContextShouldExistsAsCreated, ctxName))
			contextNames = append(contextNames, ctxName)
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(ctxName), "the active context should be recently added context")
		})
		// Test case: test 'tanzu context use' command with the specific context name (not the recently created one)
		It("use context command", func() {
			By("use context command")
			err := tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
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
			By("list context should have all added contexts")
			list := GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(len(contextNames)), "list context should have all contexts added in previous tests")
		})
		// Test case: test 'tanzu context delete' command, make sure to delete all context's created in previous test cases
		It("delete context command", func() {
			By("delete all contexts created in previous tests")
			for _, ctx := range contextNames {
				err := tf.ContextCmd.DeleteContext(ctx)
				Expect(IsContextExists(tf, ctx)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists, ctx))
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			list := GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(0), "delete context should have deleted all given contexts")
		})
		// Test case: (negative test) test 'tanzu context delete' command for context name which is not exists
		It("delete context command", func() {
			By("delete context command with random string")
			err := tf.ContextCmd.DeleteContext(framework.RandomString(4))
			Expect(err).ToNot(BeNil())
		})
	})
})
