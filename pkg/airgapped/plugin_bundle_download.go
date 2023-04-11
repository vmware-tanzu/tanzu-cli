// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package airgapped

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/verybluebot/tarinator-go"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// DownloadPluginBundleOptions defines options for downloading plugin bundle
type DownloadPluginBundleOptions struct {
	PluginInventoryImage string
	ToTar                string

	ImageProcessor carvelhelpers.ImageOperationsImpl
}

// DownloadPluginBundle download the plugin bundle based on provided plugin inventory image
// and save it as tar file
func (o *DownloadPluginBundleOptions) DownloadPluginBundle() error {
	// Create temp download directory
	tempBaseDir, err := os.MkdirTemp("", "")
	if err != nil {
		return errors.Wrap(err, "unable to create temp directory")
	}
	tempDir := filepath.Join(tempBaseDir, PluginBundleDirName)
	err = os.Mkdir(tempDir, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "unable to create temp directory")
	}
	defer os.RemoveAll(tempBaseDir)

	// Download the plugin inventory oci image to '<tempBaseDir>/db/'
	inventoryFile := filepath.Join(tempBaseDir, "db", plugininventory.SQliteDBFileName)
	if err := o.ImageProcessor.DownloadImageAndSaveFilesToDir(o.PluginInventoryImage, filepath.Dir(inventoryFile)); err != nil {
		return errors.Wrapf(err, "failed to download plugin inventory image '%s'", o.PluginInventoryImage)
	}

	// Read plugin inventory database and set pluginEntries to point to plugins that needs to be downloaded
	imagePrefix := path.Dir(o.PluginInventoryImage)
	pi := plugininventory.NewSQLiteInventory(inventoryFile, imagePrefix)
	pluginEntries, err := pi.GetAllPlugins()
	if err != nil {
		return errors.Wrap(err, "unable to get plugin details from the database")
	}

	// Download all plugin inventory database and plugins as tar file
	allImages, err := o.downloadAllPluginImages(pluginEntries, imagePrefix, tempDir)
	if err != nil {
		return errors.Wrap(err, "error while downloading plugin images")
	}

	// Save all downloaded images as part of manifest file
	err = saveManifestFile(allImages, tempDir)
	if err != nil {
		return errors.Wrap(err, "error while saving plugin bundle manifest")
	}

	log.Infof("saving plugin bundle at: %s", o.ToTar)
	// Save entire plugin bundle as a single tar file
	err = tarinator.Tarinate([]string{tempDir}, o.ToTar)
	if err != nil {
		return errors.Wrap(err, "error while creating tar file")
	}

	return nil
}

func (o *DownloadPluginBundleOptions) downloadAllPluginImages(pluginEntries []*plugininventory.PluginInventoryEntry, imagePrefix, tempDir string) ([]*ImageInfo, error) {
	allImages := []*ImageInfo{}

	// Download plugin inventory database as tar file
	pluginInventoryFileNameTar := "plugin-inventory-image.tar"
	log.Infof("downloading image %q", o.PluginInventoryImage)
	err := o.ImageProcessor.CopyImageToTar(o.PluginInventoryImage, filepath.Join(tempDir, pluginInventoryFileNameTar))
	if err != nil {
		return nil, err
	}
	allImages = append(allImages, &ImageInfo{FilePath: pluginInventoryFileNameTar, ImagePath: getImageRelativePath(o.PluginInventoryImage, imagePrefix)})

	// Process all plugin entries and download the oci image as tar file
	for _, pe := range pluginEntries {
		for version, artifacts := range pe.Artifacts {
			for _, a := range artifacts {
				log.Infof("---------------------------")
				log.Infof("downloading image %q", a.Image)
				tarfileName := fmt.Sprintf("%s-%s-%s_%s-%s.tar", pe.Name, pe.Target, a.OS, a.Arch, version)
				err = o.ImageProcessor.CopyImageToTar(a.Image, filepath.Join(tempDir, tarfileName))
				if err != nil {
					return nil, err
				}
				allImages = append(allImages, &ImageInfo{FilePath: tarfileName, ImagePath: getImageRelativePath(a.Image, imagePrefix)})
			}
		}
	}
	return allImages, nil
}

func getImageRelativePath(image, imagePrefix string) string {
	relativePathWithVersion := strings.TrimPrefix(image, imagePrefix)
	if idx := strings.LastIndex(relativePathWithVersion, ":"); idx != -1 {
		return relativePathWithVersion[:idx]
	}
	return relativePathWithVersion
}

func saveManifestFile(allImages []*ImageInfo, dir string) error {
	// Save all downloaded images as part of manifest file
	manifest := Manifest{Images: allImages}
	bytes, err := yaml.Marshal(&manifest)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(dir, PluginBundleManifestFile), bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
