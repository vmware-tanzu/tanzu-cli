// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package airgapped

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/verybluebot/tarinator-go"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

func TestAirgappedSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Airgapped package Suite")
}

// Unit tests for download and upload bundle
var _ = Describe("Unit tests for download and upload bundle", func() {

	var (
		tempTestDir         string
		err                 error
		dpbo                *DownloadPluginBundleOptions
		upbo                *UploadPluginBundleOptions
		fakeImageOperations *fakes.ImageOperationsImpl
	)

	fakeImageOperations = &fakes.ImageOperationsImpl{}

	// plugin entry to be added in the inventory database
	pluginEntry := &plugininventory.PluginInventoryEntry{
		Name:        "foo",
		Target:      "global",
		Description: "Foo plugin",
		Publisher:   "fakepublisher",
		Vendor:      "fakevendor",
		Hidden:      false,
		Artifacts: map[string]distribution.ArtifactList{
			"v0.0.2": []distribution.Artifact{
				{
					OS:     "darwin",
					Arch:   "amd64",
					Digest: "fake-digest",
					Image:  "path/darwin/amd64/global/foo:v0.0.2",
				},
				{
					OS:     "linux",
					Arch:   "amd64",
					Digest: "fake-digest",
					Image:  "path/linux/amd64/global/foo:v0.0.2",
				},
			},
		},
	}

	// Plugin bundle manifest file generated based on the above mentioned
	// plugin entry in the inventory database
	pluginBundleManifestString := `images:
    - filePath: plugin-inventory-image.tar
      imagePath: /plugin-inventory
    - filePath: foo-global-darwin_amd64-v0.0.2.tar
      imagePath: /path/darwin/amd64/global/foo
    - filePath: foo-global-linux_amd64-v0.0.2.tar
      imagePath: /path/linux/amd64/global/foo
`

	// Configure the configuration before running the tests
	BeforeEach(func() {
		tempTestDir, err = os.MkdirTemp("", "")
		Expect(tempTestDir).ToNot(BeEmpty())
		Expect(err).ToNot(HaveOccurred())
		dpbo = &DownloadPluginBundleOptions{
			PluginInventoryImage: "fake.fakerepo.abc/plugin/plugin-inventory:latest",
			ToTar:                filepath.Join(tempTestDir, "plugin_bundle.tar"),
			ImageProcessor:       fakeImageOperations,
		}
		upbo = &UploadPluginBundleOptions{
			DestinationRepo: "fake.newfakerepo.abc/plugin",
			Tar:             filepath.Join(tempTestDir, "plugin_bundle.tar"),
			ImageProcessor:  fakeImageOperations,
		}
	})
	AfterEach(func() {
		defer os.RemoveAll(tempTestDir)
	})

	// downloadImageAndSaveFilesToDirStub fakes the image downloads and puts a database
	// with the table schemas created to provided path
	downloadImageAndSaveFilesToDirStub := func(image, path string) error {
		dbFile := filepath.Join(path, plugininventory.SQliteDBFileName)
		err := utils.SaveFile(dbFile, []byte{})
		Expect(err).ToNot(HaveOccurred())

		db := plugininventory.NewSQLiteInventory(dbFile, "")
		err = db.CreateSchema()
		Expect(err).ToNot(HaveOccurred())

		err = db.InsertPlugin(pluginEntry)
		Expect(err).ToNot(HaveOccurred())
		return nil
	}

	// copyImageToTarStub fakes the image downloads and creates a fake tar.gz file for images
	copyImageToTarStub := func(image, tarfile string) error {
		_, err := os.Create(tarfile)
		Expect(err).ToNot(HaveOccurred())
		return nil
	}

	var _ = Context("Tests for downloading plugin bundle", func() {

		var _ = It("when downloading plugin inventory image fail with error, it should return an error", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirReturns(errors.New("fake error"))
			fakeImageOperations.CopyImageToTarCalls(copyImageToTarStub)

			err := dpbo.DownloadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to download plugin inventory image"))
			Expect(err.Error()).To(ContainSubstring("fake error"))
		})

		var _ = It("when downloading plugin inventory image succeeds but copy image to tar fail with error, it should return an error", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadImageAndSaveFilesToDirStub)
			fakeImageOperations.CopyImageToTarReturns(errors.New("fake error"))

			err := dpbo.DownloadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while downloading plugin images"))
			Expect(err.Error()).To(ContainSubstring("fake error"))
		})

		var _ = It("when everything works as expected, it should download plugin bundle as tar file", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadImageAndSaveFilesToDirStub)
			fakeImageOperations.CopyImageToTarCalls(copyImageToTarStub)

			err := dpbo.DownloadPluginBundle()
			Expect(err).NotTo(HaveOccurred())

			// Verify that tar file was generated correctly with untar
			tempDir, err := os.MkdirTemp("", "")
			Expect(tempDir).ToNot(BeEmpty())
			Expect(err).NotTo(HaveOccurred())
			err = tarinator.UnTarinate(tempDir, dpbo.ToTar)
			Expect(err).NotTo(HaveOccurred())

			// Verify the plugin bundle manifest file is accurate
			bytes, err := os.ReadFile(filepath.Join(tempDir, PluginBundleDirName, PluginBundleManifestFile))
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes).To(Equal([]byte(pluginBundleManifestString)))
			manifest := &Manifest{}
			err = yaml.Unmarshal(bytes, &manifest)
			Expect(err).NotTo(HaveOccurred())

			// Iterate through all the images in the manifest and verify the all image archive
			// files mentioned in the manifest exists in the bundle
			for _, pi := range manifest.Images {
				exists := utils.PathExists(filepath.Join(tempDir, PluginBundleDirName, pi.FilePath))
				Expect(exists).To(BeTrue())
			}
		})
	})

	var _ = Context("Tests for uploading plugin bundle", func() {
		JustBeforeEach(func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadImageAndSaveFilesToDirStub)
			fakeImageOperations.CopyImageToTarCalls(copyImageToTarStub)

			err := dpbo.DownloadPluginBundle()
			Expect(err).NotTo(HaveOccurred())
		})

		var _ = It("when incorrect tarfile is provided, it should return an error", func() {
			upbo.Tar = "does-not-exists.tar"

			err := upbo.UploadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to untar provided file"))
		})

		var _ = It("when incorrect tarfile is provided, it should return an error", func() {
			// create an incorrect plugin bundle tar file
			upbo.Tar = createIncorrectPluginBundleTarFile(tempTestDir)

			err := upbo.UploadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while reading plugin bundle manifest"))
		})

		var _ = It("when uploading image fail with error, it should return an error", func() {
			fakeImageOperations.CopyImageFromTarReturns(errors.New("fake error"))

			err := upbo.UploadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while uploading image"))
			Expect(err.Error()).To(ContainSubstring("fake error"))
		})

		var _ = It("when uploading image succeeds, it should not return an error", func() {
			fakeImageOperations.CopyImageFromTarReturns(nil)
			err := upbo.UploadPluginBundle()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

// Create incorrect plugin bundle tar file with empty content
func createIncorrectPluginBundleTarFile(dir string) string {
	tarFile := filepath.Join(dir, "incorrect-plugin-bundle.tar")
	emptyDirToTar := filepath.Join(dir, "empty-bundle")
	err := os.MkdirAll(emptyDirToTar, 0755)
	Expect(err).NotTo(HaveOccurred())
	err = tarinator.Tarinate([]string{emptyDirToTar}, tarFile)
	Expect(err).NotTo(HaveOccurred())
	return tarFile
}
