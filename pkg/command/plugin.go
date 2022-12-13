// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aunum/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cliv1alpha1 "github.com/vmware-tanzu/tanzu-framework/apis/cli/v1alpha1"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/command"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	cliconfig "github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
)

var (
	local        string
	version      string
	forceDelete  bool
	outputFormat string
	targetStr    string
)

func newPluginCmd(ps catalog.PluginSupplier) *cobra.Command {
	var pluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Manage CLI plugins",
		Annotations: map[string]string{
			"group": string(plugin.SystemCmdGroup),
		},
	}

	pluginCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	listPluginCmd := newListPluginCmd(ps)
	installPluginCmd := newInstallPluginCmd()
	upgradePluginCmd := newUpgradePluginCmd()
	describePluginCmd := newDescribePluginCmd()
	deletePluginCmd := newDeletePluginCmd()
	cleanPluginCmd := newCleanPluginCmd()
	syncPluginCmd := newSyncPluginCmd()
	discoverySourceCmd := newDiscoverySourceCmd()

	listPluginCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (yaml|json|table)")
	listPluginCmd.Flags().StringVarP(&local, "local", "l", "", "path to local plugin source")
	installPluginCmd.Flags().StringVarP(&local, "local", "l", "", "path to local discovery/distribution source")
	installPluginCmd.Flags().StringVarP(&version, "version", "v", cli.VersionLatest, "version of the plugin")
	deletePluginCmd.Flags().BoolVarP(&forceDelete, "yes", "y", false, "delete the plugin without asking for confirmation")

	if config.IsFeatureActivated(cliconfig.FeatureContextCommand) {
		installPluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", "target of the plugin (kubernetes[k8s]/mission-control[tmc])")
		upgradePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", "target of the plugin (kubernetes[k8s]/mission-control[tmc])")
		deletePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", "target of the plugin (kubernetes[k8s]/mission-control[tmc])")
		describePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", "target of the plugin (kubernetes[k8s]/mission-control[tmc])")
	}

	// TODO(prkalle): Should be removed if we evaluate and decide to remove the repo command as it is obsolete not being used
	command.DeprecateCommand(repoCmd, "")

	pluginCmd.AddCommand(
		// TODO(prkalle): Evaluate and remove repoCmd command since it is absolete after plugin discovery changes are added
		repoCmd,
		listPluginCmd,
		installPluginCmd,
		upgradePluginCmd,
		describePluginCmd,
		deletePluginCmd,
		cleanPluginCmd,
		syncPluginCmd,
		discoverySourceCmd,
	)
	return pluginCmd
}

func newListPluginCmd(ps catalog.PluginSupplier) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Not handling plugin discovery yet, so only showing installed plugins

			descriptors, err := ps.GetInstalledPlugins()
			if err != nil {
				return err
			}

			data := [][]string{}
			for _, desc := range descriptors {
				var exists bool
				for _, d := range data {
					if desc.Name == d[0] {
						exists = true
						break
					}
				}
				if !exists {
					data = append(data, []string{desc.Name, desc.Description, desc.Version, "installed"})
				}
			}

			// sort plugins based on their names
			sort.SliceStable(data, func(i, j int) bool {
				return strings.ToLower(data[i][0]) < strings.ToLower(data[j][0])
			})

			// TODO(prkalle): uncomment/update the below rendering code once the pluginSupplier is integrated for all the commands.
			//		if config.IsFeatureActivated(cliconfig.FeatureContextCommand) && (outputFormat == "" || outputFormat == string(component.TableOutputType)) {
			//			displayPluginListOutputSplitViewContext(availablePlugins, cmd.OutOrStdout())
			//		} else {
			//			displayPluginListOutputListView(availablePlugins, cmd.OutOrStdout())
			//		}
			output := component.NewOutputWriter(cmd.OutOrStdout(), outputFormat, "Name", "Description", "Version", "Status")
			for _, row := range data {
				vals := make([]interface{}, len(row))
				for i, val := range row {
					vals[i] = val
				}
				output.AddRow(vals...)
			}
			output.Render()
			return nil
		},
	}

	return listCmd
}

func newDescribePluginCmd() *cobra.Command {
	var describeCmd = &cobra.Command{
		Use:   "describe [name]",
		Short: "Describe a plugin",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 1 {
				return fmt.Errorf("must provide plugin name as positional argument")
			}
			pluginName := args[0]

			if !cliv1alpha1.IsValidTarget(targetStr) {
				return errors.New("invalid target specified. Please specify correct value of `--target` or `-t` flag from 'kubernetes/k8s/mission-control/tmc'")
			}

			pd, err := pluginmanager.DescribePlugin(pluginName, getTarget())
			if err != nil {
				return err
			}

			b, err := yaml.Marshal(pd)
			if err != nil {
				return errors.Wrap(err, "could not marshal plugin")
			}
			fmt.Println(string(b))
			return nil
		},
	}

	return describeCmd
}

func newInstallPluginCmd() *cobra.Command {
	var installCmd = &cobra.Command{
		Use:   "install [name]",
		Short: "Install a plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			pluginName := args[0]

			if !cliv1alpha1.IsValidTarget(targetStr) {
				return errors.New("invalid target specified. Please specify correct value of `--target` or `-t` flag from 'kubernetes/k8s/mission-control/tmc'")
			}

			// Invoke install plugin from local source if local files are provided
			if local != "" {
				// get absolute local path
				local, err = filepath.Abs(local)
				if err != nil {
					return err
				}
				err = pluginmanager.InstallPluginsFromLocalSource(pluginName, version, getTarget(), local, false)
				if err != nil {
					return err
				}
				if pluginName == cli.AllPlugins {
					log.Successf("successfully installed all plugins")
				} else {
					log.Successf("successfully installed '%s' plugin", pluginName)
				}
				return nil
			}

			// Invoke plugin sync if install all plugins is mentioned
			if pluginName == cli.AllPlugins {
				err = pluginmanager.SyncPlugins()
				if err != nil {
					return err
				}
				log.Successf("successfully installed all plugins")
				return nil
			}

			pluginVersion := version
			if pluginVersion == cli.VersionLatest {
				pluginVersion, err = pluginmanager.GetRecommendedVersionOfPlugin(pluginName, getTarget())
				if err != nil {
					return err
				}
			}

			err = pluginmanager.InstallPlugin(pluginName, pluginVersion, getTarget())
			if err != nil {
				return err
			}
			log.Successf("successfully installed '%s' plugin", pluginName)
			return nil
		},
	}

	return installCmd
}

func newUpgradePluginCmd() *cobra.Command {
	var upgradeCmd = &cobra.Command{
		Use:   "upgrade [name]",
		Short: "Upgrade a plugin",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 1 {
				return fmt.Errorf("must provide plugin name as positional argument")
			}
			pluginName := args[0]

			if !cliv1alpha1.IsValidTarget(targetStr) {
				return errors.New("invalid target specified. Please specify correct value of `--target` or `-t` flag from 'kubernetes/k8s/mission-control/tmc'")
			}

			pluginVersion, err := pluginmanager.GetRecommendedVersionOfPlugin(pluginName, getTarget())
			if err != nil {
				return err
			}

			err = pluginmanager.UpgradePlugin(pluginName, pluginVersion, getTarget())
			if err != nil {
				return err
			}
			log.Successf("successfully upgraded plugin '%s' to version '%s'", pluginName, pluginVersion)
			return nil
		},
	}

	return upgradeCmd
}

func newDeletePluginCmd() *cobra.Command {
	var deleteCmd = &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a plugin",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 1 {
				return fmt.Errorf("must provide plugin name as positional argument")
			}
			pluginName := args[0]

			if !cliv1alpha1.IsValidTarget(targetStr) {
				return errors.New("invalid target specified. Please specify correct value of `--target` or `-t` flag from 'kubernetes/k8s/mission-control/tmc'")
			}

			deletePluginOptions := pluginmanager.DeletePluginOptions{
				PluginName:  pluginName,
				Target:      getTarget(),
				ForceDelete: forceDelete,
			}

			err = pluginmanager.DeletePlugin(deletePluginOptions)
			if err != nil {
				return err
			}

			log.Successf("successfully deleted plugin '%s'", pluginName)
			return nil
		},
	}
	return deleteCmd
}

func newCleanPluginCmd() *cobra.Command {
	var cleanCmd = &cobra.Command{
		Use:   "clean",
		Short: "Clean the plugins",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			err = pluginmanager.Clean()
			if err != nil {
				return err
			}
			log.Success("successfully cleaned up all plugins")
			return nil
		},
	}
	return cleanCmd
}

func newSyncPluginCmd() *cobra.Command {
	var syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Sync the plugins",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			err = pluginmanager.SyncPlugins()
			if err != nil {
				return err
			}
			log.Success("Done")
			return nil
		},
	}
	return syncCmd
}

func getTarget() cliv1alpha1.Target {
	return cliv1alpha1.StringToTarget(strings.ToLower(targetStr))
}
