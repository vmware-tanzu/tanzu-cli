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
	"github.com/vmware-tanzu/tanzu-cli/pkg/essentials"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const fileExists = "the file '%s' already exists"

// DownloadPluginBundleOptions defines options for downloading plugin bundle
type DownloadPluginBundleOptions struct {
	PluginInventoryImage string
	ToTar                string
	Groups               []string
	Plugins              []string
	RefreshConfigOnly    bool
	DryRun               bool
	ImageProcessor       carvelhelpers.ImageOperationsImpl
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

	if o.DryRun {
		imageMetadata, err := o.getListOfImages(selectedPluginEntries)
		if err != nil {
			return err
		}

		imagesBytes, err := yaml.Marshal(imageMetadata)
		if err != nil {
			return err
		}

		log.Info("Saving the list of images to download...")
		log.Outputf("%v", string(imagesBytes))
		return nil
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

	log.Infof("Getting selected plugin information...")

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
	if len(o.Groups) == 0 && len(o.Plugins) == 0 {
		if o.RefreshConfigOnly {
			log.Infof("will only be downloading the latest plugin inventory OCI image and central configuration data")
		} else {
			selectedPluginGroups, err = pi.GetPluginGroups(plugininventory.PluginGroupFilter{IncludeHidden: true}) // Include the hidden plugin groups during plugin migration
			if err != nil {
				return nil, nil, errors.Wrap(err, "unable to read all plugin groups from database")
			}
			selectedPluginEntries, err = pi.GetPlugins(&plugininventory.PluginInventoryFilter{IncludeHidden: true}) // Include the hidden plugins during plugin migration
			if err != nil {
				return nil, nil, errors.Wrap(err, "unable to read all plugins from database")
			}
			if len(selectedPluginEntries) == 1 {
				log.Infof("will be downloading the one plugin from: %s", o.PluginInventoryImage)
			} else {
				log.Infof("will be downloading the %d plugins from: %s", len(selectedPluginEntries), o.PluginInventoryImage)
			}
		}
	} else {
		// If groups were provided as argument select only provided plugin groups and
		// plugins available from the specified plugin groups

		// Add essential plugins
		name, version := essentials.GetEssentialsPluginGroupDetails()
		essentialPluginGroup := name
		if version != "" {
			essentialPluginGroup = fmt.Sprintf("%v:%v", essentialPluginGroup, version)
		}
		o.Groups = append(o.Groups, essentialPluginGroup)
		for _, groupID := range o.Groups {
			pluginGroups, pluginEntries, err := o.getAllPluginGroupsAndPluginEntriesFromPluginGroupVersion(groupID, pi)
			if err != nil {
				// Continue to download rest of the plugin groups if essentials is not available
				if groupID == essentialPluginGroup {
					continue
				}
				return nil, nil, err
			}
			selectedPluginGroups = append(selectedPluginGroups, pluginGroups...)
			selectedPluginEntries = append(selectedPluginEntries, pluginEntries...)
		}

		for _, pluginID := range o.Plugins {
			pluginEntry, err := o.getPluginFromPluginID(pluginID, pi)
			if err != nil {
				return nil, nil, err
			}
			selectedPluginEntries = append(selectedPluginEntries, pluginEntry...)
		}
	}

	// Remove duplicate PluginInventoryEntries and PluginGroups from the selected list
	selectedPluginEntries = plugininventory.RemoveDuplicatePluginInventoryEntries(selectedPluginEntries)
	selectedPluginGroups = plugininventory.RemoveDuplicatePluginGroups(selectedPluginGroups)

	return selectedPluginEntries, selectedPluginGroups, nil
}

func (o *DownloadPluginBundleOptions) getPluginFromPluginID(pluginID string, pi plugininventory.PluginInventory) ([]*plugininventory.PluginInventoryEntry, error) {
	pluginName, pluginTarget, pluginVersion := utils.ParsePluginID(pluginID)
	if pluginVersion == "" {
		pluginVersion = cli.VersionLatest
	}

	pluginEntries, err := pi.GetPlugins(&plugininventory.PluginInventoryFilter{
		Name:          pluginName,
		Target:        configtypes.StringToTarget(pluginTarget),
		Version:       pluginVersion,
		IncludeHidden: true,
	}) // Include the hidden plugins during plugin migration
	if err != nil {
		return nil, errors.Wrap(err, "unable to read plugins from database")
	}
	if len(pluginEntries) == 0 {
		return nil, errors.Errorf("no plugins found for pluginID %q", pluginID)
	}

	// If we get more than 1 pluginEntries, this means that provided pluginID matches with more than one plugin
	// this most likely indicates that there are more than 1 plugin name with different target and we should throw an
	// error in this scenario considering the ambiguity
	if len(pluginEntries) > 1 {
		return nil, errors.Errorf("more than one plugins found for pluginID '%s'. Please specify the uniquely identifiable pluginID in the form of 'name@target'", pluginID)
	}

	log.Infof("will be downloading the %q plugin individually", pluginID)

	return pluginEntries, nil
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
		var groupPluginsCount int
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
				// Once we get the plugin entries, deleting Artifacts specific to any extra plugin versions
				// that we got (possible when a shortened version is specified for the plugin in plugin group)
				// and only keeping the Artifacts information for the RecommendedVersion
				// We are doing this because we only want to copy the latest available version (RecommendedVersion)
				// of the each plugin to the airgapped repository instead of copy all the matched versions of the plugin
				for i := range pluginEntries {
					for version := range pluginEntries[i].Artifacts {
						if version != pluginEntries[i].RecommendedVersion {
							delete(pluginEntries[i].Artifacts, version)
						}
					}
				}
				allPluginEntries = append(allPluginEntries, pluginEntries...)
				groupPluginsCount += len(pluginEntries)
			}
		}
		groupIDWithVersion := fmt.Sprintf("%s:%s", plugininventory.PluginGroupToID(pg), pg.RecommendedVersion)
		if groupPluginsCount == 1 {
			log.Infof("will be downloading the one plugin from group: %s", groupIDWithVersion)
		} else {
			log.Infof("will be downloading the %d plugins from group: %s", groupPluginsCount, groupIDWithVersion)
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

// downloadImagesAsTarFile downloads plugin inventory image and all plugin images
// as tar file to the specified directory
//
//nolint:unparam
func (o *DownloadPluginBundleOptions) getListOfImages(pluginEntries []*plugininventory.PluginInventoryEntry) (map[string]interface{}, error) {
	images := []string{}
	images = append(images, o.PluginInventoryImage)

	// Process all plugin entries and download the oci image as tar file
	for _, pe := range pluginEntries {
		for _, artifacts := range pe.Artifacts {
			for _, a := range artifacts {
				images = append(images, a.Image)
			}
		}
	}

	metadata := make(map[string]interface{})
	metadata["images"] = images
	return metadata, nil
}

// validateOptions validates the provided options and returns
// error if contains invalid option
func (o *DownloadPluginBundleOptions) validateOptions() error {
	if !o.DryRun {
		// Verify tar file to be used to save plugin bundle
		err := o.verifyTarFile()
		if err != nil {
			return err
		}
	}

	// Verify the inventory image signature before downloading the plugin inventory database
	err := sigverifier.VerifyInventoryImageSignature(o.PluginInventoryImage)
	if err != nil {
		return err
	}

	return nil
}

func (o *DownloadPluginBundleOptions) verifyTarFile() error {
	dir := filepath.Dir(o.ToTar)
	_, err := os.Stat(dir)
	if err != nil {
		return errors.Wrapf(err, "invalid path for %q", dir)
	}
	if _, err = os.Stat(o.ToTar); err == nil {
		return fmt.Errorf(fileExists, o.ToTar)
	}
	// Check the input file path for --to-tar is valid
	var empty []byte
	err = os.WriteFile(o.ToTar, empty, 0600)
	if err == nil {
		os.Remove(o.ToTar)
		return nil
	}
	return errors.Wrapf(err, "invalid path for %q", o.ToTar)
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
