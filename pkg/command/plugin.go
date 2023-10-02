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
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
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

const (
	invalidTargetMsg                = "invalid target specified. Please specify a correct value for the `--target/-t` flag from '" + common.TargetList + "'"
	errorWhileDiscoveringPlugins    = "there was an error while discovering plugins, error information: '%v'"
	errorWhileGettingContextPlugins = "there was an error while getting installed context plugins, error information: '%v'"
	pluginNameCaps                  = "PLUGIN_NAME"
)

func newPluginCmd() *cobra.Command {
	var pluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Manage CLI plugins",
		Long:  "Provides all lifecycle operations for plugins",
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
	describePluginCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (yaml|json|table)")

	installPluginCmd.Flags().StringVar(&group, "group", "", "install the plugins specified by a plugin-group version")

	// --local is renamed to --local-source
	installPluginCmd.Flags().StringVarP(&local, "local", "", "", "path to local plugin source")
	msg := "this was done in the v1.0.0 release, it will be removed following the deprecation policy (6 months). Use the --local-source flag instead.\n"
	utils.PanicOnErr(installPluginCmd.Flags().MarkDeprecated("local", msg))

	// The --local-source flag for installing plugins is only used in development testing
	// and should not be used in production.  We mark it as hidden to help convey this reality.
	installPluginCmd.Flags().StringVarP(&local, "local-source", "l", "", "path to local plugin source")
	utils.PanicOnErr(installPluginCmd.Flags().MarkHidden("local-source"))

	installPluginCmd.Flags().StringVarP(&version, "version", "v", cli.VersionLatest, "version of the plugin")
	deletePluginCmd.Flags().BoolVarP(&forceDelete, "yes", "y", false, "delete the plugin without asking for confirmation")

	targetFlagDesc := fmt.Sprintf("target of the plugin (%s)", common.TargetList)
	installPluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", targetFlagDesc)
	upgradePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", targetFlagDesc)
	deletePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", targetFlagDesc)
	describePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", targetFlagDesc)

	installPluginCmd.MarkFlagsMutuallyExclusive("group", "local")
	installPluginCmd.MarkFlagsMutuallyExclusive("group", "local-source")
	installPluginCmd.MarkFlagsMutuallyExclusive("group", "version")
	installPluginCmd.MarkFlagsMutuallyExclusive("group", "target")

	pluginCmd.AddCommand(
		listPluginCmd,
		installPluginCmd,
		upgradePluginCmd,
		describePluginCmd,
		deletePluginCmd,
		cleanPluginCmd,
		syncPluginCmd,
		discoverySourceCmd,
		newSearchPluginCmd(),
		newPluginGroupCmd(),
		newDownloadBundlePluginCmd(),
		newUploadBundlePluginCmd(),
	)

	return pluginCmd
}

func newListPluginCmd() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Long:  "List installed standalone plugins or plugins recommended by the contexts being used",
		RunE: func(cmd *cobra.Command, args []string) error {
			errorList := make([]error, 0)
			// List installed standalone plugins
			standalonePlugins, err := pluginsupplier.GetInstalledStandalonePlugins()
			if err != nil {
				errorList = append(errorList, err)
				log.Warningf("there was an error while getting installed standalone plugins, error information: '%v'", err.Error())
			}
			sort.Sort(cli.PluginInfoSorter(standalonePlugins))

			// List installed context plugins and also missing context plugins.
			// Showing missing ones guides the user to know some plugins are recommended for the
			// active contexts, but are not installed.
			installedContextPlugins, missingContextPlugins, pluginSyncRequired, err := getInstalledAndMissingContextPlugins()
			if err != nil {
				errorList = append(errorList, err)
				log.Warningf(errorWhileGettingContextPlugins, err.Error())
			}
			sort.Sort(discovery.DiscoveredSorter(installedContextPlugins))
			sort.Sort(discovery.DiscoveredSorter(missingContextPlugins))

			if outputFormat == "" || outputFormat == string(component.TableOutputType) {
				displayInstalledAndMissingSplitView(standalonePlugins, installedContextPlugins, missingContextPlugins, pluginSyncRequired, cmd.OutOrStdout())
			} else {
				displayInstalledAndMissingListView(standalonePlugins, installedContextPlugins, missingContextPlugins, cmd.OutOrStdout())
			}

			return kerrors.NewAggregate(errorList)
		},
	}

	return listCmd
}

func newDescribePluginCmd() *cobra.Command {
	var describeCmd = &cobra.Command{
		Use:   "describe " + pluginNameCaps,
		Short: "Describe a plugin",
		Long:  "Displays detailed information for a plugin",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			output := component.NewOutputWriter(cmd.OutOrStdout(), outputFormat, "name", "version", "status", "target", "description", "installationPath")
			if len(args) != 1 {
				return fmt.Errorf("must provide plugin name as positional argument")
			}
			pluginName := args[0]

			if !configtypes.IsValidTarget(targetStr, true, true) {
				return errors.New(invalidTargetMsg)
			}

			pd, err := pluginmanager.DescribePlugin(pluginName, getTarget())
			if err != nil {
				return err
			}
			output.AddRow(pd.Name, pd.Version, pd.Status, pd.Target, pd.Description, pd.InstallationPath)
			output.Render()
			return nil
		},
	}

	return describeCmd
}

func newInstallPluginCmd() *cobra.Command { //nolint:funlen
	var installCmd = &cobra.Command{
		Use:   "install [" + pluginNameCaps + "]",
		Short: "Install a plugin",
		Long:  "Install a specific plugin by name or specify all to install all plugins of a group",
		Example: `
    # Install all plugins of the vmware-tkg/default plugin group version v2.1.0
    tanzu plugin install --group vmware-tkg/default:v2.1.0

    # Install all plugins of the latest version of the vmware-tkg/default plugin group
    tanzu plugin install --group vmware-tkg/default

    # Install all plugins from the latest minor and patch of the v1 version of the vmware-tkg/default plugin group
    tanzu plugin install --group vmware-tkg/default:v1

    # Install all plugins from the latest patch of the v1.2 version of the vmware-tkg/default plugin group
    tanzu plugin install --group vmware-tkg/default:v1.2

    # Install the latest version of plugin "myPlugin"
    # If the plugin exists for more than one target, an error will be thrown
    tanzu plugin install myPlugin

    # Install the latest version of plugin "myPlugin" for target kubernetes
    tanzu plugin install myPlugin --target k8s

    # Install version v1.0.0 of plugin "myPlugin"
    tanzu plugin install myPlugin --version v1.0.0

    # Install latest patch version of v1.0 of plugin "myPlugin"
    tanzu plugin install myPlugin --version v1.0

    # Install latest minor and patch version of v1 of plugin "myPlugin"
    tanzu plugin install myPlugin --version v1`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var pluginName string

			if !configtypes.IsValidTarget(targetStr, true, true) {
				return errors.New(invalidTargetMsg)
			}

			if group != "" {
				// We are installing from a group
				if len(args) == 0 {
					// Default to 'all' when installing from a group
					pluginName = cli.AllPlugins
				} else {
					pluginName = args[0]
				}

				groupWithVersion, err := pluginmanager.InstallPluginsFromGroup(pluginName, group)
				if err != nil {
					return err
				}

				if pluginName == cli.AllPlugins {
					log.Successf("successfully installed all plugins from group '%s'", groupWithVersion)
				} else {
					log.Successf("successfully installed '%s' from group '%s'", pluginName, groupWithVersion)
				}

				return nil
			}

			// Invoke install plugin from local source if local files are provided
			if local != "" {
				if len(args) == 0 {
					return fmt.Errorf("missing plugin name or '%s' as an argument", cli.AllPlugins)
				}
				pluginName = args[0]

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

			if len(args) == 0 {
				return errors.New("missing plugin name as an argument or the use of '--group'")
			}
			pluginName = args[0]

			if pluginName == cli.AllPlugins {
				return fmt.Errorf("the '%s' argument can only be used with the '--group' flag", cli.AllPlugins)
			}

			pluginVersion := version
			err = pluginmanager.InstallStandalonePlugin(pluginName, pluginVersion, getTarget())
			if err != nil {
				return err
			}
			log.Successf("successfully installed '%s' plugin", pluginName)
			return nil
		},
	}
	return installCmd
}

func newUpgradePluginCmd() *cobra.Command {
	var upgradeCmd = &cobra.Command{
		Use:   "upgrade " + pluginNameCaps,
		Short: "Upgrade a plugin",
		Long:  "Installs the latest version available for the specified plugin",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 1 {
				return fmt.Errorf("must provide plugin name as positional argument")
			}
			pluginName := args[0]

			if !configtypes.IsValidTarget(targetStr, true, true) {
				return errors.New(invalidTargetMsg)
			}

			// With the Central Repository feature we can simply request to install
			// the recommendedVersion.
			err = pluginmanager.UpgradePlugin(pluginName, cli.VersionLatest, getTarget())
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
		Use:   "delete " + pluginNameCaps,
		Short: "Delete a plugin",
		Long:  "Uninstalls the specified plugin",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 1 {
				return fmt.Errorf("must provide plugin name as positional argument")
			}
			pluginName := args[0]

			if !configtypes.IsValidTarget(targetStr, true, true) {
				return errors.New(invalidTargetMsg)
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
		Long:  "Remove all installed plugins from the system",
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
		Short: "Installs all plugins recommended by the active contexts",
		Long: `Installs all plugins recommended by the active contexts.
Plugins installed with this command will only be available while the context remains active.`,
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

// getInstalledAndMissingContextPlugins returns any context plugins that are not installed
func getInstalledAndMissingContextPlugins() (installed, missing []discovery.Discovered, pluginSyncRequired bool, err error) {
	errorList := make([]error, 0)
	serverPlugins, err := pluginmanager.DiscoverServerPlugins()
	if err != nil {
		errorList = append(errorList, err)
		log.Warningf(errorWhileDiscoveringPlugins, err.Error())
	}

	// Note that the plugins we get here don't know from which context they were installed.
	// We need to cross-reference them with the discovered plugins.
	installedPlugins, err := pluginsupplier.GetInstalledServerPlugins()
	if err != nil {
		errorList = append(errorList, err)
		log.Warningf(errorWhileGettingContextPlugins, err.Error())
	}

	for i := range serverPlugins {
		found := false
		for j := range installedPlugins {
			if serverPlugins[i].Name != installedPlugins[j].Name || serverPlugins[i].Target != installedPlugins[j].Target {
				continue
			}

			// Store the installed plugin, which includes the context from which it was installed
			found = true
			if serverPlugins[i].RecommendedVersion != installedPlugins[j].Version {
				serverPlugins[i].Status = common.PluginStatusUpdateAvailable
				pluginSyncRequired = true
			} else {
				serverPlugins[i].Status = common.PluginStatusInstalled
			}
			serverPlugins[i].InstalledVersion = installedPlugins[j].Version
			installed = append(installed, serverPlugins[i])
			break
		}
		if !found {
			// We have a server plugin that is not installed, include it in the list
			serverPlugins[i].Status = common.PluginStatusNotInstalled
			missing = append(missing, serverPlugins[i])
			pluginSyncRequired = true
		}
	}
	return installed, missing, pluginSyncRequired, kerrors.NewAggregate(errorList)
}

func displayInstalledAndMissingSplitView(installedStandalonePlugins []cli.PluginInfo, installedContextPlugins, missingContextPlugins []discovery.Discovered, pluginSyncRequired bool, writer io.Writer) {
	// List installed standalone plugins
	cyanBold := color.New(color.FgCyan).Add(color.Bold)
	_, _ = cyanBold.Println("Standalone Plugins")

	outputStandalone := component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Version", "Status")
	for index := range installedStandalonePlugins {
		outputStandalone.AddRow(
			installedStandalonePlugins[index].Name,
			installedStandalonePlugins[index].Description,
			string(installedStandalonePlugins[index].Target),
			installedStandalonePlugins[index].Version,
			common.PluginStatusInstalled,
		)
	}
	outputStandalone.Render()

	// List installed and missing context plugins in one list.
	// First group them by context.
	contextPlugins := installedContextPlugins
	contextPlugins = append(contextPlugins, missingContextPlugins...)
	sort.Sort(discovery.DiscoveredSorter(contextPlugins))

	ctxPluginsByContext := make(map[string][]discovery.Discovered)
	for index := range contextPlugins {
		ctx := contextPlugins[index].ContextName
		ctxPluginsByContext[ctx] = append(ctxPluginsByContext[ctx], contextPlugins[index])
	}

	cyanBoldItalic := color.New(color.FgCyan).Add(color.Bold, color.Italic)

	// sort contexts to maintain consistency in the plugin list output
	contexts := make([]string, 0, len(ctxPluginsByContext))
	for context := range ctxPluginsByContext {
		contexts = append(contexts, context)
	}
	sort.Strings(contexts)
	for _, context := range contexts {
		outputWriter := component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Version", "Status")

		fmt.Println("")
		_, _ = cyanBold.Println("Plugins from Context: ", cyanBoldItalic.Sprintf(context))
		for i := range ctxPluginsByContext[context] {
			version := ctxPluginsByContext[context][i].InstalledVersion
			if ctxPluginsByContext[context][i].Status == common.PluginStatusNotInstalled {
				version = ctxPluginsByContext[context][i].RecommendedVersion
			}
			outputWriter.AddRow(
				ctxPluginsByContext[context][i].Name,
				ctxPluginsByContext[context][i].Description,
				string(ctxPluginsByContext[context][i].Target),
				version,
				ctxPluginsByContext[context][i].Status,
			)
		}
		outputWriter.Render()
	}

	if pluginSyncRequired {
		// Print a warning to the user that some context plugins are not installed or outdated and plugin sync is required to install them
		fmt.Println("")
		log.Warningf("As shown above, some recommended plugins have not been installed or are outdated. To install them please run 'tanzu plugin sync'.")
	}
}

func displayInstalledAndMissingListView(installedStandalonePlugins []cli.PluginInfo, installedContextPlugins, missingContextPlugins []discovery.Discovered, writer io.Writer) {
	outputWriter := component.NewOutputWriter(writer, outputFormat, "Name", "Description", "Target", "Version", "Status", "Context")
	for index := range installedStandalonePlugins {
		outputWriter.AddRow(
			installedStandalonePlugins[index].Name,
			installedStandalonePlugins[index].Description,
			string(installedStandalonePlugins[index].Target),
			installedStandalonePlugins[index].Version,
			installedStandalonePlugins[index].Status,
			"", // No context
		)
	}

	// List context plugins that are installed.
	for i := range installedContextPlugins {
		outputWriter.AddRow(
			installedContextPlugins[i].Name,
			installedContextPlugins[i].Description,
			string(installedContextPlugins[i].Target),
			installedContextPlugins[i].InstalledVersion,
			installedContextPlugins[i].Status,
			installedContextPlugins[i].ContextName,
		)
	}

	// List context plugins that are not installed.
	for i := range missingContextPlugins {
		outputWriter.AddRow(
			missingContextPlugins[i].Name,
			missingContextPlugins[i].Description,
			string(missingContextPlugins[i].Target),
			missingContextPlugins[i].RecommendedVersion,
			common.PluginStatusNotInstalled,
			missingContextPlugins[i].ContextName,
		)
	}
	outputWriter.Render()
}

func getTarget() configtypes.Target {
	return configtypes.StringToTarget(strings.ToLower(targetStr))
}
