// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package airgapped

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/verybluebot/tarinator-go"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
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

	// plugin entry foo to be added in the inventory database
	pluginEntryFoo := &plugininventory.PluginInventoryEntry{
		Name:               "foo",
		Target:             "global",
		Description:        "Foo plugin",
		Publisher:          "fakepublisher",
		Vendor:             "fakevendor",
		Hidden:             false,
		RecommendedVersion: "v0.0.2",
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

	// plugin entry bar to be added in the inventory database
	pluginEntryBar := &plugininventory.PluginInventoryEntry{
		Name:               "bar",
		Target:             "kubernetes",
		Description:        "Bar plugin",
		Publisher:          "fakepublisher",
		Vendor:             "fakevendor",
		Hidden:             false,
		RecommendedVersion: "v0.0.1",
		Artifacts: map[string]distribution.ArtifactList{
			"v0.0.1": []distribution.Artifact{
				{
					OS:     "darwin",
					Arch:   "amd64",
					Digest: "fake-digest-bar",
					Image:  "path/darwin/amd64/kubernetes/bar:v0.0.1",
				},
			},
		},
	}

	pluginGroupEntry := &plugininventory.PluginGroup{
		Vendor:             "fakevendor",
		Publisher:          "fakepublisher",
		Name:               "default",
		Description:        "Desc for plugin",
		Hidden:             false,
		RecommendedVersion: "v1.0.0",
		Versions: map[string][]*plugininventory.PluginGroupPluginEntry{
			"v1.0.0": {
				&plugininventory.PluginGroupPluginEntry{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "bar",
						Target:  "kubernetes",
						Version: "v0.0.1",
					},
				},
			},
		},
	}

	// plugin entry bar to be added in the inventory database
	essentialPluginEntryTelemetry := &plugininventory.PluginInventoryEntry{
		Name:               "telemetry",
		Target:             "global",
		Description:        "Telemetry plugin",
		Publisher:          "tanzucli",
		Vendor:             "vmware",
		Hidden:             false,
		RecommendedVersion: "v0.0.1",
		Artifacts: map[string]distribution.ArtifactList{
			"v0.0.1": []distribution.Artifact{
				{
					OS:     "darwin",
					Arch:   "amd64",
					Digest: "fake-digest-telemetry",
					Image:  "path/darwin/amd64/global/telemetry:v0.0.1",
				},
			},
		},
	}

	essentialPluginGroupEntry := &plugininventory.PluginGroup{
		Vendor:             "vmware",
		Publisher:          "tanzucli",
		Name:               "essentials",
		Description:        "Desc for plugin",
		Hidden:             false,
		RecommendedVersion: "v0.0.1",
		Versions: map[string][]*plugininventory.PluginGroupPluginEntry{
			"v0.0.1": {
				&plugininventory.PluginGroupPluginEntry{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "telemetry",
						Target:  "global",
						Version: "v0.0.1",
					},
				},
			},
		},
	}

	// Plugin bundle manifest file generated based on the above mentioned
	// plugin entry in the inventory database
	pluginBundleManifestCompleteRepositoryString := `relativeInventoryImagePathWithTag: /plugin-inventory:latest
inventoryMetadataImage:
    sourceFilePath: plugin_inventory_metadata.db
    relativeImagePathWithTag: /plugin-inventory-metadata:latest
imagesToCopy:
    - sourceTarFilePath: plugin-inventory-image.tar.gz
      relativeImagePath: /plugin-inventory
    - sourceTarFilePath: bar-kubernetes-darwin_amd64-v0.0.1.tar.gz
      relativeImagePath: /path/darwin/amd64/kubernetes/bar
    - sourceTarFilePath: foo-global-darwin_amd64-v0.0.2.tar.gz
      relativeImagePath: /path/darwin/amd64/global/foo
    - sourceTarFilePath: foo-global-linux_amd64-v0.0.2.tar.gz
      relativeImagePath: /path/linux/amd64/global/foo
    - sourceTarFilePath: telemetry-global-darwin_amd64-v0.0.1.tar.gz
      relativeImagePath: /path/darwin/amd64/global/telemetry
`

	// Plugin bundle manifest file generated based on the above mentioned
	// plugin entry in the inventory database with only single plugin group specified
	pluginBundleManifestDefaultGroupOnlyString := `relativeInventoryImagePathWithTag: /plugin-inventory:latest
inventoryMetadataImage:
    sourceFilePath: plugin_inventory_metadata.db
    relativeImagePathWithTag: /plugin-inventory-metadata:latest
imagesToCopy:
    - sourceTarFilePath: plugin-inventory-image.tar.gz
      relativeImagePath: /plugin-inventory
    - sourceTarFilePath: bar-kubernetes-darwin_amd64-v0.0.1.tar.gz
      relativeImagePath: /path/darwin/amd64/kubernetes/bar
    - sourceTarFilePath: telemetry-global-darwin_amd64-v0.0.1.tar.gz
      relativeImagePath: /path/darwin/amd64/global/telemetry
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
		os.Setenv(constants.PluginDiscoveryImageSignatureVerificationSkipList, dpbo.PluginInventoryImage)
	})
	AfterEach(func() {
		defer os.RemoveAll(tempTestDir)
	})

	// downloadInventoryImageAndSaveFilesToDirStub fakes the image downloads and puts a database
	// with the table schemas created to provided path
	downloadInventoryImageAndSaveFilesToDirStub := func(_, path string) error {
		dbFile := filepath.Join(path, plugininventory.SQliteDBFileName)
		err := utils.SaveFile(dbFile, []byte{})
		Expect(err).ToNot(HaveOccurred())

		db := plugininventory.NewSQLiteInventory(dbFile, "")
		err = db.CreateSchema()
		Expect(err).ToNot(HaveOccurred())

		err = db.InsertPlugin(pluginEntryFoo)
		Expect(err).ToNot(HaveOccurred())
		err = db.InsertPlugin(pluginEntryBar)
		Expect(err).ToNot(HaveOccurred())
		err = db.InsertPluginGroup(pluginGroupEntry, true)
		Expect(err).ToNot(HaveOccurred())

		err = db.InsertPlugin(essentialPluginEntryTelemetry)
		Expect(err).ToNot(HaveOccurred())
		err = db.InsertPluginGroup(essentialPluginGroupEntry, true)
		Expect(err).ToNot(HaveOccurred())
		return nil
	}

	// downloadInventoryMetadataImageWithNoExistingPlugins fakes the image downloads and puts a database
	// with the table schemas created to provided path
	downloadInventoryMetadataImageWithNoExistingPlugins := func(_, path string) error {
		dbFile := filepath.Join(path, plugininventory.SQliteInventoryMetadataDBFileName)
		err := utils.SaveFile(dbFile, []byte{})
		Expect(err).ToNot(HaveOccurred())

		db := plugininventory.NewSQLiteInventoryMetadata(dbFile)
		err = db.CreateInventoryMetadataDBSchema()
		Expect(err).ToNot(HaveOccurred())

		return nil
	}

	// downloadInventoryMetadataImageWithExistingPlugins fakes the image downloads and puts a database
	// with the table schemas created to provided path
	downloadInventoryMetadataImageWithExistingPlugins := func(_, path string) error {
		dbFile := filepath.Join(path, plugininventory.SQliteInventoryMetadataDBFileName)
		err := utils.SaveFile(dbFile, []byte{})
		Expect(err).ToNot(HaveOccurred())

		db := plugininventory.NewSQLiteInventoryMetadata(dbFile)
		err = db.CreateInventoryMetadataDBSchema()
		Expect(err).ToNot(HaveOccurred())

		err = db.InsertPluginGroupIdentifier(&plugininventory.PluginGroupIdentifier{Name: pluginGroupEntry.Name, Vendor: pluginGroupEntry.Vendor, Publisher: pluginGroupEntry.Publisher})
		Expect(err).ToNot(HaveOccurred())

		err = db.InsertPluginIdentifier(&plugininventory.PluginIdentifier{Name: pluginEntryBar.Name, Target: pluginEntryBar.Target, Version: pluginEntryBar.RecommendedVersion})
		Expect(err).ToNot(HaveOccurred())

		err = db.InsertPluginGroupIdentifier(&plugininventory.PluginGroupIdentifier{Name: essentialPluginGroupEntry.Name, Vendor: essentialPluginGroupEntry.Vendor, Publisher: essentialPluginGroupEntry.Publisher})
		Expect(err).ToNot(HaveOccurred())

		err = db.InsertPluginIdentifier(&plugininventory.PluginIdentifier{Name: essentialPluginEntryTelemetry.Name, Target: essentialPluginEntryTelemetry.Target, Version: essentialPluginEntryTelemetry.RecommendedVersion})
		Expect(err).ToNot(HaveOccurred())

		return nil
	}

	// copyImageToTarStub fakes the image downloads and creates a fake tar.gz file for images
	copyImageToTarStub := func(_, tarfile string) error {
		_, err := os.Create(tarfile)
		Expect(err).ToNot(HaveOccurred())
		return nil
	}

	var _ = Context("Tests for downloading plugin bundle", func() {

		var _ = It("when invalid tar file path is provided, it should return an error", func() {
			dpbo.ToTar = filepath.Join("/tmp", "doesnotexist", "plugin_bundle.tar.gz")

			err := dpbo.DownloadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid path for"))
		})

		var _ = It("when downloading plugin inventory image fail with error, it should return an error", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirReturns(errors.New("fake error"))
			fakeImageOperations.CopyImageToTarCalls(copyImageToTarStub)

			err := dpbo.DownloadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to download plugin inventory image"))
			Expect(err.Error()).To(ContainSubstring("fake error"))
		})

		var _ = It("when downloading plugin inventory image succeeds but copy image to tar fail with error, it should return an error", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadInventoryImageAndSaveFilesToDirStub)
			fakeImageOperations.CopyImageToTarReturns(errors.New("fake error"))

			err := dpbo.DownloadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while downloading and saving plugin images"))
			Expect(err.Error()).To(ContainSubstring("fake error"))
		})

		var _ = It("when group is not specified and everything works as expected, it should download plugin bundle as tar file", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadInventoryImageAndSaveFilesToDirStub)
			fakeImageOperations.CopyImageToTarCalls(copyImageToTarStub)

			err := dpbo.DownloadPluginBundle()
			Expect(err).NotTo(HaveOccurred())

			// Verify that tar file was generated correctly with untar
			tempDir, err := os.MkdirTemp("", "")
			Expect(tempDir).ToNot(BeEmpty())
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = tarinator.UnTarinate(tempDir, dpbo.ToTar)
			Expect(err).NotTo(HaveOccurred())

			// Verify the plugin bundle manifest file is accurate
			bytes, err := os.ReadFile(filepath.Join(tempDir, PluginBundleDirName, PluginMigrationManifestFile))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(bytes)).To(Equal(pluginBundleManifestCompleteRepositoryString))
			manifest := &PluginMigrationManifest{}
			err = yaml.Unmarshal(bytes, &manifest)
			Expect(err).NotTo(HaveOccurred())

			// Iterate through all the images in the manifest and verify the all image archive
			// files mentioned in the manifest exists in the bundle
			for _, pi := range manifest.ImagesToCopy {
				exists := utils.PathExists(filepath.Join(tempDir, PluginBundleDirName, pi.SourceTarFilePath))
				Expect(exists).To(BeTrue())
			}
		})

		var _ = It("when group specified does not exists, it should return an error", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadInventoryImageAndSaveFilesToDirStub)
			fakeImageOperations.CopyImageToTarCalls(copyImageToTarStub)

			dpbo.Groups = []string{"vmware-tanzu/does-not-exists"}
			err := dpbo.DownloadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while getting selected plugin and plugin group information"))
			Expect(err.Error()).To(ContainSubstring("incorrect plugin group \"vmware-tanzu/does-not-exists\" specified"))

			dpbo.Groups = []string{"does-not-exists"}
			err = dpbo.DownloadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while getting selected plugin and plugin group information"))
			Expect(err.Error()).To(ContainSubstring("incorrect plugin group \"does-not-exists\" specified"))
		})

		var _ = It("when group is specified and everything works as expected, it should download plugin bundle as tar file", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadInventoryImageAndSaveFilesToDirStub)
			fakeImageOperations.CopyImageToTarCalls(copyImageToTarStub)

			dpbo.Groups = []string{"fakevendor-fakepublisher/default:v1.0.0"}
			err := dpbo.DownloadPluginBundle()
			Expect(err).NotTo(HaveOccurred())

			// Verify that tar file was generated correctly with untar
			tempDir, err := os.MkdirTemp("", "")
			Expect(tempDir).ToNot(BeEmpty())
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			err = tarinator.UnTarinate(tempDir, dpbo.ToTar)
			Expect(err).NotTo(HaveOccurred())

			// Verify the plugin bundle manifest file is accurate
			bytes, err := os.ReadFile(filepath.Join(tempDir, PluginBundleDirName, PluginMigrationManifestFile))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(bytes)).To(Equal(pluginBundleManifestDefaultGroupOnlyString))
			manifest := &PluginMigrationManifest{}
			err = yaml.Unmarshal(bytes, &manifest)
			Expect(err).NotTo(HaveOccurred())

			// Iterate through all the images in the manifest and verify the all image archive
			// files mentioned in the manifest exists in the bundle
			for _, pi := range manifest.ImagesToCopy {
				exists := utils.PathExists(filepath.Join(tempDir, PluginBundleDirName, pi.SourceTarFilePath))
				Expect(exists).To(BeTrue())
			}
		})

		var _ = It("when using --dry-run option, it should work and write the images yaml to the standard output", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadInventoryImageAndSaveFilesToDirStub)
			fakeImageOperations.CopyImageToTarCalls(copyImageToTarStub)
			dpbo.ToTar = ""
			dpbo.DryRun = true

			// Setup to interject stdout for our tests
			r, w, err := os.Pipe()
			Expect(err).NotTo(HaveOccurred())
			c := make(chan []byte)
			go readOutput(r, c)
			stdout := os.Stdout
			defer func() {
				os.Stdout = stdout
			}()
			os.Stdout = w

			// Invoke actual tests
			err = dpbo.DownloadPluginBundle()
			Expect(err).NotTo(HaveOccurred())

			// Read the stdout from the channel.
			w.Close()
			stdoutBytes := <-c

			// Verify that correct information was written to the stdout
			log.Infof("%v", string(stdoutBytes))
			imageMetadata := make(map[string][]string)
			err = yaml.Unmarshal(stdoutBytes, &imageMetadata)
			Expect(err).NotTo(HaveOccurred())
			images, ok := imageMetadata["images"]
			Expect(ok).To(Equal(true))
			expectedImages := []string{
				"fake.fakerepo.abc/plugin/plugin-inventory:latest",
				"fake.fakerepo.abc/plugin/path/darwin/amd64/kubernetes/bar:v0.0.1",
				"fake.fakerepo.abc/plugin/path/darwin/amd64/global/foo:v0.0.2",
				"fake.fakerepo.abc/plugin/path/linux/amd64/global/foo:v0.0.2",
				"fake.fakerepo.abc/plugin/path/darwin/amd64/global/telemetry:v0.0.1",
			}

			Expect(images).To(ContainElements(expectedImages))
		})
	})

	var _ = Context("Tests for uploading plugin bundle when downloading entire plugin repository with all plugin", func() {
		JustBeforeEach(func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadInventoryImageAndSaveFilesToDirStub)
			fakeImageOperations.CopyImageToTarCalls(copyImageToTarStub)

			err := dpbo.DownloadPluginBundle()
			Expect(err).NotTo(HaveOccurred())
		})

		var _ = It("when incorrect tarfile is provided, it should return an error", func() {
			upbo.Tar = "does-not-exists.tar"

			err := upbo.UploadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to extract provided file"))
		})

		var _ = It("when incorrect tarfile is provided, it should return an error", func() {
			// create an incorrect plugin bundle tar file
			upbo.Tar = createIncorrectPluginBundleTarFile(tempTestDir)

			err := upbo.UploadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while reading plugin migration manifest"))
		})

		var _ = It("when uploading image fail with error, it should return an error", func() {
			fakeImageOperations.CopyImageFromTarReturns(errors.New("fake error"))

			err := upbo.UploadPluginBundle()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error while uploading image"))
			Expect(err.Error()).To(ContainSubstring("fake error"))
		})

		var _ = It("when fetching the existing inventory metadata fails, it should not return an error", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirReturns(errors.New("fake-error"))
			fakeImageOperations.CopyImageFromTarReturns(nil)
			err := upbo.UploadPluginBundle()
			Expect(err).NotTo(HaveOccurred())
		})

		var _ = It("when uploading images succeeds and fetching the existing inventory metadata returns no existing plugins, it should not return an error", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadInventoryMetadataImageWithNoExistingPlugins)
			fakeImageOperations.CopyImageFromTarReturns(nil)
			err := upbo.UploadPluginBundle()
			Expect(err).NotTo(HaveOccurred())
		})

		var _ = It("when uploading images succeeds and fetching the existing inventory metadata returns few existing plugins, merge should happen and it should not return an error", func() {
			fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(downloadInventoryMetadataImageWithExistingPlugins)
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

func readOutput(r io.Reader, c chan<- []byte) {
	data, err := io.ReadAll(r)
	Expect(err).NotTo(HaveOccurred())
	c <- data
}
