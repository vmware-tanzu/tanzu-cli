// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// tmc provides context command e2e test cases for tmc target
package tmc

import (
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	context "github.com/vmware-tanzu/tanzu-cli/test/e2e/context"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	types "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

const ContextNamePrefix = "context-endpoint-tmc-"
const ContextShouldNotExists = "the context %s should not exists"
const ContextShouldExistsAsCreated = "the context %s should exists as its been created"

// Test suite tests the context life cycle use cases for the TMC target
// Here are sequence of test cases in this suite:
// a. list and delete contexts if any exists
// b. Create context for TMC target with TMC cluster URL as endpoint
// c. (negative test) Create context for TMC target with TMC cluster "incorrect" URL as endpoint
// d. (negative test) Create context for TMC target with TMC cluster URL as endpoint when api token set as incorrect
// e. Create context for TMC target with TMC cluster URL as endpoint, and validate the active context, should be recently create context
// f. test 'tanzu context use' command with the specific context name (not the recently created one)
// g. (negative test) test 'tanzu context use' command with the specific context name (incorrect, which is not exists)
// h. test 'tanzu context list' command, should list all contexts created
// i. test 'tanzu context delete' command, make sure to delete all context's created in previous test cases
var (
	tf           *framework.Framework
	clusterInfo  *framework.ClusterInfo
	contextNames []string
)

var _ = BeforeSuite(func() {
	tf = framework.NewFramework()
	// get TMC TANZU_CLI_TMC_UNSTABLE_URL and TANZU_API_TOKEN from environment variables
	clusterInfo = context.GetTMCClusterInfo()
	Expect(clusterInfo.EndPoint).NotTo(Equal(""), "TMC cluster URL is must needed to create TMC context")
	Expect(clusterInfo.APIKey).NotTo(Equal(""), "TMC API Key is must needed to create TMC context")
	contextNames = make([]string, 0)
})

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-tmc]", func() {

	Context("Context lifecycle tests for TMC target", func() {
		// Test case: a. list and delete contexts if any exists
		It("delete context command", func() {
			list, err := tf.ContextCmd.ListContext()
			Expect(err).To(BeNil(), "list context should not return any error")
			for _, ctx := range list {
				err := tf.ContextCmd.DeleteContext(ctx.Name)
				Expect(framework.IsContextExists(tf, ctx.Name)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists, ctx.Name))
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			list, err = tf.ContextCmd.ListContext()
			Expect(err).To(BeNil(), "list context should not return any error")
			Expect(len(list)).To(Equal(0), "all contexts should be deleted")
		})
		// Test case: b. Create context for TMC target with TMC cluster URL as endpoint
		It("create tmc context with endpoint", func() {
			ctxName := ContextNamePrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithEndPointStaging(ctxName, clusterInfo.EndPoint)
			Expect(err).To(BeNil(), "context should create without any error")
			Expect(framework.IsContextExists(tf, ctxName)).To(BeTrue(), fmt.Sprintf(ContextShouldExistsAsCreated, ctxName))
			contextNames = append(contextNames, ctxName)
		})
		// Test case: c. (negative test) Create context for TMC target with TMC cluster "incorrect" URL as endpoint
		It("create tmc context with incorrect endpoint", func() {
			ctxName := ContextNamePrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithEndPointStaging(ctxName, framework.RandomString(4))
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
			Expect(framework.IsContextExists(tf, ctxName)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists, ctxName))
		})
		// Test case: d. (negative test) Create context for TMC target with TMC cluster URL as endpoint when api token set as incorrect
		It("create tmc context with endpoint and with incorrect api token", func() {
			err := os.Setenv(framework.TanzuAPIToken, framework.RandomString(4))
			Expect(err).ToNot(BeNil())
			ctxName := framework.RandomString(4)
			err = tf.ContextCmd.CreateContextWithEndPointStaging(ctxName, clusterInfo.EndPoint)
			Expect(err).ToNot(BeNil())
			err = os.Setenv(framework.TanzuAPIToken, clusterInfo.APIKey)
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
			Expect(framework.IsContextExists(tf, ctxName)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists, ctxName))
		})
		// Test case: e. Create context for TMC target with TMC cluster URL as endpoint, and validate the active context, should be recently create context
		It("create tmc context with endpoint and check active context", func() {
			ctxName := ContextNamePrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithEndPointStaging(ctxName, clusterInfo.EndPoint)
			Expect(err).To(BeNil(), "context should create without any error")
			Expect(framework.IsContextExists(tf, ctxName)).To(BeTrue(), fmt.Sprintf(ContextShouldExistsAsCreated, ctxName))
			contextNames = append(contextNames, ctxName)
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(ctxName), "the active context should be recently added context")
		})
		// Test case: f. test 'tanzu context use' command with the specific context name (not the recently created one)
		It("use context command", func() {
			err := tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be same as the context set by use context command")
		})
		// Test case: g. (negative test) test 'tanzu context use' command with the specific context name (incorrect, which is not exists)
		It("use context command with incorrect context as input", func() {
			err := tf.ContextCmd.UseContext(framework.RandomString(4))
			Expect(err).ToNot(BeNil())
		})
		// Test case: h. test 'tanzu context list' command, should list all contexts created
		It("list context should have all added contexts", func() {
			list := context.GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(len(contextNames)), "list context should exists all contexts added in previous tests")
		})
		// Test case: i. test 'tanzu context delete' command, make sure to delete all context's created in previous test cases
		It("delete context command", func() {
			for _, ctx := range contextNames {
				err := tf.ContextCmd.DeleteContext(ctx)
				Expect(framework.IsContextExists(tf, ctx)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists+" as been deleted", ctx))
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			list := context.GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(0), "deleted contexts should not be in list context")
		})
	})
})
