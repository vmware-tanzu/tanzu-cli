// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides commands
package command

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// NewRootCmd creates a root command.
func NewRootCmd(ps catalog.PluginSupplier) (*cobra.Command, error) {
	var rootCmd = &cobra.Command{
		Use: "tanzu",
		// Don't have Cobra print the error message, the CLI will
		// print it itself in a nicer format.
		SilenceErrors: true,
	}

	uFunc := cli.NewMainUsage().UsageFunc()
	rootCmd.SetUsageFunc(uFunc)
	rootCmd.AddCommand(
		newVersionCmd(),
		newPluginCmd(ps),
		loginCmd,
	)

	if ps != nil {
		plugins, err := ps.GetInstalledPlugins()
		if err != nil {
			return nil, err
		}
		for _, plugin := range plugins {
			rootCmd.AddCommand(cli.GetPluginCmd(plugin))
		}
	}

	// If the context-command feature is enabled add it under root.
	// TODO(prkalle): Comment the below line after "config" package is moved
	const FeatureContextCommand = "features.global.context-target"
	// TODO(prkalle): Uncomment the below line and remove the line after the below line after "config" package is moved
	// if config.IsFeatureActivated(config.FeatureContextCommand) {
	if config.IsFeatureActivated(FeatureContextCommand) {
		rootCmd.AddCommand(contextCmd)
		// TODO(prkalle): Add the target related commands and changes after "pluginmanager" package is moved.
	}
	return rootCmd, nil
}

// Execute executes the CLI.
func Execute() error {
	root, err := NewRootCmd(&FlatDirPluginSupplier{})
	if err != nil {
		return err
	}
	return root.Execute()
}
