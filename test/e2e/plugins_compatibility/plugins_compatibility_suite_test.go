// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugincompatibility provides plugins compatibility E2E test cases
package plugincompatibility_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"

	plugincompatibility "github.com/vmware-tanzu/tanzu-cli/test/e2e/plugins_compatibility"
)

func TestPluginsCompatibility(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PluginsCompatibility Suite")
}

var (
	tf      *framework.Framework
	plugins []*framework.PluginInfo
)

// In the BeforeSuite search for the test-plugin-'s from the TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL test central repository
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()
	// get all plugins with name prefix "test-plugin-"
	plugins = plugincompatibility.PluginsForCompatibilityTesting(tf)
	Expect(len(plugins)).NotTo(BeZero(), fmt.Sprintf("there are no test-plugin-'s in test central repo:%s , make sure its valid test central repo with test-plugins", os.Getenv(framework.TanzuCliE2ETestCentralRepositoryURL)))
})
