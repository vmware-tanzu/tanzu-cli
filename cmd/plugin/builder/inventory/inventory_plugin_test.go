// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/fakes"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

var _ = Describe("Unit tests for inventory plugin insert", func() {
	manifestFile, err := createTestManifestFile()
	Expect(err).ToNot(HaveOccurred())

	var referencedDBFile string

	fakeImgpkgWrapper := &fakes.ImgpkgWrapper{}
	iip := InventoryPluginUpdateOptions{
		Repository:        "test-repo.com",
		InventoryImageTag: "latest",
		ImgpkgOptions:     fakeImgpkgWrapper,
		Vendor:            "fakevendor",
		Publisher:         "fakepublisher",
		ManifestFile:      manifestFile,
	}

	// pullDBImageStub create new empty database with the table schemas created
	pullDBImageStub := func(image, path string) error {
		dbFile := filepath.Join(path, plugininventory.SQliteDBFileName)
		db := plugininventory.NewSQLiteInventory(dbFile, "")
		err := db.CreateSchema()
		Expect(err).ToNot(HaveOccurred())
		referencedDBFile = dbFile
		return nil
	}

	// pullDBImageStubWithPlugins create new database with the table schemas and foo plugin
	pullDBImageStubWithPlugins := func(image, path string) error {
		dbFile := filepath.Join(path, plugininventory.SQliteDBFileName)
		db := plugininventory.NewSQLiteInventory(dbFile, "")
		err := db.CreateSchema()
		Expect(err).ToNot(HaveOccurred())
		artifacts := make(map[string]distribution.ArtifactList)
		artifacts["v0.0.2"] = []distribution.Artifact{
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
		entry := &plugininventory.PluginInventoryEntry{
			Name:        "foo",
			Target:      "global",
			Description: "Foo plugin",
			Publisher:   "fakepublisher",
			Vendor:      "fakevendor",
			Hidden:      false,
			Artifacts:   artifacts,
		}
		err = db.InsertPlugin(entry)
		Expect(err).ToNot(HaveOccurred())
		referencedDBFile = dbFile
		return nil
	}

	var _ = Context("tests for the inventory plugin insert function", func() {

		var _ = It("when plugin inventory database cannot be pulled from the repository", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(errors.New("image not found"))
			fakeImgpkgWrapper.PullImageReturns(nil)
			fakeImgpkgWrapper.PullImageReturnsOnCall(0, errors.New("unable to pull inventory database"))

			err := iip.PluginInsert()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while pulling database from the image"))
			Expect(err.Error()).To(ContainSubstring("unable to pull inventory database"))
		})

		var _ = It("when plugin inventory database can be pulled from the repository but calculating plugin binary digest fails", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(errors.New("image not found"))
			fakeImgpkgWrapper.PullImageCalls(pullDBImageStub)
			fakeImgpkgWrapper.GetFileDigestFromImageReturns("", errors.New("error while getting digest"))

			err := iip.PluginInsert()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while getting digest"))
			Expect(err.Error()).To(ContainSubstring("error while getting plugin binary digest"))
		})

		var _ = It("when plugin inventory database can be pulled, plugin binary digest can be calculated by publishing image fails", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(errors.New("image not found"))
			fakeImgpkgWrapper.PullImageCalls(pullDBImageStub)
			fakeImgpkgWrapper.GetFileDigestFromImageReturns("fake-digest", nil)

			err := iip.PluginInsert()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("image not found"))
			Expect(err.Error()).To(ContainSubstring("error while publishing inventory database to the repository"))
		})

		var _ = It("when all configuration are correct and inserting plugin with DeactivatePlugins=false", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.PullImageCalls(pullDBImageStub)
			fakeImgpkgWrapper.GetFileDigestFromImageReturns("fake-digest", nil)

			iip.DeactivatePlugins = false
			err := iip.PluginInsert()
			Expect(err).NotTo(HaveOccurred())

			// verify that the local db file was updated before publishing the database to remote repository
			db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
			pluginInventoryEntries, err := db.GetAllPlugins()
			Expect(err).NotTo(HaveOccurred())
			Expect(pluginInventoryEntries).NotTo(BeNil())
			Expect(len(pluginInventoryEntries)).To(Equal(1))
			Expect(pluginInventoryEntries[0].Name).To(Equal("foo"))
			Expect(pluginInventoryEntries[0].Target).To(Equal(types.TargetGlobal))
			Expect(pluginInventoryEntries[0].Description).To(Equal("Foo plugin"))
			Expect(pluginInventoryEntries[0].Hidden).To(Equal(false))
			Expect(pluginInventoryEntries[0].Publisher).To(Equal("fakepublisher"))
			Expect(pluginInventoryEntries[0].Vendor).To(Equal("fakevendor"))
			Expect(pluginInventoryEntries[0].Artifacts["v0.0.2"]).NotTo(BeNil())
		})

		var _ = It("when all configuration are correct and inserting plugin with DeactivatePlugins=true", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.PullImageCalls(pullDBImageStub)
			fakeImgpkgWrapper.GetFileDigestFromImageReturns("fake-digest", nil)

			iip.DeactivatePlugins = true
			err := iip.PluginInsert()
			Expect(err).NotTo(HaveOccurred())

			// verify that the local db file was updated before publishing the database to remote repository
			db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
			pluginInventoryEntries, err := db.GetAllPlugins()
			Expect(err).NotTo(HaveOccurred())
			Expect(pluginInventoryEntries).NotTo(BeNil())
			Expect(len(pluginInventoryEntries)).To(Equal(1))
			Expect(pluginInventoryEntries[0].Name).To(Equal("foo"))
			Expect(pluginInventoryEntries[0].Target).To(Equal(types.TargetGlobal))
			Expect(pluginInventoryEntries[0].Description).To(Equal("Foo plugin"))
			Expect(pluginInventoryEntries[0].Hidden).To(Equal(true))
			Expect(pluginInventoryEntries[0].Artifacts["v0.0.2"]).NotTo(BeNil())
		})
	})

	var _ = Context("tests for the inventory plugin UpdatePluginActivationState function", func() {

		var _ = It("when specified pluginInventoryEntry doesn't exist in database", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.PullImageCalls(pullDBImageStub)
			fakeImgpkgWrapper.GetFileDigestFromImageReturns("fake-digest", nil)

			iip.DeactivatePlugins = false
			err := iip.UpdatePluginActivationState()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while updating plugin"))
		})

		var _ = It("when all configuration are correct and DeactivatePlugins=false", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.PullImageCalls(pullDBImageStubWithPlugins)
			fakeImgpkgWrapper.GetFileDigestFromImageReturns("fake-digest", nil)

			iip.DeactivatePlugins = false
			err := iip.UpdatePluginActivationState()
			Expect(err).NotTo(HaveOccurred())

			// verify that the local db file was updated before publishing the database to remote repository
			db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
			pluginInventoryEntries, err := db.GetAllPlugins()
			Expect(err).NotTo(HaveOccurred())
			Expect(pluginInventoryEntries).NotTo(BeNil())
			Expect(len(pluginInventoryEntries)).To(Equal(1))
			Expect(pluginInventoryEntries[0].Name).To(Equal("foo"))
			Expect(pluginInventoryEntries[0].Target).To(Equal(types.TargetGlobal))
			Expect(pluginInventoryEntries[0].Description).To(Equal("Foo plugin"))
			Expect(pluginInventoryEntries[0].Hidden).To(Equal(false))
			Expect(pluginInventoryEntries[0].Artifacts["v0.0.2"]).NotTo(BeNil())
		})

		var _ = It("when all configuration are correct and inserting plugin with DeactivatePlugins=true", func() {
			fakeImgpkgWrapper.ResolveImageReturns(nil)
			fakeImgpkgWrapper.PushImageReturns(nil)
			fakeImgpkgWrapper.PullImageCalls(pullDBImageStubWithPlugins)
			fakeImgpkgWrapper.GetFileDigestFromImageReturns("fake-digest", nil)

			iip.DeactivatePlugins = true
			err := iip.UpdatePluginActivationState()
			Expect(err).NotTo(HaveOccurred())

			// verify that the local db file was updated before publishing the database to remote repository
			db := plugininventory.NewSQLiteInventory(referencedDBFile, "")
			pluginInventoryEntries, err := db.GetAllPlugins()
			Expect(err).NotTo(HaveOccurred())
			Expect(pluginInventoryEntries).NotTo(BeNil())
			Expect(len(pluginInventoryEntries)).To(Equal(1))
			Expect(pluginInventoryEntries[0].Name).To(Equal("foo"))
			Expect(pluginInventoryEntries[0].Target).To(Equal(types.TargetGlobal))
			Expect(pluginInventoryEntries[0].Description).To(Equal("Foo plugin"))
			Expect(pluginInventoryEntries[0].Hidden).To(Equal(true))
			Expect(pluginInventoryEntries[0].Artifacts["v0.0.2"]).NotTo(BeNil())
		})
	})
})

func createTestManifestFile() (string, error) {
	manifestBytes := `created: 2023-02-24T10:10:59.093382-08:00
plugins:
    - name: foo
      target: global
      description: Foo plugin
      versions:
        - v0.0.2
`
	tempManifestFile := filepath.Join(os.TempDir(), "plugin_manifets.yaml")
	return filepath.Join(os.TempDir(), "plugin_manifets.yaml"), utils.SaveFile(tempManifestFile, []byte(manifestBytes))
}
