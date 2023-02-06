// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// tmc provides context command e2e test cases for tmc target
package tmc

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	context "github.com/vmware-tanzu/tanzu-cli/test/e2e/context"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// Test suite tests the context life cycle use cases for the TMC target
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-tmc]", func() {
	var (
		tf           *framework.Framework
		clusterInfo  *framework.ClusterInfo
		contextNames []string
	)
	BeforeSuite(func() {
		tf = framework.NewFramework()
		// get TMC TANZU_CLI_TMC_UNSTABLE_URL and TANZU_API_TOKEN from environment variables
		clusterInfo = context.GetTMCClusterInfo()
		Expect(clusterInfo.EndPoint).NotTo(Equal(""), "TMC cluster URL is must needed to create TMC context")
		Expect(clusterInfo.APIKey).NotTo(Equal(""), "TMC API Key is must needed to create TMC context")
		contextNames = make([]string, 0)
	})
	Context("Context lifecycle tests for TMC target", func() {
		// Test case: Create context for TMC target with TMC cluster URL as endpoint
		It("create tmc context with endpoint", func() {
			ctxName := "context-endpoint" + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithEndPointStaging(ctxName, clusterInfo.EndPoint)
			Expect(err).To(BeNil(), "context should create without any error")
			contextNames = append(contextNames, ctxName)
		})
		// Test case: Create context for TMC target with TMC cluster URL as endpoint, and validate the active context, should be recently create context
		It("create tmc context with endpoint and check active context", func() {
			ctxName := "context-endpoint" + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithEndPointStaging(ctxName, clusterInfo.EndPoint)
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
			Expect(active).To(Equal(contextNames[0]), "the active context should be same as the context set by use context command")
		})
		// Test case: test 'tanzu context list' command, should list all contexts created
		It("list context should have all added contexts", func() {
			list := context.GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(len(contextNames)), "list context should exists all contexts added in previous tests")
		})
		// Test case: test 'tanzu context delete' command, make sure to delete all context's created in previous test cases
		It("delete context command", func() {
			for _, ctx := range contextNames {
				err := tf.ContextCmd.DeleteContext(ctx)
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			list := context.GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(0), "deleted contexts should not be in list context")
		})
	})
})
