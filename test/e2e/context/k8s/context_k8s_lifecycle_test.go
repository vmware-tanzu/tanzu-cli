// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package contextk8s provides context command specific E2E test cases for k8s target
package contextk8s

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const ContextNameConfigPrefix = "context-config-k8s-"

// This test suite focuses on testing context use cases for the Kubernetes (K8s) target,
// as well as stress test cases specifically for the K8s target.
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-k8s-tests]", func() {

	// This test suite is dedicated to testing the basic context life cycle use cases for the Kubernetes (K8s) target.
	framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-k8s]", func() {
		k8sContextLifeCycleTests()
	})

	// This test suite includes stress test cases to evaluate the performance and
	// robustness of the context operations.
	framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-stress-tests-k8s]", func() {
		k8sContextStressTests()
	})
})

// k8sContextLifeCycleTests has test cases for context life cycle use cases for the k8s target
func k8sContextLifeCycleTests() bool {
	var active string
	contextNames := make([]string, 0)
	return Context("Context lifecycle tests for k8s target", func() {
		// Test case: Create context for k8s target with kubeconfig and its context as input
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			log.Infof("context: %s added", ctxName)
			Expect(framework.IsContextExists(tf, ctxName)).To(BeTrue(), fmt.Sprintf(framework.ContextShouldExistsAsCreated, ctxName))
			contextNames = append(contextNames, ctxName)
		})
		// Test case: (negative test) Create context for k8s target with incorrect kubeconfig file path and its context as input
		It("create context with incorrect kubeconfig and context", func() {
			By("create context with incorrect kubeconfig and context")
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithKubeconfig(ctxName, framework.RandomString(4), clusterInfo.ClusterKubeContext)
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
			Expect(framework.IsContextExists(tf, ctxName)).To(BeFalse(), fmt.Sprintf(framework.ContextShouldNotExists, ctxName))
		})
		// Test case: (negative test) Create context for k8s target with kubeconfig file path and incorrect context as input
		It("create context with kubeconfig and incorrect context", func() {
			By("create context with kubeconfig and incorrect context")
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, framework.RandomString(4))
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
			Expect(framework.IsContextExists(tf, ctxName)).To(BeFalse(), fmt.Sprintf(framework.ContextShouldNotExists, ctxName))
		})
		// Test case: Create context for k8s target with "default" kubeconfig and its context only as input value
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			ctxName := "context-defaultConfig-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithDefaultKubeconfig(ctxName, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			log.Infof("context: %s added", ctxName)
			Expect(framework.IsContextExists(tf, ctxName)).To(BeTrue(), fmt.Sprintf(framework.ContextShouldExistsAsCreated, ctxName))
			contextNames = append(contextNames, ctxName)
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(ctxName), "the active context should be recently added context")
		})
		// Test case: test 'tanzu context use' command with the specific context name (not the recently created one)
		It("use context command", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")
		})
		// Test case: context unset command: test 'tanzu context unset' command with active context name
		It("unset context command: by context name", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			stdOut, _, err := tf.ContextCmd.UnsetContext(contextNames[0])
			Expect(err).To(BeNil(), "unset context should unset context without any error")
			Expect(stdOut).To(ContainSubstring(fmt.Sprintf(framework.ContextForTargetSetInactive, contextNames[0], framework.KubernetesTarget)))
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should not be any error for the get context")
			Expect(active).To(Equal(""), "there should not be any active context as unset performed")
		})
		// Test case: context unset command: test 'tanzu context unset' command with random context name
		It("unset context command: negative use case: by random context name", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			name := framework.RandomString(5)
			_, _, err := tf.ContextCmd.UnsetContext(name)
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.ContextNotActiveOrNotExists, name)))
		})
		// Test case: context unset command: test 'tanzu context unset' command by providing target
		It("unset context command: by target", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			stdOut, _, err := tf.ContextCmd.UnsetContext("", framework.AddAdditionalFlagAndValue("--target k8s"))
			Expect(err).To(BeNil(), "unset context should unset context without any error")
			Expect(stdOut).To(ContainSubstring(fmt.Sprintf(framework.ContextForTargetSetInactive, contextNames[0], framework.KubernetesTarget)))
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should not be any error for the get context")
			Expect(active).To(Equal(""), "there should not be any active context as unset performed")
		})
		// Test case: context unset command: test 'tanzu context unset' by providing target and context name
		It("unset context command: by target and context name", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			stdOut, _, err := tf.ContextCmd.UnsetContext(contextNames[0], framework.AddAdditionalFlagAndValue("--target k8s"))
			Expect(err).To(BeNil(), "unset context should unset context without any error")
			Expect(stdOut).To(ContainSubstring(fmt.Sprintf(framework.ContextForTargetSetInactive, contextNames[0], framework.KubernetesTarget)))
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should not be any error for the get context")
			Expect(active).To(Equal(""), "there should not be any active context as unset performed")
		})
		// Test case: context unset command: test 'tanzu context unset' command with incorrect target
		It("unset context command: negative use case: by target and context name: incorrect target", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			_, _, err := tf.ContextCmd.UnsetContext(contextNames[0], framework.AddAdditionalFlagAndValue("--target tmc"))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.ContextNotExistsForTarget, contextNames[0], framework.MissionControlTarget)))
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should not be any error for the get context")
			Expect(active).To(Equal(contextNames[0]), "there should be an active context as unset failed")
		})
		// Test case: (negative test) test 'tanzu context use' command with the specific context name (incorrect, which is not exists)
		It("use context command with incorrect context as input", func() {
			err := tf.ContextCmd.UseContext(framework.RandomString(4))
			Expect(err).ToNot(BeNil())
		})
		// Test case: test 'tanzu context list' command, should list all contexts created
		It("list context should have all added contexts", func() {
			By("list context should have all added contexts")
			list := framework.GetAvailableContexts(tf, contextNames)
			Expect(len(list) >= len(contextNames)).Should(BeTrue(), "list context should have all contexts added in previous tests")
		})
		// Test case: test 'tanzu context delete' command, make sure to delete all context's created in previous test cases
		It("delete context command", func() {
			By("delete all contexts created in previous tests")
			for _, ctx := range contextNames {
				_, _, err := tf.ContextCmd.DeleteContext(ctx)
				Expect(framework.IsContextExists(tf, ctx)).To(BeFalse(), fmt.Sprintf(framework.ContextShouldNotExists, ctx))
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
		})
		// Test case: (negative test) test 'tanzu context delete' command for context name which is not exists
		It("delete context command", func() {
			By("delete context command with random string")
			_, _, err := tf.ContextCmd.DeleteContext(framework.RandomString(4))
			Expect(err).ToNot(BeNil())
		})
	})
}

// k8sContextStressTests has test suite focuses on context life cycle tests for the Kubernetes (K8s) target and includes stress test cases to evaluate the performance and robustness of the context operations.
func k8sContextStressTests() bool {
	// This suite adds stress test cases for context life cycle tests (for the k8s target)
	// Here are sequence of tests:
	// a. delete config files and initialize config
	// b. list and delete contexts if any exists already, before running test cases
	// c. create multiple contexts with kubeconfig and context
	// d. test 'tanzu context use' command with the specific context name (not the recently created one),test for multiple contexts continuously
	// e. test 'tanzu context list' command, should list all contexts created
	// f. test 'tanzu context delete' command, make sure to delete all contexts created in previous test cases
	contextNamesStress := make([]string, 0)
	return Context("Context lifecycle stress tests for k8s target", func() {
		// Test case: b. list and delete contexts if any exists already, before running test cases
		It("list and delete contexts if any exists already", func() {
			By("list and delete contexts if any exists already before running test cases")
			list, err := tf.ContextCmd.ListContext()
			Expect(err).To(BeNil(), "list context should not return any error")
			for _, ctx := range list {
				_, _, err := tf.ContextCmd.DeleteContext(ctx.Name)
				Expect(err).To(BeNil(), "delete context should delete context without any error")
				Expect(framework.IsContextExists(tf, ctx.Name)).To(BeFalse(), fmt.Sprintf(framework.ContextShouldNotExists, ctx.Name))
			}
			list, err = tf.ContextCmd.ListContext()
			Expect(err).To(BeNil(), "list context should not return any error")
			Expect(len(list)).To(Equal(0), "all contexts should be deleted")
		})
		// Test case: c. create multiple contexts with kubeconfig and context
		It("create multiple contexts with kubeconfig and context", func() {
			By("create multiple contexts with kubeconfig and context")
			for i := 0; i < ContextCreateLimit; i++ {
				ctxName := ContextNameConfigPrefix + framework.RandomString(4)
				err := tf.ContextCmd.CreateContextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
				Expect(err).To(BeNil(), "context should create without any error")
				log.Info("context: " + ctxName + " added")
				Expect(framework.IsContextExists(tf, ctxName)).To(BeTrue(), fmt.Sprintf(framework.ContextShouldExistsAsCreated, ctxName))
				contextNamesStress = append(contextNamesStress, ctxName)
			}
		})
		// Test case: d. test 'tanzu context use' command with the specific context name
		// 				(not the recently created one), test for multiple contexts continuously
		It("use context command", func() {
			By("use context command")
			for i := 0; i < len(contextNamesStress); i++ {
				err := tf.ContextCmd.UseContext(contextNamesStress[i])
				log.Info("set the corrent context as:" + contextNamesStress[i])
				Expect(err).To(BeNil(), "use context should set context without any error")
				active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
				Expect(err).To(BeNil(), "there should be a active context")
				Expect(active).To(Equal(contextNamesStress[i]), "the active context should be recently set context")
			}
		})
		// Test case: e. test 'tanzu context list' command, should list all contexts created
		It("list context should have all added contexts", func() {
			By("list context should have all added contexts")
			list := framework.GetAvailableContexts(tf, contextNamesStress)
			Expect(len(list) >= len(contextNamesStress)).Should(BeTrue(), "list context should have all contexts added in previous tests")
		})
		// Test case: f. test 'tanzu context delete' command, make sure to delete all contexts created in previous test cases
		It("delete context command", func() {
			By("delete all contexts created in previous tests")
			for _, ctx := range contextNamesStress {
				_, _, err := tf.ContextCmd.DeleteContext(ctx)
				log.Infof("context: %s deleted", ctx)
				Expect(framework.IsContextExists(tf, ctx)).To(BeFalse(), fmt.Sprintf(framework.ContextShouldNotExists, ctx))
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
		})
	})
}
