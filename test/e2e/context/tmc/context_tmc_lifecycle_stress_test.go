// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package tmc

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// This suite adds stress test cases for context life cycle tests (for the TMC target)
// Here are sequence of tests:
// a. create multiple contexts with tmc endpoint
// b. test 'tanzu context use' command with the specific context name (not the recently created one),test for multiple contexts continuously
// c. test 'tanzu context list' command, should list all contexts created
// d. test 'tanzu context delete' command, make sure to delete all contexts created in previous test cases
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-stress-tests-k8s]", func() {
	Context("Context lifecycle stress tests for k8s target", func() {
		// Test case: a. create multiple contexts with tmc endpoint
		It("create multiple contexts", func() {
			By("create tmc context with tmc endpoint")
			for i := 0; i < maxCtx; i++ {
				ctxName := prefix + framework.RandomString(5)
				_, _, err := tf.ContextCmd.CreateContextWithEndPointStaging(ctxName, tmcClusterInfo.EndPoint)
				Expect(err).To(BeNil(), "context should create without any error")
				Expect(framework.IsContextExists(tf, ctxName)).To(BeTrue(), fmt.Sprintf(ContextShouldExistsAsCreated, ctxName))
				contextNamesStress = append(contextNamesStress, ctxName)
			}
		})
		// Test case: b. test 'tanzu context use' command with the specific context name
		// 				(not the recently created one), test for multiple contexts continuously
		It("use context command", func() {
			By("use context command")
			for i := 0; i < len(ctxsStress); i++ {
				err := tf.ContextCmd.UseContext(ctxsStress[i])
				Expect(err).To(BeNil(), "use context should set context without any error")
				active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
				Expect(err).To(BeNil(), "there should be a active context")
				Expect(active).To(Equal(ctxsStress[i]), "the active context should be recently set context")
			}
		})
		// Test case: c. test 'tanzu context list' command, should list all contexts created
		It("list context should have all added contexts", func() {
			By("list context should have all added contexts")
			list := framework.GetAvailableContexts(tf, ctxsStress)
			Expect(len(list)).Should(BeNumerically(">=", len(ctxsStress)))
		})
		// Test case: d. test 'tanzu context delete' command, make sure to delete all contexts created in previous test cases
		It("delete context command", func() {
			By("delete all contexts created in previous tests")
			for _, ctx := range ctxsStress {
				err := tf.ContextCmd.DeleteContext(ctx)
				log.Infof("context: %s deleted", ctx)
				Expect(framework.IsContextExists(tf, ctx)).To(BeFalse(), fmt.Sprintf(framework.ContextShouldNotExists, ctx))
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			list := framework.GetAvailableContexts(tf, ctxsStress)
			Expect(len(list)).To(Equal(0), "delete context should have deleted all given contexts")
		})
	})
})
