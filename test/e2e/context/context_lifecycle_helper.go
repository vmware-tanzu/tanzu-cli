// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package context provides context command specific E2E test cases
package context

import (
	"os"

	gomega "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// GetTMCClusterInfo returns the TMC cluster info by reading environment variables TANZU_CLI_TMC_UNSTABLE_URL and TANZU_API_TOKEN
// Currently we are setting these env variables in GitHub action for local testing these variables need to be set by the developer on their respective machine
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

// GetAvailableServers takes list of servers and returns which are available in the 'tanzu config server list' command
func GetAvailableServers(tf *framework.Framework, serverNames []string) []string {
	var available []string
	list, err := tf.Config.ConfigServerList()
	gomega.Expect(err).To(gomega.BeNil(), "server list should not return any error")
	set := framework.SliceToSet(serverNames)
	for _, server := range list {
		if _, ok := set[server.Name]; ok {
			available = append(available, server.Name)
		}
	}
	return available
}
