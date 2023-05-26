// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package common defines generic constants and structs
package common

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

var (
	// DefaultPluginRoot is the default plugin root.
	DefaultPluginRoot = filepath.Join(xdg.DataHome, "tanzu-cli")

	// DefaultCacheDir is the default cache directory
	DefaultCacheDir = filepath.Join(xdg.Home, ".cache", "tanzu")

	// DefaultLocalPluginDistroDir is the default Local plugin distribution root directory
	// This directory will be used for local discovery and local distribute of plugins
	DefaultLocalPluginDistroDir = filepath.Join(xdg.Home, ".config", "tanzu-plugins")
)

const (
	// PluginInventoryDirName is the name of the directory where the file(s) describing
	// the inventory of the discovery will be downloaded and stored.
	// It should be used as a sub-directory of the cache directory (DefaultCacheDir).
	PluginInventoryDirName = "plugin_inventory"
)
