// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var _ = Describe("defaults test cases", func() {
	Context("default locations and repositories", func() {
		const (
			testHost1    = "registry1.vmware.com"
			testHost2    = "registry2.vmware.com"
			privateHost1 = "registry1.private.vmware.com"
			privateHost2 = "registry2.private.vmware.com"
		)
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
		Context("with config files", func() {
			var (
				configFile   *os.File
				configFileNG *os.File
				err          error
			)
			BeforeEach(func() {
				configFile, err = os.CreateTemp("", "config")
				Expect(err).To(BeNil())
				os.Setenv("TANZU_CONFIG", configFile.Name())

				configFileNG, err = os.CreateTemp("", "config_ng")
				Expect(err).To(BeNil())
				os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

				featureArray := strings.Split(constants.FeatureContextCommand, ".")
				err = configlib.SetFeature(featureArray[1], featureArray[2], "true")
				Expect(err).To(BeNil())
			})
			AfterEach(func() {
				os.Unsetenv("TANZU_CONFIG")
				os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
				os.RemoveAll(configFile.Name())
				os.RemoveAll(configFileNG.Name())
			})
			It("trusted registries should include hostname of each configured central discovery source", func() {
				testHost1 := "example.com"
				testImage1 := testHost1 + "/the/path/to/an/image:tag"
				testImage2 := testHost1 + ":12345/the/path/to/another/image:latest"
				testHost2 := "another.com"
				testImage3 := testHost2 + "/the/path/to/an/image:tag"
				testImage4 := testHost2 + ":12345/the/path/to/another/image:latest"

				err = configlib.SetCLIDiscoverySources([]types.PluginDiscovery{
					{
						OCI: &types.OCIDiscovery{
							Name:  "default1",
							Image: testImage1,
						},
					},
					{
						OCI: &types.OCIDiscovery{
							Name:  "default2",
							Image: testImage2,
						},
					},
					{
						OCI: &types.OCIDiscovery{
							Name:  "default3",
							Image: testImage3,
						},
					}, {
						OCI: &types.OCIDiscovery{
							Name:  "default4",
							Image: testImage4,
						},
					},
				})
				Expect(err).To(BeNil())

				trustedRegis := GetTrustedRegistries()
				Expect(trustedRegis).NotTo(BeNil())
				Expect(trustedRegis).Should(ContainElement(testHost1))
				Expect(trustedRegis).Should(ContainElement(testHost2))
			})
		})
		It("trusted registries should include hostname of additional discoveries for test if provided", func() {
			oldValue := os.Getenv(constants.ConfigVariableAdditionalDiscoveryForTesting)
			err := os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting,
				testHost1+"/test/path, "+testHost2+"/another/test/image")
			Expect(err).To(BeNil())

			trustedRegis := GetTrustedRegistries()
			Expect(trustedRegis).NotTo(BeNil())
			Expect(trustedRegis).Should(ContainElement(testHost1))
			Expect(trustedRegis).Should(ContainElement(testHost2))

			err = os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, oldValue)
			Expect(err).To(BeNil())
		})
		It("trusted registries should include hostname of additional private discoveries if provided", func() {
			oldValue := os.Getenv(constants.ConfigVariableAdditionalPrivateDiscoveryImages)
			err := os.Setenv(constants.ConfigVariableAdditionalPrivateDiscoveryImages,
				privateHost1+"/private/path, "+privateHost2+"/another/private/image")
			Expect(err).To(BeNil())

			trustedRegis := GetTrustedRegistries()
			Expect(trustedRegis).NotTo(BeNil())
			Expect(trustedRegis).Should(ContainElement(privateHost1))
			Expect(trustedRegis).Should(ContainElement(privateHost2))

			err = os.Setenv(constants.ConfigVariableAdditionalPrivateDiscoveryImages, oldValue)
			Expect(err).To(BeNil())
		})
		It("trusted registries should include hostname of additional private discoveries only, if both additional private plugin discoveries and additional test plugin discoveries are provided", func() {
			oldTestDiscValue := os.Getenv(constants.ConfigVariableAdditionalDiscoveryForTesting)
			err := os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting,
				testHost1+"/test/path, "+testHost2+"/another/test/image")
			Expect(err).To(BeNil())

			oldPrivateDiscValue := os.Getenv(constants.ConfigVariableAdditionalPrivateDiscoveryImages)
			err = os.Setenv(constants.ConfigVariableAdditionalPrivateDiscoveryImages,
				privateHost1+"/private/path, "+privateHost2+"/another/private/image")
			Expect(err).To(BeNil())

			trustedRegis := GetTrustedRegistries()
			Expect(trustedRegis).NotTo(BeNil())
			Expect(trustedRegis).Should(ContainElement(privateHost1))
			Expect(trustedRegis).Should(ContainElement(privateHost2))
			Expect(trustedRegis).ShouldNot(ContainElement(testHost1))
			Expect(trustedRegis).ShouldNot(ContainElement(testHost2))

			err = os.Setenv(constants.ConfigVariableAdditionalPrivateDiscoveryImages, oldPrivateDiscValue)
			Expect(err).To(BeNil())

			err = os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, oldTestDiscValue)
			Expect(err).To(BeNil())
		})
	})
})
