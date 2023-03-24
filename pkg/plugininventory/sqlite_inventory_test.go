// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugininventory

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"

	// Import the sqlite3 driver
	_ "github.com/mattn/go-sqlite3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Inventory Suite")
}

var piEntry1 = PluginInventoryEntry{
	Name:               "management-cluster",
	Target:             types.TargetK8s,
	Description:        "Kubernetes management cluster operations",
	Publisher:          "tkg",
	Vendor:             "vmware",
	RecommendedVersion: "v0.28.0",
	Hidden:             false,
	Artifacts: distribution.Artifacts{
		"v0.28.0": []distribution.Artifact{
			{
				OS:     "linux",
				Arch:   "amd64",
				Digest: "0000000000",
				Image:  "vmware/tkg/linux/amd64/k8s/management-cluster:v0.28.0",
			},
			{
				OS:     "darwin",
				Arch:   "amd64",
				Digest: "1111111111",
				Image:  "vmware/tkg/darwin/amd64/k8s/management-cluster:v0.28.0",
			},
			{
				OS:     "windows",
				Arch:   "amd64",
				Digest: "2222222222",
				Image:  "vmware/tkg/windows/amd64/k8s/management-cluster:v0.28.0",
			},
		},
	},
}
var piEntry2 = PluginInventoryEntry{
	Name:               "isolated-cluster",
	Target:             types.TargetGlobal,
	Description:        "Isolated cluster plugin",
	Publisher:          "otherpublisher",
	Vendor:             "othervendor",
	RecommendedVersion: "v1.2.3",
	Hidden:             false,
	Artifacts: distribution.Artifacts{
		"v1.2.3": []distribution.Artifact{
			{
				OS:     "linux",
				Arch:   "amd64",
				Digest: "3333333333",
				Image:  "othervendor/otherpublisher/linux/amd64/global/isolated-cluster:v1.2.3",
			},
		},
	},
}
var piEntry3 = PluginInventoryEntry{
	Name:               "management-cluster",
	Target:             types.TargetTMC,
	Description:        "Mission-control management cluster operations",
	Publisher:          "tmc",
	Vendor:             "vmware",
	RecommendedVersion: "",
	Hidden:             false,
	Artifacts: distribution.Artifacts{
		"v0.0.1": []distribution.Artifact{
			{
				OS:     "linux",
				Arch:   "amd64",
				Digest: "0000000000",
				Image:  "vmware/tmc/linux/amd64/tmc/management-cluster:v0.0.1",
			},
		},
	},
}

var pluginGroup1 = PluginGroup{
	Name:      "v1.0.0",
	Vendor:    "fakevendor",
	Publisher: "fakepublisher",
	Hidden:    false,
	Plugins: []*PluginGroupPluginEntry{
		{
			PluginIdentifier: PluginIdentifier{Name: "management-cluster", Target: types.TargetK8s, Version: "v0.28.0"},
			Mandatory:        false,
		},
	},
}

const createPluginsStmt = `
INSERT INTO PluginBinaries VALUES(
	'management-cluster',
	'kubernetes',
	'v0.28.0',
	'v0.28.0',
	'false',
	'Kubernetes management cluster operations',
	'tkg',
	'vmware',
	'linux',
	'amd64',
	'0000000000',
	'vmware/tkg/linux/amd64/k8s/management-cluster:v0.28.0');
INSERT INTO PluginBinaries VALUES(
	'management-cluster',
	'kubernetes',
	'v0.28.0',
	'v0.28.0',
	'false',
	'Kubernetes management cluster operations',
	'tkg',
	'vmware',
	'darwin',
	'amd64',
	'1111111111',
	'vmware/tkg/darwin/amd64/k8s/management-cluster:v0.28.0');
INSERT INTO PluginBinaries VALUES(
	'management-cluster',
	'kubernetes',
	'v0.28.0',
	'v0.26.0',
	'false',
	'Kubernetes management cluster operations',
	'tkg',
	'vmware',
	'windows',
	'amd64',
	'2222222222',
	'vmware/tkg/windows/amd64/k8s/management-cluster:v0.26.0');
INSERT INTO PluginBinaries VALUES(
	'isolated-cluster',
	'global',
	'v1.2.3',
	'v1.2.3',
	'false',
	'Isolated cluster plugin',
	'otherpublisher',
	'othervendor',
	'linux',
	'amd64',
	'3333333333',
	'othervendor/otherpublisher/linux/amd64/global/isolated-cluster:v1.2.3');
`
const createPluginTMCNoRecommendedVersionStmt = `
INSERT INTO PluginBinaries VALUES(
	'management-cluster',
	'mission-control',
	'',
	'v0.0.1',
	'false',
	'Mission-control management cluster operations',
	'tmc',
	'vmware',
	'linux',
	'amd64',
	'0000000000',
	'vmware/tmc/linux/amd64/tmc/management-cluster:v0.0.1');
INSERT INTO PluginBinaries VALUES(
	'management-cluster',
	'mission-control',
	'',
	'v0.0.2',
	'false',
	'Mission-control management cluster operations',
	'tmc',
	'vmware',
	'linux',
	'amd64',
	'1111111111',
	'vmware/tmc/linux/amd64/tmc/management-cluster:v0.0.2');
`
const createGroupsStmt = `
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'2.1.0',
	'management-cluster',
	'kubernetes',
	'v0.28.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'2.1.0',
	'package',
	'kubernetes',
	'v0.28.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'2.1.0',
	'feature',
	'kubernetes',
	'v0.28.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'2.1.0',
	'kubernetes-release',
	'kubernetes',
	'v0.28.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'2.1.0',
	'isolated-cluster',
	'kubernetes',
	'v0.28.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'1.6.0',
	'management-cluster',
	'kubernetes',
	'v0.26.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'1.6.0',
	'package',
	'kubernetes',
	'v0.26.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'1.6.0',
	'feature',
	'kubernetes',
	'v0.26.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'1.6.0',
	'kubernetes-release',
	'kubernetes',
	'v0.26.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'independent',
	'other',
	'mygroup',
	'plugin1',
	'kubernetes',
	'v0.1.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'independent',
	'other',
	'mygroup',
	'plugin2',
	'mission-control',
	'v0.2.0',
	'false',
	'false');
`

var _ = Describe("Unit tests for plugin inventory", func() {
	var (
		err       error
		inventory PluginInventory
		dbFile    *os.File
		tmpDir    string
	)

	Describe("Getting plugins from inventory", func() {
		Context("With an empty DB file", func() {
			BeforeEach(func() {
				tmpDir, err = os.MkdirTemp(os.TempDir(), "")
				Expect(err).To(BeNil(), "unable to create temporary directory")

				// Create empty file for the DB
				dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
				Expect(err).To(BeNil())

				inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an error", func() {
				_, err = inventory.GetAllPlugins()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to setup DB"))
			})
		})
		Context("With an empty DB table", func() {
			BeforeEach(func() {
				tmpDir, err = os.MkdirTemp(os.TempDir(), "")
				Expect(err).To(BeNil(), "unable to create temporary directory")

				// Create DB file
				dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
				Expect(err).To(BeNil())
				// Open DB with the sqlite driver
				db, err := sql.Open("sqlite3", dbFile.Name())
				Expect(err).To(BeNil(), "failed to open the DB for testing")
				defer db.Close()

				// Create the table but don't add any rows
				_, err = db.Exec(CreateTablesSchema)
				Expect(err).To(BeNil(), "failed to create DB table for testing")

				inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an empty list of plugins with no error", func() {
				plugins, err := inventory.GetAllPlugins()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(plugins)).To(Equal(0))
			})
		})
		Describe("With a DB table with two plugins", func() {
			BeforeEach(func() {
				tmpDir, err = os.MkdirTemp(os.TempDir(), "")
				Expect(err).To(BeNil(), "unable to create temporary directory")

				// Create DB file
				dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
				Expect(err).To(BeNil())
				// Open DB with the sqlite driver
				db, err := sql.Open("sqlite3", dbFile.Name())
				Expect(err).To(BeNil(), "failed to open the DB for testing")
				defer db.Close()

				// Create the table
				_, err = db.Exec(CreateTablesSchema)
				Expect(err).To(BeNil(), "failed to create DB table for testing")

				// Add a plugin entry to the DB
				_, err = db.Exec(createPluginsStmt)
				Expect(err).To(BeNil(), "failed to create plugin for testing")

				inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			Context("When getting all plugins", func() {
				It("should return a list of two plugins with no error", func() {
					plugins, err := inventory.GetAllPlugins()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(plugins)).To(Equal(2))

					for _, p := range plugins {
						if p.Name == "management-cluster" {
							Expect(p.RecommendedVersion).To(Equal("v0.28.0"))
							Expect(string(p.Target)).To(Equal("kubernetes"))
							Expect(p.Description).To(Equal("Kubernetes management cluster operations"))
							Expect(p.Vendor).To(Equal("vmware"))
							Expect(p.Publisher).To(Equal("tkg"))

							Expect(len(p.Artifacts)).To(Equal(2))
							artifactList := p.Artifacts["v0.28.0"]
							Expect(len(artifactList)).To(Equal(2))
							for _, a := range artifactList {
								if a.OS == "linux" { // nolint: goconst
									Expect(a.Arch).To(Equal("amd64"))
									Expect(a.Digest).To(Equal("0000000000"))
									Expect(a.Image).To(Equal(tmpDir + "/vmware/tkg/linux/amd64/k8s/management-cluster:v0.28.0"))
								} else {
									Expect(a.OS).To(Equal("darwin"))
									Expect(a.Arch).To(Equal("amd64"))
									Expect(a.Digest).To(Equal("1111111111"))
									Expect(a.Image).To(Equal(tmpDir + "/vmware/tkg/darwin/amd64/k8s/management-cluster:v0.28.0"))
								}
							}

							artifactList = p.Artifacts["v0.26.0"]
							Expect(len(artifactList)).To(Equal(1))

							Expect(artifactList[0].OS).To(Equal("windows"))
							Expect(artifactList[0].Arch).To(Equal("amd64"))
							Expect(artifactList[0].Digest).To(Equal("2222222222"))
							Expect(artifactList[0].Image).To(Equal(tmpDir + "/vmware/tkg/windows/amd64/k8s/management-cluster:v0.26.0"))
						} else {
							Expect(p.Name).To(Equal("isolated-cluster"))

							Expect(p.RecommendedVersion).To(Equal("v1.2.3"))
							Expect(p.Target).To(Equal(types.TargetGlobal))
							Expect(p.Description).To(Equal("Isolated cluster plugin"))
							Expect(p.Vendor).To(Equal("othervendor"))
							Expect(p.Publisher).To(Equal("otherpublisher"))

							Expect(len(p.Artifacts)).To(Equal(1))
							a := p.Artifacts["v1.2.3"]
							Expect(len(a)).To(Equal(1))

							Expect(a[0].OS).To(Equal("linux"))
							Expect(a[0].Arch).To(Equal("amd64"))
							Expect(a[0].Digest).To(Equal("3333333333"))
							Expect(a[0].Image).To(Equal(tmpDir + "/othervendor/otherpublisher/linux/amd64/global/isolated-cluster:v1.2.3"))
						}
					}
				})
			})
			Context("When getting a specific plugin version for k8s for an os/arch", func() {
				It("should return a list of one plugin with no error", func() {
					plugins, err := inventory.GetPlugins(&PluginInventoryFilter{
						Name:    "management-cluster",
						Target:  "kubernetes",
						Version: "v0.26.0",
						OS:      "windows",
						Arch:    "amd64",
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))

					p := plugins[0]
					Expect(p.Name).To(Equal("management-cluster"))

					Expect(p.RecommendedVersion).To(Equal("v0.28.0"))
					Expect(string(p.Target)).To(Equal("kubernetes"))
					Expect(p.Description).To(Equal("Kubernetes management cluster operations"))
					Expect(p.Vendor).To(Equal("vmware"))
					Expect(p.Publisher).To(Equal("tkg"))

					Expect(len(p.Artifacts)).To(Equal(1))
					artifactList := p.Artifacts["v0.26.0"]
					Expect(len(artifactList)).To(Equal(1))
					a := artifactList[0]
					Expect(a.OS).To(Equal("windows"))
					Expect(a.Arch).To(Equal("amd64"))
					Expect(a.Digest).To(Equal("2222222222"))
					Expect(a.Image).To(Equal(tmpDir + "/vmware/tkg/windows/amd64/k8s/management-cluster:v0.26.0"))
				})
			})
			Context("When getting the recommended version of a plugin for an os/arch", func() {
				It("should return a list of one plugin with no error", func() {
					plugins, err := inventory.GetPlugins(&PluginInventoryFilter{
						Name:    "management-cluster",
						Target:  "kubernetes",
						Version: cli.VersionLatest,
						OS:      "darwin",
						Arch:    "amd64",
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))

					p := plugins[0]
					Expect(p.Name).To(Equal("management-cluster"))

					Expect(p.RecommendedVersion).To(Equal("v0.28.0"))
					Expect(string(p.Target)).To(Equal("kubernetes"))
					Expect(p.Description).To(Equal("Kubernetes management cluster operations"))
					Expect(p.Vendor).To(Equal("vmware"))
					Expect(p.Publisher).To(Equal("tkg"))

					Expect(len(p.Artifacts)).To(Equal(1))
					artifactList := p.Artifacts["v0.28.0"]
					Expect(len(artifactList)).To(Equal(1))
					a := artifactList[0]
					Expect(a.OS).To(Equal("darwin"))
					Expect(a.Arch).To(Equal("amd64"))
					Expect(a.Digest).To(Equal("1111111111"))
					Expect(a.Image).To(Equal(tmpDir + "/vmware/tkg/darwin/amd64/k8s/management-cluster:v0.28.0"))
				})
			})
			Context("When getting plugins by vendor", func() {
				It("should return a list of one plugin with no error", func() {
					plugins, err := inventory.GetPlugins(&PluginInventoryFilter{
						Vendor: "vmware",
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))

					p := plugins[0]
					Expect(p.Name).To(Equal("management-cluster"))
					Expect(len(p.Artifacts)).To(Equal(2))
					Expect(p.Artifacts["v0.26.0"]).ToNot(BeNil())
					Expect(p.Artifacts["v0.28.0"]).ToNot(BeNil())

					Expect(p.RecommendedVersion).To(Equal("v0.28.0"))
					Expect(string(p.Target)).To(Equal("kubernetes"))
					Expect(p.Description).To(Equal("Kubernetes management cluster operations"))
					Expect(p.Vendor).To(Equal("vmware"))
					Expect(p.Publisher).To(Equal("tkg"))
				})
			})
			Context("When getting plugins by publisher", func() {
				It("should return a list of one plugin with no error", func() {
					plugins, err := inventory.GetPlugins(&PluginInventoryFilter{
						Publisher: "otherpublisher",
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))

					p := plugins[0]
					Expect(p.Name).To(Equal("isolated-cluster"))
					Expect(len(p.Artifacts)).To(Equal(1))
					Expect(p.Artifacts["v1.2.3"]).ToNot(BeNil())

					Expect(p.RecommendedVersion).To(Equal("v1.2.3"))
					Expect(p.Target).To(Equal(types.TargetGlobal))
					Expect(p.Description).To(Equal("Isolated cluster plugin"))
					Expect(p.Vendor).To(Equal("othervendor"))
					Expect(p.Publisher).To(Equal("otherpublisher"))
				})
			})
		})
		Describe("With a DB table with one plugin and no recommended version", func() {
			BeforeEach(func() {
				tmpDir, err = os.MkdirTemp(os.TempDir(), "")
				Expect(err).To(BeNil(), "unable to create temporary directory")

				// Create DB file
				dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
				Expect(err).To(BeNil())
				// Open DB with the sqlite driver
				db, err := sql.Open("sqlite3", dbFile.Name())
				Expect(err).To(BeNil(), "failed to open the DB for testing")
				defer db.Close()

				// Create the table
				_, err = db.Exec(CreateTablesSchema)
				Expect(err).To(BeNil(), "failed to create DB table for testing")

				// Add a plugin entry to the DB
				_, err = db.Exec(createPluginTMCNoRecommendedVersionStmt)
				Expect(err).To(BeNil(), "failed to create plugin for testing")

				inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			Context("When getting all plugins", func() {
				It("should return a list of one plugin with no error and recommended version set", func() {
					plugins, err := inventory.GetAllPlugins()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(plugins)).To(Equal(1))

					p := plugins[0]
					Expect(p.Name).To(Equal("management-cluster"))
					Expect(len(p.Artifacts)).To(Equal(2))
					Expect(p.Artifacts["v0.0.1"]).ToNot(BeNil())
					Expect(p.Artifacts["v0.0.2"]).ToNot(BeNil())

					Expect(p.RecommendedVersion).To(Equal("v0.0.2"))
					Expect(string(p.Target)).To(Equal("mission-control"))
					Expect(p.Vendor).To(Equal("vmware"))
					Expect(p.Publisher).To(Equal("tmc"))

					artifactList := p.Artifacts["v0.0.2"]
					Expect(len(artifactList)).To(Equal(1))
					a := artifactList[0]
					Expect(a.OS).To(Equal("linux"))
					Expect(a.Arch).To(Equal("amd64"))
					Expect(a.Digest).To(Equal("1111111111"))
					Expect(a.Image).To(Equal(tmpDir + "/vmware/tmc/linux/amd64/tmc/management-cluster:v0.0.2"))
				})
			})
			Context("When getting a non-existent plugin version for tmc for an os/arch", func() {
				It("should return an empty list of plugins no error", func() {
					plugins, err := inventory.GetPlugins(&PluginInventoryFilter{
						Name:    "management-cluster",
						Target:  "mission-control",
						Version: "v1.2.3",
						OS:      "windows",
						Arch:    "amd64",
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(len(plugins)).To(Equal(0))
				})
			})
		})
	})

	Describe("Getting plugin groups from inventory", func() {
		Context("With an empty DB file", func() {
			BeforeEach(func() {
				tmpDir, err = os.MkdirTemp(os.TempDir(), "")
				Expect(err).To(BeNil(), "unable to create temporary directory")

				// Create empty file for the DB
				dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
				Expect(err).To(BeNil())

				inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an error", func() {
				_, err = inventory.GetAllGroups()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to setup DB"))
			})
		})
		Context("With an empty DB table", func() {
			BeforeEach(func() {
				tmpDir, err = os.MkdirTemp(os.TempDir(), "")
				Expect(err).To(BeNil(), "unable to create temporary directory")

				// Create DB file
				dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
				Expect(err).To(BeNil())
				// Open DB with the sqlite driver
				db, err := sql.Open("sqlite3", dbFile.Name())
				Expect(err).To(BeNil(), "failed to open the DB for testing")
				defer db.Close()

				// Create the table but don't add any rows
				_, err = db.Exec(CreateTablesSchema)
				Expect(err).To(BeNil(), "failed to create DB table for testing")

				inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an empty list of plugin groups with no error", func() {
				groups, err := inventory.GetAllGroups()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(groups)).To(Equal(0))
			})
		})
		Describe("With a DB table with three groups", func() {
			BeforeEach(func() {
				tmpDir, err = os.MkdirTemp(os.TempDir(), "")
				Expect(err).To(BeNil(), "unable to create temporary directory")

				// Create DB file
				dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
				Expect(err).To(BeNil())
				// Open DB with the sqlite driver
				db, err := sql.Open("sqlite3", dbFile.Name())
				Expect(err).To(BeNil(), "failed to open the DB for testing")
				defer db.Close()

				// Create the table
				_, err = db.Exec(CreateTablesSchema)
				Expect(err).To(BeNil(), "failed to create DB table for testing")

				// Add a plugin entry to the DB
				_, err = db.Exec(createGroupsStmt)
				Expect(err).To(BeNil(), "failed to create groups for testing")

				inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			Context("When getting all groups", func() {
				It("should return a list of three groups with no error", func() {
					groups, err := inventory.GetAllGroups()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(groups)).To(Equal(3))

					sort.Sort(pluginGroupSorter(groups))

					i := 0
					Expect(groups[i].Vendor).To(Equal("independent"))
					Expect(groups[i].Publisher).To(Equal("other"))
					Expect(groups[i].Name).To(Equal("mygroup"))

					plugins := groups[i].Plugins
					Expect(len(plugins)).To(Equal(2))
					sort.Sort(pluginGroupPluginEntrySorter(plugins))
					j := 0
					Expect(plugins[j].Name).To(Equal("plugin1"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.1.0"))
					j++
					Expect(plugins[j].Name).To(Equal("plugin2"))
					Expect(plugins[j].Target).To(Equal(types.TargetTMC))
					Expect(plugins[j].Version).To(Equal("v0.2.0"))

					i++
					Expect(groups[i].Vendor).To(Equal("vmware"))
					Expect(groups[i].Publisher).To(Equal("tkg"))
					Expect(groups[i].Name).To(Equal("1.6.0"))
					Expect(len(groups[i].Plugins)).To(Equal(4))

					plugins = groups[i].Plugins
					Expect(len(plugins)).To(Equal(4))
					sort.Sort(pluginGroupPluginEntrySorter(plugins))
					j = 0
					Expect(plugins[j].Name).To(Equal("feature"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.26.0"))
					j++
					Expect(plugins[j].Name).To(Equal("kubernetes-release"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.26.0"))
					j++
					Expect(plugins[j].Name).To(Equal("management-cluster"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.26.0"))
					j++
					Expect(plugins[j].Name).To(Equal("package"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.26.0"))

					i++
					Expect(groups[i].Vendor).To(Equal("vmware"))
					Expect(groups[i].Publisher).To(Equal("tkg"))
					Expect(groups[i].Name).To(Equal("2.1.0"))
					Expect(len(groups[i].Plugins)).To(Equal(5))

					plugins = groups[i].Plugins
					Expect(len(plugins)).To(Equal(5))
					sort.Sort(pluginGroupPluginEntrySorter(plugins))
					j = 0
					Expect(plugins[j].Name).To(Equal("feature"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.28.0"))
					j++
					Expect(plugins[j].Name).To(Equal("isolated-cluster"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.28.0"))
					j++
					Expect(plugins[j].Name).To(Equal("kubernetes-release"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.28.0"))
					j++
					Expect(plugins[j].Name).To(Equal("management-cluster"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.28.0"))
					j++
					Expect(plugins[j].Name).To(Equal("package"))
					Expect(plugins[j].Target).To(Equal(types.TargetK8s))
					Expect(plugins[j].Version).To(Equal("v0.28.0"))
				})
			})
		})
	})

	Describe("Inserting plugins to inventory and verifying it with GetPlugins", func() {
		BeforeEach(func() {
			tmpDir, err = os.MkdirTemp(os.TempDir(), "")
			Expect(err).To(BeNil(), "unable to create temporary directory")

			// Create DB file
			dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
			Expect(err).To(BeNil())

			inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			err = inventory.CreateSchema()
			Expect(err).To(BeNil(), "failed to create DB schema for testing")
		})
		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})
		Context("When inserting plugins", func() {
			It("operation should be successful and getplugins should return the correct result of the plugins with no error", func() {
				err = inventory.InsertPlugin(&piEntry1)
				Expect(err).To(BeNil(), "failed to insert plugin1")
				err = inventory.InsertPlugin(&piEntry2)
				Expect(err).To(BeNil(), "failed to insert plugin2")
				err = inventory.InsertPlugin(&piEntry3)
				Expect(err).To(BeNil(), "failed to insert plugin3")

				// Verify that "management-cluster" plugin with "kubernetes" target can be retrieved and all configuration are correct
				plugins, err := inventory.GetPlugins(&PluginInventoryFilter{Name: "management-cluster", Target: types.TargetK8s})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(plugins)).To(Equal(1))
				p := plugins[0]
				Expect(p.RecommendedVersion).To(Equal("v0.28.0"))
				Expect(string(p.Target)).To(Equal("kubernetes"))
				Expect(p.Description).To(Equal("Kubernetes management cluster operations"))
				Expect(p.Vendor).To(Equal("vmware"))
				Expect(p.Publisher).To(Equal("tkg"))
				Expect(len(p.Artifacts)).To(Equal(1))
				artifactList := p.Artifacts["v0.28.0"]
				Expect(len(artifactList)).To(Equal(3))
				for _, a := range artifactList {
					if a.OS == "linux" {
						Expect(a.Arch).To(Equal("amd64"))
						Expect(a.Digest).To(Equal("0000000000"))
						Expect(a.Image).To(Equal(tmpDir + "/vmware/tkg/linux/amd64/k8s/management-cluster:v0.28.0"))
					} else if a.OS == "darwin" {
						Expect(a.OS).To(Equal("darwin"))
						Expect(a.Arch).To(Equal("amd64"))
						Expect(a.Digest).To(Equal("1111111111"))
						Expect(a.Image).To(Equal(tmpDir + "/vmware/tkg/darwin/amd64/k8s/management-cluster:v0.28.0"))
					} else if a.OS == "windows" {
						Expect(a.OS).To(Equal("windows"))
						Expect(a.Arch).To(Equal("amd64"))
						Expect(a.Digest).To(Equal("2222222222"))
						Expect(a.Image).To(Equal(tmpDir + "/vmware/tkg/windows/amd64/k8s/management-cluster:v0.28.0"))
					}
				}

				// Verify that "isolated-cluster" plugin with "global" target can be retrieved and all configuration are correct
				plugins, err = inventory.GetPlugins(&PluginInventoryFilter{Name: "isolated-cluster", Target: types.TargetGlobal})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(plugins)).To(Equal(1))
				p = plugins[0]
				Expect(p.Name).To(Equal("isolated-cluster"))
				Expect(p.RecommendedVersion).To(Equal("v1.2.3"))
				Expect(p.Target).To(Equal(types.TargetGlobal))
				Expect(p.Description).To(Equal("Isolated cluster plugin"))
				Expect(p.Vendor).To(Equal("othervendor"))
				Expect(p.Publisher).To(Equal("otherpublisher"))
				Expect(len(p.Artifacts)).To(Equal(1))
				a := p.Artifacts["v1.2.3"]
				Expect(len(a)).To(Equal(1))
				Expect(a[0].OS).To(Equal("linux"))
				Expect(a[0].Arch).To(Equal("amd64"))
				Expect(a[0].Digest).To(Equal("3333333333"))
				Expect(a[0].Image).To(Equal(tmpDir + "/othervendor/otherpublisher/linux/amd64/global/isolated-cluster:v1.2.3"))

				// Verify that "management-cluster" plugin with "mission-control" target can be retrieved and all configuration are correct
				plugins, err = inventory.GetPlugins(&PluginInventoryFilter{Name: "management-cluster", Target: types.TargetTMC})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(plugins)).To(Equal(1))
				p = plugins[0]
				Expect(p.Name).To(Equal("management-cluster"))
				Expect(p.RecommendedVersion).To(Equal("v0.0.1"))
				Expect(p.Target).To(Equal(types.TargetTMC))
				Expect(p.Description).To(Equal("Mission-control management cluster operations"))
				Expect(p.Vendor).To(Equal("vmware"))
				Expect(p.Publisher).To(Equal("tmc"))
				Expect(len(p.Artifacts)).To(Equal(1))
				a = p.Artifacts["v0.0.1"]
				Expect(len(a)).To(Equal(1))
				Expect(a[0].OS).To(Equal("linux"))
				Expect(a[0].Arch).To(Equal("amd64"))
				Expect(a[0].Digest).To(Equal("0000000000"))
				Expect(a[0].Image).To(Equal(tmpDir + "/vmware/tmc/linux/amd64/tmc/management-cluster:v0.0.1"))

				// Verify that retrieving any plugin that doesn't exist should return empty array
				plugins, err = inventory.GetPlugins(&PluginInventoryFilter{Name: "unknown", Target: types.TargetTMC})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(plugins)).To(Equal(0))
			})
		})
		Context("When inserting a plugin which already exists in the database", func() {
			BeforeEach(func() {
				err = inventory.InsertPlugin(&piEntry1)
				Expect(err).To(BeNil(), "failed to insert plugin1")
			})
			It("should return an error", func() {
				err = inventory.InsertPlugin(&piEntry1)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("unable to insert plugin row"))
				Expect(err.Error()).To(ContainSubstring("UNIQUE constraint failed"))
			})
		})
	})

	Describe("Inserting plugin-groups to inventory and verifying it with GetAllGroups", func() {
		BeforeEach(func() {
			tmpDir, err = os.MkdirTemp(os.TempDir(), "")
			Expect(err).To(BeNil(), "unable to create temporary directory")

			// Create DB file
			dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
			Expect(err).To(BeNil())

			inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			err = inventory.CreateSchema()
			Expect(err).To(BeNil(), "failed to create DB schema for testing")
			err = inventory.InsertPlugin(&piEntry1)
			Expect(err).To(BeNil(), "failed to insert plugin1")
			err = inventory.InsertPlugin(&piEntry2)
			Expect(err).To(BeNil(), "failed to insert plugin2")
			err = inventory.InsertPlugin(&piEntry3)
			Expect(err).To(BeNil(), "failed to insert plugin3")
		})
		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})
		Context("When inserting plugin-group with plugin that doesn't exists in the database", func() {
			It("should return error", func() {
				pg := &PluginGroup{
					Name:      "v1.0.0",
					Vendor:    "fakevendor",
					Publisher: "fakepublisher",
					Hidden:    false,
					Plugins: []*PluginGroupPluginEntry{
						{
							PluginIdentifier: PluginIdentifier{
								Name:    "fake-plugin",
								Target:  types.TargetGlobal,
								Version: "v1.0.0",
							},
							Mandatory: false,
						},
					},
				}
				err = inventory.InsertPluginGroup(pg, false)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("specified plugin 'name:fake-plugin', 'target:global', 'version:v1.0.0' is not present in the database"))
			})
		})
		Context("When inserting plugin-group with plugin which exists but specified version of the plugin doesn't exists in the database", func() {
			It("should return error", func() {
				pg := &PluginGroup{
					Name:      "v1.0.0",
					Vendor:    "fakevendor",
					Publisher: "fakepublisher",
					Hidden:    false,
					Plugins: []*PluginGroupPluginEntry{
						{
							PluginIdentifier: PluginIdentifier{
								Name:    "mission-control",
								Target:  types.TargetK8s,
								Version: "v1.0.0",
							},
							Mandatory: false,
						},
					},
				}
				err = inventory.InsertPluginGroup(pg, false)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("specified plugin 'name:mission-control', 'target:kubernetes', 'version:v1.0.0' is not present in the database"))
			})
		})
		Context("When inserting plugin-group with all specified plugins and their versions exist in the database", func() {
			It("should not return error and GetAllGroups should return correct result", func() {
				err = inventory.InsertPluginGroup(&pluginGroup1, false)
				Expect(err).To(BeNil())

				groups, err := inventory.GetAllGroups()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(groups)).To(Equal(1))
				Expect(groups[0].Name).To(Equal(pluginGroup1.Name))
				Expect(groups[0].Vendor).To(Equal(pluginGroup1.Vendor))
				Expect(groups[0].Publisher).To(Equal(pluginGroup1.Publisher))
				Expect(groups[0].Hidden).To(Equal(pluginGroup1.Hidden))
				Expect(len(groups[0].Plugins)).To(Equal(1))
				Expect(groups[0].Plugins[0].Name).To(Equal(pluginGroup1.Plugins[0].Name))
				Expect(groups[0].Plugins[0].Target).To(Equal(pluginGroup1.Plugins[0].Target))
				Expect(groups[0].Plugins[0].Version).To(Equal(pluginGroup1.Plugins[0].Version))
				Expect(groups[0].Plugins[0].Mandatory).To(Equal(pluginGroup1.Plugins[0].Mandatory))
			})
		})
		Context("When inserting a plugin-group which already exists in the database", func() {
			BeforeEach(func() {
				err = inventory.InsertPluginGroup(&pluginGroup1, false)
				Expect(err).To(BeNil())
			})
			It("should return an error", func() {
				err = inventory.InsertPluginGroup(&pluginGroup1, false)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("unable to insert plugin-group row"))
				Expect(err.Error()).To(ContainSubstring("UNIQUE constraint failed"))
			})
		})
		Context("When inserting a plugin-group which already exists in the database with override flag", func() {
			BeforeEach(func() {
				err = inventory.InsertPluginGroup(&pluginGroup1, false)
				Expect(err).To(BeNil())
			})
			It("should not return error and GetAllGroups should return the updated result", func() {
				pluginGroupUpdated := pluginGroup1
				pluginGroupUpdated.Hidden = true
				pluginGroupUpdated.Plugins = []*PluginGroupPluginEntry{
					{
						PluginIdentifier: PluginIdentifier{
							Name:    "management-cluster",
							Target:  types.TargetTMC,
							Version: "v0.0.1",
						},
						Mandatory: true,
					},
					{
						PluginIdentifier: PluginIdentifier{
							Name:    "isolated-cluster",
							Target:  types.TargetGlobal,
							Version: "v1.2.3",
						},
						Mandatory: false,
					},
				}

				err = inventory.InsertPluginGroup(&pluginGroupUpdated, true)
				Expect(err).To(BeNil())

				// Verify the result using GetAllGroups
				groups, err := inventory.GetAllGroups()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(groups)).To(Equal(1))
				Expect(groups[0].Name).To(Equal(pluginGroupUpdated.Name))
				Expect(groups[0].Vendor).To(Equal(pluginGroupUpdated.Vendor))
				Expect(groups[0].Publisher).To(Equal(pluginGroupUpdated.Publisher))
				Expect(groups[0].Hidden).To(Equal(pluginGroupUpdated.Hidden))
				Expect(len(groups[0].Plugins)).To(Equal(len(pluginGroupUpdated.Plugins)))
				plugins := groups[0].Plugins
				sort.Sort(pluginGroupPluginEntrySorter(plugins))
				sort.Sort(pluginGroupPluginEntrySorter(pluginGroupUpdated.Plugins))
				Expect(plugins[0].Name).To(Equal(pluginGroupUpdated.Plugins[0].Name))
				Expect(plugins[0].Target).To(Equal(pluginGroupUpdated.Plugins[0].Target))
				Expect(plugins[0].Version).To(Equal(pluginGroupUpdated.Plugins[0].Version))
				Expect(plugins[0].Mandatory).To(Equal(pluginGroupUpdated.Plugins[0].Mandatory))
				Expect(plugins[1].Name).To(Equal(pluginGroupUpdated.Plugins[1].Name))
				Expect(plugins[1].Target).To(Equal(pluginGroupUpdated.Plugins[1].Target))
				Expect(plugins[1].Version).To(Equal(pluginGroupUpdated.Plugins[1].Version))
				Expect(plugins[1].Mandatory).To(Equal(pluginGroupUpdated.Plugins[1].Mandatory))
			})
		})

	})
	Describe("Updating plugin-group activation state", func() {

		BeforeEach(func() {
			tmpDir, err = os.MkdirTemp(os.TempDir(), "")
			Expect(err).To(BeNil(), "unable to create temporary directory")

			// Create DB file
			dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
			Expect(err).To(BeNil())

			inventory = NewSQLiteInventory(dbFile.Name(), tmpDir)
			err = inventory.CreateSchema()
			Expect(err).To(BeNil(), "failed to create DB schema for testing")
			err = inventory.InsertPlugin(&piEntry1)
			Expect(err).To(BeNil(), "failed to insert plugin1")
			err = inventory.InsertPlugin(&piEntry2)
			Expect(err).To(BeNil(), "failed to insert plugin2")
			err = inventory.InsertPlugin(&piEntry3)
			Expect(err).To(BeNil(), "failed to insert plugin3")
		})
		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})

		Context("When updating the activation state of a plugin-group which already exists in the database", func() {
			BeforeEach(func() {
				err = inventory.InsertPluginGroup(&pluginGroup1, false)
				Expect(err).To(BeNil())
			})
			It("should not return error when no change has been done to the activation state and the GetAllGroups should reflect the same", func() {
				err = inventory.UpdatePluginGroupActivationState(&pluginGroup1)
				Expect(err).To(BeNil())

				// Verify the result using GetAllGroups
				groups, err := inventory.GetAllGroups()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(groups)).To(Equal(1))
				Expect(groups[0].Name).To(Equal(pluginGroup1.Name))
				Expect(groups[0].Vendor).To(Equal(pluginGroup1.Vendor))
				Expect(groups[0].Publisher).To(Equal(pluginGroup1.Publisher))
				Expect(groups[0].Hidden).To(Equal(pluginGroup1.Hidden))
				Expect(len(groups[0].Plugins)).To(Equal(len(pluginGroup1.Plugins)))
			})
			It("should not return error when the activation state has been updated and the GetAllGroups should reflect the change", func() {
				pluginGroupUpdated := pluginGroup1
				pluginGroupUpdated.Hidden = true
				err = inventory.UpdatePluginGroupActivationState(&pluginGroupUpdated)
				Expect(err).To(BeNil())

				// Verify the result using GetAllGroups
				groups, err := inventory.GetAllGroups()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(groups)).To(Equal(1))
				Expect(groups[0].Name).To(Equal(pluginGroupUpdated.Name))
				Expect(groups[0].Vendor).To(Equal(pluginGroupUpdated.Vendor))
				Expect(groups[0].Publisher).To(Equal(pluginGroupUpdated.Publisher))
				Expect(groups[0].Hidden).To(Equal(pluginGroupUpdated.Hidden))
				Expect(len(groups[0].Plugins)).To(Equal(len(pluginGroupUpdated.Plugins)))
			})
		})

		Context("When updating the activation state of a plugin-group which does not exist in the database", func() {
			BeforeEach(func() {
				err = inventory.InsertPluginGroup(&pluginGroup1, false)
				Expect(err).To(BeNil())
			})
			It("should return error", func() {
				pluginGroupUpdated := pluginGroup1
				pluginGroupUpdated.Name = "unknown"
				err = inventory.UpdatePluginGroupActivationState(&pluginGroupUpdated)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("unable to update plugin-group 'fakevendor-fakepublisher/unknown'. This might be possible because the provided plugin-group doesn't exists"))
			})
		})
	})
})

type pluginGroupSorter []*PluginGroup

func (g pluginGroupSorter) Len() int      { return len(g) }
func (g pluginGroupSorter) Swap(i, j int) { g[i], g[j] = g[j], g[i] }
func (g pluginGroupSorter) Less(i, j int) bool {
	if g[i].Vendor != g[j].Vendor {
		return g[i].Vendor < g[j].Vendor
	}
	if g[i].Publisher != g[j].Publisher {
		return g[i].Publisher < g[j].Publisher
	}
	return g[i].Name < g[j].Name
}

type pluginGroupPluginEntrySorter []*PluginGroupPluginEntry

func (g pluginGroupPluginEntrySorter) Len() int      { return len(g) }
func (g pluginGroupPluginEntrySorter) Swap(i, j int) { g[i], g[j] = g[j], g[i] }
func (g pluginGroupPluginEntrySorter) Less(i, j int) bool {
	if g[i].Target != g[j].Target {
		return g[i].Target < g[j].Target
	}
	return g[i].Name < g[j].Name
}
