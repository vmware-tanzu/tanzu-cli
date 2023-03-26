// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides commands
package command

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	cliconfig "github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
)

const cmdNameSet = "set"

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
		// Note(TODO:prkalle): The below ceip-participation command(experimental) added may be removed in the next release,
		//       If we decide to fold this functionality into existing 'tanzu telemetry' plugin
		newCEIPParticipationCmd(),
		newGenAllDocsCmd(),
	)

	// If the context and target feature is enabled, add the corresponding commands under root.
	if config.IsFeatureActivated(constants.FeatureContextCommand) {
		rootCmd.AddCommand(
			contextCmd,
			k8sCmd,
			tmcCmd,
		)
		mapTargetToCmd := map[configtypes.Target]*cobra.Command{
			configtypes.TargetK8s: k8sCmd,
			configtypes.TargetTMC: tmcCmd,
		}
		if err := addPluginsToTarget(mapTargetToCmd); err != nil {
			return nil, err
		}
	}

	plugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return nil, err
	}
	if err = config.CopyLegacyConfigDir(); err != nil {
		return nil, fmt.Errorf("failed to copy legacy configuration directory to new location: %w", err)
	}

	var maskedPlugins []string

	for i := range plugins {
		// Only add plugins that should be available as root level command
		if isPluginRootCmdTargeted(&plugins[i]) {
			cmd := cli.GetCmdForPlugin(&plugins[i])
			// check and find if a command/plugin with the same name already exists as part of the root command
			matchedCmd := findSubCommand(rootCmd, cmd)
			if matchedCmd == nil { // If the subcommand for the plugin doesn't exist add the command
				rootCmd.AddCommand(cmd)
			} else if plugins[i].Scope == common.PluginScopeContext && isStandalonePluginCommand(matchedCmd) {
				// If the subcommand already exists because of a standalone plugin but the new plugin
				// is `Context-Scoped` then the new context-scoped plugin gets higher precedence.
				// We therefore replace the existing command with the new command by removing the old and
				// adding the new one.
				maskedPlugins = append(maskedPlugins, matchedCmd.Name())
				rootCmd.RemoveCommand(matchedCmd)
				rootCmd.AddCommand(cmd)
			} else if plugins[i].Name != "login" {
				// As the `login` plugin is now part of the core Tanzu CLI command and not a plugin
				// anymore, skip the `login` plugin from adding it to the maskedPlugins array to avoid
				// the warning message from getting shown to the user on each command invocation.
				maskedPlugins = append(maskedPlugins, plugins[i].Name)
			}
		}
	}

	if len(maskedPlugins) > 0 {
		fmt.Fprintf(os.Stderr, "Warning, Masking commands for plugins %q because a core command or other plugin with that name already exists. \n", strings.Join(maskedPlugins, ", "))
	}

	duplicateAliasWarning(rootCmd)

	// Disable footers in docs generated for core commands
	rootCmd.DisableAutoGenTag = true

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
			// Prompt user for EULA and CEIP agreement if necessary, except when
			//   - performing shell completion (The shell completion setup is not interactive,
			//     so it should not trigger a prompt)
			//   - 'tanzu version'(common first command to run),
			//   - 'config set' (would be a chicken and egg issue if user tries to set CEIP configuration
			//      using "tanzu config set env.TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER yes")
			//   - 'config eula xxx', 'ceip set' (auto prompting when running these commands is confusing)
			if cmd.Name() != cobra.ShellCompRequestCmd && cmd.Name() != completionCmd.Name() &&
				cmd.Name() != newVersionCmd().Name() &&
				!(cmd.Name() == cmdNameSet && cmd.Parent().Name() == configCmd.Name()) {
				if !(cmd.Parent().Name() == newEULACmd().Name()) &&
					!(cmd.Name() == cmdNameSet && cmd.Parent().Name() == newCEIPParticipationCmd().Name()) {
					if err := cliconfig.ConfigureEULA(false); err != nil {
						return err
					}

					configVal, _ := config.GetEULAStatus()
					if configVal != config.EULAStatusAccepted {
						fmt.Fprintf(os.Stderr, "The Tanzu CLI is only usable with reduced functionality until the General Terms are agreed to.\nPlease use `tanzu config eula show` to review the terms, or `tanzu config eula accept` to accept them directly\n")
						return errors.New("terms not accepted")
					}

					if err := cliconfig.ConfigureCEIPOptIn(); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	return rootCmd
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

// Execute executes the CLI.
func Execute() error {
	root, err := NewRootCmd()
	if err != nil {
		return err
	}
	return root.Execute()
}
