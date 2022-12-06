// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package common defines generic constants and structs
package common

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

var (
	// TODO(vuil): all directories prefixed with '_', but are expected to be
	// reconsolidated back to their unprefixed equivalents once backward
	// compatibility is achieved.

	// DefaultPluginRoot is the default plugin root.
	DefaultPluginRoot = filepath.Join(xdg.DataHome, "_tanzu-cli")

	// DefaultCacheDir is the default cache directory
	DefaultCacheDir = filepath.Join(xdg.Home, ".cache", "_tanzu")

	// DefaultLocalPluginDistroDir is the default Local plugin distribution root directory
	// This directory will be used for local discovery and local distribute of plugins
	DefaultLocalPluginDistroDir = filepath.Join(xdg.Home, ".config", "_tanzu-plugins")
)
