// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package airgapped

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/verybluebot/tarinator-go"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cosignhelper/sigverifier"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// DownloadPluginBundleOptions defines options for downloading plugin bundle
type DownloadPluginBundleOptions struct {
	PluginInventoryImage string
	ToTar                string
	Groups               []string

	ImageProcessor carvelhelpers.ImageOperationsImpl
}

// DownloadPluginBundle download the plugin bundle based on provided plugin inventory image
// and save it as tar file
func (o *DownloadPluginBundleOptions) DownloadPluginBundle() error {
	// Validate the input options
	err := o.validateOptions()
	if err != nil {
		return err
	}

	// Create temp download directory
	tempBaseDir, err := os.MkdirTemp("", "")
	if err != nil {
		return errors.Wrap(err, "unable to create temp directory")
	}
	tempPluginBundleDir := filepath.Join(tempBaseDir, PluginBundleDirName)
	err = os.Mkdir(tempPluginBundleDir, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "unable to create temp directory")
	}
	defer os.RemoveAll(tempBaseDir)

	// Get selected plugin groups and plugins objects based on the inputs
	selectedPluginEntries, selectedPluginGroups, err := o.getSelectedPluginInfo()
	if err != nil {
		return errors.Wrap(err, "error while getting selected plugin and plugin group information")
	}

	// Save plugin images and get list of images that needs to be copied as part of the upload process
	relativeInventoryImagePathWithTag, imagesToCopy, err := o.saveAndGetImagesToCopy(selectedPluginEntries, tempPluginBundleDir)
	if err != nil {
		return errors.Wrap(err, "error while downloading and saving plugin images")
	}

	// Save plugin inventory metadata file and create an entry object
	inventoryMetadataImageInfo, err := o.savePluginInventoryMetadata(selectedPluginGroups, selectedPluginEntries, tempPluginBundleDir)
	if err != nil {
		return errors.Wrap(err, "error while saving plugin inventory metadata")
	}

	// Save plugin migration manifest file to the plugin bundle directory
	err = savePluginMigrationManifestFile(relativeInventoryImagePathWithTag, imagesToCopy, inventoryMetadataImageInfo, tempPluginBundleDir)
	if err != nil {
		return errors.Wrap(err, "error while saving plugin migration manifest")
	}

	// Save entire plugin bundle as a single tar file which can be used with upload-bundle
	log.Infof("saving plugin bundle at: %s", o.ToTar)
	err = tarinator.Tarinate([]string{tempPluginBundleDir}, o.ToTar)
	if err != nil {
		return errors.Wrap(err, "error while creating archive file")
	}

	return nil
}

// getSelectedPluginInfo returns the list of PluginInventoryEntry and
// PluginGroupEntry based on the DownloadPluginBundleOptions that needs to be
// considered for downloading plugin bundle.
// Downloads the plugin inventory image and selects the plugins and plugin
// groups based on the DownloadPluginBundleOptions.Groups by querying the
// plugin inventory database
func (o *DownloadPluginBundleOptions) getSelectedPluginInfo() ([]*plugininventory.PluginInventoryEntry, []*plugininventory.PluginGroup, error) {
	var err error
	tempDBDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create temp directory")
	}
	defer os.RemoveAll(tempDBDir)

	// Download the plugin inventory oci image to tempDBDir
	inventoryFile := filepath.Join(tempDBDir, plugininventory.SQliteDBFileName)
	if err := o.ImageProcessor.DownloadImageAndSaveFilesToDir(o.PluginInventoryImage, filepath.Dir(inventoryFile)); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to download plugin inventory image '%s'", o.PluginInventoryImage)
	}

	// Read plugin inventory database and set pluginEntries to point to plugins that needs to be downloaded
	pi := plugininventory.NewSQLiteInventory(inventoryFile, path.Dir(o.PluginInventoryImage))

	selectedPluginGroups := []*plugininventory.PluginGroup{}
	selectedPluginEntries := []*plugininventory.PluginInventoryEntry{}

	// If groups were not provided as argument select all available plugin groups and all available plugins
	if len(o.Groups) == 0 {
		selectedPluginGroups, err = pi.GetPluginGroups(plugininventory.PluginGroupFilter{IncludeHidden: true}) // Include the hidden plugin groups during plugin migration
		if err != nil {
			return nil, nil, errors.Wrap(err, "unable to read all plugin groups from database")
		}
		selectedPluginEntries, err = pi.GetPlugins(&plugininventory.PluginInventoryFilter{IncludeHidden: true}) // Include the hidden plugins during plugin migration
		if err != nil {
			return nil, nil, errors.Wrap(err, "unable to read all plugins from database")
		}
	} else {
		// If groups were provided as argument select only provided plugin groups and
		// plugins available from the specified plugin groups
		for _, groupID := range o.Groups {
			pluginGroups, pluginEntries, err := o.getAllPluginGroupsAndPluginEntriesFromPluginGroupVersion(groupID, pi)
			if err != nil {
				return nil, nil, err
			}
			selectedPluginGroups = append(selectedPluginGroups, pluginGroups...)
			selectedPluginEntries = append(selectedPluginEntries, pluginEntries...)
		}
	}
	return selectedPluginEntries, selectedPluginGroups, nil
}

func (o *DownloadPluginBundleOptions) getAllPluginGroupsAndPluginEntriesFromPluginGroupVersion(pgID string, pi plugininventory.PluginInventory) ([]*plugininventory.PluginGroup, []*plugininventory.PluginInventoryEntry, error) {
	pgi := plugininventory.PluginGroupIdentifierFromID(pgID)
	if pgi == nil {
		return nil, nil, errors.Errorf("incorrect plugin group %q specified", pgID)
	}
	if pgi.Version == "" {
		pgi.Version = cli.VersionLatest
	}
	pgFilter := plugininventory.PluginGroupFilter{
		IncludeHidden: true, // Include the hidden plugin groups during plugin migration
		Vendor:        pgi.Vendor,
		Publisher:     pgi.Publisher,
		Name:          pgi.Name,
		Version:       pgi.Version,
	}
	pluginGroups, err := pi.GetPluginGroups(pgFilter)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to get plugin groups")
	}

	if len(pluginGroups) == 0 {
		return nil, nil, errors.Errorf("incorrect plugin group %q specified", pgID)
	}

	var allPluginEntries []*plugininventory.PluginInventoryEntry
	for _, pg := range pluginGroups {
		for _, plugins := range pg.Versions {
			for _, p := range plugins {
				pif := &plugininventory.PluginInventoryFilter{
					Name:          p.Name,
					Target:        p.Target,
					Version:       p.Version,
					IncludeHidden: true, // Include the hidden plugins during plugin migration
				}
				pluginEntries, err := pi.GetPlugins(pif)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "unable to get plugins in plugin group %v", plugininventory.PluginGroupToID(pg))
				}
				allPluginEntries = append(allPluginEntries, pluginEntries...)
			}
		}
	}
	return pluginGroups, allPluginEntries, nil
}

// saveAndGetImagesToCopy saves the images after downloading them and
// returns the images to copy object
func (o *DownloadPluginBundleOptions) saveAndGetImagesToCopy(pluginEntries []*plugininventory.PluginInventoryEntry, downloadDir string) (string, []*ImageCopyInfo, error) {
	// Download all plugin inventory database and plugins as tar file
	return o.downloadImagesAsTarFile(pluginEntries, downloadDir)
}

// downloadImagesAsTarFile downloads plugin inventory image and all plugin images
// as tar file to the specified directory
func (o *DownloadPluginBundleOptions) downloadImagesAsTarFile(pluginEntries []*plugininventory.PluginInventoryEntry, downloadDir string) (string, []*ImageCopyInfo, error) {
	allImages := []*ImageCopyInfo{}

	// Download plugin inventory database as tar file
	pluginInventoryFileNameTar := "plugin-inventory-image.tar.gz"
	log.Infof("downloading image %q", o.PluginInventoryImage)
	err := o.ImageProcessor.CopyImageToTar(o.PluginInventoryImage, filepath.Join(downloadDir, pluginInventoryFileNameTar))
	if err != nil {
		return "", nil, err
	}

	relativeInventoryImagePathWithTag := GetImageRelativePath(o.PluginInventoryImage, path.Dir(o.PluginInventoryImage), true)

	allImages = append(allImages, &ImageCopyInfo{
		SourceTarFilePath: pluginInventoryFileNameTar,
		RelativeImagePath: GetImageRelativePath(o.PluginInventoryImage, path.Dir(o.PluginInventoryImage), false),
	})

	// Process all plugin entries and download the oci image as tar file
	for _, pe := range pluginEntries {
		for version, artifacts := range pe.Artifacts {
			for _, a := range artifacts {
				log.Infof("---------------------------")
				log.Infof("downloading image %q", a.Image)
				tarfileName := fmt.Sprintf("%s-%s-%s_%s-%s.tar.gz", pe.Name, pe.Target, a.OS, a.Arch, version)
				err = o.ImageProcessor.CopyImageToTar(a.Image, filepath.Join(downloadDir, tarfileName))
				if err != nil {
					return "", nil, err
				}
				allImages = append(allImages, &ImageCopyInfo{
					SourceTarFilePath: tarfileName,
					RelativeImagePath: GetImageRelativePath(a.Image, path.Dir(o.PluginInventoryImage), false),
				})
			}
		}
	}
	return relativeInventoryImagePathWithTag, allImages, nil
}

// validateOptions validates the provided options and returns
// error if contains invalid option
func (o *DownloadPluginBundleOptions) validateOptions() error {
	_, err := os.Stat(filepath.Dir(o.ToTar))
	if err != nil {
		return errors.Wrapf(err, "invalid path for %q", o.ToTar)
	}

	// Verify the inventory image signature before downloading the plugin inventory database
	err = sigverifier.VerifyInventoryImageSignature(o.PluginInventoryImage)
	if err != nil {
		return err
	}

	return nil
}

// savePluginMigrationManifestFile save the plugin_migration_manifest.yaml file
// to the provided pluginBundleDir
func savePluginMigrationManifestFile(relativeInventoryImagePathWithTag string, imagesToCopy []*ImageCopyInfo, inventoryMetadataImageInfo *ImagePublishInfo, pluginBundleDir string) error {
	// Save all downloaded images as part of manifest file
	manifest := PluginMigrationManifest{
		RelativeInventoryImagePathWithTag: relativeInventoryImagePathWithTag,
		ImagesToCopy:                      imagesToCopy,
		InventoryMetadataImage:            inventoryMetadataImageInfo,
	}
	bytes, err := yaml.Marshal(&manifest)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(pluginBundleDir, PluginMigrationManifestFile), bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

// savePluginInventoryMetadata saves the plugin inventory metadata database file
// and returns ImagePublishInfo object containing the details on where to publish
// the metadata database file as an oci image
func (o *DownloadPluginBundleOptions) savePluginInventoryMetadata(pgs []*plugininventory.PluginGroup, pes []*plugininventory.PluginInventoryEntry, pluginBundleDir string) (*ImagePublishInfo, error) {
	inventoryMetadataDBFileName := plugininventory.SQliteInventoryMetadataDBFileName
	inventoryMetadataDBFilePath := filepath.Join(pluginBundleDir, inventoryMetadataDBFileName)
	inventoryMetadataDB := plugininventory.NewSQLiteInventoryMetadata(inventoryMetadataDBFilePath)

	err := inventoryMetadataDB.CreateInventoryMetadataDBSchema()
	if err != nil {
		return nil, err
	}

	for _, pe := range pes {
		for version := range pe.Artifacts {
			err := inventoryMetadataDB.InsertPluginIdentifier(&plugininventory.PluginIdentifier{Name: pe.Name, Target: pe.Target, Version: version})
			if err != nil {
				return nil, err
			}
		}
	}
	for _, pg := range pgs {
		for version := range pg.Versions {
			err := inventoryMetadataDB.InsertPluginGroupIdentifier(&plugininventory.PluginGroupIdentifier{Vendor: pg.Vendor, Publisher: pg.Publisher, Name: pg.Name, Version: version})
			if err != nil {
				return nil, err
			}
		}
	}

	pluginInventoryMetadataImage, err := GetPluginInventoryMetadataImage(o.PluginInventoryImage)
	if err != nil {
		return nil, err
	}

	imagePublishInfo := &ImagePublishInfo{
		SourceFilePath:           inventoryMetadataDBFileName,
		RelativeImagePathWithTag: GetImageRelativePath(pluginInventoryMetadataImage, path.Dir(o.PluginInventoryImage), true),
	}

	return imagePublishInfo, nil
}
