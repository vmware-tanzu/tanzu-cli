// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
)

var (
	listVersions bool
)

const searchLongDesc = `Search provides the ability to search for plugins that can be installed.
Without an argument, the command lists all plugins currently available.
The search command can also be used with a keyword filter to filter the
list of available plugins. If the filter is flanked with slashes, the
filter will be treated as a regex.
`

func newSearchPluginCmd() *cobra.Command {
	var searchCmd = &cobra.Command{
		Use:               "search [keyword|/regex/]",
		Short:             "Search for a keyword or regex in the list of available plugins",
		Long:              searchLongDesc,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(khouzam): Implement the below flags
			if len(args) == 1 {
				return fmt.Errorf("the filter argument is not yet implemented")
			}
			if len(targetStr) > 0 {
				return fmt.Errorf("filtering by target is not yet implemented")
			}
			if listVersions {
				return fmt.Errorf("listing versions is not yet implemented")
			}

			var err error
			var allPlugins []discovery.Discovered
			if local != "" {
				// The user requested the list of plugins from a local path
				local, err = filepath.Abs(local)
				if err != nil {
					return err
				}
				allPlugins, err = pluginmanager.AvailablePluginsFromLocalSource(local)
				if err != nil {
					return err
				}
			} else {
				allPlugins, err = pluginmanager.AvailablePlugins()
				if err != nil {
					return err
				}
			}
			sort.Sort(discovery.DiscoveredSorter(allPlugins))
			displayPluginsFound(allPlugins, cmd.OutOrStdout())

			return nil
		},
	}

	f := searchCmd.Flags()
	f.BoolVar(&listVersions, "list-versions", false, "show the long listing, with each available version of plugins")
	f.StringVarP(&outputFormat, "output", "o", "", "Output format (yaml|json|table)")
	f.StringVarP(&local, "local", "l", "", "path to local plugin source")
	f.StringVarP(&targetStr, "target", "t", "", "list plugins for the specified target (kubernetes[k8s]/mission-control[tmc])")

	return searchCmd
}

func displayPluginsFound(plugins []discovery.Discovered, writer io.Writer) {
	outputWriter := component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Version", "Status", "Context")

	for i := range plugins {
		context := ""
		if plugins[i].Scope == common.PluginScopeContext {
			context = plugins[i].ContextName
		}

		version := plugins[i].RecommendedVersion
		status := common.PluginStatusNotInstalled
		if plugins[i].Status == common.PluginStatusInstalled ||
			plugins[i].Status == common.PluginStatusUpdateAvailable {
			version = plugins[i].InstalledVersion

			// TODO(khouzam): For the moment, only show the plugin as installed.
			// Please see https://github.com/vmware-tanzu/tanzu-cli/issues/65
			status = common.PluginStatusInstalled
		}

		outputWriter.AddRow(
			plugins[i].Name,
			plugins[i].Description,
			string(plugins[i].Target),
			version,
			status,
			context)
	}

	outputWriter.Render()
}
