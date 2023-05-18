// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// tmc provides context command e2e test cases for tmc target
package tmc

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	types "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// Test suite tests the context life cycle use cases for the TMC target
// Here are sequence of test cases in this suite:
// Use case 1: delete all contexts if any available
// Use case 2: create both tmc and k8s contexts, make sure both are active
// Use case 3: create multiple tmc and k8s contexts, make sure most recently created contexts for both tmc and k8s are active

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-tmc-k8s]", func() {

	// Use case 1: delete config files and initialize config
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
	// use case 2: create both tmc and k8s contexts, make sure both are active
	// Test case: a. create tmc context
	// Test case: b. create k8s context, make sure its active
	// Test case: c. list all active contexts, make both tmc and k8s contexts are active
	// Test case: d. delte both k8s and tmc contexts
	Context("Context lifecycle tests for TMC target", func() {
		var k8sCtx, tmcCtx string
		// Test case: a. create tmc context
		It("create tmc context with endpoint and check active context", func() {
			tmcCtx = prefix + framework.RandomString(4)
			_, _, err := tf.ContextCmd.CreateContextWithEndPointStaging(tmcCtx, tmcClusterInfo.EndPoint)
			Expect(err).To(BeNil(), "context should create without any error")
			Expect(framework.IsContextExists(tf, tmcCtx)).To(BeTrue(), fmt.Sprintf(ContextShouldExistsAsCreated, tmcCtx))
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(tmcCtx), "the active context should be recently added context")
		})

		// Test case: b. create k8s context, make sure its active
		It("create k8s context", func() {
			k8sCtx = prefixK8s + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithKubeconfig(k8sCtx, k8sClusterInfo.KubeConfigPath, k8sClusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			Expect(framework.IsContextExists(tf, k8sCtx)).To(BeTrue(), fmt.Sprintf(framework.ContextShouldExistsAsCreated, k8sCtx))

			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(k8sCtx), "the active context should be recently added context")

		})

		// Test case: c. list all active contexts, make both tmc and k8s contexts are active
		It("list all active contexts", func() {
			cts, err := tf.ContextCmd.GetActiveContexts()
			Expect(err).To(BeNil(), "there should no error for list contexts")
			m := framework.ContextInfoToMap(cts)
			_, ok := m[k8sCtx]
			Expect(ok).To(BeTrue(), "k8s context should exists and active")
			_, ok = m[tmcCtx]
			Expect(ok).To(BeTrue(), "tmc context should exists and active")
		})

		// Test case: d. delete both k8s and tmc contexts
		It("delete context command", func() {
			err := tf.ContextCmd.DeleteContext(k8sCtx)
			Expect(err).To(BeNil(), "delete context should delete context without any error")
			Expect(framework.IsContextExists(tf, k8sCtx)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists+" as been deleted", k8sCtx))
			err = tf.ContextCmd.DeleteContext(tmcCtx)
			Expect(err).To(BeNil(), "delete context should delete context without any error")
			Expect(framework.IsContextExists(tf, tmcCtx)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists+" as been deleted", tmcCtx))
		})
	})
	// Use case 3: create multiple tmc and k8s contexts, make sure most recently created contexts for both tmc and k8s are active
	// Test case: a. create tmc contexts
	// Test case: b. create k8s contexts
	// Test case: c. list all active contexts, make both tmc and k8s contexts are active
	// Test case: d. list all contexts
	// Test case: e. delete all contexts
	Context("Context lifecycle tests for TMC target", func() {
		var k8sCtx, tmcCtx string
		k8sCtxs := make([]string, 0)
		tmcCtxs := make([]string, 0)
		// Test case: a. create tmc context
		It("create tmc context with endpoint and check active context", func() {
			for i := 0; i < 5; i++ {
				tmcCtx = prefix + framework.RandomString(4)
				_, _, err := tf.ContextCmd.CreateContextWithEndPointStaging(tmcCtx, tmcClusterInfo.EndPoint)
				Expect(err).To(BeNil(), "context should create without any error")
				Expect(framework.IsContextExists(tf, tmcCtx)).To(BeTrue(), fmt.Sprintf(ContextShouldExistsAsCreated, tmcCtx))
				active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
				Expect(err).To(BeNil(), "there should be a active context")
				Expect(active).To(Equal(tmcCtx), "the active context should be recently added context")
				tmcCtxs = append(tmcCtxs, tmcCtx)
			}

		})

		// Test case: b. create k8s contexts
		It("create k8s context", func() {
			for i := 0; i < 5; i++ {
				k8sCtx = prefixK8s + framework.RandomString(4)
				err := tf.ContextCmd.CreateContextWithKubeconfig(k8sCtx, k8sClusterInfo.KubeConfigPath, k8sClusterInfo.ClusterKubeContext)
				Expect(err).To(BeNil(), "context should create without any error")
				Expect(framework.IsContextExists(tf, k8sCtx)).To(BeTrue(), fmt.Sprintf(framework.ContextShouldExistsAsCreated, k8sCtx))

				active, err := tf.ContextCmd.GetActiveContext(string(types.TargetK8s))
				Expect(err).To(BeNil(), "there should be a active context")
				Expect(active).To(Equal(k8sCtx), "the active context should be recently added context")
				k8sCtxs = append(k8sCtxs, k8sCtx)
			}
		})

		// Test case: c. list all active contexts, make both tmc and k8s contexts are active
		It("list all active contexts", func() {
			cts, err := tf.ContextCmd.GetActiveContexts()
			Expect(err).To(BeNil(), "there should no error for list contexts")
			Expect(len(cts)).To(Equal(2), "there should be only 2 contexts active")
			m := framework.ContextInfoToMap(cts)
			_, ok := m[k8sCtx]
			Expect(ok).To(BeTrue(), "latest k8s context should exists and active")
			_, ok = m[tmcCtx]
			Expect(ok).To(BeTrue(), "latest tmc context should exists and active")
		})

		// Test case: d. list all contexts
		It("list all contexts", func() {
			cts, err := tf.ContextCmd.ListContext()
			Expect(err).To(BeNil(), "there should no error for list contexts")
			Expect(len(cts)).To(Equal(len(k8sCtxs)+len(tmcCtxs)), "there should be all k8s+tmc contexts")
			m := framework.ContextInfoToMap(cts)
			for _, ctx := range k8sCtxs {
				_, ok := m[ctx]
				Expect(ok).To(BeTrue(), "all k8s contexts should exists")
				delete(m, ctx)
			}
			for _, ctx := range tmcCtxs {
				_, ok := m[ctx]
				Expect(ok).To(BeTrue(), "all tmc contexts should exists")
				delete(m, ctx)
			}
			Expect(len(m)).To(Equal(0), "after tmc and k8s context deleted, there should not be any contexts exists")
		})

		// Test case: e. delete all contexts
		It("delete all contexts", func() {

			for _, ctx := range k8sCtxs {
				err := tf.ContextCmd.DeleteContext(ctx)
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			for _, ctx := range tmcCtxs {

				err := tf.ContextCmd.DeleteContext(ctx)
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			cts, err := tf.ContextCmd.ListContext()
			Expect(err).To(BeNil(), "there should no error for list contexts")
			Expect(len(cts)).To(Equal(0), "there should be no contexts available after deleting all contexts")
		})
	})
})
