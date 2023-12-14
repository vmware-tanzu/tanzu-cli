// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
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
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				_, _, err := dbDiscovery.checkImageCache()
				Expect(err).To(Not(BeNil()), "expected error when checking an invalid image")
				Expect(err.Error()).To(ContainSubstring(`plugins discovery image resolution failed. Please check that the repository image URL "test-image:latest" is correct: error getting the image digest: GET https://index.docker.io/v2/library/test-image/manifests/latest`))
			})
		})

		Context("checkDigestFileExistence function", func() {
			const (
				validDigest   = "1234567890"
				discoveryName = "test-discovery"
				imageURI      = "test-image:latest"
			)
			var dbDir, digestFile string
			var err error
			BeforeEach(func() {
				// Create a fake db file
				dbDir, err = os.MkdirTemp("", "test-cache-dir")
				Expect(err).To(BeNil())

				common.DefaultCacheDir = dbDir

				// Create the directory for the DB file
				pluginDBdir := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, discoveryName)
				err := os.MkdirAll(pluginDBdir, 0755)
				Expect(err).To(BeNil())
				// Create the DB file
				pluginDBFile := filepath.Join(pluginDBdir, plugininventory.SQliteDBFileName)
				file, err := os.Create(pluginDBFile)
				Expect(err).To(BeNil())
				file.Close()

				// Create a the digest file with the URI of the image as its content
				digestFile = filepath.Join(pluginDBdir, "digest."+validDigest)
				file, err = os.Create(digestFile)
				Expect(err).To(BeNil())
				_, err = file.WriteString(imageURI)
				Expect(err).To(BeNil())
				file.Close()
			})
			AfterEach(func() {
				os.RemoveAll(dbDir)
			})

			It("should return empty if the digest matches", func() {
				discovery := NewOCIDiscovery(discoveryName, imageURI)
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				digestFileName := dbDiscovery.checkDigestFileExistence(validDigest, "")
				Expect(digestFileName).To(BeEmpty(), "expected an empty digest filename")

				Expect(checkFileContentIsEqual(digestFile, imageURI)).To(BeTrue(), "expected the digest file to have the same content as the image URI")
			})
			It("should have update the URI in the digest file if the digest matches but the URI has changed", func() {
				newImageURI := "test-image"
				discovery := NewOCIDiscovery(discoveryName, newImageURI)
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				digestFileName := dbDiscovery.checkDigestFileExistence(validDigest, "")
				Expect(digestFileName).To(BeEmpty(), "expected an empty digest filename")

				Expect(checkFileContentIsEqual(digestFile, newImageURI)).To(BeTrue(), "expected the digest file to have the same content as the image URI")
			})
			It("should return a new digest file name if the digest does not match", func() {
				discovery := NewOCIDiscovery(discoveryName, imageURI)
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				newdigest := "0987654321"
				expectedDigestFileName := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, discoveryName, "digest."+newdigest)
				digestFileName := dbDiscovery.checkDigestFileExistence(newdigest, "")
				Expect(digestFileName).To(Equal(expectedDigestFileName), "expected a new digest filename")

				// Check that the existing digest file was removed
				_, err := os.Stat(digestFile)
				Expect(os.IsNotExist(err)).To(BeTrue(), "expected the old digest file to be removed")
			})
		})

		Context("cacheTTLExpired and resetCacheTTL functions", func() {
			var dbDir, expiredDigest, nonExpiredDigest string
			var err error
			BeforeEach(func() {
				// Create a fake db file
				dbDir, err = os.MkdirTemp("", "test-cache-dir")
				Expect(err).To(BeNil())

				common.DefaultCacheDir = dbDir

				// Create an expired and an non-expired discovery in the cache directory
				for _, discoveryName := range []string{"test-expired", "test-notexpired"} {
					imageURI := discoveryName + "-image:latest"

					// Create the discovery in the config file but not for the "test additional" ones
					discovery := configtypes.PluginDiscovery{
						OCI: &configtypes.OCIDiscovery{
							Name:  discoveryName,
							Image: imageURI,
						}}
					err = configlib.SetCLIDiscoverySource(discovery)
					Expect(err).To(BeNil())

					// Create the directory for the DB file
					pluginDBdir := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, discoveryName)
					err := os.MkdirAll(pluginDBdir, 0755)
					Expect(err).To(BeNil())
					// Create the DB file
					pluginDBFile := filepath.Join(pluginDBdir, plugininventory.SQliteDBFileName)
					file, err := os.Create(pluginDBFile)
					Expect(err).To(BeNil())
					file.Close()

					// Create a the digest file with the URI of the image as its content
					digestFile := filepath.Join(pluginDBdir, "digest.1234567890")
					file, err = os.Create(digestFile)
					Expect(err).To(BeNil())
					_, err = file.WriteString(imageURI)
					Expect(err).To(BeNil())
					file.Close()

					// Set an expired timestamp for the digest file and a non-expired timestamp for the other
					if strings.Contains(discoveryName, "notexpired") {
						// Set an non-expired time of 2 seconds ago so the TTL is not expired
						err = os.Chtimes(digestFile, time.Now(), time.Now().Add(-2*time.Second))
						Expect(err).To(BeNil())
						nonExpiredDigest = digestFile
					} else {
						// Set an expired time of 30 hours ago so the TTL is expired
						err = os.Chtimes(digestFile, time.Now(), time.Now().Add(-30*time.Hour))
						Expect(err).To(BeNil())
						expiredDigest = digestFile
					}
				}
			})
			AfterEach(func() {
				os.RemoveAll(dbDir)
				os.Unsetenv(constants.ConfigVariablePluginDBCacheTTL)
			})
			It("cacheTTLExpired should return true when TTL is expired", func() {
				discovery := NewOCIDiscovery("test-expired", "test-expired-image:latest")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				Expect(dbDiscovery.cacheTTLExpired()).To(BeTrue())
			})
			It("cacheTTLExpired should return false when TTL is not expired", func() {
				discovery := NewOCIDiscovery("test-notexpired", "test-notexpired-image:latest")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				Expect(dbDiscovery.cacheTTLExpired()).To(BeFalse())
			})
			It("cacheTTLExpired should return true when URI changed", func() {
				discovery := NewOCIDiscovery("test-notexpired", "changedURI:latest")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				Expect(dbDiscovery.cacheTTLExpired()).To(BeTrue())
			})
			It("cacheTTLExpired should return true when we shorten the TTL to an expired value", func() {
				discovery := NewOCIDiscovery("test-notexpired", "test-notexpired-image:latest")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Set the TTL to 1 second, which should expire this digest
				os.Setenv(constants.ConfigVariablePluginDBCacheTTL, "1")
				Expect(dbDiscovery.cacheTTLExpired()).To(BeTrue())
			})
			It("cacheTTLExpired should return true when there is no DB, even if the TTL has not expired", func() {
				discovery := NewOCIDiscovery("test-notexpired", "test-notexpired-image:latest")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Remove the cache, as if a plugin clean had occurred
				os.RemoveAll(filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName))

				// Make sure the cache is considered expired
				Expect(dbDiscovery.cacheTTLExpired()).To(BeTrue())
			})

			It("resetCacheTTL should reset the digest file to time.Now()", func() {
				discovery := NewOCIDiscovery("test-notexpired", "test-image:latest")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				dbDiscovery.resetCacheTTL()

				// The mtime of the digest files should be very very close to the current time
				// Let's check that it is within 1 second of the current time.
				stat, err := os.Stat(expiredDigest)
				Expect(err).To(BeNil())
				Expect(time.Since(stat.ModTime()).Seconds()).Should(BeNumerically("<", 1*time.Second))

				stat, err = os.Stat(nonExpiredDigest)
				Expect(err).To(BeNil())
				Expect(time.Since(stat.ModTime()).Seconds()).Should(BeNumerically("<", 1*time.Second))
			})
		})
	})
})

func checkFileContentIsEqual(filename, content string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		firstLine := scanner.Text()
		return firstLine == content
	}
	return false
}
