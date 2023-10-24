// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides commands
package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	cliconfig "github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-cli/pkg/telemetry"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

// NewRootCmd creates a root command.
func NewRootCmd() (*cobra.Command, error) {
	var rootCmd = newRootCmd()
	uFunc := cli.NewMainUsage().UsageFunc()
	rootCmd.SetUsageFunc(uFunc)

	// Configure defined environment variables found in the config file
	cliconfig.ConfigureEnvVariables()

	rootCmd.AddCommand(
		newVersionCmd(),
		newPluginCmd(),
		loginCmd,
		initCmd,
		completionCmd,
		configCmd,
		contextCmd,
		k8sCmd,
		tmcCmd,
		genAllDocsCmd,
		// Note(TODO:prkalle): The below ceip-participation command(experimental) added may be removed in the next release,
		//       If we decide to fold this functionality into existing 'tanzu telemetry' plugin
		newCEIPParticipationCmd(),
	)
	if _, err := ensureCLIInstanceID(); err != nil {
		return nil, errors.Wrap(err, "failed to ensure CLI ID")
	}

	mapTargetToCmd := map[configtypes.Target]*cobra.Command{
		configtypes.TargetK8s: k8sCmd,
		configtypes.TargetTMC: tmcCmd,
	}
	if err := addPluginsToTarget(mapTargetToCmd); err != nil {
		return nil, err
	}

	plugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return nil, err
	}
	telemetry.Client().SetInstalledPlugins(plugins)
	if err = config.CopyLegacyConfigDir(); err != nil {
		return nil, fmt.Errorf("failed to copy legacy configuration directory to new location: %w", err)
	}

	var maskedPluginsWithPluginOverlap []string
	var maskedPluginsWithCoreCmdOverlap []string

	for i := range plugins {
		// Only add plugins that should be available as root level command
		if isPluginRootCmdTargeted(&plugins[i]) {
			cmd := cli.GetCmdForPlugin(&plugins[i])
			// check and find if a command/plugin with the same name already exists as part of the root command
			matchedCmd := findSubCommand(rootCmd, cmd)
			if matchedCmd == nil { // If the subcommand for the plugin doesn't exist add the command
				rootCmd.AddCommand(cmd)
			} else if (plugins[i].Scope == common.PluginScopeContext ||
				plugins[i].Target == configtypes.TargetGlobal) && isStandalonePluginCommand(matchedCmd) {
				// If the subcommand already exists because of a standalone plugin but the new plugin
				// is `Context-Scoped` then the new context-scoped plugin gets higher precedence.
				// Also, if the subcommand already exists because of a standalone plugin but the new plugin
				// is explicitly using the global target, it gets higher precedence also. This allows a plugin
				// developer to move their plugin from a k8s target to a global target; during the transition
				// the previous version of that plugin may be installed and target k8s, so we want to make sure
				// that the new version which targets global will be properly installed at the root level.
				// We therefore replace the existing command with the new command by removing the old and
				// adding the new one.
				maskedPluginsWithPluginOverlap = append(maskedPluginsWithPluginOverlap, matchedCmd.Name())
				rootCmd.RemoveCommand(matchedCmd)
				rootCmd.AddCommand(cmd)
			} else if plugins[i].Name != "login" {
				// As the `login` plugin is now part of the core Tanzu CLI command and not a plugin
				// anymore, skip the `login` plugin from adding it to the maskedPlugins array to avoid
				// the warning message from getting shown to the user on each command invocation.
				if isPluginCommand(matchedCmd) {
					maskedPluginsWithPluginOverlap = append(maskedPluginsWithPluginOverlap, plugins[i].Name)
				} else {
					maskedPluginsWithCoreCmdOverlap = append(maskedPluginsWithCoreCmdOverlap, plugins[i].Name)
				}
			}
		}
	}

	if len(maskedPluginsWithPluginOverlap) > 0 {
		catalog.DeleteIncorrectPluginEntriesFromCatalog()
	}
	if len(maskedPluginsWithCoreCmdOverlap) > 0 {
		fmt.Fprintf(os.Stderr, "Warning, masking commands for plugins %q because a core command with that name already exists. \n", strings.Join(maskedPluginsWithCoreCmdOverlap, ", "))
	}

	duplicateAliasWarning(rootCmd)

	return rootCmd, nil
}

func newRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use: "tanzu",
		// Don't have Cobra print the error message, the CLI will
		// print it itself in a nicer format.
		SilenceErrors: true,
		// silencing usage for now as we are getting double usage from plugins on errors
		SilenceUsage: true,
		// Flag parsing must be deactivated because the root plugin won't know about all flags.
		DisableFlagParsing: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Ensure mutual exclusion in current contexts just in case if any plugins with old
			// plugin-runtime sets k8s context as current when tae context is already set as current
			if err := utils.EnsureMutualExclusiveCurrentContexts(); err != nil {
				return err
			}

			if !shouldSkipTelemetryCollection(cmd) {
				if err := telemetry.Client().UpdateCmdPreRunMetrics(cmd, args); err != nil {
					telemetry.LogError(err, "")
				}
			}

			// Prompt user for EULA agreement if necessary
			if !shouldSkipPrompts(cmd) {
				if err := cliconfig.ConfigureEULA(false); err != nil {
					return err
				}
				configVal, _ := config.GetEULAStatus()
				if configVal != config.EULAStatusAccepted {
					fmt.Fprintf(os.Stderr, "The Tanzu CLI is only usable with reduced functionality until the General Terms are agreed to.\nPlease use `tanzu config eula show` to review the terms, or `tanzu config eula accept` to accept them directly\n")
					return errors.New("terms not accepted")
				}
			}

			// Install or update essential plugins
			InstallEssentialPlugins(cmd)

			// Prompt for CEIP agreement
			if !shouldSkipPrompts(cmd) {
				if err := cliconfig.ConfigureCEIPOptIn(); err != nil {
					return err
				}
			}

			setupActiveHelp(cmd, args)

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// Ensure mutual exclusion in current contexts just in case if any plugins with old
			// plugin-runtime sets k8s context as current when tae context is already set as current
			if err := utils.EnsureMutualExclusiveCurrentContexts(); err != nil {
				return err
			}
			return nil
		},
	}
	return rootCmd
}

func InstallEssentialPlugins(cmd *cobra.Command) {
	skipCommandsForEssentials := []string{
		// The shell completion setup is not interactive, so it should not trigger a prompt
		"tanzu __complete",
		"tanzu completion",
		// Common first command to run,
		"tanzu version",
		// It would be a chicken and egg issue if user tries to set CEIP configuration
		// using "tanzu config set env.TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER yes"
		"tanzu config set",
		// Auto prompting when running these commands is confusing
		"tanzu config eula",
		"tanzu ceip-participation set",
		// This command is being invoked by the kubectl exec binary where the user doesn't
		// get to see the prompts and the kubectl command execution just gets stuck, and it
		// is very hard for users to figure out what is going wrong
		"tanzu pinniped-auth",
		// Avoid trying to install essential plugins when user want to remove all plugins using tanzu plugin clean
		"tanzu plugin clean",
	}
	skipEssentials := false
	for _, cmdPath := range skipCommandsForEssentials {
		if strings.HasPrefix(cmd.CommandPath(), cmdPath) {
			skipEssentials = true
			break
		}
	}
	if !skipEssentials {
		// Check if essential plugins are installed and up to date if not Install or Upgrade the Essential plugins
		_, _ = pluginmanager.InstallPluginsFromEssentialPluginGroup()
	}
}

var k8sCmd = &cobra.Command{
	Use:     "kubernetes",
	Short:   "Tanzu CLI plugins that target a Kubernetes cluster",
	Aliases: []string{"k8s"},
	Annotations: map[string]string{
		"group": string(plugin.TargetCmdGroup),
	},
}

var tmcCmd = &cobra.Command{
	Use:     "mission-control",
	Short:   "Tanzu CLI plugins that target a Tanzu Mission Control endpoint",
	Aliases: []string{"tmc"},
	Annotations: map[string]string{
		"group": string(plugin.TargetCmdGroup),
	},
}

func addPluginsToTarget(mapTargetToCmd map[configtypes.Target]*cobra.Command) error {
	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return fmt.Errorf("unable to find installed plugins: %w", err)
	}

	for i := range installedPlugins {
		if cmd, exists := mapTargetToCmd[installedPlugins[i].Target]; exists {
			cmd.AddCommand(cli.GetCmdForPlugin(&installedPlugins[i]))
		}
	}
	return nil
}

func findSubCommand(rootCmd, subCmd *cobra.Command) *cobra.Command {
	arrSubCmd := rootCmd.Commands()
	for i := range arrSubCmd {
		if arrSubCmd[i].Name() == subCmd.Name() {
			return arrSubCmd[i]
		}
	}
	return nil
}

func isPluginRootCmdTargeted(pluginInfo *cli.PluginInfo) bool {
	// Plugins are considered "root-targeted" if their target is one of:
	// - global
	// - k8s
	// - unknown (backwards-compatibility: old designation for "global")
	return pluginInfo != nil &&
		(pluginInfo.Target == configtypes.TargetGlobal ||
			pluginInfo.Target == configtypes.TargetK8s ||
			pluginInfo.Target == configtypes.TargetUnknown)
}

func isStandalonePluginCommand(cmd *cobra.Command) bool {
	scope, exists := cmd.Annotations["scope"]
	return exists && scope == common.PluginScopeStandalone
}

func isPluginCommand(cmd *cobra.Command) bool {
	t, exists := cmd.Annotations["type"]
	return exists && t == common.CommandTypePlugin
}

func duplicateAliasWarning(rootCmd *cobra.Command) {
	var aliasMap = make(map[string][]string)
	for _, command := range rootCmd.Commands() {
		for _, alias := range command.Aliases {
			aliases, ok := aliasMap[alias]
			if !ok {
				aliasMap[alias] = []string{command.Name()}
			} else {
				aliasMap[alias] = append(aliases, command.Name())
			}
		}
	}

	for alias, plugins := range aliasMap {
		if len(plugins) > 1 {
			fmt.Fprintf(os.Stderr, "Warning, the alias %s is duplicated across plugins: %s\n\n", alias, strings.Join(plugins, ", "))
		}
	}
}
func ensureCLIInstanceID() (string, error) {
	cliID, _ := config.GetCLIId()
	if cliID != "" {
		return cliID, nil
	}
	cliID = uuid.New().String()
	err := config.SetCLIId(cliID)
	if err != nil {
		return "", err
	}
	return cliID, nil
}

// isSkipCommand returns true if the command is part of the skip list by checking the prefix of
// the command's command path matches with one of the item in the skip command list
func isSkipCommand(skipCommandList []string, commandPath string) bool {
	skipCommand := false
	for _, cmdPath := range skipCommandList {
		if strings.HasPrefix(commandPath, cmdPath) {
			skipCommand = true
			break
		}
	}
	return skipCommand
}

// shouldSkipTelemetryCollection checks if the command should be skipped for telemetry collection
func shouldSkipTelemetryCollection(cmd *cobra.Command) bool {
	skipTelemetryCollectionCommands := []string{
		// The shell completion setup is not interactive, so it should not trigger a prompt
		"tanzu __complete",
		"tanzu completion",
		// Common first command to run,
		"tanzu version",
		// should skip telemetry for "telemetry" plugin
		"tanzu telemetry",
	}
	return isSkipCommand(skipTelemetryCollectionCommands, cmd.CommandPath())
}

// shouldSkipPrompts checks if the prompts should be skipped for the command
func shouldSkipPrompts(cmd *cobra.Command) bool {
	// Prompt user for EULA and CEIP agreement if necessary, except for
	skipCommands := []string{
		// The shell completion setup is not interactive, so it should not trigger a prompt
		"tanzu __complete",
		"tanzu completion",
		// Common first command to run,
		"tanzu version",
		// It would be a chicken and egg issue if user tries to set CEIP configuration
		// using "tanzu config set env.TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER yes"
		"tanzu config set",
		// Auto prompting when running these commands is confusing
		"tanzu config eula",
		"tanzu ceip-participation set",
		// This command is being invoked by the kubectl exec binary where the user doesn't
		// get to see the prompts and the kubectl command execution just gets stuck, and it
		// is very hard for users to figure out what is going wrong
		"tanzu pinniped-auth",
	}
	return isSkipCommand(skipCommands, cmd.CommandPath())
}

// Execute executes the CLI.
func Execute() error {
	root, err := NewRootCmd()
	if err != nil {
		return err
	}
	executionErr := root.Execute()
	exitCode := 0
	if executionErr != nil {
		exitCode = 1
		if errStr, ok := executionErr.(*exec.ExitError); ok {
			// If a plugin exited with an error, we don't want to print its
			// exit status as a string, but want to use it as our own exit code.
			exitCode = (errStr.ExitCode())
		}
	}

	postRunMetrics := &telemetry.PostRunMetrics{ExitCode: exitCode}
	if updateErr := telemetry.Client().UpdateCmdPostRunMetrics(postRunMetrics); updateErr != nil {
		telemetry.LogError(updateErr, "")
	} else if saveErr := telemetry.Client().SaveMetrics(); saveErr != nil {
		telemetry.LogError(saveErr, "")
	} else if sendErr := telemetry.Client().SendMetrics(context.Background(), 2); sendErr != nil {
		telemetry.LogError(sendErr, "")
	}
	return executionErr
}

// ====================================
// Shell completion functions
// ====================================
func setupActiveHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != cobra.ShellCompRequestCmd {
		// We only setup ActiveHelp when we are dealing
		// with the __complete command since that is the
		// time shell completion is being performed.
		return
	}

	printShortDescOfCmdInActiveHelp(cmd, args)
}

// printShortDescOfCmdInActiveHelp sets up a ValidArgsFunction for the
// final command being run to print that command's "short" text as activeHelp.
// For example, if the user does
//
//	tanzu context list <TAB>
//
// this function will add a ValidArgsFunction for the "context list"
// command to print its short text as activeHelp.
func printShortDescOfCmdInActiveHelp(cmd *cobra.Command, args []string) {
	activeHelpConfig := os.Getenv("TANZU_ACTIVE_HELP")
	if strings.Contains(activeHelpConfig, "no_short_help") {
		return
	}

	// Find the final command that is being shell completed
	finalCmd, _, err := cmd.Root().Find(args)

	// Add the extra ValidArgsFunction to core commands only.
	// This feature will be handled by tanzu-plugin-runtime for plugins.
	if err == nil && finalCmd != nil && !isPluginCommand(finalCmd) {
		// If there is already a ValidArgsFunction, we must continue
		// using it once we have dealt with our extra activeHelp
		originalValidArgsFunc := finalCmd.ValidArgsFunction

		finalCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var comps []string
			if cmd.Short != "" {
				// Add the short text to activeHelp
				// We need to prefix it with something to differentiate it from
				// other active help text telling the user what to do.
				comps = cobra.AppendActiveHelp(comps, fmt.Sprintf("Command help: %s", cmd.Short))
			}

			// By default don't provide file completion.
			// This is important when we are doing sub-command
			// completion such as "tanzu context <TAB>"; normally
			// cobra would turn off file completion in this case,
			// but the below will be overriding cobra's directive.
			// For the cases that need file completion, we'll have
			// to add a ValidArgsFunction.  Such cases are much more
			// rare than needing to disable file completion.
			directive := cobra.ShellCompDirectiveNoFileComp
			if originalValidArgsFunc != nil {
				var oriComps []string
				oriComps, directive = originalValidArgsFunc(cmd, args, toComplete)
				comps = append(comps, oriComps...)
			}
			return comps, directive
		}
	}
}
