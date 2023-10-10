// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

type inventoryFilterInError struct {
	pluginFilter *plugininventory.PluginInventoryFilter
	groupFilter  *plugininventory.PluginGroupFilter
}

func (i inventoryFilterInError) Error() string {
	return ""
}

type stubInventory struct{}

func (stub *stubInventory) GetAllPlugins() ([]*plugininventory.PluginInventoryEntry, error) {
	return stub.GetPlugins(&plugininventory.PluginInventoryFilter{})
}
func (stub *stubInventory) GetPlugins(filter *plugininventory.PluginInventoryFilter) ([]*plugininventory.PluginInventoryEntry, error) {
	// Return the plugin filter so the tests can verify if it is correct
	return nil, inventoryFilterInError{pluginFilter: filter}
}
func (stub *stubInventory) GetPluginGroups(filter plugininventory.PluginGroupFilter) ([]*plugininventory.PluginGroup, error) {
	// Return the group filter so the tests can verify if it is correct
	return nil, inventoryFilterInError{groupFilter: &filter}
}
func (stub *stubInventory) CreateSchema() error {
	return nil
}
func (stub *stubInventory) InsertPlugin(pluginInventoryEntry *plugininventory.PluginInventoryEntry) error {
	return nil
}
func (stub *stubInventory) InsertPluginGroup(pg *plugininventory.PluginGroup, override bool) error {
	return nil
}
func (stub *stubInventory) UpdatePluginActivationState(pluginInventoryEntry *plugininventory.PluginInventoryEntry) error {
	return nil
}
func (stub *stubInventory) UpdatePluginGroupActivationState(pg *plugininventory.PluginGroup) error {
	return nil
}

var _ = Describe("Unit tests for DB-backed OCI discovery", func() {
	var (
		err          error
		tmpDir       string
		configFile   *os.File
		configFileNG *os.File
	)

	Describe("List plugins from inventory", func() {
		BeforeEach(func() {
			tmpDir, err = os.MkdirTemp(os.TempDir(), "")
			Expect(err).To(BeNil(), "unable to create temporary directory")

			configFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", configFile.Name())

			configFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.Unsetenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting)
			os.RemoveAll(configFile.Name())
			os.RemoveAll(configFileNG.Name())
			os.RemoveAll(tmpDir)
		})
		Context("Without any criteria", func() {
			It("should have a filter that only ignores hidden plugins", func() {
				discovery := NewOCIDiscovery("test-discovery", "test-image:latest")
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Inject the stub inventory and data dir
				dbDiscovery.pluginDataDir = tmpDir
				dbDiscovery.inventory = &stubInventory{}

				plugins, err := dbDiscovery.listPluginsFromInventory()
				Expect(plugins).To(BeNil())
				Expect(err).ToNot(BeNil())
				filterInErr, ok := err.(inventoryFilterInError)
				Expect(ok).To(BeTrue())
				Expect(*filterInErr.pluginFilter).To(Equal(plugininventory.PluginInventoryFilter{
					IncludeHidden: false,
				}))
			})
			It("with TANZU_CLI_INCLUDE_DEACTIVATED_PLUGINS_TEST_ONLY=1 the filter should include hidden plugins", func() {
				discovery := NewOCIDiscovery("test-discovery", "test-image:latest")
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Inject the stub inventory and data dir
				dbDiscovery.pluginDataDir = tmpDir
				dbDiscovery.inventory = &stubInventory{}

				err = os.Setenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting, "true")
				defer os.Unsetenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting)
				Expect(err).To(BeNil())

				plugins, err := dbDiscovery.listPluginsFromInventory()
				Expect(plugins).To(BeNil())
				Expect(err).ToNot(BeNil())
				filterInErr, ok := err.(inventoryFilterInError)
				Expect(ok).To(BeTrue())
				Expect(*filterInErr.pluginFilter).To(Equal(plugininventory.PluginInventoryFilter{
					IncludeHidden: true,
				}))
			})
		})
		Context("With a criteria", func() {
			const (
				filteredName    = "cluster"
				filteredTarget  = configtypes.TargetK8s
				filteredVersion = "v0.26.0"
				filteredOS      = "darwin"
				filteredArch    = "amd64"
			)
			It("should have a filter that matches the criteria and ignores hidden plugins", func() {
				criteria := &PluginDiscoveryCriteria{
					Name:    filteredName,
					Target:  filteredTarget,
					Version: filteredVersion,
					OS:      filteredOS,
					Arch:    filteredArch,
				}
				discovery := NewOCIDiscovery("test-discovery", "test-image:latest", WithPluginDiscoveryCriteria(criteria))
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Inject the stub inventory and data dir
				dbDiscovery.pluginDataDir = tmpDir
				dbDiscovery.inventory = &stubInventory{}

				plugins, err := dbDiscovery.listPluginsFromInventory()
				Expect(plugins).To(BeNil())
				Expect(err).ToNot(BeNil())
				filterInErr, ok := err.(inventoryFilterInError)
				Expect(ok).To(BeTrue())
				Expect(*filterInErr.pluginFilter).To(Equal(plugininventory.PluginInventoryFilter{
					Name:          filteredName,
					Target:        filteredTarget,
					Version:       filteredVersion,
					OS:            filteredOS,
					Arch:          filteredArch,
					IncludeHidden: false,
				}))
			})
			It("with TANZU_CLI_INCLUDE_DEACTIVATED_PLUGINS_TEST_ONLY=1 the filter should include hidden plugin", func() {
				criteria := &PluginDiscoveryCriteria{
					Name:    filteredName,
					Target:  filteredTarget,
					Version: filteredVersion,
					OS:      filteredOS,
					Arch:    filteredArch,
				}
				discovery := NewOCIDiscovery("test-discovery", "test-image:latest", WithPluginDiscoveryCriteria(criteria))
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Inject the stub inventory and data dir
				dbDiscovery.pluginDataDir = tmpDir
				dbDiscovery.inventory = &stubInventory{}

				err = os.Setenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting, "true")
				defer os.Unsetenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting)
				Expect(err).To(BeNil())

				plugins, err := dbDiscovery.listPluginsFromInventory()
				Expect(plugins).To(BeNil())
				Expect(err).ToNot(BeNil())
				filterInErr, ok := err.(inventoryFilterInError)
				Expect(ok).To(BeTrue())
				Expect(*filterInErr.pluginFilter).To(Equal(plugininventory.PluginInventoryFilter{
					Name:          filteredName,
					Target:        filteredTarget,
					Version:       filteredVersion,
					OS:            filteredOS,
					Arch:          filteredArch,
					IncludeHidden: true,
				}))
			})
		})
	})

	Describe("List plugin groups from inventory", func() {
		BeforeEach(func() {
			tmpDir, err = os.MkdirTemp(os.TempDir(), "")
			Expect(err).To(BeNil(), "unable to create temporary directory")

			configFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", configFile.Name())

			configFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(configFile.Name())
			os.RemoveAll(configFileNG.Name())
			os.RemoveAll(tmpDir)
		})
		Context("Without any criteria", func() {
			It("should use a filter that ignores hidden groups", func() {
				discovery := NewOCIDiscovery("test-discovery", "test-image:latest")
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Inject the stub inventory and data dir
				dbDiscovery.pluginDataDir = tmpDir
				dbDiscovery.inventory = &stubInventory{}

				groups, err := dbDiscovery.listGroupsFromInventory()
				Expect(groups).To(BeNil())
				Expect(err).ToNot(BeNil())
				filterInErr, ok := err.(inventoryFilterInError)
				Expect(ok).To(BeTrue())
				Expect(*filterInErr.groupFilter).To(Equal(plugininventory.PluginGroupFilter{
					IncludeHidden: false,
				}))
			})
			It("with TANZU_CLI_INCLUDE_DEACTIVATED_PLUGINS_TEST_ONLY=1 the filter should include hidden groups", func() {
				discovery := NewOCIDiscovery("test-discovery", "test-image:latest")
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Inject the stub inventory and data dir
				dbDiscovery.pluginDataDir = tmpDir
				dbDiscovery.inventory = &stubInventory{}

				err = os.Setenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting, "true")
				defer os.Unsetenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting)
				Expect(err).To(BeNil())

				groups, err := dbDiscovery.listGroupsFromInventory()
				Expect(groups).To(BeNil())
				Expect(err).ToNot(BeNil())
				filterInErr, ok := err.(inventoryFilterInError)
				Expect(ok).To(BeTrue())
				Expect(*filterInErr.groupFilter).To(Equal(plugininventory.PluginGroupFilter{
					IncludeHidden: true,
				}))
			})
		})
		Context("With a criteria", func() {
			const (
				filteredVendor    = "vmware"
				filteredPublisher = "tkg"
				filteredName      = "groupname"
			)
			It("should have a filter that matches the criteria and ignores hidden groups", func() {
				criteria := &GroupDiscoveryCriteria{
					Vendor:    filteredVendor,
					Publisher: filteredPublisher,
					Name:      filteredName,
				}
				discovery := NewOCIGroupDiscovery("test-discovery", "test-image:latest", WithGroupDiscoveryCriteria(criteria))
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Inject the stub inventory and data dir
				dbDiscovery.pluginDataDir = tmpDir
				dbDiscovery.inventory = &stubInventory{}

				groups, err := dbDiscovery.listGroupsFromInventory()
				Expect(groups).To(BeNil())
				Expect(err).ToNot(BeNil())
				filterInErr, ok := err.(inventoryFilterInError)
				Expect(ok).To(BeTrue())
				Expect(*filterInErr.groupFilter).To(Equal(plugininventory.PluginGroupFilter{
					Vendor:        filteredVendor,
					Publisher:     filteredPublisher,
					Name:          filteredName,
					IncludeHidden: false,
				}))
			})
			It("with TANZU_CLI_INCLUDE_DEACTIVATED_PLUGINS_TEST_ONLY=1 the filter should include hidden groups", func() {
				criteria := &GroupDiscoveryCriteria{
					Vendor:    filteredVendor,
					Publisher: filteredPublisher,
					Name:      filteredName,
				}
				discovery := NewOCIGroupDiscovery("test-discovery", "test-image:latest", WithGroupDiscoveryCriteria(criteria))
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Inject the stub inventory and data dir
				dbDiscovery.pluginDataDir = tmpDir
				dbDiscovery.inventory = &stubInventory{}

				err = os.Setenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting, "true")
				defer os.Unsetenv(constants.ConfigVariableIncludeDeactivatedPluginsForTesting)
				Expect(err).To(BeNil())

				groups, err := dbDiscovery.listGroupsFromInventory()
				Expect(groups).To(BeNil())
				Expect(err).ToNot(BeNil())
				filterInErr, ok := err.(inventoryFilterInError)
				Expect(ok).To(BeTrue())

				Expect(*filterInErr.groupFilter).To(Equal(plugininventory.PluginGroupFilter{
					Vendor:        filteredVendor,
					Publisher:     filteredPublisher,
					Name:          filteredName,
					IncludeHidden: true,
				}))
			})
		})
	})
	Describe("Fetch image", func() {
		BeforeEach(func() {
			configFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", configFile.Name())

			configFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(configFile.Name())
			os.RemoveAll(configFileNG.Name())
		})
		Context("checkImageCache function", func() {
			It("should show a detailed error", func() {
				discovery := NewOCIDiscovery("test-discovery", "test-image:latest")
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				_, _, err := dbDiscovery.checkImageCache()
				Expect(err).To(Not(BeNil()), "expected error when checking an invalid image")
				Expect(err.Error()).To(ContainSubstring(`plugins discovery image resolution failed. Please check that the repository image URL "test-image:latest" is correct: error getting the image digest: GET https://index.docker.io/v2/library/test-image/manifests/latest`))
			})
		})
	})
})
