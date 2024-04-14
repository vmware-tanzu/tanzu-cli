// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package clilifecycle

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// These tests verify that when the central config file is missing from the cache,
// a global initializer is triggered to invalidate the cache.
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Initialize-central-config-cache]", func() {
	var (
		tf *framework.Framework
	)
	BeforeEach(func() {
		tf = framework.NewFramework()
	})
	Context("tests for the central config global initializer", func() {
		const initializationStr = "Some initialization of the CLI is required"
		const initErrorStr = "initialization encountered an error"

		It("no initialization in a normal situation", func() {
			// Setup the plugin cache which will include the central config
			_, err := tf.PluginCmd.InitPluginDiscoverySource()
			Expect(err).To(BeNil(), "should not get any error for plugin source init")

			// Run any command to see that the global initializer is not triggered
			_, _, errStream, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil())
			Expect(errStream).ToNot(ContainSubstring(initializationStr))
			Expect(errStream).ToNot(ContainSubstring(initErrorStr))
		})
		It("initialization when config file is missing", func() {
			err := deleteCentralConfigFile()
			Expect(err).To(BeNil())

			// Run any command to see that the global initializer is triggered
			_, _, errStream, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil())
			Expect(errStream).To(ContainSubstring(initializationStr))
			Expect(errStream).ToNot(ContainSubstring(initErrorStr))

			// Run the command again to see that the global initializer is not triggered
			// since the cache was fixed
			_, _, errStream, err = tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil())
			Expect(errStream).ToNot(ContainSubstring(initializationStr))
			Expect(errStream).ToNot(ContainSubstring(initErrorStr))
		})
		It("initialization when config file is missing a second time", func() {
			// Test that if the central config file is missing again, the global initializer
			// is triggered again.  This is to simulate if an older CLI version is used
			// after the cache was cleaned up.
			err := deleteCentralConfigFile()
			Expect(err).To(BeNil())

			// Run any command to see that the global initializer is triggered
			_, _, errStream, err := tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil())
			Expect(errStream).To(ContainSubstring(initializationStr))
			Expect(errStream).ToNot(ContainSubstring(initErrorStr))

			// Run the command again to see that the global initializer is not triggered
			// since the cache was fixed
			_, _, errStream, err = tf.PluginCmd.ListPlugins()
			Expect(err).To(BeNil())
			Expect(errStream).ToNot(ContainSubstring(initializationStr))
			Expect(errStream).ToNot(ContainSubstring(initErrorStr))
		})
	})
})

func deleteCentralConfigFile() error {
	centralCfg := filepath.Join(framework.TestHomeDir, ".cache", "tanzu", common.PluginInventoryDirName, config.DefaultStandaloneDiscoveryName, constants.CentralConfigFileName)

	return os.Remove(centralCfg)
}
