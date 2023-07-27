// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
)

var (
	showDetails bool
	pluginName  string
)

const searchLongDesc = `Search provides the ability to search for plugins that can be installed.
The command lists all plugins currently available for installation.
The search command also provides flags to limit the scope of the search.
`

func newSearchPluginCmd() *cobra.Command {
	var searchCmd = &cobra.Command{
		Use:               "search",
		Short:             "Search for available plugins",
		Long:              searchLongDesc,
		Args:              cobra.MaximumNArgs(0),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !common.IsValidTarget(targetStr, true, true) {
				return errors.New(invalidTargetMsg)
			}
			errorList := make([]error, 0)
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
					errorList = append(errorList, fmt.Errorf("there was an error while discovering plugins from local source, error information: '%w'", err))
				}
			} else {
				// Show plugins found in the central repos
				criteria := &discovery.PluginDiscoveryCriteria{
					Name:   pluginName,
					Target: common.StringToTargetString(targetStr),
				}
				allPlugins, err = pluginmanager.DiscoverStandalonePlugins(discovery.WithPluginDiscoveryCriteria(criteria))
				if err != nil {
					errorList = append(errorList, fmt.Errorf("there was an error while discovering standalone plugins, error information: '%w'", err))
				}
			}
			sort.Sort(discovery.DiscoveredSorter(allPlugins))

			if !showDetails {
				displayPluginsFound(allPlugins, cmd.OutOrStdout())
			} else {
				displayPluginDetails(allPlugins, cmd.OutOrStdout())
			}

			return kerrors.NewAggregate(errorList)
		},
	}

	f := searchCmd.Flags()
	f.BoolVar(&showDetails, "show-details", false, "show the details of the specified plugin, including all available versions")
	f.StringVarP(&pluginName, "name", "n", "", "limit the search to plugins with the specified name")
	f.StringVarP(&outputFormat, "output", "o", "", "output format (yaml|json|table)")

	f.StringVarP(&local, "local", "", "", "path to local plugin source")
	msg := fmt.Sprintf("this was done in the %q release, it will be removed following the deprecation policy (6 months). Use the %q flag instead.\n", "v1.0.0", "--local-source")
	if err := f.MarkDeprecated("local", msg); err != nil {
		// Will only fail if the flag does not exist, which would indicate a coding error,
		// so let's panic so we notice immediately.
		panic(err)
	}
	f.StringVarP(&local, "local-source", "l", "", "path to local plugin source")
	// We hide the "local-source" flag because installing from a local-source is not supported in production.
	// See the "local-source" flag of the "plugin install" command.
	if err := f.MarkHidden("local-source"); err != nil {
		// Will only fail if the flag does not exist, which would indicate a coding error,
		// so let's panic so we notice immediately.
		panic(err)
	}
	f.StringVarP(&targetStr, "target", "t", "", fmt.Sprintf("limit the search to plugins of the specified target (%s)", common.TargetList))

	searchCmd.MarkFlagsMutuallyExclusive("local", "name")
	searchCmd.MarkFlagsMutuallyExclusive("local", "target")
	searchCmd.MarkFlagsMutuallyExclusive("local", "show-details")

	return searchCmd
}

func displayPluginsFound(plugins []discovery.Discovered, writer io.Writer) {
	outputWriter := component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Latest")

	for i := range plugins {
		outputWriter.AddRow(
			plugins[i].Name,
			plugins[i].Description,
			plugins[i].Target,
			plugins[i].RecommendedVersion)
	}

	outputWriter.Render()
}

func displayPluginDetails(plugins []discovery.Discovered, writer io.Writer) {
	// Create a specific object format so it gets printed properly in yaml or json
	type detailedObject struct {
		Name        string
		Description string
		Target      string
		Latest      string
		Versions    []string
	}

	// For the table format, we will use individual yaml output for each plugin
	if outputFormat == "" || outputFormat == string(component.TableOutputType) {
		for i := range plugins {
			if i > 0 {
				fmt.Println()
			}
			details := detailedObject{
				Name:        plugins[i].Name,
				Description: plugins[i].Description,
				Target:      plugins[i].Target,
				Latest:      plugins[i].RecommendedVersion,
				Versions:    plugins[i].SupportedVersions,
			}
			component.NewObjectWriter(writer, string(component.YAMLOutputType), details).Render()
		}
		return
	}

	// Non-table format.
	// Here we use an objectWriter so that the array of versions is printed as an array
	// and not a long string.
	var details []detailedObject
	for i := range plugins {
		details = append(details, detailedObject{
			Name:        plugins[i].Name,
			Description: plugins[i].Description,
			Target:      plugins[i].Target,
			Latest:      plugins[i].RecommendedVersion,
			Versions:    plugins[i].SupportedVersions,
		})
	}
	component.NewObjectWriter(writer, outputFormat, details).Render()
}
