// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/aunum/log"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
)

var descriptor = plugin.PluginDescriptor{
	Name:        "test",
	Description: "Test the CLI",
	Group:       plugin.AdminCmdGroup,
	Version:     buildinfo.Version,
	BuildSHA:    buildinfo.SHA,
}

var local string

func init() {
	fetchCmd.Flags().StringVarP(&local, "local", "l", "", "path to local repository")
	_ = fetchCmd.MarkFlagRequired("local")
}

func main() {
	p, err := plugin.NewPlugin(&descriptor)
	if err != nil {
		log.Fatal(err)
	}

	p.AddCommands(
		fetchCmd,
		pluginsCmd,
	)

	installedPlugins, err := pluginsupplier.GetInstalledStandalonePlugins()
	if err != nil {
		log.Fatal(err)
	}

	for i := range installedPlugins {
		// Check if test plugin binary installed. If available add a plugin command
		_, err := os.Stat(cli.TestPluginPathFromPluginPath(installedPlugins[i].InstallationPath))
		if err != nil {
			continue
		}
		pluginsCmd.AddCommand(cli.GetTestCmdForPlugin(&installedPlugins[i]))
	}

	if err := p.Execute(); err != nil {
		log.Fatal(err)
	}
}

var pluginsCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Plugin tests",
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch the plugin tests",
	RunE: func(cmd *cobra.Command, args []string) error {
		return pluginmanager.InstallPluginsFromLocalSource("all", "", "", local, true)
	},
}
