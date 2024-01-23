// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

var _ = Describe("Unit tests for inventory plugin-group add", func() {
	pluginGroupManifestFile, err := createTestPluginGroupManifestFile()
	Expect(err).ToNot(HaveOccurred())

	var referencedDBFile string
	var ipgu InventoryPluginGroupUpdateOptions
	fakeImgpkgWrapper := &fakes.ImageOperationsImpl{}

	// pullDBImageStub create new empty database with the table schemas created
	//nolint:unparam
	pullDBImageStub := func(_, path string) error {
		dbFile := filepath.Join(path, plugininventory.SQliteDBFileName)
		db := plugininventory.NewSQLiteInventory(dbFile, "")
		err := db.CreateSchema()
		Expect(err).ToNot(HaveOccurred())
		referencedDBFile = dbFile
		return nil
	}

	// pullDBImageStubWithPlugins create new database with the table schemas and foo plugin
	pullDBImageStubWithPlugins := func(image, path string) error {
		err := pullDBImageStub(image, path)
		if err != nil {
			return err
		}
		db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
		artifactsFoo := make(map[string]distribution.ArtifactList)
		artifactsFoo["v0.0.2"] = []distribution.Artifact{
			{
				OS:     "darwin",
				Arch:   "amd64",
				Digest: "fake-digest",
				Image:  "fake-uri",
			},
			{
				OS:     "linux",
				Arch:   "amd64",
				Digest: "fake-digest",
				Image:  "fake-uri",
			},
		}
		artifactsBar := make(map[string]distribution.ArtifactList)
		artifactsBar["v0.0.3"] = []distribution.Artifact{
			{
				OS:     "darwin",
				Arch:   "amd64",
				Digest: "fake-digest",
				Image:  "fake-uri",
			},
			{
				OS:     "linux",
				Arch:   "amd64",
				Digest: "fake-digest",
				Image:  "fake-uri",
			},
		}
		entryFoo := &plugininventory.PluginInventoryEntry{
			Name:        "foo",
			Target:      "global",
			Description: "Foo plugin",
			Publisher:   "fakepublisher",
			Vendor:      "fakevendor",
			Hidden:      false,
			Artifacts:   artifactsFoo,
		}
		entryBar := &plugininventory.PluginInventoryEntry{
			Name:        "bar",
			Target:      "mission-control",
			Description: "Bar plugin",
			Publisher:   "fakepublisher",
			Vendor:      "fakevendor",
			Hidden:      false,
			Artifacts:   artifactsBar,
		}
		err = db.InsertPlugin(entryFoo)
		Expect(err).ToNot(HaveOccurred())
		err = db.InsertPlugin(entryBar)
		Expect(err).ToNot(HaveOccurred())
		return nil
	}

	// pullDBImageStubWithPluginGroups create new database with the table schemas and plugin groups
	pullDBImageStubWithPluginGroups := func(image, path string) error {
		err := pullDBImageStubWithPlugins(image, path)
		if err != nil {
			return err
		}
		db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
		pgEntry := plugininventory.PluginGroup{
			Vendor:      "fakevendor",
			Publisher:   "fakepublisher",
			Name:        "default",
			Description: "Desc for plugin",
			Hidden:      false,
			Versions: map[string][]*plugininventory.PluginGroupPluginEntry{
				"v1.0.0": {
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "foo", Target: "global", Version: "v0.0.2"},
						Mandatory:        false,
					},
				},
				"v2.0.0": {
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "foo", Target: "global", Version: "v0.0.2"},
						Mandatory:        false,
					},
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "bar", Target: "mission-control", Version: "v0.0.3"},
						Mandatory:        false,
					},
				},
			},
		}
		err = db.InsertPluginGroup(&pgEntry, false)
		Expect(err).ToNot(HaveOccurred())
		return nil
	}

	var _ = Context("tests for the inventory plugin-group add function", func() {

		BeforeEach(func() {
			ipgu = InventoryPluginGroupUpdateOptions{
				Repository:              "test-repo.com",
				InventoryImageTag:       "latest",
				ImageOperationsImpl:     fakeImgpkgWrapper,
				Vendor:                  "fakevendor",
				Publisher:               "fakepublisher",
				PluginGroupManifestFile: pluginGroupManifestFile,
				GroupName:               "default",
				GroupVersion:            "v1.0.0",
				Description:             "Desc for plugin",
				DeactivatePluginGroup:   false,
				Override:                false,
			}
		})

		var _ = It("when plugin inventory database cannot be pulled from the repository", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(errors.New("image not found"))
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirReturns(errors.New("unable to pull inventory database"))

			err := ipgu.PluginGroupAdd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while pulling database from the image"))
			Expect(err.Error()).To(ContainSubstring("unable to pull inventory database"))
		})

		var _ = It("when specified manifest file doesn't exists, adding plugin group should throw error", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(errors.New("image not found"))
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStub)

			ipgu.PluginGroupManifestFile = "does-not-exists.yaml"
			err := ipgu.PluginGroupAdd()
			Expect(referencedDBFile).NotTo(BeEmpty())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while reading plugin group"))
		})

		var _ = It("when specified plugins in the plugin-group doesn't exist in the inventory database, adding plugin group should throw error", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStub)

			err := ipgu.PluginGroupAdd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while inserting plugin group"))
			Expect(err.Error()).To(ContainSubstring("specified plugin"))
			Expect(err.Error()).To(ContainSubstring("not present in the database"))
		})

		var _ = It("when specified plugins exists and the plugin-group doesn't exist in the inventory database, adding plugin group should be successful", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStubWithPlugins)

			err := ipgu.PluginGroupAdd()
			Expect(err).NotTo(HaveOccurred())

			// verify that the local db file was updated correctly before publishing the database to remote repository
			db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
			pgEntries, err := db.GetPluginGroups(plugininventory.PluginGroupFilter{})
			Expect(err).NotTo(HaveOccurred())
			Expect(pgEntries).NotTo(BeNil())
			Expect(len(pgEntries)).To(Equal(1))
			Expect(pgEntries[0].Name).To(Equal("default"))
			Expect(pgEntries[0].Publisher).To(Equal("fakepublisher"))
			Expect(pgEntries[0].Vendor).To(Equal("fakevendor"))
			Expect(pgEntries[0].Description).To(Equal("Desc for plugin"))
			Expect(pgEntries[0].Hidden).To(Equal(ipgu.DeactivatePluginGroup))
			Expect(len(pgEntries[0].Versions)).To(Equal(1))

			plugins := pgEntries[0].Versions["v1.0.0"]
			Expect(len(plugins)).To(Equal(2))
		})

		var _ = It("when specified plugin-group already exist in the inventory database and override is not provided, adding plugin group should throw error", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStubWithPluginGroups)

			err := ipgu.PluginGroupAdd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while inserting plugin group"))
			Expect(err.Error()).To(ContainSubstring("unable to insert plugin-group"))
			Expect(err.Error()).To(ContainSubstring("NIQUE constraint failed"))
		})

		var _ = It("when specified plugin-group already exist in the inventory database and override is provided, adding plugin group should be successful", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStubWithPluginGroups)

			ipgu.Override = true
			ipgu.DeactivatePluginGroup = false
			err := ipgu.PluginGroupAdd()
			Expect(err).NotTo(HaveOccurred())

			// verify that the local db file was updated correctly before publishing the database to remote repository
			db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
			pgEntries, err := db.GetPluginGroups(plugininventory.PluginGroupFilter{})
			Expect(err).NotTo(HaveOccurred())
			Expect(pgEntries).NotTo(BeNil())
			Expect(len(pgEntries)).To(Equal(1))
			Expect(pgEntries[0].Name).To(Equal("default"))
			Expect(pgEntries[0].Publisher).To(Equal("fakepublisher"))
			Expect(pgEntries[0].Vendor).To(Equal("fakevendor"))
			Expect(pgEntries[0].Description).To(Equal("Desc for plugin"))
			Expect(pgEntries[0].Hidden).To(Equal(ipgu.DeactivatePluginGroup))
			Expect(len(pgEntries[0].Versions)).To(Equal(2))

			plugins := pgEntries[0].Versions["v1.0.0"]
			Expect(len(plugins)).To(Equal(2))
			plugins = pgEntries[0].Versions["v2.0.0"]
			Expect(len(plugins)).To(Equal(2))
		})

		var _ = It("when inventory database cannot be published from the repository", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(errors.New("unable to publish image"))
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStubWithPlugins)

			err := ipgu.PluginGroupAdd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while publishing inventory database to the repository as image"))
			Expect(err.Error()).To(ContainSubstring("unable to publish image"))
		})
	})

	var _ = Context("tests for the inventory plugin-group UpdatePluginGroupActivationState function", func() {

		BeforeEach(func() {
			ipgu = InventoryPluginGroupUpdateOptions{
				Repository:            "test-repo.com",
				InventoryImageTag:     "latest",
				ImageOperationsImpl:   fakeImgpkgWrapper,
				Vendor:                "fakevendor",
				Publisher:             "fakepublisher",
				GroupName:             "default",
				GroupVersion:          "v1.0.0",
				Description:           "Desc for plugin",
				DeactivatePluginGroup: false,
			}
		})

		var _ = It("when plugin inventory database cannot be pulled from the repository", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(errors.New("image not found"))
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirReturns(errors.New("unable to pull inventory database"))

			err := ipgu.UpdatePluginGroupActivationState()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while pulling database from the image"))
			Expect(err.Error()).To(ContainSubstring("unable to pull inventory database"))
		})

		var _ = It("when specified plugin-group doesn't exist in the inventory database, updating the activation state should throw error", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStub)

			err := ipgu.UpdatePluginGroupActivationState()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while updating activation state of plugin group"))
			Expect(err.Error()).To(ContainSubstring("unable to update plugin-group 'fakevendor-fakepublisher/default:v1.0.0'. This might be possible because the provided plugin-group version doesn't exists"))
		})

		var _ = It("when specified plugin-group exists in the inventory database, updating the activation state with 'DeactivatePluginGroup=true' should be successful", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStubWithPluginGroups)

			ipgu.DeactivatePluginGroup = true
			err := ipgu.UpdatePluginGroupActivationState()
			Expect(err).NotTo(HaveOccurred())

			// verify that the local db file was updated correctly before publishing the database to remote repository
			db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
			pgEntries, err := db.GetPluginGroups(plugininventory.PluginGroupFilter{IncludeHidden: false})
			Expect(err).NotTo(HaveOccurred())
			Expect(pgEntries).NotTo(BeNil())
			Expect(len(pgEntries)).To(Equal(1))
			Expect(pgEntries[0].Name).To(Equal("default"))
			Expect(pgEntries[0].Publisher).To(Equal("fakepublisher"))
			Expect(pgEntries[0].Vendor).To(Equal("fakevendor"))
			Expect(pgEntries[0].Description).To(Equal("Desc for plugin"))
			Expect(pgEntries[0].Hidden).To(Equal(false))
			Expect(len(pgEntries[0].Versions)).To(Equal(1))

			plugins := pgEntries[0].Versions["v2.0.0"]
			Expect(len(plugins)).To(Equal(2))
		})

		var _ = It("when specified plugin-group exists in the inventory database, updating the activation state with 'DeactivatePluginGroup=false' should be successful", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStubWithPluginGroups)

			ipgu.DeactivatePluginGroup = false
			err := ipgu.UpdatePluginGroupActivationState()
			Expect(err).NotTo(HaveOccurred())

			// verify that the local db file was updated correctly before publishing the database to remote repository
			db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
			pgEntries, err := db.GetPluginGroups(plugininventory.PluginGroupFilter{})
			Expect(err).NotTo(HaveOccurred())
			Expect(pgEntries).NotTo(BeNil())
			Expect(len(pgEntries)).To(Equal(1))
			Expect(pgEntries[0].Name).To(Equal("default"))
			Expect(pgEntries[0].Publisher).To(Equal("fakepublisher"))
			Expect(pgEntries[0].Vendor).To(Equal("fakevendor"))
			Expect(pgEntries[0].Description).To(Equal("Desc for plugin"))
			Expect(pgEntries[0].Hidden).To(Equal(ipgu.DeactivatePluginGroup))
			Expect(len(pgEntries[0].Versions)).To(Equal(2))

			plugins := pgEntries[0].Versions["v1.0.0"]
			Expect(len(plugins)).To(Equal(1))
			plugins = pgEntries[0].Versions["v2.0.0"]
			Expect(len(plugins)).To(Equal(2))
		})

		var _ = It("when inventory database cannot be published from the repository", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(errors.New("unable to publish image"))
			fakeImgpkgWrapper.DownloadImageAndSaveFilesToDirCalls(pullDBImageStubWithPluginGroups)

			err := ipgu.UpdatePluginGroupActivationState()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while publishing inventory database to the repository as image"))
			Expect(err.Error()).To(ContainSubstring("unable to publish image"))
		})
	})
})

func createTestPluginGroupManifestFile() (string, error) {
	manifestBytes := `created: 2023-02-24T10:10:59.093382-08:00
plugins:
    - name: foo
      target: global
      scope: Standalone
      version: v0.0.2
    - name: bar
      target: mission-control
      scope: Context
      version: v0.0.3
`
	tempManifestFile := filepath.Join(os.TempDir(), "plugin_group_manifets.yaml")
	return filepath.Join(os.TempDir(), "plugin_group_manifets.yaml"), utils.SaveFile(tempManifestFile, []byte(manifestBytes))
}
