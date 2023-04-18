// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"net/url"
	"os"

	. "github.com/onsi/ginkgo/v2"
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
		It("trusted registries should include hostname of additional discoveries", func() {
			testHost1 := "registry1.vmware.com"
			testHost2 := "registry2.vmware.com"
			oldValue := os.Getenv(constants.ConfigVariableAdditionalDiscoveryForTesting)
			err := os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting,
				testHost1+"/test/path, "+testHost2+"/another/test/image")
			Expect(err).To(BeNil())

			trustedRegis := GetTrustedRegistries()
			Expect(trustedRegis).NotTo(BeNil())
			Expect(trustedRegis).Should(ContainElement(testHost1))
			Expect(trustedRegis).Should(ContainElement(testHost2))

			err = os.Setenv(constants.ConfigVariablePreReleasePluginRepoImage, oldValue)
			Expect(err).To(BeNil())
		})
		It("trusted registries should include hostname of default central discovery", func() {
			u, err := url.ParseRequestURI("https://" + constants.TanzuCLIDefaultCentralPluginDiscoveryImage)
			Expect(err).To(BeNil())
			Expect(u).NotTo(BeNil())

			trustedRegis := GetTrustedRegistries()
			Expect(trustedRegis).NotTo(BeNil())
			Expect(trustedRegis).Should(ContainElement(u.Hostname()))
		})
	})
})
