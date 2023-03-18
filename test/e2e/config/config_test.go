// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config_e2e_test provides config command specific E2E test cases
package config_e2e_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

const TRUE = "true"

// This test suite tests `tanzu config` life cycle tests
// tests `tanzu config init` by deleting the existing config files
// tests `tanzu config init` and make sure previous set flags are not deleted
// tests `tanzu config set` and `tanzu config unset` commands
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:config]", func() {
	var (
		tf                *framework.Framework
		randomFeatureFlag string
	)
	BeforeEach(func() {
		tf = framework.NewFramework()
	})
	Context("config feature flag operations", func() {
		When("new config flag set with value", func() {
			It("should set flag and unset flag successfully", func() {
				flagName := "e2e-test-" + framework.RandomString(4)
				randomFeatureFlagPath := "features.global." + flagName
				flagVal := TRUE
				err := tf.Config.ConfigSetFeatureFlag(randomFeatureFlagPath, flagVal)
				Expect(err).To(BeNil())

				val, err := tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath)
				Expect(err).To(BeNil())
				Expect(val).Should(Equal(TRUE))

				err = tf.Config.ConfigUnsetFeature(randomFeatureFlagPath)
				Expect(err).To(BeNil())

				val, err = tf.Config.ConfigGetFeatureFlag(randomFeatureFlagPath)
				Expect(err).To(BeNil())
				Expect(val).Should(Equal(""))
			})
		})
		When("config init called when config files not exists and test re-init use case", func() {
			It("should initialize configuration successfully", func() {
				// delete config files
				err := tf.Config.DeleteCLIConfigurationFiles()
				Expect(err).To(BeNil())
				// call init
				err = tf.Config.ConfigInit()
				Expect(err).To(BeNil())
				// should create config files
				Expect(tf.Config.IsCLIConfigurationFilesExists()).To(BeTrue())
			})
			It("should able to set new feature flag", func() {
				// set feature flag
				randomFeatureFlag = "features.global." + "e2e-test-" + framework.RandomString(4)
				err := tf.Config.ConfigSetFeatureFlag(randomFeatureFlag, TRUE)
				Expect(err).To(BeNil())

				val, err := tf.Config.ConfigGetFeatureFlag(randomFeatureFlag)
				Expect(err).To(BeNil())
				Expect(val).Should(Equal(TRUE))
			})
			It("re-init and should not remove previous set flags", func() {
				// call init
				err := tf.Config.ConfigInit()
				Expect(err).To(BeNil())
				// second run of init should not remove the existing feature flag
				val, err := tf.Config.ConfigGetFeatureFlag(randomFeatureFlag)
				Expect(err).To(BeNil())
				Expect(val).Should(Equal(TRUE))

				// unset the feature flag
				err = tf.Config.ConfigUnsetFeature(randomFeatureFlag)
				Expect(err).To(BeNil())
			})
		})
	})
})
