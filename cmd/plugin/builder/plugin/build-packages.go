// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugin implements plugin specific publishing functions
package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aunum/log"
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/imgpkg"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

type BuildPluginPackageOptions struct {
	BinaryArtifactDir  string
	PackageArtifactDir string
	LocalOCIRegistry   string
	ImgpkgOptions      imgpkg.ImgpkgWrapper

	pluginManifestFile string
}

func (bpo *BuildPluginPackageOptions) BuildPluginPackages() error {
	if bpo.pluginManifestFile == "" {
		bpo.pluginManifestFile = filepath.Join(bpo.BinaryArtifactDir, cli.PluginManifestFileName)
	}
	if !utils.PathExists(bpo.PackageArtifactDir) {
		err := os.MkdirAll(bpo.PackageArtifactDir, 0755)
		if err != nil {
			return err
		}
	}

	pluginManifest, err := helpers.ReadPluginManifest(bpo.pluginManifestFile)
	if err != nil {
		return err
	}

	log.Infof("Using plugin binary artifacts from %q", bpo.BinaryArtifactDir)

	for i := range pluginManifest.Plugins {
		for _, osArch := range cli.AllOSArch {
			for _, version := range pluginManifest.Plugins[i].Versions {
				pluginBinaryFilePath := filepath.Join(bpo.BinaryArtifactDir, osArch.OS(), osArch.Arch(),
					pluginManifest.Plugins[i].Target, pluginManifest.Plugins[i].Name, version,
					cli.MakeArtifactName(pluginManifest.Plugins[i].Name, osArch))

				if !utils.PathExists(pluginBinaryFilePath) {
					continue
				}

				pluginTarFilePath := filepath.Join(bpo.PackageArtifactDir, helpers.GetPluginArchiveRelativePath(pluginManifest.Plugins[i], osArch, version))
				image := fmt.Sprintf("%s/plugins/%s/%s/%s:%s", bpo.LocalOCIRegistry, osArch.OS(), osArch.Arch(), pluginManifest.Plugins[i].Name, version)

				log.Infof("Generating plugin package for 'plugin:%s' 'target:%s' 'os:%s' 'arch:%s' 'version:%s'", pluginManifest.Plugins[i].Name, pluginManifest.Plugins[i].Target, osArch.OS(), osArch.Arch(), version)

				err := bpo.ImgpkgOptions.PushImage(image, pluginBinaryFilePath)
				if err != nil {
					return errors.Wrapf(err, "unable to push package to temporary registry for plugin: %s, target: %s, os: %s, arch: %s, version: %s", pluginManifest.Plugins[i].Name, pluginManifest.Plugins[i].Target, osArch.OS(), osArch.Arch(), version)
				}

				err = bpo.ImgpkgOptions.CopyImageToArchive(image, pluginTarFilePath)
				if err != nil {
					return errors.Wrapf(err, "unable to generate package for plugin: %s, target: %s, os: %s, arch: %s, version: %s", pluginManifest.Plugins[i].Name, pluginManifest.Plugins[i].Target, osArch.OS(), osArch.Arch(), version)
				}

				log.Infof("Generated plugin package at %q", pluginTarFilePath)
			}
		}
	}

	// copy plugin_manifest.yaml to PackageArtifactDir
	err = utils.CopyFile(bpo.pluginManifestFile, filepath.Join(bpo.PackageArtifactDir, cli.PluginManifestFileName))
	if err != nil {
		return errors.Wrap(err, "unable to copy plugin manifest file")
	}
	log.Infof("Saved plugin manifest at %q", filepath.Join(bpo.PackageArtifactDir, cli.PluginManifestFileName))

	return nil
}
