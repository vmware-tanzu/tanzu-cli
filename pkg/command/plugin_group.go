// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
)

func newPluginGroupCmd() *cobra.Command {
	var pluginGroupCmd = &cobra.Command{
		Use:   "group",
		Short: "Manage plugin groups",
		Long:  "Manage plugin groups. A plugin group provides a list of plugins name/version combinations which can be installed in one step.",
	}
	pluginGroupCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	pluginGroupCmd.AddCommand(
		newSearchCmd(),
	)

	return pluginGroupCmd
}

func newSearchCmd() *cobra.Command {
	var searchCmd = &cobra.Command{
		Use:   "search",
		Short: "Search for available plugin groups",
		Long:  "Search from the list of available plugin groups.  A plugin group provides a list of plugin name/version combinations which can be installed in one step.",
		RunE: func(cmd *cobra.Command, args []string) error {
			output := component.NewOutputWriter(cmd.OutOrStdout(), outputFormat, "group", "source")

			groupsByDiscovery, err := pluginmanager.DiscoverPluginGroups()
			if err != nil {
				return err
			}
			for _, discAndGroups := range groupsByDiscovery {
				for _, group := range discAndGroups.Groups {
					output.AddRow(fmt.Sprintf("%s/%s", group.Publisher, group.Name), discAndGroups.Source)
				}
			}
			output.Render()

			return nil
		},
	}

	searchCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (yaml|json|table)")

	return searchCmd
}
