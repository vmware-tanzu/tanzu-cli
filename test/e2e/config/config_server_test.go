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
	Context("tanzu config server command test cases ", func() {
		// Test case: delete servers if any exists, with command 'tanzu config server delete'
		It("list and delete servers if any exists before running tests", func() {
			By("delete servers if any exists before running tests")
			list, err := tf.Config.ConfigServerList()
			Expect(err).To(BeNil(), "server list should not return any error")
			for _, ctx := range list {
				err := tf.Config.ConfigServerDelete(ctx.Name)
				Expect(err).To(BeNil(), "delete server should delete server without any error")
			}
			list, err = tf.Config.ConfigServerList()
			Expect(err).To(BeNil(), "server list should not return any error")
			Expect(len(list)).To(Equal(0), "all servers should be deleted")
		})
		// Test case: Create context for k8s target with kubeconfig and its context as input
		It("create context with kubeconfig and context", func() {
			By("create context with kubeconfig and context")
			ctxName := ContextNameConfigPrefix + framework.RandomString(4)
			err := tf.ContextCmd.CreateConextWithKubeconfig(ctxName, clusterInfo.KubeConfigPath, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			Expect(context.IsContextExists(tf, ctxName)).To(BeTrue(), "context should be available")
			contextNames = append(contextNames, ctxName)
		})
		// Test case: Create context for k8s target with "default" kubeconfig and its context only as input value
		It("create context with default kubeconfig and context", func() {
			By("create context with default kubeconfig and context")
			ctxName := "context-defaultConfig-" + framework.RandomString(4)
			err := tf.ContextCmd.CreateContextWithDefaultKubeconfig(ctxName, clusterInfo.ClusterContext)
			Expect(err).To(BeNil(), "context should create without any error")
			Expect(context.IsContextExists(tf, ctxName)).To(BeTrue(), "context should be available")
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
})
