// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package helpers implements helper function for builder plugin
package helpers

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// ReadPluginManifest reads the PluginManifest file and returns Manifest object
func ReadPluginManifest(pluginManifestFile string) (*cli.Manifest, error) {
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

// ReadPluginGroupManifest reads the PluginGroupManifest file and returns PluginGroupManifest object
func ReadPluginGroupManifest(pluginGroupManifestFile string) (*cli.PluginGroupManifest, error) {
	data, err := os.ReadFile(pluginGroupManifestFile)
	if err != nil {
		return nil, errors.Wrap(err, "fail to read the plugin-group manifest file")
	}

	pluginGroupManifest := &cli.PluginGroupManifest{}
	err = yaml.Unmarshal(data, pluginGroupManifest)
	if err != nil {
		return nil, errors.Wrap(err, "fail to read the plugin-group manifest file")
	}
	return pluginGroupManifest, nil
}

// GetPluginArchiveRelativePath creates plugin archive relative path from metadata
func GetPluginArchiveRelativePath(plugin cli.Plugin, osArch cli.Arch, version string) string {
	pluginTarFileName := fmt.Sprintf("%s-%s.tar.gz", plugin.Name, osArch.String())
	return filepath.Join(osArch.OS(), osArch.Arch(), plugin.Target, plugin.Name, version, pluginTarFileName)
}

// GetDigest computes the sha256 digest of the specified file
func GetDigest(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
