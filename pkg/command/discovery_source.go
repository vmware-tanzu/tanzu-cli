// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"strings"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

var (
	discoverySourceType, discoverySourceName, uri string
)

func newDiscoverySourceCmd() *cobra.Command {
	var discoverySourceCmd = &cobra.Command{
		Use:   "source",
		Short: "Manage plugin discovery sources",
		Long:  "Manage plugin discovery sources. Discovery source provides metadata about the list of available plugins, their supported versions and how to download them.",
	}
	discoverySourceCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	listDiscoverySourceCmd := newListDiscoverySourceCmd()
	updateDiscoverySourceCmd := newUpdateDiscoverySourceCmd()
	deleteDiscoverySourceCmd := newDeleteDiscoverySourceCmd()

	listDiscoverySourceCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (yaml|json|table)")

	discoverySourceCmd.AddCommand(
		listDiscoverySourceCmd,
		updateDiscoverySourceCmd,
		deleteDiscoverySourceCmd,
	)

	if !configlib.IsFeatureActivated(constants.FeatureDisableCentralRepositoryForTesting) {
		discoverySourceCmd.AddCommand(newInitDiscoverySourceCmd())
		updateDiscoverySourceCmd.Flags().StringVarP(&uri, "uri", "u", "", "URI for discovery source. The URI must be of an OCI image")
	} else {
		updateDiscoverySourceCmd.Flags().StringVarP(&discoverySourceType, "type", "t", "", "type of discovery source")
		updateDiscoverySourceCmd.Flags().StringVarP(&uri, "uri", "u", "", "URI for discovery source. The URI format might be different based on the type of discovery source")

		// The "add" and "delete" plugin source commands are not needed for the central repo
		addDiscoverySourceCmd := newAddDiscoverySourceCmd()

		// TODO: when reactivating the "plugin source add" command, we need to replace the --name flag
		// with a argument for consistency with other commands
		addDiscoverySourceCmd.Flags().StringVarP(&discoverySourceName, "name", "n", "", "name of discovery source")
		addDiscoverySourceCmd.Flags().StringVarP(&discoverySourceType, "type", "t", "", "type of discovery source")
		addDiscoverySourceCmd.Flags().StringVarP(&uri, "uri", "u", "", "URI for discovery source. The URI format might be different based on the type of discovery source")

		// Not handling errors below because cobra handles the error when flag user doesn't provide these required flags
		_ = cobra.MarkFlagRequired(addDiscoverySourceCmd.Flags(), "name")
		_ = cobra.MarkFlagRequired(addDiscoverySourceCmd.Flags(), "type")
		_ = cobra.MarkFlagRequired(addDiscoverySourceCmd.Flags(), "uri")

		discoverySourceCmd.AddCommand(
			addDiscoverySourceCmd,
		)
	}
	return discoverySourceCmd
}

func newListDiscoverySourceCmd() *cobra.Command {
	var listDiscoverySourceCmd = &cobra.Command{
		Use:   "list",
		Short: "List available discovery sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !configlib.IsFeatureActivated(constants.FeatureDisableCentralRepositoryForTesting) {
				output := component.NewOutputWriter(cmd.OutOrStdout(), outputFormat, "name", "image")
				discoverySources, _ := configlib.GetCLIDiscoverySources()
				for _, ds := range discoverySources {
					if ds.OCI != nil {
						output.AddRow(ds.OCI.Name, ds.OCI.Image)
					}
				}

				output.Render()
				return nil
			}

			output := component.NewOutputWriter(cmd.OutOrStdout(), outputFormat, "name", "type", "scope")

			// List standalone scoped discoveries
			discoverySources, _ := configlib.GetCLIDiscoverySources()
			if discoverySources != nil {
				outputFromDiscoverySources(discoverySources, common.PluginScopeStandalone, output)
			}

			// If context-target feature is activated, get discovery sources from all active context
			// else get discovery sources from current server
			if configlib.IsFeatureActivated(constants.FeatureContextCommand) {
				mapContexts, err := configlib.GetAllCurrentContextsMap()
				if err == nil {
					for _, context := range mapContexts {
						outputFromDiscoverySources(context.DiscoverySources, common.PluginScopeContext, output)
					}
				}
			} else {
				server, err := configlib.GetCurrentServer() // nolint:staticcheck // Deprecated
				if err == nil && server != nil {
					outputFromDiscoverySources(server.DiscoverySources, common.PluginScopeContext, output)
				}
			}

			output.Render()

			return nil
		},
	}
	return listDiscoverySourceCmd
}

func outputFromDiscoverySources(discoverySources []configtypes.PluginDiscovery, scope string, output component.OutputWriter) {
	for _, ds := range discoverySources {
		dsName, dsType := discoverySourceNameAndType(ds)
		output.AddRow(dsName, dsType, scope)
	}
}
func newAddDiscoverySourceCmd() *cobra.Command {
	var addDiscoverySourceCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a discovery source",
		Long:  "Add a discovery source. Supported discovery types are: oci, local",
		Example: `
    # Add a local discovery source. If URI is relative path,
    # $HOME/.config/tanzu-plugins will be considered based path
    tanzu plugin source add --name standalone-local --type local --uri path/to/local/discovery

    # Add an OCI discovery source. URI should be an OCI image.
    tanzu plugin source add --name standalone-oci --type oci --uri projects.registry.vmware.com/tkg/tanzu-plugins/standalone:latest`,

		RunE: func(cmd *cobra.Command, args []string) error {
			newDiscoverySource, err := createDiscoverySource(discoverySourceType, discoverySourceName, uri)
			if err != nil {
				return err
			}

			err = configlib.SetCLIDiscoverySource(newDiscoverySource)
			if err != nil {
				return err
			}

			log.Successf("successfully added discovery source %s", discoverySourceName)
			return nil
		},
	}
	return addDiscoverySourceCmd
}

func newUpdateDiscoverySourceCmd() *cobra.Command {
	var updateDiscoverySourceCmd = &cobra.Command{
		Use:   "update SOURCE_NAME",
		Short: "Update a discovery source configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			discoveryName := args[0]

			discoverySource, _ := configlib.GetCLIDiscoverySource(discoveryName)
			if discoverySource == nil {
				return fmt.Errorf("discovery %q does not exist", discoveryName)
			}

			if !configlib.IsFeatureActivated(constants.FeatureDisableCentralRepositoryForTesting) {
				// With the central discovery, there is no more --type flag
				discoverySourceType = common.DiscoveryTypeOCI
			}
			newDiscoverySource, err := createDiscoverySource(discoverySourceType, discoveryName, uri)
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

	if !configlib.IsFeatureActivated(constants.FeatureDisableCentralRepositoryForTesting) {
		updateDiscoverySourceCmd.Example = `
    # Update the discovery source for an air-gapped scenario. The URI must be an OCI image.
    tanzu plugin source update default --uri registry.example.com/tanzu/plugin-inventory:latest`
	} else {
		updateDiscoverySourceCmd.Example = `
    # Update a local discovery source. If URI is relative path,
    # $HOME/.config/tanzu-plugins will be considered base path
    tanzu plugin source update standalone-local --type local --uri new/path/to/local/discovery

    # Update an OCI discovery source. URI should be an OCI image.
    tanzu plugin source update standalone-oci --type oci --uri projects.registry.vmware.com/tkg/tanzu-plugins/standalone:v1.0`
	}

	return updateDiscoverySourceCmd
}

func newDeleteDiscoverySourceCmd() *cobra.Command {
	var deleteDiscoverySourceCmd = &cobra.Command{
		Use:    "delete SOURCE_NAME",
		Short:  "Delete a discovery source",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		Example: `
    # Delete a discovery source
    tanzu plugin discovery delete default`,
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

func createDiscoverySource(dsType, dsName, uri string) (configtypes.PluginDiscovery, error) {
	pluginDiscoverySource := configtypes.PluginDiscovery{}
	if dsType == "" {
		return pluginDiscoverySource, errors.New("discovery source type cannot be empty")
	}
	if dsName == "" {
		return pluginDiscoverySource, errors.New("discovery source name cannot be empty")
	}

	switch strings.ToLower(dsType) {
	case common.DiscoveryTypeLocal:
		pluginDiscoverySource.Local = createLocalDiscoverySource(dsName, uri)
	case common.DiscoveryTypeOCI:
		pluginDiscoverySource.OCI = createOCIDiscoverySource(dsName, uri)
	case common.DiscoveryTypeREST:
		pluginDiscoverySource.REST = createRESTDiscoverySource(dsName, uri)
	case common.DiscoveryTypeGCP, common.DiscoveryTypeKubernetes:
		return pluginDiscoverySource, errors.Errorf("discovery source type '%s' is not yet supported", dsType)
	default:
		return pluginDiscoverySource, errors.Errorf("unknown discovery source type '%s'", dsType)
	}

	err := checkDiscoverySource(pluginDiscoverySource)
	return pluginDiscoverySource, err
}

// checkDiscoverySource attempts to access the content of the discovery to
// confirm it is valid
func checkDiscoverySource(source configtypes.PluginDiscovery) error {
	discObject, err := discovery.CreateDiscoveryFromV1alpha1(source, nil)
	if err != nil {
		return err
	}
	_, err = discObject.List()
	return err
}

func createLocalDiscoverySource(discoveryName, uri string) *configtypes.LocalDiscovery {
	return &configtypes.LocalDiscovery{
		Name: discoveryName,
		Path: uri,
	}
}

func createOCIDiscoverySource(discoveryName, uri string) *configtypes.OCIDiscovery {
	return &configtypes.OCIDiscovery{
		Name:  discoveryName,
		Image: uri,
	}
}

func createRESTDiscoverySource(discoveryName, uri string) *configtypes.GenericRESTDiscovery {
	return &configtypes.GenericRESTDiscovery{
		Name:     discoveryName,
		Endpoint: uri,
	}
}

func discoverySourceNameAndType(ds configtypes.PluginDiscovery) (string, string) {
	switch {
	case ds.GCP != nil: // nolint:staticcheck // Deprecated
		return ds.GCP.Name, common.DiscoveryTypeGCP // nolint:staticcheck // Deprecated
	case ds.Kubernetes != nil:
		return ds.Kubernetes.Name, common.DiscoveryTypeKubernetes
	case ds.Local != nil:
		return ds.Local.Name, common.DiscoveryTypeLocal
	case ds.OCI != nil:
		return ds.OCI.Name, common.DiscoveryTypeOCI
	case ds.REST != nil:
		return ds.REST.Name, common.DiscoveryTypeREST
	default:
		return "-", "Unknown" // Unknown discovery source found
	}
}
