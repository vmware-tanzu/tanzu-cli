// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
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
	invalidTargetMsg                = "invalid target specified. Please specify a correct value for the `--target` flag from '" + common.TargetList + "'"
	errorWhileDiscoveringPlugins    = "there was an error while discovering plugins, error information: '%v'"
	errorWhileGettingContextPlugins = "there was an error while discovering context plugins, error information: '%v'"
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
	utils.PanicOnErr(listPluginCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))
	listPluginCmd.Flags().BoolVar(&showAllColumns, "wide", false, "display additional columns for plugins")

	describePluginCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (yaml|json|table)")
	utils.PanicOnErr(describePluginCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))

	installPluginCmd.Flags().StringVar(&group, "group", "", "install the plugins specified by a plugin-group version")
	utils.PanicOnErr(installPluginCmd.RegisterFlagCompletionFunc("group", completeGroupsAndVersion))

	// --local is renamed to --local-source
	installPluginCmd.Flags().StringVarP(&local, "local", "", "", "path to local plugin source")
	msg := "this was done in the v1.0.0 release, it will be removed following the deprecation policy (6 months). Use the --local-source flag instead.\n"
	utils.PanicOnErr(installPluginCmd.Flags().MarkDeprecated("local", msg))

	// The --local-source flag for installing plugins is only used in development testing
	// and should not be used in production.  We mark it as hidden to help convey this reality.
	// Shell completion for this flag is the default behavior of doing file completion
	installPluginCmd.Flags().StringVarP(&local, "local-source", "l", "", "path to local plugin source")
	utils.PanicOnErr(installPluginCmd.Flags().MarkHidden("local-source"))

	installPluginCmd.Flags().StringVarP(&version, "version", "v", cli.VersionLatest, "version of the plugin")
	utils.PanicOnErr(installPluginCmd.RegisterFlagCompletionFunc("version", completePluginVersions))

	deletePluginCmd.Flags().BoolVarP(&forceDelete, "yes", "y", false, "uninstall the plugin without asking for confirmation")

	targetFlagDesc := fmt.Sprintf("target of the plugin (%s)", common.TargetList)
	installPluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", targetFlagDesc)
	utils.PanicOnErr(installPluginCmd.RegisterFlagCompletionFunc("target", completeTargetsForAllPlugins))

	upgradePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", targetFlagDesc)
	utils.PanicOnErr(upgradePluginCmd.RegisterFlagCompletionFunc("target", completeTargetsForAllPlugins))

	deletePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", targetFlagDesc)
	utils.PanicOnErr(deletePluginCmd.RegisterFlagCompletionFunc("target", completeTargetsForInstalledPlugins))

	describePluginCmd.Flags().StringVarP(&targetStr, "target", "t", "", targetFlagDesc)
	utils.PanicOnErr(describePluginCmd.RegisterFlagCompletionFunc("target", completeTargetsForInstalledPlugins))

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
		Use:               "list",
		Short:             "List installed plugins",
		Long:              "List installed plugins and plugins recommended by the active contexts",
		ValidArgsFunction: noMoreCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			errorList := make([]error, 0)
			// List installed plugins
			installedPlugins, err := pluginsupplier.GetInstalledPlugins()
			if err != nil {
				errorList = append(errorList, err)
				log.Warningf("there was an error while getting installed plugins, error information: '%v'", err.Error())
			}

			// Get List of discovered Server Plugins
			discoveredServerPlugins, err := pluginmanager.DiscoverServerPlugins()
			if err != nil {
				errorList = append(errorList, err)
				log.Warningf(errorWhileGettingContextPlugins, err.Error())
			}

			displayInstalledPlugins(installedPlugins, discoveredServerPlugins, cmd.OutOrStdout())

			return kerrors.NewAggregate(errorList)
		},
	}

	return listCmd
}

func newDescribePluginCmd() *cobra.Command {
	var describeCmd = &cobra.Command{
		Use:               "describe " + pluginNameCaps,
		Short:             "Describe a plugin",
		Long:              "Displays detailed information for a plugin",
		ValidArgsFunction: completeInstalledPlugins,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if outputFormat == "" {
				outputFormat = string(component.ListTableOutputType)
				fmt.Fprintln(cmd.OutOrStdout())
				defer fmt.Fprintln(cmd.OutOrStdout())
			}
			output := component.NewOutputWriterWithOptions(cmd.OutOrStdout(), outputFormat, []component.OutputWriterOption{}, "name", "version", "status", "target", "description", "installationPath")
			if len(args) != 1 {
				return fmt.Errorf("must provide one plugin name as a positional argument")
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

func newInstallPluginCmd() *cobra.Command {
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
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeAllPluginsToInstall,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var pluginName string

			if !configtypes.IsValidTarget(targetStr, true, true) {
				return errors.New(invalidTargetMsg)
			}

			if group != "" {
				return installPluginsForPluginGroup(cmd, args)
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

func installPluginsForPluginGroup(cmd *cobra.Command, args []string) error {
	var pluginName string
	// We are installing from a group
	if len(args) == 0 {
		// Default to 'all' when installing from a group
		pluginName = cli.AllPlugins
	} else {
		pluginName = args[0]
	}

	if pluginName == cli.AllPlugins {
		pg, err := pluginmanager.GetPluginGroup(group)
		if err != nil {
			return err
		}
		groupIDAndVersion := fmt.Sprintf("%s-%s/%s:%s", pg.Vendor, pg.Publisher, pg.Name, pg.RecommendedVersion)
		log.Infof("The following plugins will be installed from plugin group '%s'", groupIDAndVersion)
		// list plugins if we are installing all plugins from the plugin group
		displayGroupContentAsTable(pg, pg.RecommendedVersion, "", false, false, cmd.ErrOrStderr())
		groupWithVersion, err := pluginmanager.InstallPluginsFromGivenPluginGroup(pluginName, groupIDAndVersion, pg)
		if err != nil {
			return err
		}
		log.Successf("successfully installed all plugins from group '%s'", groupWithVersion)
	} else {
		groupWithVersion, err := pluginmanager.InstallPluginsFromGroup(pluginName, group)
		if err != nil {
			return err
		}
		log.Successf("successfully installed '%s' from group '%s'", pluginName, groupWithVersion)
	}
	return nil
}

func newUpgradePluginCmd() *cobra.Command {
	var upgradeCmd = &cobra.Command{
		Use:               "upgrade " + pluginNameCaps,
		Short:             "Upgrade a plugin",
		Long:              "Installs the latest version available for the specified plugin",
		ValidArgsFunction: completeAllPluginsToInstall,
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
		Use:               "uninstall " + pluginNameCaps,
		Aliases:           []string{"delete"},
		Short:             "Uninstall a plugin",
		Long:              "Uninstall the specified plugin or specify 'all' to uninstall all plugins of a target",
		ValidArgsFunction: completeDeletePlugin,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) != 1 {
				return fmt.Errorf("must provide one plugin name as a positional argument")
			}
			pluginName := args[0]

			if !configtypes.IsValidTarget(targetStr, true, true) {
				return errors.New(invalidTargetMsg)
			}

			target := getTarget()
			if pluginName == cli.AllPlugins {
				if target == configtypes.TargetUnknown {
					return fmt.Errorf("the '%s' argument can only be used with the '--target' flag", cli.AllPlugins)
				}
			}

			deletePluginOptions := pluginmanager.DeletePluginOptions{
				PluginName:  pluginName,
				Target:      target,
				ForceDelete: forceDelete,
			}

			err = pluginmanager.DeletePlugin(deletePluginOptions)
			if err != nil {
				return err
			}

			if pluginName == cli.AllPlugins {
				log.Successf("successfully uninstalled all plugins of target '%s'", target)
			} else {
				log.Successf("successfully uninstalled plugin '%s'", pluginName)
			}
			return nil
		},
	}
	return deleteCmd
}

func newCleanPluginCmd() *cobra.Command {
	var cleanCmd = &cobra.Command{
		Use:               "clean",
		Short:             "Clean the plugins",
		Long:              "Remove all installed plugins from the system",
		ValidArgsFunction: noMoreCompletions,
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
		ValidArgsFunction: noMoreCompletions,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			err = syncPlugins(cmd)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return syncCmd
}

// syncPlugins installs all plugins recommended by the active contexts and lists the plugins it's going to install
func syncPlugins(cmd *cobra.Command) error {
	contextMap, err := config.GetAllActiveContextsMap()
	if err != nil {
		return err
	}
	errList := make([]error, 0)
	count := 0
	for _, context := range contextMap {
		if strings.TrimSpace(context.Name) != "" {
			count++
		}
	}
	if count == 0 {
		log.Warning("No active contexts available to perform plugin sync")
		return nil
	}

	for contextType, context := range contextMap {
		err = syncContextPlugins(cmd, contextType, context.Name)
		if err != nil {
			errList = append(errList, err)
		}
	}
	return kerrors.NewAggregate(errList)
}

type pluginListInfo struct {
	name        string
	description string
	target      string
	installed   string
	recommended string
	status      string
	active      bool
}

// pluginListInfoSorter sorts pluginListInfo objects.
type pluginListInfoSorter []pluginListInfo

func (d pluginListInfoSorter) Len() int      { return len(d) }
func (d pluginListInfoSorter) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d pluginListInfoSorter) Less(i, j int) bool {
	if d[i].name != d[j].name {
		return d[i].name < d[j].name
	}
	return d[i].target < d[j].target
}

func displayInstalledPlugins(installedPlugins []cli.PluginInfo, recommendedContextPlugins []discovery.Discovered, writer io.Writer) {
	pluginSyncRequired := false

	getRecommendedPluginVersion := func(installedPlugin cli.PluginInfo) string {
		for index := range recommendedContextPlugins {
			if installedPlugin.Name == recommendedContextPlugins[index].Name && installedPlugin.Target == recommendedContextPlugins[index].Target {
				recommendedContextPlugins[index].Status = common.PluginStatusInstalled
				return recommendedContextPlugins[index].RecommendedVersion
			}
		}
		return ""
	}

	plugins := []pluginListInfo{}

	for index := range installedPlugins {
		p := pluginListInfo{
			name:        installedPlugins[index].Name,
			description: installedPlugins[index].Description,
			target:      string(installedPlugins[index].Target),
			installed:   installedPlugins[index].Version,
			recommended: getRecommendedPluginVersion(installedPlugins[index]),
			status:      common.PluginStatusInstalled,
			active:      pluginsupplier.IsPluginActive(&installedPlugins[index]),
		}
		if p.recommended != "" && p.installed != p.recommended {
			p.status = common.PluginStatusRecommendUpdate
			pluginSyncRequired = true
		}
		plugins = append(plugins, p)
	}
	for index := range recommendedContextPlugins {
		if recommendedContextPlugins[index].Status != common.PluginStatusInstalled {
			p := pluginListInfo{
				name:        recommendedContextPlugins[index].Name,
				description: recommendedContextPlugins[index].Description,
				target:      string(recommendedContextPlugins[index].Target),
				installed:   "",
				recommended: recommendedContextPlugins[index].RecommendedVersion,
				status:      common.PluginStatusRecommendInstall,
				active:      false,
			}
			plugins = append(plugins, p)
			pluginSyncRequired = true
		}
	}

	sort.Sort(pluginListInfoSorter(plugins))

	outputPluginWriter := component.NewOutputWriterWithOptions(writer, outputFormat, []component.OutputWriterOption{})
	if isTableOutputFormat() {
		columnsNames := []string{"Name", "Description", "Target", "Installed", "Recommended", "Status"}
		if showAllColumns {
			columnsNames = append(columnsNames, "Active")
		}
		outputPluginWriter.SetKeys(columnsNames...)
		outputPluginWriter.MarkDynamicKeys("Recommended") // Marking this column as dynamic so that it will only be shown if at least one row is non-empty
		for index := range plugins {
			outputPluginWriter.AddRow(plugins[index].name, plugins[index].description, plugins[index].target, plugins[index].installed, plugins[index].recommended, plugins[index].status, plugins[index].active)
		}
	} else {
		outputPluginWriter.SetKeys("Name", "Description", "Target", "Installed", "Recommended", "Status", "Active", "Context", "Version") // Add 'Context' and 'Version' fields for backwards compatibility
		for index := range plugins {
			outputPluginWriter.AddRow(plugins[index].name, plugins[index].description, plugins[index].target, plugins[index].installed, plugins[index].recommended, plugins[index].status, plugins[index].active, "", plugins[index].installed)
		}
	}

	outputPluginWriter.Render()

	if pluginSyncRequired && isTableOutputFormat() {
		// Print a warning to the user that some context plugins are not installed or outdated and plugin sync is required to install them
		fmt.Println("")
		fmt.Printf("Note: As shown above, some recommended plugins have not been installed or are outdated. To install them please run %s.\n", "'tanzu plugin sync'")
	}
}

func getTarget() configtypes.Target {
	return configtypes.StringToTarget(strings.ToLower(targetStr))
}

// ====================================
// Shell completion functions
// ====================================
func completeInstalledPlugins(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 1 {
		// Too many args
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 {
		// Check if the plugin name specified applies to more than one plugin
		if needTargetFlag := compCheckIfTargetFlagNeededForInstalled(cmd, args[0]); needTargetFlag {
			// The target flag needs to be used
			return []string{"--target"}, cobra.ShellCompDirectiveNoFileComp
		}
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	// Need to complete the names of installed plugins

	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var comps []string
	target := getTarget()
	// Complete all plugin names as long as the target matches and let the shell filter
	for i := range installedPlugins {
		if target == configtypes.TargetUnknown || target == installedPlugins[i].Target {
			// Make sure the name of the plugin is part of the description so that
			// zsh does not lump many plugins that have the same description
			comps = append(comps, fmt.Sprintf("%[1]s\tTarget: %[2]s for %[1]s", installedPlugins[i].Name, installedPlugins[i].Target))
		}
	}

	comps = completionMergeSimilarPlugins(comps)

	return comps, cobra.ShellCompDirectiveNoFileComp
}

func completeAllPluginsToInstall(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 1 {
		// Too many args
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 {
		// Check if the plugin name specified applies to more than one discovered plugin
		if needTargetFlag := compCheckIfTargetFlagNeededForAllPlugins(cmd, args[0]); needTargetFlag {
			// The target flag needs to be used
			return []string{"--target"}, cobra.ShellCompDirectiveNoFileComp
		}
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	return completionAllPlugins(), cobra.ShellCompDirectiveNoFileComp
}

func completePluginVersions(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// We can't complete the version if we don't have a plugin name
		comps := cobra.AppendActiveHelp(nil, "You must first specify a plugin name to be able to complete its version")
		return comps, cobra.ShellCompDirectiveNoFileComp
	}

	criteria := &discovery.PluginDiscoveryCriteria{
		Name:   args[0],
		Target: configtypes.StringToTarget(targetStr),
	}

	plugins, err := pluginmanager.DiscoverStandalonePlugins(
		discovery.WithPluginDiscoveryCriteria(criteria),
		discovery.WithUseLocalCacheOnly())

	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if len(plugins) == 0 {
		var comps []string
		if targetStr == "" {
			comps = cobra.AppendActiveHelp(nil, fmt.Sprintf("Unable to find plugin '%s'", args[0]))
		} else {
			comps = cobra.AppendActiveHelp(nil, fmt.Sprintf("Unable to find plugin '%s' for target '%s'", args[0], targetStr))
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	}

	// There could be more than one plugin if the target was not specified and
	// the plugin name exists for multiple targets.  It would be confusing to
	// do completion for versions of different plugins, so instead, as the user
	// to provide the target
	if len(plugins) > 1 {
		comps := cobra.AppendActiveHelp(nil, "Unable to uniquely identify this plugin. Please specify a target using the `--target` flag")
		return comps, cobra.ShellCompDirectiveNoFileComp
	}

	// The versions are already sorted, but in ascending order.
	// Since more recent versions are more likely to be of interest
	// lets reverse the order and then tell the shell to respect
	// that order using cobra.ShellCompDirectiveKeepOrder
	versions := plugins[0].SupportedVersions
	comps := make([]string, len(versions))
	for i := range versions {
		comps[len(versions)-1-i] = versions[i]
	}
	return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

func completeDeletePlugin(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 1 {
		// Too many arguments
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	targetFlag := cmd.Flags().Lookup("target")
	if len(args) == 1 {
		if args[0] == cli.AllPlugins {
			// With 'all' the '--target' flag must be used
			if !targetFlag.Changed {
				// The target flag needs to be used
				return []string{"--target"}, cobra.ShellCompDirectiveNoFileComp
			}
		} else {
			// Check if the plugin name specified applies to more than one installed plugin
			if needTargetFlag := compCheckIfTargetFlagNeededForInstalled(cmd, args[0]); needTargetFlag {
				// The target flag needs to be used
				return []string{"--target"}, cobra.ShellCompDirectiveNoFileComp
			}
		}
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	var comps []string
	compsForInstalledPlugins, directive := completeInstalledPlugins(cmd, args, toComplete)
	if !targetFlag.Changed {
		// Add the 'all' completion as the first one and ask the shell to preserve the order
		comps = append(comps, "all\tAll plugins for a target. You will need to use the --target flag.")
	} else {
		target := configtypes.StringToTarget(targetFlag.Value.String())
		// Add the 'all' completion as the first one and ask the shell to preserve the order
		comps = append(comps, fmt.Sprintf("all\tAll plugins of target %s", target))
	}
	comps = append(comps, compsForInstalledPlugins...)
	return comps, directive | cobra.ShellCompDirectiveKeepOrder
}

func completionAllPluginsFromLocal() []string {
	// The user requested the list of plugins from a local path
	var err error
	local, err = filepath.Abs(local)
	if err != nil {
		return nil
	}
	allPlugins, err := pluginmanager.DiscoverPluginsFromLocalSource(local)

	if err != nil {
		return nil
	}

	var comps []string
	target := getTarget()
	for i := range allPlugins {
		if target == configtypes.TargetUnknown || target == allPlugins[i].Target {
			comps = append(comps, fmt.Sprintf("%s\t%s", allPlugins[i].Name, allPlugins[i].Description))
		}
	}

	// When using the --local-source flag, the "all" keyword can be used
	if len(comps) > 0 {
		comps = append(comps, fmt.Sprintf("%s\t%s", cli.AllPlugins, "All plugins of the local source"))

		comps = completionMergeSimilarPlugins(comps)
	}
	return comps
}

func completionAllPluginsFromGroup() []string {
	groupIdentifier := plugininventory.PluginGroupIdentifierFromID(group)
	if groupIdentifier == nil {
		return nil
	}

	if groupIdentifier.Version == "" {
		groupIdentifier.Version = cli.VersionLatest
	}

	groups, err := pluginmanager.DiscoverPluginGroups(
		discovery.WithGroupDiscoveryCriteria(&discovery.GroupDiscoveryCriteria{
			Vendor:    groupIdentifier.Vendor,
			Publisher: groupIdentifier.Publisher,
			Name:      groupIdentifier.Name,
			Version:   groupIdentifier.Version,
		}),
		discovery.WithUseLocalCacheOnly())
	if err != nil || len(groups) == 0 {
		return nil
	}

	var comps []string
	for _, plugin := range groups[0].Versions[groups[0].RecommendedVersion] {
		if showNonMandatory || plugin.Mandatory {
			// To get the description we would need to query the central repo again.
			// Let's avoid that extra delay and simply not provide a description.
			comps = append(comps, plugin.Name)
		}
	}

	// When using the --group flag, the "all" keyword can be used
	comps = append(comps, cli.AllPlugins)

	comps = completionMergeSimilarPlugins(comps)

	return comps
}

func completionAllPlugins() []string {
	if local != "" {
		return completionAllPluginsFromLocal()
	}

	if group != "" {
		return completionAllPluginsFromGroup()
	}

	// Show plugins found in the central repos
	allPlugins, err := pluginmanager.DiscoverStandalonePlugins(
		discovery.WithPluginDiscoveryCriteria(&discovery.PluginDiscoveryCriteria{
			Target: configtypes.StringToTarget(targetStr),
		}),
		discovery.WithUseLocalCacheOnly())

	if err != nil {
		return nil
	}

	if len(allPlugins) == 0 {
		// If no plugin was returned it probably means the cache is empty.
		// Try the call again but allow it to download the plugin DB.
		allPlugins, err = pluginmanager.DiscoverStandalonePlugins(
			discovery.WithPluginDiscoveryCriteria(&discovery.PluginDiscoveryCriteria{
				Target: configtypes.StringToTarget(targetStr),
			}))

		if err != nil {
			return nil
		}
	}

	var comps []string
	for i := range allPlugins {
		comps = append(comps, fmt.Sprintf("%s\t%s", allPlugins[i].Name, allPlugins[i].Description))
	}

	comps = completionMergeSimilarPlugins(comps)

	return comps
}

// completionMergeSimilarPlugins A plugin completion is made up as the plugin name as
// the completion choice and a description, the two separated by a '\t'.
// This function will merge multiple entries with the same plugin name into a single one
// and update the description accordingly.  We do this because zsh and fish, when receiving
// two identical completions with only the description different, will only show the first
// completion. E.g.,
// $ tanzu plugin install cluster<TAB>
// cluster       -- A TMC managed Kubernetes cluster
// clustergroup  -- A group of Kubernetes clusters
//
// There should have been a second "cluster" entry for target Kubernetes.
// This can be confusing to users, as if there is no cluster plugin for Kubernetes.
func completionMergeSimilarPlugins(completions []string) []string {
	// Sort the completions so we can get duplicates to be sequential
	sort.Strings(completions)

	var mergedCompletions []string
	var prevName, mergedComp string
	for _, comp := range completions {
		pluginName, _, _ := strings.Cut(comp, "\t")

		if pluginName != prevName {
			// New plugin name.  The completion of the previous plugin can be stored.
			if mergedComp != "" {
				mergedCompletions = append(mergedCompletions, mergedComp)
			}
			prevName = pluginName
			mergedComp = comp
		} else {
			// Duplicate plugin name
			mergedComp = fmt.Sprintf("%[1]s\tMultiple entries for plugin %[1]s. You will need to use the --target flag.", pluginName)
		}
	}
	// Store the last completion now that the loop is done
	mergedCompletions = append(mergedCompletions, mergedComp)

	return mergedCompletions
}

func compCheckIfTargetFlagNeededForInstalled(cmd *cobra.Command, name string) bool {
	targetFlag := cmd.Flags().Lookup("target")
	if targetFlag.Changed {
		// The target flag is already on the command-line
		return false
	}

	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return false
	}

	// Check if the pluginName applies to more than one installed plugin
	matchingCount := 0
	for i := range installedPlugins {
		if installedPlugins[i].Name == name {
			matchingCount++
			if matchingCount > 1 {
				return true
			}
		}
	}

	return false
}

func compCheckIfTargetFlagNeededForAllPlugins(cmd *cobra.Command, pluginName string) bool {
	targetFlag := cmd.Flags().Lookup("target")
	if targetFlag.Changed {
		// The target flag is already on the command-line
		return false
	}

	// Check if the pluginName applies to more than one installed plugin
	plugins, err := pluginmanager.DiscoverStandalonePlugins(
		discovery.WithPluginDiscoveryCriteria(&discovery.PluginDiscoveryCriteria{
			Name: pluginName,
		}),
		discovery.WithUseLocalCacheOnly())

	if err != nil {
		return false
	}

	return len(plugins) > 1
}

func completeTargetsForInstalledPlugins(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 1 {
		// Only suggest targets that match the specified plugin
		pluginName := args[0]
		if pluginName == cli.AllPlugins {
			// Suggest all targets
			return []string{compGlobalTarget, compK8sTarget, compTMCTarget, compOpsTarget}, cobra.ShellCompDirectiveNoFileComp
		}

		installedPlugins, err := pluginsupplier.GetInstalledPlugins()
		if err != nil {
			return []string{compGlobalTarget, compK8sTarget, compTMCTarget, compOpsTarget}, cobra.ShellCompDirectiveNoFileComp
		}

		// Find all plugins matching the pluginName.  Each of the corresponding target should be suggested
		var availableTargets []string
		for i := range installedPlugins {
			if installedPlugins[i].Name == pluginName {
				availableTargets = append(availableTargets, compTargetToCompString(installedPlugins[i].Target))
			}
		}

		// If we found no plugins with the correct name, just complete all targets
		if len(availableTargets) == 0 {
			return []string{compGlobalTarget, compK8sTarget, compTMCTarget, compOpsTarget}, cobra.ShellCompDirectiveNoFileComp
		}

		sort.Strings(availableTargets)
		return availableTargets, cobra.ShellCompDirectiveNoFileComp
	}

	// Suggest all targets
	return []string{compGlobalTarget, compK8sTarget, compTMCTarget, compOpsTarget}, cobra.ShellCompDirectiveNoFileComp
}

func completeTargetsForAllPlugins(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 1 {
		// Only suggest targets that match the specified plugin
		pluginName := args[0]
		plugins, err := pluginmanager.DiscoverStandalonePlugins(
			discovery.WithPluginDiscoveryCriteria(&discovery.PluginDiscoveryCriteria{
				Name: pluginName,
			}),
			discovery.WithUseLocalCacheOnly())

		// If we found no plugins with the correct name, just complete all targets
		if err != nil || len(plugins) == 0 {
			return []string{compGlobalTarget, compK8sTarget, compTMCTarget, compOpsTarget}, cobra.ShellCompDirectiveNoFileComp
		}

		// For all plugins withe the specified name, the corresponding target should be suggested
		var availableTargets []string
		for i := range plugins {
			availableTargets = append(availableTargets, compTargetToCompString(plugins[i].Target))
		}
		sort.Strings(availableTargets)
		return availableTargets, cobra.ShellCompDirectiveNoFileComp
	}

	// Suggest all targets
	return []string{compGlobalTarget, compK8sTarget, compTMCTarget, compOpsTarget}, cobra.ShellCompDirectiveNoFileComp
}

func compTargetToCompString(target configtypes.Target) string {
	switch target {
	case configtypes.TargetGlobal:
		return compGlobalTarget
	case configtypes.TargetK8s:
		return compK8sTarget
	case configtypes.TargetTMC:
		return compTMCTarget
	case configtypes.TargetOperations:
		return compOpsTarget
	}
	return string(target)
}
