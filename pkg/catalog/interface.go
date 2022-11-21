// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package catalog ...
package catalog

import (
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// PluginSupplier is responsible for keeping an inventory of installed plugins
type PluginSupplier interface {

	// GetInstalledPlugins returns a list of installed plugins
	GetInstalledPlugins() ([]*cli.PluginInfo, error)
}
