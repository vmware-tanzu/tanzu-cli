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
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// This test suite includes tests for the TMC context,
// as well as tests for both the TMC and Kubernetes (K8s) contexts simultaneously.
// Additionally, it includes stress test cases.
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Context-tmc-tests]", func() {

	// The test suite includes context life cycle tests specifically for the TMC target.
	framework.CLICoreDescribe("[Tests:E2E][Feature:Context-basic-lifecycle-tmc]", func() {
		tmcLifeCycleTests()
	})

	// This test suite includes context life cycle tests where both Kubernetes (K8s) and TMC contexts coexist.
	framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-tmc-k8s]", func() {
		tmcAndK8sContextTests()
	})

	// The test suite is dedicated to stress testing the context life cycle for the TMC target.
	framework.CLICoreDescribe("[Tests:E2E][Feature:Context-lifecycle-stress-tests-tmc]", func() {
		tmcStressTestCases()
	})
})

// tmcStressTestCases has test stress test cases for tmc target
func tmcStressTestCases() bool {
	// Here are sequence of tests:
	// a. create multiple contexts with tmc endpoint
	// b. test 'tanzu context use' command with the specific context name (not the recently created one),test for multiple contexts continuously
	// c. test 'tanzu context list' command, should list all contexts created
	// d. test 'tanzu context delete' command, make sure to delete all contexts created in previous test cases
	contextNamesStress := make([]string, 0)
	return Context("Context lifecycle stress tests for tmc target", func() {
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
			for i := 0; i < len(contextNamesStress); i++ {
				err := tf.ContextCmd.UseContext(contextNamesStress[i])
				Expect(err).To(BeNil(), "use context should set context without any error")
				active, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
				Expect(err).To(BeNil(), "there should be a active context")
				Expect(active).To(Equal(contextNamesStress[i]), "the active context should be recently set context")
			}
		})
		// Test case: c. test 'tanzu context list' command, should list all contexts created
		It("list context should have all added contexts", func() {
			By("list context should have all added contexts")
			list := framework.GetAvailableContexts(tf, contextNamesStress)
			Expect(len(list)).Should(BeNumerically(">=", len(contextNamesStress)))
		})
		// Test case: d. test 'tanzu context delete' command, make sure to delete all contexts created in previous test cases
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

// tmcAndK8sContextTests has test cases for k8s and tmc context co-existing use cases
func tmcAndK8sContextTests() bool {
	// This Test suite has the context life cycle tests with k8s and tmc contexts co-existing
	// Here are sequence of test cases in this suite:
	// Use case 1: create both tmc and k8s contexts, make sure both are active
	// Use case 2: create multiple tmc and k8s contexts, make sure most recently created contexts for both tmc and k8s are active
	return Describe("[Tests:E2E][Feature:Context-lifecycle-tmc-k8s]", func() {
		// use case 1: create both tmc and k8s contexts, make sure both are active
		// Test case: a. create tmc context
		// Test case: b. create k8s context, make sure its active
		// Test case: c. list all active contexts, make both tmc and k8s contexts are active
		// Test case: d. delte both k8s and tmc contexts
		Context("Context lifecycle tests for TMC and k8s targets", func() {
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
				_, _, err := tf.ContextCmd.DeleteContext(k8sCtx)
				Expect(err).To(BeNil(), "delete context should delete context without any error")
				Expect(framework.IsContextExists(tf, k8sCtx)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists+" as been deleted", k8sCtx))
				_, _, err = tf.ContextCmd.DeleteContext(tmcCtx)
				Expect(err).To(BeNil(), "delete context should delete context without any error")
				Expect(framework.IsContextExists(tf, tmcCtx)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists+" as been deleted", tmcCtx))
			})
		})

		// Use case 2: create multiple tmc and k8s contexts, make sure most recently created contexts for both tmc and k8s are active
		// Test case: a. create tmc contexts
		// Test case: b. create k8s contexts
		// Test case: c. list all active contexts, make both tmc and k8s contexts are active
		// Test case: d. list all contexts
		// Test case: e. delete all contexts
		Context("Context lifecycle tests for TMC and k8s targets, with multiple contexts", func() {
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
			})

			// Test case: e. delete all contexts
			It("delete all contexts", func() {
				for _, ctx := range k8sCtxs {
					_, _, err := tf.ContextCmd.DeleteContext(ctx)
					Expect(err).To(BeNil(), "delete context should delete context without any error")
				}
				for _, ctx := range tmcCtxs {
					_, _, err := tf.ContextCmd.DeleteContext(ctx)
					Expect(err).To(BeNil(), "delete context should delete context without any error")
				}
			})
		})
	})
}

// tmcLifeCycleTests has test cases for context life cycle tests for tmc target
func tmcLifeCycleTests() bool {
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
	var active string
	contextNames := make([]string, 0)
	return Context("Context lifecycle tests for TMC target", func() {
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
			activeCtx, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(activeCtx).To(Equal(ctxName), "the active context should be recently added context")
		})
		// Test case: f. test 'tanzu context use' command with the specific context name (not the recently created one)
		It("use context command", func() {
			err := tf.ContextCmd.UseContext(contextNames[0])
			Expect(err).To(BeNil(), "use context should set context without any error")
			activeCtx, err := tf.ContextCmd.GetActiveContext(string(types.TargetTMC))
			Expect(err).To(BeNil(), "there should be a active context")
			Expect(activeCtx).To(Equal(contextNames[0]), "the active context should be same as the context set by use context command")
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
			Expect(len(list) >= len(contextNames)).Should(BeTrue(), "list context should exists all contexts added in previous tests")
		})
		// Test case: n. test 'tanzu context delete' command, make sure to delete all context's created in previous test cases
		It("delete context command", func() {
			for _, ctx := range contextNames {
				_, _, err := tf.ContextCmd.DeleteContext(ctx)
				Expect(framework.IsContextExists(tf, ctx)).To(BeFalse(), fmt.Sprintf(ContextShouldNotExists+" as been deleted", ctx))
				Expect(err).To(BeNil(), "delete context should delete context without any error")
			}
		})
	})
}
