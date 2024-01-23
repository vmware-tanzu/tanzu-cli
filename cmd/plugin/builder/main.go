// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo"
)

var descriptor = plugin.PluginDescriptor{
	Name:        "builder",
	Target:      types.TargetGlobal,
	Description: "Build Tanzu components",
	Group:       plugin.AdminCmdGroup,
	Version:     buildinfo.Version,
	BuildSHA:    buildinfo.SHA,
}

func main() {
	p, err := plugin.NewPlugin(&descriptor)
	if err != nil {
		log.Fatal(err, "")
	}

	p.AddCommands(
		NewCLICmd(),
		NewInitCmd(),
		NewPluginCmd(),
		newInventoryCmd(),
	)

	if err := p.Execute(); err != nil {
		log.Fatal(err, "")
	}
}
