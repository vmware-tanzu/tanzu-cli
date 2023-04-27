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

var _ = Describe("Populate default central discovery", func() {
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
		os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")

		featureArray := strings.Split(constants.FeatureContextCommand, ".")
		err = configlib.SetFeature(featureArray[1], featureArray[2], "true")
		Expect(err).To(BeNil())
	})
	AfterEach(func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())
	})
	Context("when no discovery exists", func() {
		It("should create the default central discovery when 'force==false'", func() {
			err = PopulateDefaultCentralDiscovery(false)
			Expect(err).To(BeNil())

			discoverySources, err := configlib.GetCLIDiscoverySources()
			Expect(err).To(BeNil())
			Expect(len(discoverySources)).To(Equal(1))
			// It should be an OCI discovery with a specific name and image
			Expect(discoverySources[0].OCI).ToNot(BeNil())
			Expect(discoverySources[0].OCI.Name).To(Equal(DefaultStandaloneDiscoveryName))
			Expect(discoverySources[0].OCI.Image).To(Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage))
		})
		It("should create the default central discovery when 'force==true'", func() {
			err = PopulateDefaultCentralDiscovery(true)
			Expect(err).To(BeNil())

			discoverySources, err := configlib.GetCLIDiscoverySources()
			Expect(err).To(BeNil())
			Expect(len(discoverySources)).To(Equal(1))
			// It should be an OCI discovery with a specific name and image
			Expect(discoverySources[0].OCI).ToNot(BeNil())
			Expect(discoverySources[0].OCI.Name).To(Equal(DefaultStandaloneDiscoveryName))
			Expect(discoverySources[0].OCI.Image).To(Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage))
		})
	})
	Context("when the default discovery already exists", func() {
		BeforeEach(func() {
			err = PopulateDefaultCentralDiscovery(false)
			Expect(err).To(BeNil())
		})
		It("should keep the default central discovery when 'force==false'", func() {
			err = PopulateDefaultCentralDiscovery(false)
			Expect(err).To(BeNil())

			discoverySources, err := configlib.GetCLIDiscoverySources()
			Expect(err).To(BeNil())
			Expect(len(discoverySources)).To(Equal(1))
			// It should be an OCI discovery with a specific name and image
			Expect(discoverySources[0].OCI).ToNot(BeNil())
			Expect(discoverySources[0].OCI.Name).To(Equal(DefaultStandaloneDiscoveryName))
			Expect(discoverySources[0].OCI.Image).To(Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage))
		})
		It("should keep the default central discovery when 'force==true'", func() {
			err = PopulateDefaultCentralDiscovery(true)
			Expect(err).To(BeNil())

			discoverySources, err := configlib.GetCLIDiscoverySources()
			Expect(err).To(BeNil())
			Expect(len(discoverySources)).To(Equal(1))
			// It should be an OCI discovery with a specific name and image
			Expect(discoverySources[0].OCI).ToNot(BeNil())
			Expect(discoverySources[0].OCI.Name).To(Equal(DefaultStandaloneDiscoveryName))
			Expect(discoverySources[0].OCI.Image).To(Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage))
		})
	})
	Context("when a different discovery already exists", func() {
		const imageName = "different/image"
		BeforeEach(func() {
			err := configlib.SetCLIDiscoverySource(types.PluginDiscovery{
				OCI: &types.OCIDiscovery{
					Name:  DefaultStandaloneDiscoveryName,
					Image: imageName,
				},
			})
			Expect(err).To(BeNil())
		})
		It("should keep the existing discovery when 'force==false'", func() {
			err = PopulateDefaultCentralDiscovery(false)
			Expect(err).To(BeNil())

			discoverySources, err := configlib.GetCLIDiscoverySources()
			Expect(err).To(BeNil())
			Expect(len(discoverySources)).To(Equal(1))
			// It should be an OCI discovery with a specific name and image
			Expect(discoverySources[0].OCI).ToNot(BeNil())
			Expect(discoverySources[0].OCI.Name).To(Equal(DefaultStandaloneDiscoveryName))
			Expect(discoverySources[0].OCI.Image).To(Equal(imageName))
		})
		It("should replace the existing discovery when 'force==true'", func() {
			err = PopulateDefaultCentralDiscovery(true)
			Expect(err).To(BeNil())

			discoverySources, err := configlib.GetCLIDiscoverySources()
			Expect(err).To(BeNil())
			Expect(len(discoverySources)).To(Equal(1))
			// It should be an OCI discovery with a specific name and image
			Expect(discoverySources[0].OCI).ToNot(BeNil())
			Expect(discoverySources[0].OCI.Name).To(Equal(DefaultStandaloneDiscoveryName))
			Expect(discoverySources[0].OCI.Image).To(Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage))
		})
	})
	Context("when a the default central discovery was deleted by the user", func() {
		BeforeEach(func() {
			err = PopulateDefaultCentralDiscovery(false)
			Expect(err).To(BeNil())

			err := configlib.DeleteCLIDiscoverySource(DefaultStandaloneDiscoveryName)
			Expect(err).To(BeNil())
			discoverySources, err := configlib.GetCLIDiscoverySources()
			Expect(err).To(BeNil())
			Expect(discoverySources).ToNot(BeNil())
			Expect(len(discoverySources)).To(Equal(0))
		})
		It("should not add the default discovery when 'force==false'", func() {
			err = PopulateDefaultCentralDiscovery(false)
			Expect(err).To(BeNil())

			discoverySources, err := configlib.GetCLIDiscoverySources()
			Expect(err).To(BeNil())
			Expect(discoverySources).ToNot(BeNil())
			Expect(len(discoverySources)).To(Equal(0))
		})
		It("should add the default discovery when 'force==true'", func() {
			err = PopulateDefaultCentralDiscovery(true)
			Expect(err).To(BeNil())

			discoverySources, err := configlib.GetCLIDiscoverySources()
			Expect(err).To(BeNil())
			Expect(len(discoverySources)).To(Equal(1))
			// It should be an OCI discovery with a specific name and image
			Expect(discoverySources[0].OCI).ToNot(BeNil())
			Expect(discoverySources[0].OCI.Name).To(Equal(DefaultStandaloneDiscoveryName))
			Expect(discoverySources[0].OCI.Image).To(Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage))
		})
	})
})
