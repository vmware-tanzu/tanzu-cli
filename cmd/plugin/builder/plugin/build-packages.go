// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugin implements plugin specific publishing functions
package plugin

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/crane"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

type BuildPluginPackageOptions struct {
	BinaryArtifactDir  string
	PackageArtifactDir string
	LocalOCIRegistry   string
	CraneOptions       crane.CraneWrapper

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

	// Limit the number of concurrent operations we perform so we don't overwhelm the system.
	maxConcurrent := helpers.GetMaxParallelism()
	guard := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	// Mix up IDs so we don't always get the same set.
	randSkew := rand.Intn(len(helpers.Identifiers)) // nolint:gosec
	errorList := []error{}

	generatePluginPackage := func(p cli.Plugin, osArch cli.Arch, version, id string) {
		defer func() {
			<-guard
			wg.Done()
		}()

		pluginBinaryFilePath := filepath.Join(bpo.BinaryArtifactDir, osArch.OS(), osArch.Arch(),
			p.Target, p.Name, version,
			cli.MakeArtifactName(p.Name, osArch))

		if !utils.PathExists(pluginBinaryFilePath) {
			return
		}

		// Check whether the binary exec file is valid
		valid, err := helpers.ValidatePluginBinary(pluginBinaryFilePath)

		// Return err if plugin binary file validation fails
		if err != nil {
			errorList = append(errorList, err)
			return
		}

		// Return err if plugin binary file is not valid
		if !valid {
			errorList = append(errorList, fmt.Errorf("invalid plugin binary :%v", pluginBinaryFilePath))
			return
		}

		pluginTarFilePath := filepath.Join(bpo.PackageArtifactDir, helpers.GetPluginArchiveRelativePath(p, osArch, version))
		image := fmt.Sprintf("%s/plugins/%s/%s/%s:%s", bpo.LocalOCIRegistry, osArch.OS(), osArch.Arch(), p.Name, version)

		log.Infof("Generating plugin package for 'plugin:%s' 'target:%s' 'os:%s' 'arch:%s' 'version:%s'", p.Name, p.Target, osArch.OS(), osArch.Arch(), version)

		err = carvelhelpers.NewImageOperationsImpl().PushImage(image, []string{pluginBinaryFilePath})
		if err != nil {
			errorList = append(errorList, errors.Wrapf(err, "unable to push package to temporary registry for plugin: %s, target: %s, os: %s, arch: %s, version: %s", p.Name, p.Target, osArch.OS(), osArch.Arch(), version))
			return
		}

		err = bpo.CraneOptions.SaveImage(image, pluginTarFilePath)
		if err != nil {
			errorList = append(errorList, errors.Wrapf(err, "unable to generate package for plugin: %s, target: %s, os: %s, arch: %s, version: %s", p.Name, p.Target, osArch.OS(), osArch.Arch(), version))
			return
		}

		log.Infof("Generated plugin package at %q", pluginTarFilePath)
	}

	id := 0
	for i := range pluginManifest.Plugins {
		for _, osArch := range cli.AllOSArch {
			for _, version := range pluginManifest.Plugins[i].Versions {
				wg.Add(1)
				guard <- struct{}{}
				go generatePluginPackage(pluginManifest.Plugins[i], osArch, version, helpers.GetID(id+randSkew))
				id++
			}
		}
	}

	wg.Wait()

	if len(errorList) != 0 {
		return kerrors.NewAggregate(errorList)
	}

	// copy plugin_manifest.yaml to PackageArtifactDir
	err = utils.CopyFile(bpo.pluginManifestFile, filepath.Join(bpo.PackageArtifactDir, cli.PluginManifestFileName))
	if err != nil {
		return errors.Wrap(err, "unable to copy plugin manifest file")
	}
	log.Infof("Saved plugin manifest at %q", filepath.Join(bpo.PackageArtifactDir, cli.PluginManifestFileName))

	return nil
}
