// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package airgapped provides helper functions to download and upload plugin bundles
package airgapped

const PluginBundleDirName = "plugin_bundle"
const PluginBundleManifestFile = "plugin_bundle_manifest.yaml"

// Manifest defines struct for plugin bundle manifest
type Manifest struct {
	Images []*ImageInfo `yaml:"images"`
}

// ImageInfo maps the relative image path and local relative file path
type ImageInfo struct {
	FilePath  string `yaml:"filePath"`
	ImagePath string `yaml:"imagePath"`
}
