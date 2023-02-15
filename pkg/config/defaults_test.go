// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

var _ = Describe("defaults test cases", func() {
	Context("default locations and repositories", func() {
		It("should initialize ClientOptions", func() {
			artLocations := GetTrustedArtifactLocations()
			Expect(artLocations).NotTo(BeNil())
		})
		It("trusted registries should return value", func() {
			DefaultAllowedPluginRepositories = "https://storage.googleapis.com"
			trustedRegis := GetTrustedRegistries()
			Expect(trustedRegis).NotTo(BeNil())
			DefaultAllowedPluginRepositories = ""
		})
		It("trusted registries should include hostname of env var", func() {
			testHost := "example.com"
			oldValue := os.Getenv(constants.ConfigVariablePreReleasePluginRepoImage)
			err := os.Setenv(constants.ConfigVariablePreReleasePluginRepoImage, testHost+"/test/path")
			Expect(err).To(BeNil())

			trustedRegis := GetTrustedRegistries()
			Expect(trustedRegis).NotTo(BeNil())
			Expect(trustedRegis).Should(ContainElement(testHost))

			err = os.Setenv(constants.ConfigVariablePreReleasePluginRepoImage, oldValue)
			Expect(err).To(BeNil())
		})
	})
})
