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
	// TODO(vuil): reconsolidate back to legacy plugin location
	DefaultPluginRoot = filepath.Join(xdg.DataHome, ".tanzu-cli")
)
