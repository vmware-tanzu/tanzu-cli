// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

var (
	groupID          string
	showNonMandatory bool
)

func newPluginGroupCmd() *cobra.Command {
	var pluginGroupCmd = &cobra.Command{
		Use:   "group",
		Short: "Manage plugin-groups",
		Long:  "Manage plugin-groups. A plugin-group provides a list of plugins name/version combinations which can be installed in one step.",
	}
	pluginGroupCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	pluginGroupCmd.AddCommand(
		newSearchCmd(),
		newGetCmd(),
	)

	return pluginGroupCmd
}

func newSearchCmd() *cobra.Command {
	var searchCmd = &cobra.Command{
		Use:   "search",
		Short: "Search for available plugin-groups",
		Long:  "Search from the list of available plugin-groups.  A plugin-group provides a list of plugin name/version combinations which can be installed in one step.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var criteria *discovery.GroupDiscoveryCriteria
			if groupID != "" {
				groupIdentifier := plugininventory.PluginGroupIdentifierFromID(groupID)
				if groupIdentifier == nil {
					return errors.Errorf("incorrect plugin-group %q specified", groupID)
				}

				criteria = &discovery.GroupDiscoveryCriteria{
					Vendor:    groupIdentifier.Vendor,
					Publisher: groupIdentifier.Publisher,
					Name:      groupIdentifier.Name,
				}
			}
			groups, err := pluginmanager.DiscoverPluginGroups(discovery.WithGroupDiscoveryCriteria(criteria))
			if err != nil {
				return err
			}

			if !showDetails {
				displayGroupsFound(groups, cmd.OutOrStdout())
			} else {
				displayGroupDetails(groups, cmd.OutOrStdout())
			}
			return nil
		},
	}

	f := searchCmd.Flags()
	f.StringVarP(&groupID, "name", "n", "", "limit the search to the plugin-group with the specified name")
	f.BoolVar(&showDetails, "show-details", false, "show the details of the specified group, including all available versions")
	f.StringVarP(&outputFormat, "output", "o", "", "output format (yaml|json|table)")

	return searchCmd
}

func newGetCmd() *cobra.Command {
	var getCmd = &cobra.Command{
		Use:   "get GROUP_NAME",
		Short: "Get the content of the specified plugin-group",
		Long:  "Get the content of the specified plugin-group.  A plugin-group provides a list of plugin name/version combinations which can be installed in one step.  This command allows to see the list of plugins included in the specified group.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := args[0]

			groupIdentifier := plugininventory.PluginGroupIdentifierFromID(groupID)
			if groupIdentifier == nil {
				return errors.Errorf("incorrect plugin-group %q specified", groupID)
			}

			if groupIdentifier.Version == "" {
				groupIdentifier.Version = cli.VersionLatest
			}

			criteria := &discovery.GroupDiscoveryCriteria{
				Vendor:    groupIdentifier.Vendor,
				Publisher: groupIdentifier.Publisher,
				Name:      groupIdentifier.Name,
				Version:   groupIdentifier.Version,
			}
			groups, err := pluginmanager.DiscoverPluginGroups(discovery.WithGroupDiscoveryCriteria(criteria))
			if err != nil {
				return err
			}
			if len(groups) == 0 {
				return errors.Errorf("plugin-group %q cannot be found", groupID)
			}

			if len(groups) > 1 {
				log.Warningf("unexpectedly found %d entries for group %q. Using the first one", len(groups), groupID)
			}

			if outputFormat == "" || outputFormat == string(component.TableOutputType) {
				displayGroupContentAsTable(groups[0], cmd.OutOrStdout())
			} else {
				displayGroupContentAsList(groups[0], cmd.OutOrStdout())
			}
			return nil
		},
	}

	f := getCmd.Flags()
	f.StringVarP(&outputFormat, "output", "o", "", "output format (yaml|json|table)")
	f.BoolVarP(&showNonMandatory, "all", "", false, "include the non-mandatory plugins")
	_ = f.MarkHidden("all")

	return getCmd
}

func displayGroupsFound(groups []*plugininventory.PluginGroup, writer io.Writer) {
	output := component.NewOutputWriterWithOptions(writer, outputFormat, []component.OutputWriterOption{}, "group", "description", "latest")

	for _, pg := range groups {
		id := plugininventory.PluginGroupToID(pg)
		output.AddRow(id, pg.Description, pg.RecommendedVersion)
	}
	output.Render()
}

func displayGroupDetails(groups []*plugininventory.PluginGroup, writer io.Writer) {
	// Create a specific object format so it gets printed properly in yaml or json
	type detailedObject struct {
		Name        string
		Description string
		Latest      string
		Versions    []string
	}

	// For the table format, we will use individual yaml output for each group
	if outputFormat == "" || outputFormat == string(component.TableOutputType) {
		first := true
		for _, pg := range groups {
			if !first {
				fmt.Println()
			}
			first = false
			var supportedVersions []string
			for version := range pg.Versions {
				supportedVersions = append(supportedVersions, version)
			}
			_ = utils.SortVersions(supportedVersions)
			details := detailedObject{
				Name:        plugininventory.PluginGroupToID(pg),
				Description: pg.Description,
				Latest:      pg.RecommendedVersion,
				Versions:    supportedVersions,
			}
			component.NewObjectWriter(writer, string(component.YAMLOutputType), details).Render()
		}

		return
	}

	// Non-table format.
	// Here we use an objectWriter so that the array of versions is printed as an array
	// and not a long string.
	var details []detailedObject
	for _, pg := range groups {
		var supportedVersions []string
		for version := range pg.Versions {
			supportedVersions = append(supportedVersions, version)
		}
		_ = utils.SortVersions(supportedVersions)
		details = append(details, detailedObject{
			Name:        plugininventory.PluginGroupToID(pg),
			Description: pg.Description,
			Latest:      pg.RecommendedVersion,
			Versions:    supportedVersions,
		})
	}
	component.NewObjectWriter(writer, outputFormat, details).Render()
}

func displayGroupContentAsTable(group *plugininventory.PluginGroup, writer io.Writer) {
	cyanBold := color.New(color.FgCyan).Add(color.Bold)
	cyanBoldItalic := color.New(color.FgCyan).Add(color.Bold, color.Italic)
	output := component.NewOutputWriterWithOptions(writer, outputFormat, []component.OutputWriterOption{}, "Name", "Target", "Version")

	groupID := plugininventory.PluginGroupToID(group)
	_, _ = cyanBold.Println("Plugins in Group: ", cyanBoldItalic.Sprintf("%s:%s", groupID, group.RecommendedVersion))

	for _, plugin := range group.Versions[group.RecommendedVersion] {
		if showNonMandatory || plugin.Mandatory {
			output.AddRow(plugin.Name, plugin.Target, plugin.Version)
		}
	}
	output.Render()
}

func displayGroupContentAsList(group *plugininventory.PluginGroup, writer io.Writer) {
	output := component.NewOutputWriterWithOptions(writer, outputFormat, []component.OutputWriterOption{}, "Group", "PluginName", "PluginTarget", "PluginVersion")

	groupID := fmt.Sprintf("%s:%s", plugininventory.PluginGroupToID(group), group.RecommendedVersion)
	for _, plugin := range group.Versions[group.RecommendedVersion] {
		if showNonMandatory || plugin.Mandatory {
			output.AddRow(groupID, plugin.Name, plugin.Target, plugin.Version)
		}
	}
	output.Render()
}
