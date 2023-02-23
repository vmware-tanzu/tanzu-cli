// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"
	"path/filepath"

	"github.com/aunum/log"
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

type PublishPluginPackageImpl interface {
	BuildPluginPackages() error
}

type PublishPluginPackageOptions struct {
	PackageArtifactDir string
	Publisher          string
	Vendor             string
	Repository         string
	DryRun             bool

	pluginManifestFile string
}

func (ppo *PublishPluginPackageOptions) PublishPluginPackages() error {
	if ppo.pluginManifestFile == "" {
		ppo.pluginManifestFile = filepath.Join(ppo.PackageArtifactDir, cli.PluginManifestFileName)
	}
	if !utils.PathExists(ppo.PackageArtifactDir) {
		return errors.Errorf("invalid package artifact directory %q", ppo.PackageArtifactDir)
	}

	pluginManifest, err := readPluginManifest(ppo.pluginManifestFile)
	if err != nil {
		return err
	}

	log.Infof("Using plugin package artifacts from %q", ppo.PackageArtifactDir)

	for i := range pluginManifest.Plugins {
		for _, osArch := range cli.AllOSArch {
			for _, version := range pluginManifest.Plugins[i].Versions {
				pluginTarFilePath := filepath.Join(ppo.PackageArtifactDir, getPluginArchiveRelativePath(pluginManifest.Plugins[i], osArch, version))
				if !utils.PathExists(pluginTarFilePath) {
					continue
				}

				imageRepo := fmt.Sprintf("%s/%s/%s/%s/%s/%s", ppo.Repository, ppo.Vendor, ppo.Publisher, osArch.OS(), osArch.Arch(), pluginManifest.Plugins[i].Name)
				log.Infof("Publishing plugin 'name:%s' 'target:%s' 'os:%s' 'arch:%s' 'version:%s'", pluginManifest.Plugins[i].Name, pluginManifest.Plugins[i].Target, osArch.OS(), osArch.Arch(), version)

				if ppo.DryRun {
					log.Infof("Command: 'imgpkg copy --tar %s --to-repo %s", pluginTarFilePath, imageRepo)
				} else {
					err = copyArchiveToRepo(imageRepo, pluginTarFilePath)
					if err != nil {
						return errors.Wrapf(err, "unable to publish plugin (name:%s, target:%s, os:%s, arch:%s, version:%s)", pluginManifest.Plugins[i].Name, pluginManifest.Plugins[i].Target, osArch.OS(), osArch.Arch(), version)
					}
					log.Infof("Published plugin at '%s:%s'", imageRepo, version)
				}
			}
		}
	}

	return nil
}
