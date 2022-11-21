// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides commands
package command

import (
	"github.com/spf13/cobra"

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
	return rootCmd, nil
}

// Execute executes the CLI.
func Execute() error {
	root, err := NewRootCmd(nil)
	if err != nil {
		return err
	}
	return root.Execute()
}
