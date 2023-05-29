// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package clilifecycle

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Command-init-version]", func() {
	var (
		tf *framework.Framework
	)
	BeforeEach(func() {
		tf = framework.NewFramework()
	})
	Context("tests for tanzu init and version commands", func() {
		When("init command executed", func() {
			It("should initialize cli successfully", func() {
				err := tf.CLIInit()
				Expect(err).To(BeNil())
			})
		})
		When("version command executed", func() {
			It("should return version info", func() {
				version, err := tf.CLIVersion()
				Expect(version).NotTo(BeNil())
				Expect(err).To(BeNil())
			})
		})
	})
})
