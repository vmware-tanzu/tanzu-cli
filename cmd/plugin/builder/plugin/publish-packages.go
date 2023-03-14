// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/imgpkg"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

type PublishPluginPackageOptions struct {
	PackageArtifactDir string
	Publisher          string
	Vendor             string
	Repository         string
	DryRun             bool
	ImgpkgOptions      imgpkg.ImgpkgWrapper

	pluginManifestFile string
}

func (ppo *PublishPluginPackageOptions) PublishPluginPackages() error {
	if ppo.pluginManifestFile == "" {
		ppo.pluginManifestFile = filepath.Join(ppo.PackageArtifactDir, cli.PluginManifestFileName)
	}
	if !utils.PathExists(ppo.PackageArtifactDir) {
		return errors.Errorf("invalid package artifact directory %q", ppo.PackageArtifactDir)
	}

	pluginManifest, err := helpers.ReadPluginManifest(ppo.pluginManifestFile)
	if err != nil {
		return err
	}

	log.Infof("using plugin package artifacts from %q", ppo.PackageArtifactDir)

	for i := range pluginManifest.Plugins {
		for _, osArch := range cli.AllOSArch {
			for _, version := range pluginManifest.Plugins[i].Versions {
				pluginTarFilePath := filepath.Join(ppo.PackageArtifactDir, helpers.GetPluginArchiveRelativePath(pluginManifest.Plugins[i], osArch, version))
				if !utils.PathExists(pluginTarFilePath) {
					continue
				}

				imageRepo := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", ppo.Repository, ppo.Vendor, ppo.Publisher, osArch.OS(), osArch.Arch(), pluginManifest.Plugins[i].Target, pluginManifest.Plugins[i].Name)
				log.Infof("publishing plugin 'name:%s' 'target:%s' 'os:%s' 'arch:%s' 'version:%s'", pluginManifest.Plugins[i].Name, pluginManifest.Plugins[i].Target, osArch.OS(), osArch.Arch(), version)

				if ppo.DryRun {
					log.Infof("command: 'imgpkg copy --tar %s --to-repo %s", pluginTarFilePath, imageRepo)
				} else {
					err = ppo.ImgpkgOptions.CopyArchiveToRepo(imageRepo, pluginTarFilePath)
					if err != nil {
						return errors.Wrapf(err, "unable to publish plugin (name:%s, target:%s, os:%s, arch:%s, version:%s)", pluginManifest.Plugins[i].Name, pluginManifest.Plugins[i].Target, osArch.OS(), osArch.Arch(), version)
					}
					log.Infof("published plugin at '%s:%s'", imageRepo, version)
				}
			}
		}
	}

	return nil
}
