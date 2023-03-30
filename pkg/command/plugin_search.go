// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

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
				allPlugins, err = pluginmanager.DiscoverPluginsFromLocalSource(local)
				if err != nil {
					return err
				}
			} else {
				// Show plugins found in the central repos
				allPlugins, err = pluginmanager.DiscoverStandalonePlugins()
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
	outputWriter := component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Latest")

	for i := range plugins {
		outputWriter.AddRow(
			plugins[i].Name,
			plugins[i].Description,
			string(plugins[i].Target),
			plugins[i].RecommendedVersion)
	}

	outputWriter.Render()
}
