// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config_e2e_test provides config command specific E2E test cases
package config_e2e_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/context"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

const ContextNameConfigPrefix = "config-k8s-"

// This test suite tests the 'tanzu config server' use cases
// As part of this suite, create a KIND cluster, and creates context's
// tests the 'tanzu config server list' and 'tanzu config server delete' commands
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Config-Server]", func() {
	var (
		tf           *framework.Framework
		clusterInfo  *framework.ClusterInfo
		contextNames []string
	)

	BeforeSuite(func() {
		tf = framework.NewFramework()
		// Create KIND cluster, which is used in test cases to create server's/context's
		clusterInfo = context.CreateKindCluster(tf, "config-e2e-"+framework.RandomNumber(4))
		contextNames = make([]string, 0)
	})
	Context("tanzu config server command test cases ", func() {
		// Test case: Create context for k8s target with kubeconfig and its context as input
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			contextNames = append(contextNames, ctxName)
		})
		// Test case: Create context for k8s target with "default" kubeconfig and its context only as input value
		It("create context with default kubeconfig and context", func() {
			By("create context with default kubeconfig and context")
			ctxName := "context-defaultConfig-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithDefaultKubeconfig(ctxName, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			contextNames = append(contextNames, ctxName)
		})
		// Test case: test 'tanzu config server list' command, should list all contexts created as servers
		It("list servers should have all added contexts", func() {
			By("list servers should have all added contexts")
			list, err := tf.Config.ConfigServerList()
			Expect(err).To(BeNil(), "config server list command should list available servers")
			Expect(len(list)).To(Equal(len(contextNames)), "list context should have all contexts (as servers) added in previous tests")
		})
		// Test case: test 'tanzu config server delete' command, make sure this command deletes server entry by deleting all contexts created in previous test cases
		It("delete server command", func() {
			By("delete servers which are created in previous tests")
			for _, ctx := range contextNames {
				err := tf.Config.ConfigServerDelete(ctx)
				Expect(err).To(BeNil(), "delete server should delete server without any error")
			}
			list := context.GetAvailableServers(tf, contextNames)
			Expect(len(list)).To(Equal(0), "delete server should have deleted all given server names")
		})
		// Test case: (negative test) test 'tanzu context delete' command for context name which is not exists
		It("delete server which is not exists", func() {
			By("delete server which is not exists")
			err := tf.Config.ConfigServerDelete(framework.RandomString(4))
			Expect(err).ToNot(BeNil())
		})
	})
	AfterSuite(func() {
		// delete the KIND cluster which was created in the suite setup
		_, err := tf.KindCluster.DeleteCluster(clusterInfo.Name)
		Expect(err).To(BeNil(), "kind cluster should be deleted without any error")
	})
})
