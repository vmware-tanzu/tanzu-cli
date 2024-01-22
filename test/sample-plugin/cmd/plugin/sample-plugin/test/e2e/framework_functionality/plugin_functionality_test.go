// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package frameworkfunctionality implements CLI E2E framework API usage test cases
package frameworkfunctionality

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

var clusterInfo *framework.ClusterInfo
var _ = framework.CLICoreDescribe("[Tests:Sample-Plugin-E2E][Feature:plugin-specific]", func() {
	Context("tanzu plugin - related API", func() {
		// Test case: 'tanzu plugin source list' API should work
		It("tanzu plugin source list api", func() {
			sourceList, err := tf.PluginCmd.ListPluginSources()
			Expect(err).To(BeNil(), "should not occur any error for the plugin source list command")
			Expect(len(sourceList)).To(Equal(1), "should have only one plugin source")
		})
	})
	Context("tanzu version api", func() {
		// Test case: 'tanzu version' API should work
		It("tanzu version api", func() {
			version, err := tf.CLIVersion()
			Expect(err).To(BeNil(), "should not occur any error for the plugin source list command")
			Expect(version).NotTo(BeNil(), "tanzu version should not be nil")
		})
	})
	Context("tanzu config api", func() {
		It("should set flag and unset flag successfully", func() {
			flagName := "e2e-test-" + framework.RandomString(4)
			randomFeatureFlagPath := "features.global." + flagName
			flagVal := "true"
			// Set random feature flag
			err := tf.Config.ConfigSetFeatureFlag(randomFeatureFlagPath, flagVal)
			Expect(err).To(BeNil())
			// Validate the value of random feature flag set in previous step
			val, err := tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath)
			Expect(err).To(BeNil())
			Expect(val).Should(Equal(flagVal))
		})
	})
	Context("tanzu context api and kind cluster api", func() {
		var contextName string
		// Test case: create KIND cluster
		It("create kind cluster", func() {
			var err error
			clusterInfo, err = framework.CreateKindCluster(tf, "context-e2e-"+framework.RandomNumber(4))
			Expect(err).To(BeNil(), "should not get any error for KIND cluster creation")
		})
		// Test case: Create context for k8s target with kubeconfig and its context as input
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			contextName = "k8s-context-" + framework.RandomString(4)
			_, _, err := tf.ContextCmd.CreateContextWithKubeconfig(contextName, clusterInfo.KubeConfigPath, clusterInfo.ClusterKubeContext)
			Expect(err).To(BeNil(), "context should create without any error")
			log.Infof("context: %s added", contextName)
			Expect(framework.IsContextExists(tf, contextName)).To(BeTrue(), fmt.Sprintf(framework.ContextShouldExistsAsCreated, contextName))
		})
		It("delete context and KIND cluster", func() {
			_, _, err := tf.ContextCmd.DeleteContext(contextName)
			Expect(err).To(BeNil(), "context should be deleted without any error")

			_, _, err = tf.KindCluster.DeleteCluster(clusterInfo.Name)
			Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
		})
	})
})
