// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugin implements business logic for `builder plugin` command
package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// getPluginManifest reads the PluginManifest file and returns Manifest object
func readPluginManifest(pluginManifestFile string) (*cli.Manifest, error) {
	data, err := os.ReadFile(pluginManifestFile)
	if err != nil {
		return nil, errors.Wrap(err, "fail to read the plugin manifest file")
	}

	pluginManifest := &cli.Manifest{}
	err = yaml.Unmarshal(data, pluginManifest)
	if err != nil {
		return nil, errors.Wrap(err, "fail to read the plugin manifest file")
	}
	return pluginManifest, nil
}

func pushImage(image, filePath string) error {
	output, err := exec.Command("imgpkg", "push", "-i", image, "-f", filePath).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

func copyArchiveToRepo(imageRepo, archivePath string) error {
	output, err := exec.Command("imgpkg", "copy", "--tar", archivePath, "--to-repo", imageRepo).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

func copyImageToArchive(image, archivePath string) error {
	err := os.MkdirAll(filepath.Dir(archivePath), 0755)
	if err != nil {
		return err
	}

	output, err := exec.Command("imgpkg", "copy", "-i", image, "--to-tar", archivePath).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

func getPluginArchiveRelativePath(plugin cli.Plugin, osArch cli.Arch, version string) string {
	pluginTarFileName := fmt.Sprintf("%s-%s.tar.gz", plugin.Name, osArch.String())
	return filepath.Join(osArch.OS(), osArch.Arch(), plugin.Target, plugin.Name, version, pluginTarFileName)
}
