// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
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
		discovery    Discovery
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
			It("should have a filter that only ignore hidden plugins", func() {
				discovery = NewOCIDiscovery("test-discovery", "test-image:latest", nil)
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
				discovery = NewOCIDiscovery("test-discovery", "test-image:latest", nil)
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
			It("should have a filter that matches the criteria and ignore hidden plugins", func() {
				filteredName := "cluster" // nolint:goconst
				filteredTarget := configtypes.TargetK8s
				filteredVersion := "v0.26.0" // nolint:goconst
				filteredOS := "darwin"       // nolint:goconst
				filteredArch := "amd64"      // nolint:goconst

				discovery = NewOCIDiscovery("test-discovery", "test-image:latest", &PluginDiscoveryCriteria{
					Name:    filteredName,
					Target:  filteredTarget,
					Version: filteredVersion,
					OS:      filteredOS,
					Arch:    filteredArch,
				})
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
				filteredName := "cluster"
				filteredTarget := configtypes.TargetK8s
				filteredVersion := "v0.26.0"
				filteredOS := "darwin"
				filteredArch := "amd64"

				discovery = NewOCIDiscovery("test-discovery", "test-image:latest", &PluginDiscoveryCriteria{
					Name:    filteredName,
					Target:  filteredTarget,
					Version: filteredVersion,
					OS:      filteredOS,
					Arch:    filteredArch,
				})
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
		It("should use a filter that ignores hidden groups", func() {
			discovery = NewOCIDiscovery("test-discovery", "test-image:latest", nil)
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
			discovery = NewOCIDiscovery("test-discovery", "test-image:latest", nil)
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

	Describe("Verify inventory image signature", func() {
		var (
			cosignVerifier *fakes.Cosignhelperfake
			dbDiscovery    *DBBackedOCIDiscovery
			ok             bool
		)
		BeforeEach(func() {
			tmpDir, err = os.MkdirTemp(os.TempDir(), "")
			Expect(err).To(BeNil(), "unable to create temporary directory")

			configFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", configFile.Name())

			configFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

			discovery = NewOCIDiscovery("test-discovery", "test-image:latest", nil)
			Expect(err).To(BeNil(), "unable to create discovery")
			dbDiscovery, ok = discovery.(*DBBackedOCIDiscovery)
			Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")
		})
		AfterEach(func() {
			os.Unsetenv(constants.PluginDiscoveryImageSignatureVerificationSkipList)
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(configFile.Name())
			os.RemoveAll(configFileNG.Name())
			os.RemoveAll(tmpDir)
		})
		Context("Cosign signature verification is success", func() {
			It("should return success", func() {
				cosignVerifier = &fakes.Cosignhelperfake{}
				cosignVerifier.VerifyReturns(nil)
				err = dbDiscovery.verifyInventoryImageSignature(cosignVerifier)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When Cosign signature verification failed and TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST environment variable is set", func() {
			It("should skip signature verification and return success", func() {
				cosignVerifier = &fakes.Cosignhelperfake{}
				cosignVerifier.VerifyReturns(fmt.Errorf("signature verification fake error"))
				os.Setenv(constants.PluginDiscoveryImageSignatureVerificationSkipList, dbDiscovery.image)
				err = dbDiscovery.verifyInventoryImageSignature(cosignVerifier)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Cosign signature verification failed", func() {
			It("should return error", func() {
				cosignVerifier = &fakes.Cosignhelperfake{}
				cosignVerifier.VerifyReturns(fmt.Errorf("signature verification fake error"))
				err = dbDiscovery.verifyInventoryImageSignature(cosignVerifier)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("signature verification fake error"))
			})
		})
	})
})
