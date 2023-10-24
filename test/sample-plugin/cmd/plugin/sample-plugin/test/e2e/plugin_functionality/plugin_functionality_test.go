// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package pluginfunctionality implements plugin functionality specific E2E test cases
package pluginfunctionality

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

var _ = framework.CLICoreDescribe("[Tests:Sample-Plugin-E2E][Feature:Plugin-life-cycle]", func() {
	Context("sample-plugin basic functionality", func() {
		// Test case: test sample-plugin sub-command functionality
		It("sample-plugin should be installed", func() {
			list, _, _, _ := tf.PluginCmd.ListPlugins()
			installed := false
			for _, plugin := range list {
				if plugin.Name == "sample-plugin" {
					installed = true
					break
				}
			}
			Expect(installed).To(BeTrue(), "sample-plugin should be installed")
		})

		// Test case: test sample-plugin sub-command functionality
		It("sample-plugin sub-command echo should work", func() {
			output, err := tf.PluginCmd.ExecuteSubCommand("sample-plugin echo hello")
			Expect(err).To(BeNil(), "should not occur any error when plugin sample-plugin sub-command echo executed")
			Expect(output).To(ContainSubstring("hello"), "should print hello")
		})
	})
})
