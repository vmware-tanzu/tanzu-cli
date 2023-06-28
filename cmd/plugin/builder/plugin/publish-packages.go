// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/crane"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

type PublishPluginPackageOptions struct {
	PackageArtifactDir string
	Publisher          string
	Vendor             string
	Repository         string
	DryRun             bool
	CraneOptions       crane.CraneWrapper

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

	// Limit the number of concurrent operations we perform so we don't overwhelm the system.
	maxConcurrent := helpers.GetMaxParallelism()
	guard := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	// Mix up IDs so we don't always get the same set.
	randSkew := rand.Intn(len(helpers.Identifiers)) // nolint:gosec
	errorList := []error{}

	publishPluginPackage := func(p cli.Plugin, osArch cli.Arch, version, id string) {
		defer func() {
			<-guard
			wg.Done()
		}()

		pluginTarFilePath := filepath.Join(ppo.PackageArtifactDir, helpers.GetPluginArchiveRelativePath(p, osArch, version))
		if !utils.PathExists(pluginTarFilePath) {
			return
		}

		imageToPush := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s:%s", ppo.Repository, ppo.Vendor, ppo.Publisher, osArch.OS(), osArch.Arch(), p.Target, p.Name, version)

		if ppo.DryRun {
			log.Infof("command: 'crane push %s %s'", pluginTarFilePath, imageToPush)
		} else {
			log.Infof("publishing plugin 'name:%s' 'target:%s' 'os:%s' 'arch:%s' 'version:%s'", p.Name, p.Target, osArch.OS(), osArch.Arch(), version)
			err = ppo.CraneOptions.PushImage(pluginTarFilePath, imageToPush)
			if err != nil {
				errorList = append(errorList, errors.Wrapf(err, "unable to publish plugin (name:%s, target:%s, os:%s, arch:%s, version:%s)", p.Name, p.Target, osArch.OS(), osArch.Arch(), version))
				return
			}
			log.Infof("published plugin at '%s'", imageToPush)
		}
	}

	id := 0
	for i := range pluginManifest.Plugins {
		for _, osArch := range cli.AllOSArch {
			for _, version := range pluginManifest.Plugins[i].Versions {
				wg.Add(1)
				guard <- struct{}{}
				go publishPluginPackage(pluginManifest.Plugins[i], osArch, version, helpers.GetID(id+randSkew))
				id++
			}
		}
	}

	// wait for all WaitGroup to complete before continuing
	wg.Wait()

	return kerrors.NewAggregate(errorList)
}
