// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

var (
	local        string
	version      string
	forceDelete  bool
	outputFormat string
	targetStr    string
	group        string
)

func newPluginCmd() *cobra.Command {
	var pluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Manage CLI plugins",
		Annotations: map[string]string{
			"group": string(plugin.SystemCmdGroup),
		},
	}

	pluginCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	listPluginCmd := newListPluginCmd()
	installPluginCmd := newInstallPluginCmd()
	upgradePluginCmd := newUpgradePluginCmd()
	describePluginCmd := newDescribePluginCmd()
	deletePluginCmd := newDeletePluginCmd()
	cleanPluginCmd := newCleanPluginCmd()
	syncPluginCmd := newSyncPluginCmd()
	discoverySourceCmd := newDiscoverySourceCmd()

	listPluginCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (yaml|json|table)")
	listPluginCmd.Flags().StringVarP(&local, "local", "l", "", "path to local plugin source")
	if config.IsFeatureActivated(constants.FeatureCentralRepository) {
		// The --local flag no longer applies to the "list" command.
		// Instead of removing it completely, we mark it hidden and print out an error
		// in the RunE() function if it is used.  This provides better guidance to the user.
		if err := listPluginCmd.Flags().MarkHidden("local"); err != nil {
			// Will only fail if the flag does not exist, which would indicate a coding error,
			// so let's panic so we notice immediately.
			panic(err)
		}
		installPluginCmd.Flags().StringVar(&group, "group", "", "install the plugins specified in a plugin group")
	}
	installPluginCmd.Flags().StringVarP(&local, "local", "l", "", "path to local discovery/distribution source")
	installPluginCmd.Flags().StringVarP(&version, "version", "v", cli.VersionLatest, "version of the plugin")
	deletePluginCmd.Flags().BoolVarP(&forceDelete, "yes", "y", false, "delete the plugin without asking for confirmation")

	if config.IsFeatureActivated(constants.FeatureContextCommand) {
		installPluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", "target of the plugin (kubernetes[k8s]/mission-control[tmc])")
		upgradePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", "target of the plugin (kubernetes[k8s]/mission-control[tmc])")
		deletePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", "target of the plugin (kubernetes[k8s]/mission-control[tmc])")
		describePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", "target of the plugin (kubernetes[k8s]/mission-control[tmc])")
	}

	pluginCmd.AddCommand(
		listPluginCmd,
		installPluginCmd,
		upgradePluginCmd,
		describePluginCmd,
		deletePluginCmd,
		cleanPluginCmd,
		syncPluginCmd,
		discoverySourceCmd,
	)

	if config.IsFeatureActivated(constants.FeatureCentralRepository) {
		installPluginCmd.MarkFlagsMutuallyExclusive("group", "local")
		installPluginCmd.MarkFlagsMutuallyExclusive("group", "version")
		installPluginCmd.MarkFlagsMutuallyExclusive("group", "target")

		pluginCmd.AddCommand(
			newSearchPluginCmd(),
			newPluginGroupCmd())
	}

	return pluginCmd
}

func newListPluginCmd() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.IsFeatureActivated(constants.FeatureCentralRepository) {
				if local != "" {
					return fmt.Errorf("the '--local' flag does not apply to this command. Please use 'tanzu plugin search --local'")
				}

				// List installed plugins
				installedPlugins, err := pluginsupplier.GetInstalledPlugins()
				if err != nil {
					return err
				}
				sort.Sort(cli.PluginInfoSorter(installedPlugins))

				// Also list context plugins even if they are not installed.
				// This guides the user to know some plugins are recommended for the
				// active contexts, but are not installed.
				missingPlugins, err := getMissingContextPlugins(installedPlugins)
				if err != nil {
					// In this case, just don't show the context plugins, to at least
					// show the installed plugins
					log.Warningf("Unable to get the possible list of missing context-plugins: '%s'", err.Error())
					missingPlugins = []discovery.Discovered{}
				}
				sort.Sort(discovery.DiscoveredSorter(missingPlugins))

				if config.IsFeatureActivated(constants.FeatureContextCommand) && (outputFormat == "" || outputFormat == string(component.TableOutputType)) {
					displayInstalledAndMissingSplitView(installedPlugins, missingPlugins, cmd.OutOrStdout())
				} else {
					displayInstalledAndMissingListView(installedPlugins, missingPlugins, cmd.OutOrStdout())
				}

				return nil
			}

			// Plugin listing before the Central Repository feature
			var err error
			var availablePlugins []discovery.Discovered
			if local != "" {
				// get absolute local path
				local, err = filepath.Abs(local)
				if err != nil {
					return err
				}
				availablePlugins, err = pluginmanager.AvailablePluginsFromLocalSource(local)
			} else {
				availablePlugins, err = pluginmanager.AvailablePlugins()
			}

			if err != nil {
				return err
			}
			sort.Sort(discovery.DiscoveredSorter(availablePlugins))

			if config.IsFeatureActivated(constants.FeatureContextCommand) && (outputFormat == "" || outputFormat == string(component.TableOutputType)) {
				displayPluginListOutputSplitViewContext(availablePlugins, cmd.OutOrStdout())
			} else {
				displayPluginListOutputListView(availablePlugins, cmd.OutOrStdout())
			}

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

			if !configtypes.IsValidTarget(targetStr, true, true) {
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
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var pluginName string

			if !configtypes.IsValidTarget(targetStr, true, true) {
				return errors.New("invalid target specified. Please specify correct value of `--target` or `-t` flag from 'kubernetes/k8s/mission-control/tmc'")
			}

			if !config.IsFeatureActivated(constants.FeatureCentralRepository) {
				return legacyPluginInstall(cmd, args)
			}

			if group != "" {
				// We are installing from a group
				if len(args) == 0 {
					// Default to 'all' when installing from a group
					pluginName = cli.AllPlugins
				} else {
					pluginName = args[0]
				}

				err = pluginmanager.InstallPluginsFromGroup(pluginName, group)
				if err != nil {
					return err
				}
				if pluginName == cli.AllPlugins {
					log.Successf("successfully installed all plugins from group '%s'", group)
				} else {
					log.Successf("successfully installed '%s' from group '%s'", pluginName, group)
				}

				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("missing plugin name or '%s' as an argument, or the use of '--group'", cli.AllPlugins)
			}
			pluginName = args[0]

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
					log.Success("successfully installed all plugins")
				} else {
					log.Successf("successfully installed '%s' plugin", pluginName)
				}
				return nil
			}

			if pluginName == cli.AllPlugins {
				return fmt.Errorf("the '%s' argument can only be used with the --group or --local flags",
					cli.AllPlugins)
			}

			pluginVersion := version
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

func legacyPluginInstall(cmd *cobra.Command, args []string) error {
	var err error
	if len(args) == 0 {
		return fmt.Errorf("missing plugin name or '%s' as an argument", cli.AllPlugins)
	}
	pluginName := args[0]

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

			if !configtypes.IsValidTarget(targetStr, true, true) {
				return errors.New("invalid target specified. Please specify correct value of `--target` or `-t` flag from 'kubernetes/k8s/mission-control/tmc'")
			}

			var pluginVersion string
			if config.IsFeatureActivated(constants.FeatureCentralRepository) {
				// With the Central Repository feature we can simply request to install
				// the recommendedVersion.
				pluginVersion = cli.VersionLatest
			} else {
				pluginVersion, err = pluginmanager.GetRecommendedVersionOfPlugin(pluginName, getTarget())
				if err != nil {
					return err
				}
			}

			err = pluginmanager.UpgradePlugin(pluginName, pluginVersion, getTarget())
			if err != nil {
				return err
			}
			log.Successf("successfully upgraded plugin '%s'", pluginName)
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

			if !configtypes.IsValidTarget(targetStr, true, true) {
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

// getInstalledElseAvailablePluginVersion return installed plugin version if plugin is installed
// if not installed it returns available recommended plugin version
func getInstalledElseAvailablePluginVersion(p *discovery.Discovered) string {
	installedOrAvailableVersion := p.InstalledVersion
	if installedOrAvailableVersion == "" {
		installedOrAvailableVersion = p.RecommendedVersion
	}
	return installedOrAvailableVersion
}

// getMissingContextPlugins returns any context plugins that are not installed
func getMissingContextPlugins(installedPlugins []cli.PluginInfo) ([]discovery.Discovered, error) {
	serverPlugins, err := pluginmanager.DiscoverServerPlugins()
	if err != nil {
		return nil, err
	}

	var missingPlugins []discovery.Discovered
	for i := range serverPlugins {
		found := false
		for j := range installedPlugins {
			if serverPlugins[i].Name == installedPlugins[j].Name && serverPlugins[i].Target == installedPlugins[j].Target {
				found = true
				break
			}
		}
		if !found {
			// We have a server plugin that is not installed, include it in the list
			missingPlugins = append(missingPlugins, serverPlugins[i])
		}
	}
	return missingPlugins, nil
}

func displayPluginListOutputListView(availablePlugins []discovery.Discovered, writer io.Writer) {
	var data [][]string
	var output component.OutputWriter

	for index := range availablePlugins {
		data = append(data, []string{availablePlugins[index].Name, availablePlugins[index].Description, availablePlugins[index].Scope,
			availablePlugins[index].Source, getInstalledElseAvailablePluginVersion(&availablePlugins[index]), availablePlugins[index].Status})
	}
	output = component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Scope", "Discovery", "Version", "Status")

	for _, row := range data {
		vals := make([]interface{}, len(row))
		for i, val := range row {
			vals[i] = val
		}
		output.AddRow(vals...)
	}
	output.Render()
}

func displayPluginListOutputSplitViewContext(availablePlugins []discovery.Discovered, writer io.Writer) {
	var dataStandalone [][]string
	var outputStandalone component.OutputWriter
	dataContext := make(map[string][][]string)
	outputContext := make(map[string]component.OutputWriter)

	outputStandalone = component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Discovery", "Version", "Status")

	for index := range availablePlugins {
		if availablePlugins[index].Scope == common.PluginScopeStandalone {
			newRow := []string{availablePlugins[index].Name, availablePlugins[index].Description, string(availablePlugins[index].Target),
				availablePlugins[index].Source, getInstalledElseAvailablePluginVersion(&availablePlugins[index]), availablePlugins[index].Status}
			dataStandalone = append(dataStandalone, newRow)
		} else {
			newRow := []string{availablePlugins[index].Name, availablePlugins[index].Description, string(availablePlugins[index].Target),
				getInstalledElseAvailablePluginVersion(&availablePlugins[index]), availablePlugins[index].Status}
			outputContext[availablePlugins[index].ContextName] = component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Version", "Status")
			data := dataContext[availablePlugins[index].ContextName]
			data = append(data, newRow)
			dataContext[availablePlugins[index].ContextName] = data
		}
	}

	addDataToOutputWriter := func(output component.OutputWriter, data [][]string) {
		for _, row := range data {
			vals := make([]interface{}, len(row))
			for i, val := range row {
				vals[i] = val
			}
			output.AddRow(vals...)
		}
	}

	cyanBold := color.New(color.FgCyan).Add(color.Bold)
	cyanBoldItalic := color.New(color.FgCyan).Add(color.Bold, color.Italic)

	_, _ = cyanBold.Println("Standalone Plugins")
	addDataToOutputWriter(outputStandalone, dataStandalone)
	outputStandalone.Render()

	for context, writer := range outputContext {
		fmt.Println("")
		_, _ = cyanBold.Println("Plugins from Context: ", cyanBoldItalic.Sprintf(context))
		data := dataContext[context]
		addDataToOutputWriter(writer, data)
		writer.Render()
	}
}

func displayInstalledAndMissingSplitView(installedPlugins []cli.PluginInfo, missingPlugins []discovery.Discovered, writer io.Writer) {
	// List installed plugins
	cyanBold := color.New(color.FgCyan).Add(color.Bold)
	_, _ = cyanBold.Println("Installed Plugins")

	outputWriter := component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Version", "Status" /*, "Context"*/)
	for index := range installedPlugins {
		// // TODO(khouzam): Fix after pre-alpha: We need to figure out what would be an appropriate
		// // format to indicate to the user that a plugin was installed due to a context
		// // and make sure that information is always available in the data structs we process
		// context := ""
		// if installedPlugins[index].Scope == common.PluginScopeContext {
		//    context = installedPlugins[index].ContextName
		// }

		outputWriter.AddRow(
			installedPlugins[index].Name,
			installedPlugins[index].Description,
			string(installedPlugins[index].Target),
			installedPlugins[index].Version,
			common.PluginStatusInstalled,
			// context,
		)
	}
	outputWriter.Render()

	// List context plugins that are not installed.
	// First group them by context.
	missingByContext := make(map[string][]discovery.Discovered)
	for index := range missingPlugins {
		ctx := missingPlugins[index].ContextName
		missingByContext[ctx] = append(missingByContext[ctx], missingPlugins[index])
	}

	cyanBoldItalic := color.New(color.FgCyan).Add(color.Bold, color.Italic)
	for context := range missingByContext {
		outputWriter := component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Version", "Status")

		fmt.Println("")
		_, _ = cyanBold.Println("Missing Plugins from Context: ", cyanBoldItalic.Sprintf(context))
		for i := range missingByContext[context] {
			outputWriter.AddRow(
				missingByContext[context][i].Name,
				missingByContext[context][i].Description,
				string(missingByContext[context][i].Target),
				missingByContext[context][i].RecommendedVersion,
				common.PluginStatusNotInstalled,
			)
		}
		outputWriter.Render()
	}
	if len(missingPlugins) > 0 {
		// Print a warning to the user that some context plugins are not installed, and how to install them
		fmt.Println("")
		log.Warningf("As shown above, some recommended plugins have not been installed. To install them please run 'tanzu plugin sync'.")
	}
}

func displayInstalledAndMissingListView(installedPlugins []cli.PluginInfo, missingPlugins []discovery.Discovered, writer io.Writer) {
	outputWriter := component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Version", "Status" /*, "Context"*/)
	for index := range installedPlugins {
		// // TODO(khouzam): Fix after pre-alpha: We need to figure out what would be an appropriate
		// // format to indicate to the user that a plugin was installed due to a context
		// // and make sure that information is always available in the data structs we process
		// context := ""
		// if installedPlugins[index].Scope == common.PluginScopeContext {
		//    context = installedPlugins[index].ContextName
		// }

		outputWriter.AddRow(
			installedPlugins[index].Name,
			installedPlugins[index].Description,
			string(installedPlugins[index].Target),
			installedPlugins[index].Version,
			common.PluginStatusInstalled,
			// context,
		)
	}

	// List context plugins that are not installed.
	for i := range missingPlugins {
		outputWriter.AddRow(
			missingPlugins[i].Name,
			missingPlugins[i].Description,
			string(missingPlugins[i].Target),
			missingPlugins[i].RecommendedVersion,
			common.PluginStatusNotInstalled,
		)
	}
	outputWriter.Render()
}

func getTarget() configtypes.Target {
	return configtypes.StringToTarget(strings.ToLower(targetStr))
}
