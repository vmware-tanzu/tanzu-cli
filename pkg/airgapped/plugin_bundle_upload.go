// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package airgapped

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/verybluebot/tarinator-go"

	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// UploadPluginBundleOptions defines options for uploading plugin bundle
type UploadPluginBundleOptions struct {
	Tar             string
	DestinationRepo string

	ImageProcessor carvelhelpers.ImageOperationsImpl
}

// UploadPluginBundle uploads the given plugin bundle to the specified remote repository
func (o *UploadPluginBundleOptions) UploadPluginBundle() error {
	// create a temporary directory
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return errors.Wrap(err, "unable to create temp directory")
	}
	defer os.RemoveAll(tempDir)

	// Untar the specified plugin bundle to the temp directory
	log.Infof("extracting %q for processing...", o.Tar)
	err = tarinator.UnTarinate(tempDir, o.Tar)
	if err != nil {
		return errors.Wrap(err, "unable to extract provided file")
	}

	// Read the plugin migration manifest file
	pluginBundleDir := filepath.Join(tempDir, PluginBundleDirName)
	bytes, err := os.ReadFile(filepath.Join(pluginBundleDir, PluginMigrationManifestFile))
	if err != nil {
		return errors.Wrap(err, "error while reading plugin migration manifest")
	}
	manifest := &PluginMigrationManifest{}
	err = yaml.Unmarshal(bytes, &manifest)
	if err != nil {
		return errors.Wrap(err, "error while parsing plugin migration manifest")
	}

	// Iterate through all the images and publish them to the remote repository
	for _, ic := range manifest.ImagesToCopy {
		imageTar := filepath.Join(pluginBundleDir, ic.SourceTarFilePath)
		repoImagePath, err := utils.JoinURL(o.DestinationRepo, ic.RelativeImagePath)
		if err != nil {
			return errors.Wrap(err, "error while constructing the repo image path")
		}
		log.Infof("---------------------------")
		log.Infof("uploading image %q", repoImagePath)
		err = o.ImageProcessor.CopyImageFromTar(imageTar, repoImagePath)
		if err != nil {
			return errors.Wrap(err, "error while uploading image")
		}
	}
	log.Infof("---------------------------")
	log.Infof("---------------------------")

	// Publish plugin inventory metadata image after merging inventory metadata
	log.Infof("publishing plugin inventory metadata image...")
	bundledPluginInventoryMetadataDBFilePath := filepath.Join(pluginBundleDir, manifest.InventoryMetadataImage.SourceFilePath)
	pluginInventoryMetadataImageWithTag, err := utils.JoinURL(o.DestinationRepo, manifest.InventoryMetadataImage.RelativeImagePathWithTag)
	if err != nil {
		return errors.Wrap(err, "error while constructing the plugin inventory metadata image with tag")
	}
	err = o.mergePluginInventoryMetadata(pluginInventoryMetadataImageWithTag, bundledPluginInventoryMetadataDBFilePath, tempDir)
	if err != nil {
		return errors.Wrap(err, "error while merging the plugin inventory metadata database before uploading metadata image")
	}

	log.Infof("uploading image %q", pluginInventoryMetadataImageWithTag)
	err = o.ImageProcessor.PushImage(pluginInventoryMetadataImageWithTag, []string{bundledPluginInventoryMetadataDBFilePath})
	if err != nil {
		return errors.Wrap(err, "error while uploading image")
	}

	log.Infof("---------------------------")

	joinedURL, err := utils.JoinURL(o.DestinationRepo, manifest.RelativeInventoryImagePathWithTag)
	if err != nil {
		return errors.Wrap(err, "error while constructing the image URL")
	}
	log.Infof("successfully published all plugin images to %q", joinedURL)

	return nil
}

/*
func (o *UploadPluginBundleOptions) uploadImage(imageTar, repoImagePath string, totalImages, imagesUploaded int, spinner component.OutputWriterSpinner) error {
	uploadingMsg := fmt.Sprintf("[%d/%d] uploading image %q", totalImages, imagesUploaded, repoImagePath)
	errorMsg := fmt.Sprintf("[%d/%d] error while uploading image %q", totalImages, imagesUploaded, repoImagePath)
	uploadedMsg := "[%d/%d] uploaded image %q"

	if component.IsTTYEnabled() {
		spinner.SetText(uploadingMsg)
		spinner.SetFinalText(errorMsg, log.LogTypeERROR)
		spinner.StartSpinner()
	} else {
		log.Infof(uploadingMsg, totalImages, imagesUploaded, repoImagePath)
	}
	if err := o.ImageProcessor.CopyImageFromTar(imageTar, repoImagePath); err != nil {
		return errors.Wrapf(err, errorMsg, repoImagePath)
	}
	uploadedMsg = fmt.Sprintf(uploadedMsg, totalImages, imagesUploaded+1, repoImagePath)
	if spinner == nil {
		spinner.SetFinalText("", "")
	} else {
		log.Infof(uploadedMsg, totalImages, imagesUploaded, repoImagePath)
	}

	return nil
}
*/
// mergePluginInventoryMetadata merges the downloaded plugin inventory metadata with
// existing plugin inventory metadata available on the remote repository
func (o *UploadPluginBundleOptions) mergePluginInventoryMetadata(pluginInventoryMetadataImageWithTag, bundledPluginInventoryMetadataDBFilePath, tempDir string) error {
	tempPluginInventoryMetadataDir := filepath.Join(tempDir, "inventory-metadata")
	err := o.ImageProcessor.DownloadImageAndSaveFilesToDir(pluginInventoryMetadataImageWithTag, tempPluginInventoryMetadataDir)
	if err == nil {
		downloadedPluginInventoryMetadataDBFilePath := filepath.Join(tempPluginInventoryMetadataDir, plugininventory.SQliteInventoryMetadataDBFileName)
		pluginInventoryDB := plugininventory.NewSQLiteInventoryMetadata(bundledPluginInventoryMetadataDBFilePath)
		err = pluginInventoryDB.MergeInventoryMetadataDatabase(downloadedPluginInventoryMetadataDBFilePath)
		if err != nil {
			return err
		}
		log.Infof("plugin inventory metadata image %q is present. Merging the plugin inventory metadata", pluginInventoryMetadataImageWithTag)
	} else {
		log.Infof("plugin inventory metadata image %q is not present. Skipping merging of the plugin inventory metadata", pluginInventoryMetadataImageWithTag)
	}
	return nil
}
