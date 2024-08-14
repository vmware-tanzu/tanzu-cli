// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides commands
package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	commonauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/csp"
	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	cliconfig "github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/datastore"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/globalinit"
	"github.com/vmware-tanzu/tanzu-cli/pkg/lastversion"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-cli/pkg/recommendedversion"
	"github.com/vmware-tanzu/tanzu-cli/pkg/telemetry"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

const isCLIContextsUpdatedToTCSPIssuers = "isCLIContextsUpdatedToTCSPIssuers"

var interruptChannel = make(chan os.Signal, 1)

// interruptHandle listens for Ctrl+C signal
// stops all spinners and exits the CLI command prompt
var interruptHandle = func() {
	sig := <-interruptChannel
	if sig != nil {
		component.StopAllSpinners()
	}
	os.Exit(128 + int(sig.(syscall.Signal)))
}

// init registers the signal handler for SIGINT and SIGTERM
func init() {
	signal.Notify(interruptChannel, syscall.SIGINT, syscall.SIGTERM)
}

// temporary function to still support invokedAs data by converting them to
// CommandMapEntry's. Will remove its use before the next minor update, at
// which point we will no longer recognized mapping information via invokedAs
func convertInvokedAs(plugins []cli.PluginInfo) {
	for i := range plugins {
		for _, invokedAsPath := range (plugins)[i].InvokedAs {
			for _, mapEntry := range (plugins)[i].CommandMap {
				if mapEntry.DestinationCommandPath == invokedAsPath {
					continue
				}
			}

			(plugins)[i].CommandMap = append((plugins)[i].CommandMap, plugin.CommandMapEntry{DestinationCommandPath: invokedAsPath})
		}
	}
}

// NewRootCmd creates a root command.
func NewRootCmd() (*cobra.Command, error) { //nolint: gocyclo,funlen
	go interruptHandle()
	var rootCmd = newRootCmd()
	uFunc := cli.NewMainUsage().UsageFunc()
	rootCmd.SetUsageFunc(uFunc)

	// Configure defined environment variables found in the config file
	cliconfig.ConfigureEnvVariables()

	rootCmd.AddCommand(
		newVersionCmd(),
		newPluginCmd(),
		newLoginCmd(),
		newInitCmd(),
		newCompletionCmd(),
		newConfigCmd(),
		newContextCmd(),
		k8sCmd,
		tmcCmd,
		opsCmd,
		tpeCmd,
		// Note(TODO:prkalle): The below ceip-participation command(experimental) added may be removed in the next release,
		//       If we decide to fold this functionality into existing 'tanzu telemetry' plugin
		newCEIPParticipationCmd(),
		newGenAllDocsCmd(),
	)
	if _, err := ensureCLIInstanceID(); err != nil {
		return nil, errors.Wrap(err, "failed to ensure CLI ID")
	}

	// Setup the commands for the plugins under the k8s and tmc targets
	if err := setupTargetPlugins(); err != nil {
		return nil, err
	}

	plugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return nil, err
	}

	convertInvokedAs(plugins)

	telemetry.Client().SetInstalledPlugins(plugins)
	if err = config.CopyLegacyConfigDir(); err != nil {
		return nil, fmt.Errorf("failed to copy legacy configuration directory to new location: %w", err)
	}

	var maskedPluginsWithPluginOverlap []string
	var maskedPluginsWithCoreCmdOverlap []string

	for i := range plugins {
		// Only add plugins that should be available as root level command
		if isPluginRootCmdTargeted(&plugins[i]) {
			cmd := cli.GetUnmappedCmdForPlugin(&plugins[i])
			if cmd == nil {
				// plugin is being remapped, will be processed in the second pass
				continue
			}
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

	// When feature flag is set, plugin-level mapping that might overwrite existing
	// commands will be allowed to do so only if the active context type
	// matches the supportedContextType of the mapping. So identify plugins that
	// need to be subjected to the active context check.
	if config.IsFeatureActivated(constants.FeaturePluginOverrideOnActiveContextType) {
		pluginsToCheck, remainingPlugins := findPluginsRequiringActiveContextCheck(rootCmd, plugins)
		pluginsAllowed, err := pluginsupplier.FilterPluginsByActiveContextType(pluginsToCheck)
		if err == nil {
			plugins = remainingPlugins
			plugins = append(plugins, pluginsAllowed...)
		}
	}

	remapCommandTree(rootCmd, plugins)
	updateTargetCommandGroupVisibility()
	updateConfigWithTanzuCSPIssuer(csp.GetIssuerUpdateFlagFromCentralConfig, datastore.GetDataStoreValue)
	updateConfigWithTanzuPlatformEndpointChanges()

	if len(maskedPluginsWithPluginOverlap) > 0 {
		catalog.DeleteIncorrectPluginEntriesFromCatalog()
	}
	if len(maskedPluginsWithCoreCmdOverlap) > 0 {
		fmt.Fprintf(os.Stderr, "Warning, masking commands for plugins %q because a core command with that name already exists. \n", strings.Join(maskedPluginsWithCoreCmdOverlap, ", "))
	}
	duplicateAliasWarning(rootCmd)

	// Disable footers in docs generated for core commands
	rootCmd.DisableAutoGenTag = true

	return rootCmd, nil
}

func remapCommandTree(rootCmd *cobra.Command, plugins []cli.PluginInfo) {
	cmdMap := buildReplacementMap(plugins)
	for pathKey, cmd := range cmdMap {
		matchedCmd, parentCmd := findSubCommandByPath(rootCmd, pathKey)

		if parentCmd != nil && isPluginCommand(parentCmd) {
			fmt.Fprintf(os.Stderr, "Remap of plugin into command tree (%s) associated with another plugin is not supported\n", parentCmd.Name())
			continue
		}

		if matchedCmd == nil {
			if parentCmd != nil {
				parentCmd.AddCommand(cmd)
			} else {
				fmt.Fprintf(os.Stderr, "Unable to remap %s at %q\n", cmd.Name(), pathKey)
			}
		} else {
			if parentCmd != nil {
				parentCmd.RemoveCommand(matchedCmd)
				if cmd != nil {
					parentCmd.AddCommand(cmd)
				}
			}
		}
	}
}

func buildReplacementMap(plugins []cli.PluginInfo) map[string]*cobra.Command {
	var maskedRemappedPlugins []string
	result := map[string]*cobra.Command{}

	activeContextMap, err := config.GetAllActiveContextsMap()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get active contexts : %v\n", err)
		return result
	}

	cmp := cli.NewCommandMapProcessor(activeContextMap)
	if cmp == nil {
		fmt.Fprintf(os.Stderr, "Unable to process command map from plugins")
		return result
	}

	for i := range plugins {
		cmdMap := cmp.GetCommandMapForPlugin(&plugins[i])
		for pathKey, newCmd := range cmdMap {
			if _, ok := result[pathKey]; ok {
				// Remapping a remapped command is unexpected! Note it and skip the attempt.
				if newCmd != nil {
					maskedRemappedPlugins = append(maskedRemappedPlugins, newCmd.Name())
				}
			} else {
				result[pathKey] = newCmd
			}
		}
	}

	if len(maskedRemappedPlugins) > 0 {
		// TODO(vuil) improve on usefulness of message
		fmt.Fprintf(os.Stderr, "Warning, multiple command groups are being remapped to the same command names : %q.\n", strings.Join(maskedRemappedPlugins, ", "))
	}

	return result
}

// updateTargetCommandGroupVisibility hides commands associated with target
// command group if latter did not acquire any child commands
func updateTargetCommandGroupVisibility() {
	for _, targetCmd := range []*cobra.Command{k8sCmd, tmcCmd, opsCmd, tpeCmd} {
		if len(targetCmd.Commands()) == 0 {
			targetCmd.Hidden = true
		}
	}
}

func findPluginsRequiringActiveContextCheck(rootCmd *cobra.Command, plugins []cli.PluginInfo) ([]cli.PluginInfo, []cli.PluginInfo) {
	var pluginsToCheck []cli.PluginInfo
	var remaining []cli.PluginInfo

	for i := range plugins {
		toCheck := false
		for _, mapEntry := range plugins[i].CommandMap {
			if mapEntry.SourceCommandPath == "" {
				cmd, _ := findSubCommandByPath(rootCmd, mapEntry.DestinationCommandPath)
				if cmd != nil {
					pluginsToCheck = append(pluginsToCheck, plugins[i])
					toCheck = true
					break
				}
			}
		}
		if !toCheck {
			remaining = append(remaining, plugins[i])
		}
	}
	return pluginsToCheck, remaining
}

// setupTargetPlugins sets up the commands for the plugins under the k8s and tmc targets
func setupTargetPlugins() error {
	mapTargetToCmd := map[configtypes.Target]*cobra.Command{
		configtypes.TargetK8s:        k8sCmd,
		configtypes.TargetTMC:        tmcCmd,
		configtypes.TargetOperations: opsCmd,
	}

	plugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return fmt.Errorf("unable to find installed plugins: %w", err)
	}

	convertInvokedAs(plugins)

	// Insert the plugin commands under the appropriate target command
	for i := range plugins {
		if targetCmd, exists := mapTargetToCmd[plugins[i].Target]; exists {
			cmd := cli.GetUnmappedCmdForPlugin(&plugins[i])
			if cmd != nil {
				targetCmd.AddCommand(cmd)
			}
		}
	}

	return nil
}

// updateConfigWithTanzuCSPIssuer updates the "tanzu" and "mission-control" CLI contexts issuers with TCSP if the
// issuer is VCSP Issuer, and invalidate the refresh token and token expiry time if these contexts token is
// of the type id-token, so that CLI would re-trigger the interactive login with updated issuer.
// This is done for only once.
func updateConfigWithTanzuCSPIssuer(centralConfigIssuerUpdateFlagGetter func() bool,
	cliContextUpdateStatusGetter func(string, interface{}) error) {

	issuerUpdateFlag := centralConfigIssuerUpdateFlagGetter()
	if !issuerUpdateFlag {
		return
	}

	cliContextsCSPIssuerUpdated := false
	_ = cliContextUpdateStatusGetter(isCLIContextsUpdatedToTCSPIssuers, &cliContextsCSPIssuerUpdated)
	if cliContextsCSPIssuerUpdated {
		return
	}
	cfg, err := config.GetClientConfig()
	if err != nil {
		return
	}
	if cfg == nil || len(cfg.KnownContexts) == 0 {
		return
	}
	updateSuccess := true
	for idx := range cfg.KnownContexts {
		ctx := cfg.KnownContexts[idx]
		if eligible := isEligibleForTCSPIssuerUpdate(ctx); !eligible {
			continue
		}
		if ctx.GlobalOpts.Auth.Issuer == csp.StgIssuer {
			ctx.GlobalOpts.Auth.Issuer = csp.StgIssuerTCSP
		} else if ctx.GlobalOpts.Auth.Issuer == csp.ProdIssuer {
			ctx.GlobalOpts.Auth.Issuer = csp.ProdIssuerTCSP
		}
		// invalidate only for interactive login token(id_token) and not for API Token type (API Tokens are carried over to TCSP)
		if ctx.GlobalOpts.Auth.Type == commonauth.IDTokenType {
			ctx.GlobalOpts.Auth.Expiration = time.Now().Local().Add(-10 * time.Second)
			ctx.GlobalOpts.Auth.RefreshToken = "Invalid"
		}
		if err := config.SetContext(ctx, false); err != nil {
			updateSuccess = false
		}
	}
	// if all the contexts are updated successfully, update the flag in the data store
	if updateSuccess {
		_ = datastore.SetDataStoreValue(isCLIContextsUpdatedToTCSPIssuers, &updateSuccess)
		log.Info("The CLI contexts have been updated to use the Tanzu CSP issuer. Any existing tokens obtained through interactive login are now invalid and CLI will automatically obtain a new token through interactive login using the new Tanzu CSP issuer")
	}
}

func isEligibleForTCSPIssuerUpdate(ctx *configtypes.Context) bool {
	if ctx.ContextType != configtypes.ContextTypeTanzu && ctx.ContextType != configtypes.ContextTypeTMC {
		return false
	}

	if (ctx.GlobalOpts == nil) || (ctx.GlobalOpts.Auth.Issuer != csp.StgIssuer && ctx.GlobalOpts.Auth.Issuer != csp.ProdIssuer) {
		return false
	}
	return true
}

func newRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "tanzu",
		Short: "The Tanzu CLI",
		// Don't have Cobra print the error message, the CLI will
		// print it itself in a nicer format.
		SilenceErrors: true,
		// silencing usage for now as we are getting double usage from plugins on errors
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Sets the verbosity of the logger if TANZU_CLI_LOG_LEVEL is set
			setLoggerVerbosity()

			// Perform some global initialization of the CLI if necessary
			// We do this as early as possible to make sure the CLI is ready for use
			// for any other logic below.
			if !shouldSkipGlobalInit(cmd) {
				checkGlobalInit(cmd)

				// Store the last executed CLI version in the datastore.
				// This can be useful for future features.
				// This must be done after the global initialization so initializers can
				// use the previously stored version if necessary.
				//
				// Note that we cannot run this in the PersistentPostRunE because if the command fails,
				// the PersistentPostRunE will not be called.  This could lead to the global initialization
				// running again on the next command execution.
				lastversion.SetLastExecutedCLIVersion()
			}

			// Ensure mutual exclusion in current contexts just in case if any plugins with old
			// plugin-runtime sets k8s context as current when tanzu context is already set as current
			if err := utils.EnsureMutualExclusiveCurrentContexts(); err != nil {
				return err
			}

			if !shouldSkipTelemetryCollection(cmd) {
				if err := telemetry.Client().UpdateCmdPreRunMetrics(cmd, args); err != nil {
					telemetry.LogError(err, "")
				}
			}

			if !shouldSkipPrompts(cmd) {
				// Prompt user for EULA agreement if necessary
				if err := cliconfig.ConfigureEULA(false); err != nil {
					return err
				}
				configVal, _ := config.GetEULAStatus()
				if configVal != config.EULAStatusAccepted {
					fmt.Fprintf(os.Stderr, "The Tanzu CLI is only usable with reduced functionality until the Broadcom Foundation Agreement is accepted.\nPlease use `tanzu config eula show` to review the Agreement, or `tanzu config eula accept` to accept it directly\n")
					return errors.New("agreement not accepted")
				}

				// Prompt for CEIP agreement
				if err := cliconfig.ConfigureCEIPOptIn(); err != nil {
					return err
				}
			}

			// Install or update essential plugins
			if !shouldSkipEssentialPlugins(cmd) {
				installEssentialPlugins()
			}

			setupActiveHelp(cmd, args)

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if !shouldSkipVersionCheck(cmd) {
				recommendedversion.CheckRecommendedCLIVersion(cmd)
			}

			// Ensure mutual exclusion in current contexts just in case if any plugins with old
			// plugin-runtime sets k8s context as current when tanzu context is already set as current
			return utils.EnsureMutualExclusiveCurrentContexts()
		},
	}
	return rootCmd
}

// setLoggerVerbosity sets the verbosity of the logger if TANZU_CLI_LOG_LEVEL is set
func setLoggerVerbosity() {
	// Configure the log level if env variable TANZU_CLI_LOG_LEVEL is set
	logLevel := os.Getenv(log.EnvTanzuCLILogLevel)
	if logLevel != "" {
		logValue, err := strconv.ParseInt(logLevel, 10, 32)
		if err == nil {
			log.SetVerbosity(int32(logValue))
		}
	}
}

func checkGlobalInit(cmd *cobra.Command) {
	if globalinit.InitializationRequired() {
		outStream := cmd.OutOrStderr()

		fmt.Fprintf(outStream, "Some initialization of the CLI is required.\n")
		fmt.Fprintf(outStream, "Let's set things up for you.  This will just take a few seconds.\n\n")

		err := globalinit.PerformInitializations(outStream)
		if err != nil {
			log.Warningf("The initialization encountered the following error: %v", err)
		}

		fmt.Fprintln(outStream)
		fmt.Fprintln(outStream, "Initialization done!")
		fmt.Fprintln(outStream, "==")
	}
}

func installEssentialPlugins() {
	_ = discovery.RefreshDatabase()

	// Check if all essential plugins are installed and up to date
	// if not install or upgrade them
	_, _ = pluginmanager.InstallPluginsFromEssentialPluginGroup()
}

func handleCommandGroupHelp(cmd *cobra.Command, args []string) {
	// If there are no plugins installed for this command group, print a message
	if len(cmd.Commands()) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Note: No plugins are currently installed for %[1]q.\n\n", cmd.Name())
	}
	// Always print the help for the command
	cmd.HelpFunc()(cmd, args)
}

var k8sCmd = &cobra.Command{
	Use:     "kubernetes",
	Short:   "Commands that interact with a Kubernetes endpoint",
	Aliases: []string{"k8s"},
	// We are moving away from the 'kubernetes' target.
	// All commands under this target are accessible as sub-commands of the root command.
	// For backwards compatibility, we are keeping the target but are hiding it.
	Deprecated: `you should invoke its sub-commands directly without the "kubernetes" prefix.`,
	Annotations: map[string]string{
		"group": string(plugin.TargetCmdGroup),
	},
	Run: func(cmd *cobra.Command, args []string) {
		handleCommandGroupHelp(cmd, args)
	},
}

var tmcCmd = &cobra.Command{
	Use:     "mission-control",
	Short:   "Commands that provide functionality for Tanzu Mission Control",
	Aliases: []string{"tmc"},
	Annotations: map[string]string{
		"group": string(plugin.TargetCmdGroup),
	},
	Run: func(cmd *cobra.Command, args []string) {
		handleCommandGroupHelp(cmd, args)
	},
}

var opsCmd = &cobra.Command{
	Use:     "operations",
	Short:   "Commands that support Kubernetes operations for Tanzu Platform for Kubernetes",
	Aliases: []string{"ops"},
	Annotations: map[string]string{
		"group": string(plugin.TargetCmdGroup),
	},
	Run: func(cmd *cobra.Command, args []string) {
		handleCommandGroupHelp(cmd, args)
	},
}

// Experiemental: placeholder group for platform-engineering commands
// For motivation and how this can be leveraged see
// https://github.com/vmware-tanzu/tanzu-cli/blob/main/docs/plugindev/README.md#reorganizing-the-plugin-commands-under-a-different-category-group-for-plugin
var tpeCmd = &cobra.Command{
	Use:     "platform-engineering",
	Short:   "Commands that provide functionality for Tanzu Platform Engineering",
	Aliases: []string{"tpe"},
	Annotations: map[string]string{
		"group": string(plugin.ExtraCmdGroup),
	},
	Run: func(cmd *cobra.Command, args []string) {
		handleCommandGroupHelp(cmd, args)
	},
}

func matchOnCommandNameAndAliases(cmd *cobra.Command, value string) bool {
	if cmd.Name() == value {
		return true
	}
	for _, alias := range cmd.Aliases {
		if alias == value {
			return true
		}
	}
	return false
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

func findSubCommandByHierarchy(cmd *cobra.Command, hierarchy []string, matcher func(*cobra.Command, string) bool) (*cobra.Command, *cobra.Command) {
	childCmds := cmd.Commands()
	for i := range childCmds {
		if len(hierarchy) == 1 {
			if matcher(childCmds[i], hierarchy[0]) {
				return childCmds[i], childCmds[i].Parent()
			}
		} else {
			if childCmds[i].Name() == hierarchy[0] {
				return findSubCommandByHierarchy(childCmds[i], hierarchy[1:], matcher)
			}
		}
	}
	if len(hierarchy) == 1 {
		return nil, cmd
	}
	return nil, nil
}

func findSubCommandByPath(rootCmd *cobra.Command, path string) (*cobra.Command, *cobra.Command) {
	var cmd, cmdParent *cobra.Command
	cmd = rootCmd
	cmdParent = rootCmd.Parent()

	cmdHierarchy := strings.Split(path, " ")
	if len(cmdHierarchy) > 0 {
		cmd, cmdParent = findSubCommandByHierarchy(rootCmd, cmdHierarchy, matchOnCommandNameAndAliases)
	}

	return cmd, cmdParent
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
		// Can be used to set the prompt on every shell command
		"tanzu context current",
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
		// Can be used to set the prompt on every shell command
		"tanzu context current",
		// This command is being invoked by the kubectl exec binary where the user doesn't
		// get to see the prompts and the kubectl command execution just gets stuck, and it
		// is very hard for users to figure out what is going wrong
		"tanzu context get-token",
	}
	return isSkipCommand(skipCommands, cmd.CommandPath())
}

func shouldSkipEssentialPlugins(cmd *cobra.Command) bool {
	skipCommandsForEssentials := []string{
		// The shell completion logic is not interactive, so it should not trigger
		// the installation of essential plugins which would print messages to the user
		// and break shell completion
		"tanzu __complete",
		"tanzu completion",
		// Common first command to run
		"tanzu version",

		"tanzu config set",

		// Can be used to set the prompt on every shell command
		"tanzu context current",
		// This command is being invoked by the kubectl exec binary where the user doesn't
		// get to see the output, so it is better to avoid printing essential plugins
		// installation messages
		"tanzu context get-token",
		"tanzu config eula",
		"tanzu ceip-participation set",
		// This command is being invoked by the kubectl exec binary where the user doesn't
		// get to see the output so, it is better to avoid printing essential plugins
		// installation messages
		"tanzu pinniped-auth",
		// Avoid trying to install essential plugins when the user wants to remove all plugins.
		// The plugin clean command would just uninstall the essential plugins we just installed
		"tanzu plugin clean",
		// Avoid trying to install essential plugins when the user initializes or updates the plugin
		// source information since the essential plugins installation would use the old plugin source
		"tanzu plugin source",
	}

	return isSkipCommand(skipCommandsForEssentials, cmd.CommandPath())
}

// shouldSkipVersionCheck checks if the CLI recommended version check should be skipped
// for the specified command
func shouldSkipVersionCheck(cmd *cobra.Command) bool {
	skipVersionCheckCommands := []string{
		// The shell completion logic is not interactive, so it should not trigger
		// extra printouts to the user for recommending a new version of the CLI
		"tanzu __complete",
		"tanzu completion",
		// Common first command to run, let's not recommend a new version of the CLI
		"tanzu version",
		// Can be used to set the prompt on every shell command
		"tanzu context current",
		// This command is being invoked by the kubectl exec binary where the user doesn't
		// get to see the output so, we should avoid printing the new version availability
		"tanzu context get-token",
		// This command is being invoked by the kubectl exec binary where the user doesn't
		// get to see the output so, we should avoid printing the new version availability
		"tanzu pinniped-auth",
	}
	return isSkipCommand(skipVersionCheckCommands, cmd.CommandPath())
}

// shouldSkipGlobalInit checks if the initialization of a new CLI version should be skipped
// for the specified command
func shouldSkipGlobalInit(cmd *cobra.Command) bool {
	skipGlobalInitCommands := []string{
		// The shell completion logic is not interactive, so it should not trigger
		// the global initialization of the CLI
		"tanzu __complete",
		"tanzu completion",
		// Common first command to run, let's not perform extra tasks
		"tanzu version",
		// Can be used to set the prompt on every shell command
		"tanzu context current",
		// This command is being invoked by the kubectl exec binary, and it is not interactive,
		// so it should not trigger the global initialization of the CLI
		"tanzu context get-token",
	}
	return isSkipCommand(skipGlobalInitCommands, cmd.CommandPath())
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
	} else if sendErr := telemetry.Client().SendMetrics(context.Background(), 0); sendErr != nil {
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
	activeHelpConfig := os.Getenv(constants.ConfigVariableActiveHelp)
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
