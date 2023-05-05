// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
)

var (
	groupID string
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
			var criteria *discovery.GroupDiscoveryCriteria
			if groupID != "" {
				groupIdentifier := plugininventory.PluginGroupIdentifierFromID(groupID)
				criteria = &discovery.GroupDiscoveryCriteria{
					Vendor:    groupIdentifier.Vendor,
					Publisher: groupIdentifier.Publisher,
					Name:      groupIdentifier.Name,
				}
			}
			groupsByDiscovery, err := pluginmanager.DiscoverPluginGroups(criteria)
			if err != nil {
				return err
			}

			displayGroupsFound(groupsByDiscovery, cmd.OutOrStdout())

			return nil
		},
	}

	f := searchCmd.Flags()
	f.StringVarP(&groupID, "name", "n", "", "limit the search to the plugin group with the specified name")
	f.StringVarP(&outputFormat, "output", "o", "", "output format (yaml|json|table)")

	return searchCmd
}

func displayGroupsFound(groupsByDiscovery []*discovery.DiscoveredPluginGroups, writer io.Writer) {
	output := component.NewOutputWriter(writer, outputFormat, "group")

	discoveriesByGroupID := make(map[string][]string)
	for _, discAndGroups := range groupsByDiscovery {
		for _, group := range discAndGroups.Groups {
			id := plugininventory.PluginGroupToID(group)
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
}
