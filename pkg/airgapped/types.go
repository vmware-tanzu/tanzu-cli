// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package airgapped provides helper functions to download and upload plugin bundles
package airgapped

const PluginBundleDirName = "plugin_bundle"
const PluginMigrationManifestFile = "plugin_migration_manifest.yaml"

// PluginMigrationManifest defines struct for plugin bundle manifest
type PluginMigrationManifest struct {
	RelativeInventoryImagePathWithTag string            `yaml:"relativeInventoryImagePathWithTag"`
	InventoryMetadataImage            *ImagePublishInfo `yaml:"inventoryMetadataImage"`
	ImagesToCopy                      []*ImageCopyInfo  `yaml:"imagesToCopy"`
}

// ImageCopyInfo maps the relative image path and local relative file path
type ImageCopyInfo struct {
	SourceTarFilePath string `yaml:"sourceTarFilePath"`
	RelativeImagePath string `yaml:"relativeImagePath"`
}

// ImagePublishInfo maps the relative image path and local relative file path
type ImagePublishInfo struct {
	SourceFilePath           string `yaml:"sourceFilePath"`
	RelativeImagePathWithTag string `yaml:"relativeImagePathWithTag"`
}
