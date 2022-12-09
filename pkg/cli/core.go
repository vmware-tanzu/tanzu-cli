// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo"
)

// CoreName is the name of the core binary.
const CoreName = "core"

const coreDescription = "The core Tanzu CLI"

// CoreDescriptor is the core descriptor.
var CoreDescriptor = plugin.PluginDescriptor{
	Name:        CoreName,
	Description: coreDescription,
	Version:     buildinfo.Version,
	BuildSHA:    buildinfo.SHA,
}

// CorePlugin is the core plugin.
var CorePlugin = Plugin{
	Name:        CoreName,
	Description: coreDescription,
}
