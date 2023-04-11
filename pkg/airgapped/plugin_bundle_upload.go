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
	err = tarinator.UnTarinate(tempDir, o.Tar)
	if err != nil {
		return errors.Wrap(err, "unable to untar provided file")
	}

	// Read the plugin_bundle_manifest file
	pluginBundleDir := filepath.Join(tempDir, PluginBundleDirName)
	bytes, err := os.ReadFile(filepath.Join(pluginBundleDir, PluginBundleManifestFile))
	if err != nil {
		return errors.Wrap(err, "error while reading plugin bundle manifest")
	}

	manifest := &Manifest{}
	err = yaml.Unmarshal(bytes, &manifest)
	if err != nil {
		return errors.Wrap(err, "error while parsing plugin bundle manifest")
	}

	// Iterate through all the images and publish them to the remote repository
	for _, pi := range manifest.Images {
		imageTar := filepath.Join(pluginBundleDir, pi.FilePath)
		repoImagePath := filepath.Join(o.DestinationRepo, pi.ImagePath)
		log.Infof("---------------------------")
		log.Infof("uploading image %q", repoImagePath)
		err = o.ImageProcessor.CopyImageFromTar(imageTar, repoImagePath)
		if err != nil {
			return errors.Wrap(err, "error while uploading image")
		}
	}
	log.Infof("---------------------------")
	log.Infof("successfully published all images")

	return nil
}
