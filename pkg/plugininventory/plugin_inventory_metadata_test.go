// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugininventory

import (
	"os"
	"path/filepath"

	// Import the sqlite driver
	_ "modernc.org/sqlite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var pluginIdentifier1 = PluginIdentifier{
	Name:    "plugin1",
	Target:  "global",
	Version: "v1.0.0",
}
var pluginIdentifier2 = PluginIdentifier{
	Name:    "plugin2",
	Target:  "kubernetes",
	Version: "v2.0.0",
}
var pluginGroupIdentifier1 = PluginGroupIdentifier{
	Vendor:    "fakevendor",
	Publisher: "fakepublisher",
	Name:      "fake1:v1.0.0",
}
var pluginGroupIdentifier2 = PluginGroupIdentifier{
	Vendor:    "fakevendor",
	Publisher: "fakepublisher",
	Name:      "fake2:v2.0.0",
}

var pluginEntry1 = PluginInventoryEntry{
	Name:               "plugin1",
	Target:             types.TargetGlobal,
	Description:        "plugin1 description",
	Publisher:          "fakepublisher",
	Vendor:             "fakevendor",
	RecommendedVersion: "v1.0.0",
	Hidden:             false,
	Artifacts: distribution.Artifacts{
		"v1.0.0": []distribution.Artifact{
			{
				OS:     "linux",
				Arch:   "amd64",
				Digest: "0000000000",
				Image:  "vmware/tkg/linux/amd64/global/plugin1:v1.0.0",
			},
		},
	},
}
var pluginEntry2 = PluginInventoryEntry{
	Name:               "plugin2",
	Target:             types.TargetK8s,
	Description:        "plugin2 description",
	Publisher:          "otherpublisher",
	Vendor:             "othervendor",
	RecommendedVersion: "v1.2.3",
	Hidden:             false,
	Artifacts: distribution.Artifacts{
		"v2.0.0": []distribution.Artifact{
			{
				OS:     "linux",
				Arch:   "amd64",
				Digest: "3333333333",
				Image:  "othervendor/otherpublisher/linux/amd64/k8s/plugin2:v2.0.0",
			},
		},
	},
}
var pluginEntry3 = PluginInventoryEntry{
	Name:               "plugin3",
	Target:             types.TargetTMC,
	Description:        "plugin3 description",
	Publisher:          "otherpublisher",
	Vendor:             "othervendor",
	RecommendedVersion: "v3.0.0",
	Hidden:             false,
	Artifacts: distribution.Artifacts{
		"v3.0.0": []distribution.Artifact{
			{
				OS:     "linux",
				Arch:   "amd64",
				Digest: "0000000000",
				Image:  "vmware/tmc/linux/amd64/tmc/plugin3:v3.0.0",
			},
		},
	},
}

var pluginGroupEntry1 = PluginGroup{
	Name:      "fake1:v1.0.0",
	Vendor:    "fakevendor",
	Publisher: "fakepublisher",
	Hidden:    false,
	Plugins: []*PluginGroupPluginEntry{
		{
			PluginIdentifier: PluginIdentifier{Name: "plugin1", Target: types.TargetGlobal, Version: "v1.0.0"},
			Mandatory:        false,
		},
	},
}
var pluginGroupEntry2 = PluginGroup{
	Name:      "fake2:v2.0.0",
	Vendor:    "fakevendor",
	Publisher: "fakepublisher",
	Hidden:    false,
	Plugins: []*PluginGroupPluginEntry{
		{
			PluginIdentifier: PluginIdentifier{Name: "plugin2", Target: types.TargetK8s, Version: "v2.0.0"},
			Mandatory:        false,
		},
	},
}

var _ = Describe("Unit tests for plugin inventory metadata", func() {
	var (
		err                                 error
		metadataInventory                   PluginInventoryMetadata
		additionalMetadataInventory         PluginInventoryMetadata
		pluginInventory                     PluginInventory
		pluginInventoryFilePath             string
		additionalMetadataInventoryFilePath string
		tmpDir                              string
	)

	createInventoryMetadataDB := func(createSchema bool) (PluginInventoryMetadata, string) {
		tmpDir, err = os.MkdirTemp(os.TempDir(), "")
		Expect(err).To(BeNil(), "unable to create temporary directory")
		// Create empty file for the DB
		dbFile, err := os.Create(filepath.Join(tmpDir, SQliteInventoryMetadataDBFileName))
		Expect(err).To(BeNil())
		mi := NewSQLiteInventoryMetadata(dbFile.Name())
		if createSchema {
			err = mi.CreateInventoryMetadataDBSchema()
			Expect(err).To(BeNil())
		}
		return mi, dbFile.Name()
	}

	createInventoryDB := func(createSchema bool) (PluginInventory, string) {
		tmpDir, err = os.MkdirTemp(os.TempDir(), "")
		Expect(err).To(BeNil(), "unable to create temporary directory")

		// Create empty file for the DB
		dbFile, err := os.Create(filepath.Join(tmpDir, SQliteDBFileName))
		Expect(err).To(BeNil())

		inventory := NewSQLiteInventory(dbFile.Name(), tmpDir)
		if createSchema {
			err = inventory.CreateSchema()
			Expect(err).To(BeNil(), "failed to create DB table for testing")
		}
		return inventory, dbFile.Name()
	}

	Describe("Insert plugin identifier", func() {
		Context("With an empty DB file", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(false)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an error", func() {
				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to insert plugin identifier"))
				Expect(err.Error()).To(ContainSubstring("no such table: AvailablePluginBinaries"))
			})
		})

		Context("With an empty DB tables", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should not return an error", func() {
				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier1)
				Expect(err).NotTo(HaveOccurred())

				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier2)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When same plugin indentifier entry already exists", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)

				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier1)
				Expect(err).NotTo(HaveOccurred())

			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an error", func() {
				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to insert plugin identifier"))
				Expect(err.Error()).To(ContainSubstring("UNIQUE constraint failed"))
			})
		})
	})

	Describe("Insert plugin group identifier", func() {
		Context("With an empty DB file", func() {
			BeforeEach(func() {
				tmpDir, err = os.MkdirTemp(os.TempDir(), "")
				Expect(err).To(BeNil(), "unable to create temporary directory")

				// Create empty file for the DB
				dbFile, err := os.Create(filepath.Join(tmpDir, SQliteInventoryMetadataDBFileName))
				Expect(err).To(BeNil())

				metadataInventory = NewSQLiteInventoryMetadata(dbFile.Name())
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an error", func() {
				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to insert plugin group identifier"))
				Expect(err.Error()).To(ContainSubstring("no such table: AvailablePluginGroups"))
			})
		})

		Context("With an empty DB tables", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should not return an error", func() {
				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier1)
				Expect(err).NotTo(HaveOccurred())

				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier2)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When same plugin group indentifier entry already exists", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)

				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier1)
				Expect(err).NotTo(HaveOccurred())

			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an error", func() {
				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to insert plugin group identifier"))
				Expect(err.Error()).To(ContainSubstring("UNIQUE constraint failed"))
			})
		})
	})

	Describe("Merge Inventory Metadata Database", func() {
		Context("when one of the database does not have tables created", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)
				_, additionalMetadataInventoryFilePath = createInventoryMetadataDB(false)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an error", func() {
				err = metadataInventory.MergeInventoryMetadataDatabase(additionalMetadataInventoryFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to execute the query"))
			})
		})

		Context("when both the databases have empty tables for plugins and plugin groups", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)
				_, additionalMetadataInventoryFilePath = createInventoryMetadataDB(true)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should not return an error", func() {
				err = metadataInventory.MergeInventoryMetadataDatabase(additionalMetadataInventoryFilePath)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when both inventory metadata databases does not have any overlap of plugins and plugin groups", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)
				additionalMetadataInventory, additionalMetadataInventoryFilePath = createInventoryMetadataDB(true)

				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier1)
				Expect(err).NotTo(HaveOccurred())
				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier1)
				Expect(err).NotTo(HaveOccurred())

				err = additionalMetadataInventory.InsertPluginIdentifier(&pluginIdentifier2)
				Expect(err).NotTo(HaveOccurred())
				err = additionalMetadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier2)
				Expect(err).NotTo(HaveOccurred())

			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should not return an error", func() {
				err = metadataInventory.MergeInventoryMetadataDatabase(additionalMetadataInventoryFilePath)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when both inventory metadata databases does have some overlap of plugins and plugin groups", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)
				additionalMetadataInventory, additionalMetadataInventoryFilePath = createInventoryMetadataDB(true)

				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier1)
				Expect(err).NotTo(HaveOccurred())
				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier2)
				Expect(err).NotTo(HaveOccurred())
				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier1)
				Expect(err).NotTo(HaveOccurred())
				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier2)
				Expect(err).NotTo(HaveOccurred())

				err = additionalMetadataInventory.InsertPluginIdentifier(&pluginIdentifier2)
				Expect(err).NotTo(HaveOccurred())
				err = additionalMetadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier2)
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should not return an error", func() {
				err = metadataInventory.MergeInventoryMetadataDatabase(additionalMetadataInventoryFilePath)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Update Plugin Inventory Database based on Metadata Database", func() {
		Context("when plugin inventory database provided is invalid and does not have tables created", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)
				_, pluginInventoryFilePath = createInventoryDB(false)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an error", func() {
				err = metadataInventory.UpdatePluginInventoryDatabase(pluginInventoryFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error while updating plugin inventory database"))
			})
		})

		Context("when plugin inventory metadata database provided is invalid and does not have tables created", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(false)
				_, pluginInventoryFilePath = createInventoryDB(true)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should return an error", func() {
				err = metadataInventory.UpdatePluginInventoryDatabase(pluginInventoryFilePath)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error while updating plugin inventory database"))
			})
		})

		Context("when both the databases have empty tables and no plugins or plugin groups", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)
				_, pluginInventoryFilePath = createInventoryDB(true)
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("should not return an error", func() {
				err = metadataInventory.UpdatePluginInventoryDatabase(pluginInventoryFilePath)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when plugin inventory database and metadata database are valid", func() {
			BeforeEach(func() {
				metadataInventory, _ = createInventoryMetadataDB(true)
				pluginInventory, pluginInventoryFilePath = createInventoryDB(true)

				err = pluginInventory.InsertPlugin(&pluginEntry1)
				Expect(err).NotTo(HaveOccurred())
				err = pluginInventory.InsertPlugin(&pluginEntry2)
				Expect(err).NotTo(HaveOccurred())
				err = pluginInventory.InsertPlugin(&pluginEntry3)
				Expect(err).NotTo(HaveOccurred())
				err = pluginInventory.InsertPluginGroup(&pluginGroupEntry1, true)
				Expect(err).NotTo(HaveOccurred())
				err = pluginInventory.InsertPluginGroup(&pluginGroupEntry2, true)
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})
			It("when metadata database tables are empty, it should not return an error and remove all plugin and plugin groups from inventory database", func() {
				err = metadataInventory.UpdatePluginInventoryDatabase(pluginInventoryFilePath)
				Expect(err).NotTo(HaveOccurred())

				pluginEntries, err := pluginInventory.GetAllPlugins()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(pluginEntries)).To(Equal(0))

				pluginGroupEntries, err := pluginInventory.GetPluginGroups(PluginGroupFilter{})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(pluginGroupEntries)).To(Equal(0))
			})

			It("when metadata database tables has overlapping entries, it should not return an error and remove all plugin and plugin groups from inventory database that are not present in metadata database - 1", func() {
				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier1)
				Expect(err).NotTo(HaveOccurred())
				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier1)
				Expect(err).NotTo(HaveOccurred())

				err = metadataInventory.UpdatePluginInventoryDatabase(pluginInventoryFilePath)
				Expect(err).NotTo(HaveOccurred())

				pluginEntries, err := pluginInventory.GetAllPlugins()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(pluginEntries)).To(Equal(1))

				pluginGroupEntries, err := pluginInventory.GetPluginGroups(PluginGroupFilter{})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(pluginGroupEntries)).To(Equal(1))
			})

			It("when metadata database tables has overlapping entries, it should not return an error and remove all plugin and plugin groups from inventory database that are not present in metadata database - 2", func() {
				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier1)
				Expect(err).NotTo(HaveOccurred())
				err = metadataInventory.InsertPluginIdentifier(&pluginIdentifier2)
				Expect(err).NotTo(HaveOccurred())
				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier1)
				Expect(err).NotTo(HaveOccurred())
				err = metadataInventory.InsertPluginGroupIdentifier(&pluginGroupIdentifier2)
				Expect(err).NotTo(HaveOccurred())

				err = metadataInventory.UpdatePluginInventoryDatabase(pluginInventoryFilePath)
				Expect(err).NotTo(HaveOccurred())

				pluginEntries, err := pluginInventory.GetAllPlugins()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(pluginEntries)).To(Equal(2))

				pluginGroupEntries, err := pluginInventory.GetPluginGroups(PluginGroupFilter{})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(pluginGroupEntries)).To(Equal(2))
			})
		})
	})
})
