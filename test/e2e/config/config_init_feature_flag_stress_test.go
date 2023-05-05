// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config_e2e_test provides config command specific E2E test cases
package config_e2e_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

// This test suite tests adds `tanzu config` stress tests
// adds 100 flags, validates the get flag, unset flag use cases.
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:config]", func() {
	var (
		fflag string
	)
	Context("config feature flag operations", func() {
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
			It("should able to set new feature flags", func() {
				// set feature flag
				for i := 0; i < numberOfFlagsForStressTest; i++ {
					fflag = featureFlagPrefix + framework.RandomString(5)
					err := tf.Config.ConfigSetFeatureFlag(fflag, TRUE)
					Expect(err).To(BeNil(), noErrorForFeatureFlagSet)

					val, err := tf.Config.ConfigGetFeatureFlag(fflag)
					Expect(err).To(BeNil(), noErrorForFeatureFlagGet)
					Expect(val).Should(Equal(TRUE))
					flags = append(flags, fflag)
				}
			})
			It("re-init and should not remove previous set flags", func() {
				// call init
				err := tf.Config.ConfigInit()
				Expect(err).To(BeNil(), noErrorForConfigInit)

				// validate the feature flag values
				for i := 0; i < numberOfFlagsForStressTest; i++ {
					val, err := tf.Config.ConfigGetFeatureFlag(flags[i])
					Expect(err).To(BeNil(), noErrorForFeatureFlagGet)
					Expect(val).Should(Equal(TRUE), "the value should be same as set")
					// unset the flag
					err = tf.Config.ConfigUnsetFeature(flags[i])
					Expect(err).To(BeNil())
				}
			})
			It("validate flags after unset", func() {
				// validate the feature flag values
				for i := 0; i < numberOfFlagsForStressTest; i++ {
					val, err := tf.Config.ConfigGetFeatureFlag(flags[i])
					Expect(err).To(BeNil(), noErrorForFeatureFlagGet)
					Expect(val).To(BeEmpty(), "flag should not exist")
				}
			})
		})
	})
})
