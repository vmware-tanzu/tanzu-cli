// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides commands
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	cliv1alpha1 "github.com/vmware-tanzu/tanzu-framework/apis/cli/v1alpha1"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	cliconfig "github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
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
		initCmd,
		completionCmd,
		configCmd,
		genAllDocsCmd,
	)

	if ps != nil {
		plugins, err := ps.GetInstalledPlugins()
		if err != nil {
			return nil, err
		}
		for _, plugin := range plugins {
			rootCmd.AddCommand(cli.GetCmdForPlugin(plugin))
		}
	}

	// If the context and target feature is enabled, add the corresponding commands under root.
	if config.IsFeatureActivated(cliconfig.FeatureContextCommand) {
		rootCmd.AddCommand(
			contextCmd,
			k8sCmd,
			tmcCmd,
		)
		mapTargetToCmd := map[cliv1alpha1.Target]*cobra.Command{
			cliv1alpha1.TargetK8s: k8sCmd,
			cliv1alpha1.TargetTMC: tmcCmd,
		}
		if err := addPluginsToTarget(mapTargetToCmd); err != nil {
			return nil, err
		}
	}
	return rootCmd, nil
}

var k8sCmd = &cobra.Command{
	Use:     "kubernetes",
	Short:   "Tanzu CLI plugins that target a Kubernetes cluster",
	Aliases: []string{"k8s"},
	Annotations: map[string]string{
		"group": string(plugin.TargetCmdGroup),
	},
}

var tmcCmd = &cobra.Command{
	Use:     "mission-control",
	Short:   "Tanzu CLI plugins that target a Tanzu Mission Control endpoint",
	Aliases: []string{"tmc"},
	Annotations: map[string]string{
		"group": string(plugin.TargetCmdGroup),
	},
}

func addPluginsToTarget(mapTargetToCmd map[cliv1alpha1.Target]*cobra.Command) error {
	installedPlugins, standalonePlugins, err := pluginmanager.InstalledPlugins()
	if err != nil {
		return fmt.Errorf("unable to find installed plugins: %w", err)
	}

	installedPlugins = append(installedPlugins, standalonePlugins...)

	for i := range installedPlugins {
		if cmd, exists := mapTargetToCmd[installedPlugins[i].Target]; exists {
			cmd.AddCommand(cli.GetCmdForPlugin(&installedPlugins[i]))
		}
	}
	return nil
}

// Execute executes the CLI.
func Execute() error {
	root, err := NewRootCmd(&FlatDirPluginSupplier{})
	if err != nil {
		return err
	}
	return root.Execute()
}
