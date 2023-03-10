// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// context provides context command specific E2E test cases
package context

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// This suite adds stress test cases for context life cycle tests (for the k8s target)
// Here are sequence of tests:
// a. delete config files and initialize config
// b. list and delete contexts if any exists already, before running test cases
// c. create multiple contexts with kubeconfig and context
// d. test 'tanzu context use' command with the specific context name (not the recently created one),test for multiple contexts continuously
// e. test 'tanzu context list' command, should list all contexts created
// f. test 'tanzu context delete' command, make sure to delete all contexts created in previous test cases
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-stress-tests-k8s]", func() {
	Context("Context lifecycle stress tests for k8s target", func() {
		// Test case: a. delete config files and initialize config
		It("should initialize configuration successfully", func() {
			// delete config files
			err := tf.Config.DeleteCLIConfigurationFiles()
			Expect(err).To(BeNil())
			// call init
			err = tf.Config.ConfigInit()
			Expect(err).To(BeNil())
			// should create config files
			Expect(tf.Config.IsCLIConfigurationFilesExists()).To(BeTrue())
		})
		// Test case: b. list and delete contexts if any exists already, before running test cases
		It("list and delete contexts if any exists already", func() {
			By("list and delete contexts if any exists already before running test cases")
			list, err := tf.ContextCmd.ListContext()
			Expect(err).To(BeNil(), "list context should not return any error")
			for _, ctx := range list {
				err := tf.ContextCmd.DeleteContext(ctx.Name)
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
			list := GetAvailableContexts(tf, contextNamesStress)
			Expect(len(list)).To(Equal(len(contextNamesStress)), "list context should have all contexts added in previous tests")
		})
		// Test case: f. test 'tanzu context delete' command, make sure to delete all contexts created in previous test cases
		It("delete context command", func() {
			By("delete all contexts created in previous tests")
			for _, ctx := range contextNamesStress {
				err := tf.ContextCmd.DeleteContext(ctx)
				log.Infof("context: %s deleted", ctx)
				Expect(framework.IsContextExists(tf, ctx)).To(BeFalse(), fmt.Sprintf(framework.ContextShouldNotExists, ctx))
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			list := GetAvailableContexts(tf, contextNamesStress)
			Expect(len(list)).To(Equal(0), "delete context should have deleted all given contexts")
		})
	})
})
