// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// tmc provides context command e2e test cases for tmc target
package tmc

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestTmc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Context TMC E2E Test Suite")
}

var (
	tf           *framework.Framework
	clusterInfo  *framework.ClusterInfo
	contextNames []string
	ctxsStress   []string
)

const prefix = "ctx-tmc-"
const maxCtx = 25
const ContextShouldNotExists = "the context %s should not exists"
const ContextShouldExistsAsCreated = "the context %s should exists as its been created"

var _ = BeforeSuite(func() {
	tf = framework.NewFramework()

	// Check whether the TMC token TANZU_API_TOKEN and tmc url TANZU_CLI_TMC_UNSTABLE_URL are set or not
	Expect(os.Getenv(framework.TanzuAPIToken)).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with TMC API Token", framework.TanzuAPIToken))
	Expect(os.Getenv(framework.TanzuCliTmcUnstableURL)).NotTo(BeEmpty(), fmt.Sprintf("environment variable %s should set with TMC endpoint URL", framework.TanzuCliTmcUnstableURL))

	// get TMC TANZU_CLI_TMC_UNSTABLE_URL and TANZU_API_TOKEN from environment variables
	clusterInfo = framework.GetTMCClusterInfo()
	Expect(clusterInfo.EndPoint).NotTo(Equal(""), "TMC cluster URL is must needed to create TMC context")
	Expect(clusterInfo.APIKey).NotTo(Equal(""), "TMC API Key is must needed to create TMC context")
	contextNames = make([]string, 0)
	ctxsStress = make([]string, 0)
})
