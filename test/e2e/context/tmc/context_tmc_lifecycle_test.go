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

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	types "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// This suite has context (for tmc target) life cycle tests

// Test suite tests the context life cycle use cases for the TMC target
// Here are sequence of test cases in this suite:
// Test case: a. delete config files and initialize config
// Test case: b. Create context for TMC target with TMC cluster URL as endpoint
// Test case: c. (negative test) Create context for TMC target with TMC cluster "incorrect" URL as endpoint
// Test case: d. (negative test) Create context for TMC target with TMC cluster URL as endpoint when api token set as incorrect
// Test case: e. Create context for TMC target with TMC cluster URL as endpoint, and validate the active context, should be recently create context
// Test case: f. test 'tanzu context use' command with the specific context name (not the recently created one)
// Test case: g. context unset command: test 'tanzu context unset' command with active context name
// Test case: h. context unset command: test 'tanzu context unset' command with random context name
// Test case: i.context unset command: test 'tanzu context unset' command by providing valid target
// Test case: j. context unset command: test 'tanzu context unset' by providing target and context name
// Test case: k. context unset command: test 'tanzu context unset' command with incorrect target
// Test case: l. (negative test) test 'tanzu context use' command with the specific context name (incorrect, which is not exists)
// Test case: m. test 'tanzu context list' command, should list all contexts created
// Test case: n. test 'tanzu context delete' command, make sure to delete all context's created in previous test cases

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-tmc]", func() {
	var active string

	Context("Context lifecycle tests for TMC target", func() {
		// Test case: b. Create context for TMC target with TMC cluster URL as endpoint
		It("create tmc context with endpoint", func() {
			ctxName := prefix + framework.RandomString(4)
			_, _, err := tf.ContextCmd.CreateContextWithEndPointStaging(ctxName, tmcClusterInfo.EndPoint)
			Expect(err).To(BeNil(), "context should create without any error")
			Expect(framework.IsContextExists(tf, ctxName)).To(BeTrue(), fmt.Sprintf(ContextShouldExistsAsCreated, ctxName))
			contextNames = append(contextNames, ctxName)
		})
		// Test case: c. (negative test) Create context for TMC target with TMC cluster "incorrect" URL as endpoint
		It("create tmc context with incorrect endpoint", func() {
			ctxName := prefix + framework.RandomString(4)
			_, _, err := tf.ContextCmd.CreateContextWithEndPointStaging(ctxName, framework.RandomString(4))
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
			Expect(framework.IsContextExists(tf, ctxName)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists, ctxName))
		})
		// Test case: d. (negative test) Create context for TMC target with TMC cluster URL as endpoint when api token set as incorrect
		It("create tmc context with endpoint and with incorrect api token", func() {
			err := os.Setenv(framework.TanzuAPIToken, framework.RandomString(4))
			Expect(err).To(BeNil(), "There should not be any error in setting environment variables.")
			ctxName := framework.RandomString(4)
			_, _, err = tf.ContextCmd.CreateContextWithEndPointStaging(ctxName, tmcClusterInfo.EndPoint)
			os.Setenv(framework.TanzuAPIToken, tmcClusterInfo.APIKey)
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), framework.FailedToCreateContext)).To(BeTrue())
			Expect(framework.IsContextExists(tf, ctxName)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists, ctxName))
			err = os.Setenv(framework.TanzuAPIToken, tmcClusterInfo.APIKey)
			Expect(err).To(BeNil(), "There should not be any error in setting environment variables.")
		})
		// Test case: e. Create context for TMC target with TMC cluster URL as endpoint, and validate the active context, should be recently create context
		It("create tmc context with endpoint and check active context", func() {
			ctxName := prefix + framework.RandomString(4)
			_, _, err := tf.ContextCmd.CreateContextWithEndPointStaging(ctxName, tmcClusterInfo.EndPoint)
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
		// context unset command related use cases:::
		// Test case: g. context unset command: test 'tanzu context unset' command with active context name
		It("unset context command: by context name", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			stdOut, _, err := tf.ContextCmd.UnsetContext(contextNames[0])
			Expect(err).To(BeNil(), "unset context should unset context without any error")
			Expect(stdOut).To(ContainSubstring(fmt.Sprintf(framework.ContextForTargetSetInactive, contextNames[0], framework.MissionControlTarget)))
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should not be any error for the get context")
			Expect(active).To(Equal(""), "there should not be any active context as unset performed")
		})
		// Test case: h. context unset command: test 'tanzu context unset' command with random context name
		It("unset context command: negative use case: by random context name", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			name := framework.RandomString(5)
			_, _, err := tf.ContextCmd.UnsetContext(name)
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.ContextNotActiveOrNotExists, name)))
		})
		// Test case: i.context unset command: test 'tanzu context unset' command by providing valid target
		It("unset context command: by target", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			stdOut, _, err := tf.ContextCmd.UnsetContext("", framework.AddAdditionalFlagAndValue("--target tmc"))
			Expect(err).To(BeNil(), "unset context should unset context without any error")
			Expect(stdOut).To(ContainSubstring(fmt.Sprintf(framework.ContextForTargetSetInactive, contextNames[0], framework.MissionControlTarget)))
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should not be any error for the get context")
			Expect(active).To(Equal(""), "there should not be any active context as unset performed")
		})
		// Test case: j. context unset command: test 'tanzu context unset' by providing target and context name
		It("unset context command: by target and context name", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			stdOut, _, err := tf.ContextCmd.UnsetContext(contextNames[0], framework.AddAdditionalFlagAndValue("--target tmc"))
			Expect(err).To(BeNil(), "unset context should unset context without any error")
			Expect(stdOut).To(ContainSubstring(fmt.Sprintf(framework.ContextForTargetSetInactive, contextNames[0], framework.MissionControlTarget)))
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should not be any error for the get context")
			Expect(active).To(Equal(""), "there should not be any active context as unset performed")
		})
		// Test case: k. context unset command: test 'tanzu context unset' command with incorrect target
		It("unset context command: negative use case: by target and context name: incorrect target", func() {
			err = tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(active).To(Equal(contextNames[0]), "the active context should be recently set context")

			_, _, err := tf.ContextCmd.UnsetContext(contextNames[0], framework.AddAdditionalFlagAndValue("--target k8s"))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(framework.ContextNotExistsForTarget, contextNames[0], framework.KubernetesTarget)))
			active, err = tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should not be any error for the get context")
			Expect(active).To(Equal(contextNames[0]), "there should be an active context as unset failed")
		})
		// Test case: l. (negative test) test 'tanzu context use' command with the specific context name (incorrect, which is not exists)
		It("use context command with incorrect context as input", func() {
			err := tf.ContextCmd.UseContext(framework.RandomString(4))
			Expect(err).ToNot(BeNil())
		})
		// Test case: m. test 'tanzu context list' command, should list all contexts created
		It("list context should have all added contexts", func() {
			list := framework.GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(len(contextNames)), "list context should exists all contexts added in previous tests")
		})
		// Test case: n. test 'tanzu context delete' command, make sure to delete all context's created in previous test cases
		It("delete context command", func() {
			for _, ctx := range contextNames {
				err := tf.ContextCmd.DeleteContext(ctx)
				Expect(framework.IsContextExists(tf, ctx)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists+" as been deleted", ctx))
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
			list := framework.GetAvailableContexts(tf, contextNames)
			Expect(len(list)).To(Equal(0), "deleted contexts should not be in list context")
		})
	})
})
