// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

var (
	uri string
)

func newDiscoverySourceCmd() *cobra.Command {
	var discoverySourceCmd = &cobra.Command{
		Use:   "source",
		Short: "Manage plugin discovery sources",
		Long:  "Manage plugin discovery sources. Discovery source provides metadata about the list of available plugins, their supported versions and how to download them.",
	}
	discoverySourceCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	discoverySourceCmd.AddCommand(
		newListDiscoverySourceCmd(),
		newUpdateDiscoverySourceCmd(),
		newDeleteDiscoverySourceCmd(),
		newInitDiscoverySourceCmd(),
	)

	return discoverySourceCmd
}

func newListDiscoverySourceCmd() *cobra.Command {
	var listDiscoverySourceCmd = &cobra.Command{
		Use:               "list",
		Short:             "List available discovery sources",
		ValidArgsFunction: noMoreCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			output := component.NewOutputWriterWithOptions(cmd.OutOrStdout(), outputFormat, []component.OutputWriterOption{}, "name", "image")
			discoverySources, err := configlib.GetCLIDiscoverySources()
			for _, ds := range discoverySources {
				if ds.OCI != nil {
					output.AddRow(ds.OCI.Name, ds.OCI.Image)
				}
			}
			testPluginSources := pluginmanager.GetAdditionalTestPluginDiscoveries()
			for _, ds := range testPluginSources {
				if ds.OCI != nil {
					output.AddRow(ds.OCI.Name+" (test only)", ds.OCI.Image)
				}
			}
			output.Render()
			return err
		},
	}

	listDiscoverySourceCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (yaml|json|table)")
	utils.PanicOnErr(listDiscoverySourceCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))

	return listDiscoverySourceCmd
}

func newUpdateDiscoverySourceCmd() *cobra.Command {
	var updateDiscoverySourceCmd = &cobra.Command{
		Use:   "update SOURCE_NAME --uri <URI>",
		Short: "Update a discovery source configuration",
		// We already include the only flag in the use text,
		// we therefore don't show '[flags]' in the usage text.
		DisableFlagsInUseLine: true,
		Example: `
    # Update the discovery source for an air-gapped scenario. The URI must be an OCI image.
    tanzu plugin source update default --uri registry.example.com/tanzu/plugin-inventory:latest`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeUpdateDiscoverySource,
		RunE: func(cmd *cobra.Command, args []string) error {
			discoveryName := args[0]

			discoverySource, _ := configlib.GetCLIDiscoverySource(discoveryName)
			if discoverySource == nil {
				return fmt.Errorf("discovery %q does not exist", discoveryName)
			}

			newDiscoverySource, err := createDiscoverySource(discoveryName, uri)
			if err != nil {
				return err
			}

			err = configlib.SetCLIDiscoverySource(newDiscoverySource)
			if err != nil {
				return err
			}

			log.Successf("updated discovery source %s", discoveryName)
			return nil
		},
	}

	updateDiscoverySourceCmd.Flags().StringVarP(&uri, "uri", "u", "", "URI for discovery source. The URI must be of an OCI image")
	_ = updateDiscoverySourceCmd.MarkFlagRequired("uri")
	utils.PanicOnErr(updateDiscoverySourceCmd.RegisterFlagCompletionFunc("uri", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cobra.AppendActiveHelp(nil, "Please enter the uri of the OCI image for plugin discovery"), cobra.ShellCompDirectiveNoFileComp
	}))

	return updateDiscoverySourceCmd
}

func newDeleteDiscoverySourceCmd() *cobra.Command {
	var deleteDiscoverySourceCmd = &cobra.Command{
		Use:   "delete SOURCE_NAME",
		Short: "Delete a discovery source",
		// There are no flags
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		Hidden:                true,
		Example: `
    # Delete a discovery source
    tanzu plugin discovery delete default`,
		ValidArgsFunction: completeDiscoverySources,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			discoveryName := args[0]

			discoverySource, _ := configlib.GetCLIDiscoverySource(discoveryName)
			if discoverySource == nil {
				return fmt.Errorf("discovery %q does not exist", discoveryName)
			}

			err = configlib.DeleteCLIDiscoverySource(discoveryName)
			if err != nil {
				return err
			}
			log.Successf("deleted discovery source %s", discoveryName)
			return nil
		},
	}
	return deleteDiscoverySourceCmd
}

func newInitDiscoverySourceCmd() *cobra.Command {
	var initDiscoverySourceCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize the discovery source to its default value",
		Args:  cobra.MaximumNArgs(0),
		// There are no flags
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     noMoreCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := config.PopulateDefaultCentralDiscovery(true)
			if err != nil {
				return err
			}

			log.Successf("successfully initialized discovery source")
			return nil
		},
	}
	return initDiscoverySourceCmd
}

func createDiscoverySource(dsName, uri string) (configtypes.PluginDiscovery, error) {
	pluginDiscoverySource := configtypes.PluginDiscovery{}

	if dsName == "" {
		return pluginDiscoverySource, errors.New("discovery source name cannot be empty")
	}

	pluginDiscoverySource = configtypes.PluginDiscovery{
		OCI: &configtypes.OCIDiscovery{
			Name:  dsName,
			Image: uri,
		}}
	err := checkDiscoverySource(pluginDiscoverySource)
	return pluginDiscoverySource, err
}

// checkDiscoverySource attempts to access the content of the discovery to
// confirm it is valid
func checkDiscoverySource(source configtypes.PluginDiscovery) error {
	discObject, err := discovery.CreateDiscoveryFromV1alpha1(source)
	if err != nil {
		return err
	}
	_, err = discObject.List()
	return err
}

// ====================================
// Shell completion functions
// ====================================
func completeDiscoverySources(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	var comps []string
	discoverySources, _ := configlib.GetCLIDiscoverySources()
	for _, ds := range discoverySources {
		if ds.OCI != nil {
			comps = append(comps, fmt.Sprintf("%s\t%s", ds.OCI.Name, ds.OCI.Image))
		}
	}
	// Sort the completion to make testing easier
	sort.Strings(comps)

	return comps, cobra.ShellCompDirectiveNoFileComp
}

func completeUpdateDiscoverySource(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 && uri == "" {
		// The --uri flag is required, so completion will be provided for it
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// The user has provided enough information
	return completeDiscoverySources(cmd, args, toComplete)
}
