// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package context provides context command specific E2E test cases
package context

import (
	"os"

	gomega "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// CreateKindCluster create the k8s KIND cluster in the local Docker environment
func CreateKindCluster(tf *framework.Framework, name string) *framework.ClusterInfo {
	ci := &framework.ClusterInfo{Name: name}
	_, err := tf.KindCluster.CreateCluster(name)
	gomega.Expect(err).To(gomega.BeNil(), "the kind cluster creation should be successful without any error")
	endpoint, err := tf.KindCluster.GetClusterEndpoint(name)
	gomega.Expect(err).To(gomega.BeNil(), "we need cluster endpoint")
	ci.EndPoint = endpoint
	ci.ClusterContext = tf.KindCluster.GetClusterContext(name)
	ci.KubeConfigPath = tf.KindCluster.GetKubeconfigPath()
	return ci
}

// GetTMCClusterInfo returns the TMC cluster info by reading environment variables TANZU_CLI_TMC_UNSTABLE_URL and TANZU_API_TOKEN
func GetTMCClusterInfo() *framework.ClusterInfo {
	return &framework.ClusterInfo{EndPoint: os.Getenv(framework.TanzuCliTmcUnstableURL), APIKey: os.Getenv(framework.TanzuAPIToken)}
}

// GetAvailableContexts takes list of contexts and returns which are available in the 'tanzu context list' command
func GetAvailableContexts(tf *framework.Framework, contextNames []string) []string {
	var available []string
	list, err := tf.ContextCmd.ListContext()
	gomega.Expect(err).To(gomega.BeNil(), "list context should not return any error")
	set := framework.SliceToSet(contextNames)
	for _, context := range list {
		if _, ok := set[context.Name]; ok {
			available = append(available, context.Name)
		}
	}
	return available
}
