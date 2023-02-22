// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var pluginEntries = []*plugininventory.PluginInventoryEntry{
	{
		Name:               "cluster",
		Target:             configtypes.TargetK8s,
		Description:        "cluster plugin for k8s",
		Publisher:          "tkg",
		Vendor:             "vmware",
		RecommendedVersion: "v0.28.0",
		Artifacts: distribution.Artifacts{
			"v0.26.0": []distribution.Artifact{
				{
					Image:  "darwin/amd64/k8s/cluster:v0.26.0",
					URI:    "",
					Digest: "1a11111",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "linux/amd64/k8s/cluster:v0.26.0",
					URI:    "",
					Digest: "1a22222",
					OS:     "linux",
					Arch:   "amd64",
				},
				{
					Image:  "windows/amd64/k8s/cluster:v0.26.0",
					URI:    "",
					Digest: "1a33333",
					OS:     "windows",
					Arch:   "amd64",
				},
				{
					Image:  "darwin/arm64/k8s/cluster:v0.26.0",
					URI:    "",
					Digest: "1a44444",
					OS:     "darwin",
					Arch:   "arm64",
				},
			},
			"v0.28.0": []distribution.Artifact{
				{
					Image:  "darwin/amd64/k8s/cluster:v0.28.0",
					URI:    "",
					Digest: "1b11111",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "linux/amd64/k8s/cluster:v0.28.0",
					URI:    "",
					Digest: "1b22222",
					OS:     "linux",
					Arch:   "amd64",
				},
				{
					Image:  "windows/amd64/k8s/cluster:v0.28.0",
					URI:    "",
					Digest: "1b33333",
					OS:     "windows",
					Arch:   "amd64",
				},
				{
					Image:  "darwin/arm64/k8s/cluster:v0.28.0",
					URI:    "",
					Digest: "1b44444",
					OS:     "darwin",
					Arch:   "arm64",
				},
			},
		},
	},
	{
		Name:               "cluster",
		Target:             configtypes.TargetTMC,
		Description:        "Cluster plugin for tmc",
		Publisher:          "tmc",
		Vendor:             "vmware",
		RecommendedVersion: "v0.2.0",
		Artifacts: distribution.Artifacts{
			"v0.0.1": []distribution.Artifact{
				{
					Image:  "darwin/amd64/k8s/cluster:v0.0.1",
					URI:    "",
					Digest: "2a11111",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "linux/amd64/k8s/cluster:v0.0.1",
					URI:    "",
					Digest: "2a22222",
					OS:     "linux",
					Arch:   "amd64",
				},
				{
					Image:  "windows/amd64/k8s/cluster:v0.0.1",
					URI:    "",
					Digest: "2a33333",
					OS:     "windows",
					Arch:   "amd64",
				},
			},
			"v0.2.0": []distribution.Artifact{
				{
					Image:  "darwin/amd64/k8s/cluster:v0.2.0",
					URI:    "",
					Digest: "2b11111",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "linux/amd64/k8s/cluster:v0.2.0",
					URI:    "",
					Digest: "2b22222",
					OS:     "linux",
					Arch:   "amd64",
				},
				{
					Image:  "windows/amd64/k8s/cluster:v0.2.0",
					URI:    "",
					Digest: "2b33333",
					OS:     "windows",
					Arch:   "amd64",
				},
			},
		},
	},
	{
		Name:               "telemetry",
		Target:             configtypes.TargetK8s,
		Description:        "telemetry plugin for k8s",
		Publisher:          "tkg",
		Vendor:             "vmware",
		RecommendedVersion: "v0.28.0",
		Artifacts: distribution.Artifacts{
			"v0.26.0": []distribution.Artifact{
				{
					Image:  "darwin/amd64/k8s/telemetry:v0.26.0",
					URI:    "",
					Digest: "3a11111",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "linux/amd64/k8s/telemetry:v0.26.0",
					URI:    "",
					Digest: "3a22222",
					OS:     "linux",
					Arch:   "amd64",
				},
				{
					Image:  "windows/amd64/k8s/telemetry:v0.26.0",
					URI:    "",
					Digest: "3a33333",
					OS:     "windows",
					Arch:   "amd64",
				},
			},
			"v0.28.0": []distribution.Artifact{
				{
					Image:  "darwin/amd64/k8s/telemetry:v0.28.0",
					URI:    "",
					Digest: "3b11111",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "linux/amd64/k8s/telemetry:v0.28.0",
					URI:    "",
					Digest: "3b22222",
					OS:     "linux",
					Arch:   "amd64",
				},
				{
					Image:  "windows/amd64/k8s/telemetry:v0.28.0",
					URI:    "",
					Digest: "3b33333",
					OS:     "windows",
					Arch:   "amd64",
				},
			},
		},
	},
}

type stubInventory struct{}

func (stub *stubInventory) GetAllPlugins() ([]*plugininventory.PluginInventoryEntry, error) {
	return pluginEntries, nil
}

// nolint: gocyclo
func (stub *stubInventory) GetPlugins(filter *plugininventory.PluginInventoryFilter) ([]*plugininventory.PluginInventoryEntry, error) {
	var matchingEntries []*plugininventory.PluginInventoryEntry
	// First find the matching plugins
	for _, entry := range pluginEntries {
		if filter.Name != "" && entry.Name != filter.Name {
			continue
		}
		if filter.Target != "" && entry.Target != filter.Target {
			continue
		}
		if filter.Publisher != "" && entry.Publisher != filter.Publisher {
			continue
		}
		if filter.Vendor != "" && entry.Vendor != filter.Vendor {
			continue
		}
		matchingEntries = append(matchingEntries, entry)
	}

	// Now only keep the matching artifacts
	for _, entry := range matchingEntries {
		if filter.Version != "" {
			if _, found := entry.Artifacts[filter.Version]; found {
				// Only keep the matching version
				filteredArtifacts := make(distribution.Artifacts, 0)
				filteredArtifacts[filter.Version] = entry.Artifacts[filter.Version]
				entry.Artifacts = filteredArtifacts
			} else {
				// Couldn't find the version.  Remove all artifacts.
				entry.Artifacts = distribution.Artifacts{}
				continue
			}
		}

		if filter.OS != "" || filter.Arch != "" {
			var filteredArtifactList distribution.ArtifactList
			for version, artifactList := range entry.Artifacts {
				for _, artifact := range artifactList {
					if (filter.OS == "" || artifact.OS == filter.OS) &&
						(filter.Arch == "" || artifact.Arch == filter.Arch) {
						filteredArtifactList = append(filteredArtifactList, artifact)
					}
				}
				entry.Artifacts[version] = filteredArtifactList
			}
		}
	}
	return matchingEntries, nil
}

var _ = Describe("Unit tests for DB-backed OCI discovery", func() {
	var (
		err             error
		tmpDir          string
		discovery       Discovery
		tkgConfigFile   *os.File
		tkgConfigFileNG *os.File
	)

	Describe("List plugins from inventory", func() {
		BeforeEach(func() {
			tmpDir, err = os.MkdirTemp(os.TempDir(), "")
			Expect(err).To(BeNil(), "unable to create temporary directory")

			tkgConfigFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

			tkgConfigFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())

			// Turn on central repo feature
			featureArray := strings.Split(constants.FeatureCentralRepository, ".")
			err = config.SetFeature(featureArray[1], featureArray[2], "true")
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tkgConfigFile.Name())
			os.RemoveAll(tkgConfigFileNG.Name())
			os.RemoveAll(tmpDir)
		})
		Context("Without any criteria", func() {
			It("should list all plugins", func() {
				discovery = NewOCIDiscovery("test-discovery", "test-image:latest", nil)
				Expect(err).To(BeNil(), "unable to create discovery")
				dbDiscovery, ok := discovery.(*DBBackedOCIDiscovery)
				Expect(ok).To(BeTrue(), "oci discovery is not of type DBBackedOCIDiscovery")

				// Inject the stub inventory and data dir
				dbDiscovery.pluginDataDir = tmpDir
				dbDiscovery.inventory = &stubInventory{}

				plugins, err := dbDiscovery.listPluginsFromInventory()
				Expect(err).To(BeNil())
				Expect(plugins).ToNot(BeNil())
				Expect(len(plugins)).To(Equal(len(pluginEntries)))

				for _, p := range plugins {
					entry := findMatchingPluginEntry(pluginEntries, p.Name, p.Target)

					Expect(p.Description).To(Equal(entry.Description))
					Expect(p.RecommendedVersion).To(Equal(entry.RecommendedVersion))
					Expect(p.InstalledVersion).To(Equal(""))
					Expect(p.SupportedVersions).To(Equal(getSupportedVersions(entry.Artifacts)))
					Expect(p.Distribution).To(Equal(entry.Artifacts))
					Expect(p.Optional).To(Equal(false))
					Expect(p.Scope).To(Equal(common.PluginScopeStandalone))
					Expect(p.Source).To(Equal("test-discovery"))
					Expect(p.DiscoveryType).To(Equal(common.DiscoveryTypeOCI))
				}
			})
		})
		Context("With a criteria", func() {
			It("should list only matching info", func() {
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

				plugins, err := dbDiscovery.listPluginsFromInventory()
				Expect(err).To(BeNil())
				Expect(plugins).ToNot(BeNil())
				// Only the "cluster" plugin should be returned
				Expect(len(plugins)).To(Equal(1))

				p := plugins[0]
				Expect(p.Name).To(Equal(filteredName))
				Expect(p.Target).To(Equal(filteredTarget))

				entry := findMatchingPluginEntry(pluginEntries, p.Name, p.Target)

				Expect(p.Description).To(Equal(entry.Description))
				Expect(p.RecommendedVersion).To(Equal(entry.RecommendedVersion))
				Expect(p.InstalledVersion).To(Equal(""))
				// We only asked for a single version
				Expect(p.SupportedVersions).To(Equal([]string{filteredVersion}))
				Expect(p.Optional).To(Equal(false))
				Expect(p.Scope).To(Equal(common.PluginScopeStandalone))
				Expect(p.Source).To(Equal("test-discovery"))
				Expect(p.DiscoveryType).To(Equal(common.DiscoveryTypeOCI))

				Expect(p.Distribution).ToNot(BeNil())
				artifacts, ok := p.Distribution.(distribution.Artifacts)
				Expect(ok).To(BeTrue(), "distribution is not of type Artifacts")
				// We only asked for a single version
				Expect(len(artifacts)).To(Equal(1))
				artifactList, ok := artifacts[filteredVersion]
				Expect(ok).To(BeTrue(), "artifacts don't contain the requested version")

				// We only asked for a single os/arch combination
				Expect(len(artifactList)).To(Equal(1))
				artifact := artifactList[0]
				Expect(artifact.Arch).To(Equal(filteredArch))
				Expect(artifact.OS).To(Equal(filteredOS))
			})
		})
	})
})

func getSupportedVersions(artifacts distribution.Artifacts) []string {
	var versions []string
	for v := range artifacts {
		versions = append(versions, v)
	}
	err := utils.SortVersions(versions)
	Expect(err).To(BeNil(), "error parsing versions for plugin")

	return versions
}

// findMatchingPluginEntry returns the pluginInventoryEntry that matches the specified name and target.
func findMatchingPluginEntry(entries []*plugininventory.PluginInventoryEntry, pluginName string, target configtypes.Target) *plugininventory.PluginInventoryEntry {
	for _, entry := range entries {
		if pluginName == entry.Name && target == entry.Target {
			return entry
		}
	}
	return nil
}
