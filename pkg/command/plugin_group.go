// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
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
			output := component.NewOutputWriter(cmd.OutOrStdout(), outputFormat, "group")

			groupsByDiscovery, err := pluginmanager.DiscoverPluginGroups()
			if err != nil {
				return err
			}

			discoveriesByGroupID := make(map[string][]string)
			for _, discAndGroups := range groupsByDiscovery {
				for _, group := range discAndGroups.Groups {
					id := fmt.Sprintf("%s-%s/%s", group.Vendor, group.Publisher, group.Name)
					output.AddRow(id)
					discoveriesByGroupID[id] = append(discoveriesByGroupID[id], discAndGroups.Source)
				}
			}

			// Check if one or more groups was discovered in different discoveries.
			var duplicateMsg string
			for id, discoveries := range discoveriesByGroupID {
				if len(discoveries) > 1 {
					// This group was found in multiple discoveries.
					if duplicateMsg != "" {
						duplicateMsg = fmt.Sprintf("%s, ", duplicateMsg)
					}
					duplicateMsg = fmt.Sprintf("%s%s was found in more than one source: %v", duplicateMsg, id, discoveries)
				}
			}
			if duplicateMsg != "" {
				log.Warning(duplicateMsg)
			}
			output.Render()
			return nil
		},
	}

	searchCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (yaml|json|table)")

	return searchCmd
}
