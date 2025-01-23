// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/otiai10/copy"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var _ = Describe("metrics helper tests", func() {
	var (
		configFile   *os.File
		configFileNG *os.File
		err          error
	)
	const (
		testProjectName      = "project-A"
		testProjectID        = "project-A-ID"
		testSpaceName        = "space-A"
		testClusterGroupName = "clustergroup-A"
	)
	BeforeEach(func() {
		configFile, err = os.CreateTemp("", "config")
		Expect(err).To(BeNil())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), configFile.Name())
		Expect(err).To(BeNil(), "Error while copying tanzu config file for testing")
		os.Setenv("TANZU_CONFIG", configFile.Name())

		configFileNG, err = os.CreateTemp("", "config_ng")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng.yaml"), configFileNG.Name())
		Expect(err).To(BeNil(), "Error while copying tanzu-ng config file for testing")
	})
	AfterEach(func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())
	})

	Context("when the plugin target is k8s and the current active context is of type 'kubernetes'", func() {
		It("should return the hash of kubernetes active context prefixed with 'kubernetes'", func() {
			pluginInfo := &cli.PluginInfo{
				Name:    "k8s-plugin",
				Version: "1.0.0",
				Target:  configtypes.TargetK8s,
			}
			epHashStr := getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).ToNot(BeEmpty())

			ctx, err := configlib.GetActiveContext(configtypes.ContextTypeK8s)
			Expect(err).ToNot(HaveOccurred())
			Expect(epHashStr).To(Equal(string(configtypes.ContextTypeK8s) + ":" + computeEndpointSHAForK8sContext(ctx)))
		})
	})
	Context("when the plugin target is k8s and the current active context is of type 'tanzu'", func() {
		It("should return the hash of tanzu active context prefixed with 'tanzu'", func() {
			pluginInfo := &cli.PluginInfo{
				Name:    "k8s-plugin",
				Version: "1.0.0",
				Target:  configtypes.TargetK8s,
			}
			// when tanzu context has active org
			ctx, err := configlib.GetContext("test-tanzu-context")
			Expect(err).ToNot(HaveOccurred())
			ctx.AdditionalMetadata[configlib.ProjectNameKey] = ""
			ctx.AdditionalMetadata[configlib.SpaceNameKey] = ""
			// Also when the ClusterGroupNameKey is not configured under AdditionalMetadata
			err = configlib.SetContext(ctx, true)
			Expect(err).ToNot(HaveOccurred())

			epHashStr := getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).ToNot(BeEmpty())
			Expect(epHashStr).To(Equal(string(configtypes.ContextTypeTanzu) + ":" + computeEndpointSHAForTanzuContext(ctx)))

			// when tanzu context has active project
			ctx.AdditionalMetadata[configlib.ProjectNameKey] = testProjectName
			ctx.AdditionalMetadata[configlib.ProjectIDKey] = testProjectID
			ctx.AdditionalMetadata[configlib.SpaceNameKey] = ""
			ctx.AdditionalMetadata[configlib.ClusterGroupNameKey] = ""
			err = configlib.SetContext(ctx, true)
			Expect(err).ToNot(HaveOccurred())

			epHashStr = getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).ToNot(BeEmpty())
			Expect(epHashStr).To(Equal(string(configtypes.ContextTypeTanzu) + ":" + computeEndpointSHAForTanzuContext(ctx)))

			// when tanzu context has active space
			ctx.AdditionalMetadata[configlib.ProjectNameKey] = testProjectName
			ctx.AdditionalMetadata[configlib.ProjectIDKey] = testProjectID
			ctx.AdditionalMetadata[configlib.SpaceNameKey] = testSpaceName
			ctx.AdditionalMetadata[configlib.ClusterGroupNameKey] = ""
			err = configlib.SetContext(ctx, true)
			Expect(err).ToNot(HaveOccurred())

			// when tanzu context has active clustergroup
			ctx.AdditionalMetadata[configlib.ProjectNameKey] = testProjectName
			ctx.AdditionalMetadata[configlib.ProjectIDKey] = testProjectID
			ctx.AdditionalMetadata[configlib.SpaceNameKey] = ""
			ctx.AdditionalMetadata[configlib.ClusterGroupNameKey] = testClusterGroupName
			err = configlib.SetContext(ctx, true)
			Expect(err).ToNot(HaveOccurred())

			epHashStr = getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).ToNot(BeEmpty())
			Expect(epHashStr).To(Equal(string(configtypes.ContextTypeTanzu) + ":" + computeEndpointSHAForTanzuContext(ctx)))
		})
	})
	Context("when the plugin target is k8s and there is no active context of type kubernetes/tanzu", func() {
		It("should return the empty string", func() {
			pluginInfo := &cli.PluginInfo{
				Name:    "k8s-plugin",
				Version: "1.0.0",
				Target:  configtypes.TargetK8s,
			}
			// remove the active contexts of type k8s
			err := configlib.RemoveActiveContext(configtypes.ContextTypeK8s)
			Expect(err).ToNot(HaveOccurred())

			epHashStr := getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).To(BeEmpty())
		})
	})
	Context("when the plugin target is global and the current active context is of type 'tanzu'", func() {
		It("should return the hash of tanzu active context prefixed with 'tanzu'", func() {
			pluginInfo := &cli.PluginInfo{
				Name:    "global-plugin",
				Version: "1.0.0",
				Target:  configtypes.TargetGlobal,
			}
			// when tanzu context has active org
			ctx, err := configlib.GetContext("test-tanzu-context")
			Expect(err).ToNot(HaveOccurred())
			ctx.AdditionalMetadata[configlib.ProjectNameKey] = ""
			ctx.AdditionalMetadata[configlib.SpaceNameKey] = ""
			// Also when the ClusterGroupNameKey is not configured under AdditionalMetadata
			err = configlib.SetContext(ctx, true)
			Expect(err).ToNot(HaveOccurred())

			epHashStr := getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).ToNot(BeEmpty())
			Expect(epHashStr).To(Equal(string(configtypes.ContextTypeTanzu) + ":" + computeEndpointSHAForTanzuContext(ctx)))

			// when tanzu context has active project
			ctx.AdditionalMetadata[configlib.ProjectNameKey] = testProjectName
			ctx.AdditionalMetadata[configlib.ProjectIDKey] = testProjectID
			ctx.AdditionalMetadata[configlib.SpaceNameKey] = ""
			ctx.AdditionalMetadata[configlib.ClusterGroupNameKey] = ""
			err = configlib.SetContext(ctx, true)
			Expect(err).ToNot(HaveOccurred())

			epHashStr = getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).ToNot(BeEmpty())
			Expect(epHashStr).To(Equal(string(configtypes.ContextTypeTanzu) + ":" + computeEndpointSHAForTanzuContext(ctx)))

			// when tanzu context has active space
			ctx.AdditionalMetadata[configlib.ProjectNameKey] = testProjectName
			ctx.AdditionalMetadata[configlib.ProjectIDKey] = testProjectID
			ctx.AdditionalMetadata[configlib.SpaceNameKey] = testSpaceName
			ctx.AdditionalMetadata[configlib.ClusterGroupNameKey] = ""
			err = configlib.SetContext(ctx, true)
			Expect(err).ToNot(HaveOccurred())

			// when tanzu context has active clustergroup
			ctx.AdditionalMetadata[configlib.ProjectNameKey] = testProjectName
			ctx.AdditionalMetadata[configlib.ProjectIDKey] = testProjectID
			ctx.AdditionalMetadata[configlib.SpaceNameKey] = ""
			ctx.AdditionalMetadata[configlib.ClusterGroupNameKey] = testClusterGroupName
			err = configlib.SetContext(ctx, true)
			Expect(err).ToNot(HaveOccurred())

			epHashStr = getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).ToNot(BeEmpty())
			Expect(epHashStr).To(Equal(string(configtypes.ContextTypeTanzu) + ":" + computeEndpointSHAForTanzuContext(ctx)))
		})
	})
	Context("when the plugin target is global and the current active context is of type 'kubernetes'", func() {
		It("should return the hash of kubernetes active context prefixed with 'kubernetes'", func() {
			pluginInfo := &cli.PluginInfo{
				Name:    "global-plugin",
				Version: "1.0.0",
				Target:  configtypes.TargetGlobal,
			}
			epHashStr := getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).ToNot(BeEmpty())

			ctx, err := configlib.GetActiveContext(configtypes.ContextTypeK8s)
			Expect(err).ToNot(HaveOccurred())
			Expect(epHashStr).To(Equal(string(configtypes.ContextTypeK8s) + ":" + computeEndpointSHAForK8sContext(ctx)))
		})
	})
	Context("when the plugin target is global and there is no active context of type tanzu or k8s", func() {
		It("should return the empty string", func() {
			pluginInfo := &cli.PluginInfo{
				Name:    "global-plugin",
				Version: "1.0.0",
				Target:  configtypes.TargetK8s,
			}
			// remove the active contexts of type k8s
			err := configlib.RemoveActiveContext(configtypes.ContextTypeK8s)
			Expect(err).ToNot(HaveOccurred())

			epHashStr := getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).To(BeEmpty())
		})
	})
	Context("when the plugin target is mission-control and the current active context is of context type 'mission-control'", func() {
		It("should return the hash of mission-control active context prefixed with 'mission-control'", func() {
			pluginInfo := &cli.PluginInfo{
				Name:    "tmc-plugin",
				Version: "1.0.0",
				Target:  configtypes.TargetTMC,
			}
			epHashStr := getEndpointSHAWithCtxTypePrefix(pluginInfo)
			Expect(epHashStr).ToNot(BeEmpty())

			ctx, err := configlib.GetActiveContext(configtypes.ContextTypeTMC)
			Expect(err).ToNot(HaveOccurred())
			Expect(epHashStr).To(Equal(string(configtypes.ContextTypeTMC) + ":" + computeEndpointSHAForTMCContext(ctx)))
		})
	})
})
