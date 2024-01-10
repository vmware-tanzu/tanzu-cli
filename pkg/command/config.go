// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// ConfigLiterals used with set/unset commands
const (
	ConfigLiteralFeatures = "features"
	ConfigLiteralEnv      = "env"
)

func init() {
	configCmd.SetUsageFunc(cli.SubCmdUsageFunc)
	configCmd.AddCommand(
		getConfigCmd,
		initConfigCmd,
		setConfigCmd,
		unsetConfigCmd,
		newEULACmd(),
		newCertCmd(),
	)
}

var unattended bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration for the CLI",
	Annotations: map[string]string{
		"group": string(plugin.SystemCmdGroup),
	},
}

var getConfigCmd = &cobra.Command{
	Use:               "get",
	Short:             "Get the current configuration",
	ValidArgsFunction: noMoreCompletions,
	Args:              cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := configlib.GetClientConfig()
		if err != nil {
			return err
		}

		// Print the entire config
		b, err := yaml.Marshal(&cfg)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(b)))

		warningForShadowedEnvVars(cmd.ErrOrStderr())

		return nil
	},
}

// Check if any of the variables of the config file are shadowed by
// a variable defined in the shell.  If so, warn the user.
func warningForShadowedEnvVars(writer io.Writer) {
	varsInConfig := configlib.GetEnvConfigurations()
	varNames := make([]string, 0, len(varsInConfig))
	for k := range varsInConfig {
		varNames = append(varNames, k)
	}
	sort.Strings(varNames)

	first := true
	for _, name := range varNames {
		configValue := varsInConfig[name]
		envValue := os.Getenv(name)
		if envValue != configValue {
			if first {
				first = false
				fmt.Fprintln(writer, "\nNote: The following variables set in the current shell take precedence over the ones of the same name set in the tanzu config:")
			}
			if envValue == "" {
				envValue = "''"
			}
			fmt.Fprintf(writer, "    - %s: %s\n", name, envValue)
		}
	}
}

var setConfigCmd = &cobra.Command{
	Use:               "set PATH <value>",
	Short:             "Set config values at the given PATH",
	Long:              "Set config values at the given PATH. Supported PATH values: [features.global.<feature>, features.<plugin>.<feature>, env.<variable>]",
	ValidArgsFunction: completeSetConfig,
	Example: `
    # Sets a custom CA cert for a proxy that requires it
    tanzu config set env.PROXY_CA_CERT b329baa034afn3.....
    # Enables a specific plugin feature
    tanzu config set features.management-cluster.custom_nameservers true
    # Enables a general CLI feature
    tanzu config set features.global.abcd true`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.Errorf("both PATH and <value> are required")
		}
		if len(args) > 2 {
			return errors.Errorf("only PATH and <value> are allowed")
		}

		// Acquire tanzu config lock
		configlib.AcquireTanzuConfigLock()
		defer configlib.ReleaseTanzuConfigLock()

		cfg, err := configlib.GetClientConfigNoLock()
		if err != nil {
			return err
		}

		err = setConfiguration(cfg, args[0], args[1])
		if err != nil {
			return err
		}

		return configlib.StoreClientConfig(cfg)
	},
}

// setConfiguration sets the key-value pair for the given path
func setConfiguration(cfg *configtypes.ClientConfig, pathParam, value string) error {
	// parse the param
	paramArray := strings.Split(pathParam, ".")
	if len(paramArray) < 2 {
		return errors.New("unable to parse config path parameter into parts [" + pathParam + "]  (was expecting 'features.<plugin>.<feature>' or 'env.<env_variable>')")
	}

	configLiteral := paramArray[0]

	switch configLiteral {
	case ConfigLiteralFeatures:
		return setFeatures(cfg, paramArray, value)
	case ConfigLiteralEnv:
		return setEnvs(cfg, paramArray, value)
	default:
		return errors.New("unsupported config path parameter [" + configLiteral + "] (was expecting 'features.<plugin>.<feature>' or 'env.<env_variable>')")
	}
}

func setFeatures(cfg *configtypes.ClientConfig, paramArray []string, value string) error {
	if len(paramArray) != 3 {
		return errors.New("unable to parse config path parameter into three parts [" + strings.Join(paramArray, ".") + "]  (was expecting 'features.<plugin>.<feature>'")
	}
	pluginName := paramArray[1]
	featureName := paramArray[2]

	if cfg.ClientOptions == nil {
		cfg.ClientOptions = &configtypes.ClientOptions{}
	}
	if cfg.ClientOptions.Features == nil {
		cfg.ClientOptions.Features = make(map[string]configtypes.FeatureMap)
	}
	if cfg.ClientOptions.Features[pluginName] == nil {
		cfg.ClientOptions.Features[pluginName] = configtypes.FeatureMap{}
	}
	cfg.ClientOptions.Features[pluginName][featureName] = value
	return nil
}

func setEnvs(cfg *configtypes.ClientConfig, paramArray []string, value string) error {
	if len(paramArray) != 2 {
		return errors.New("unable to parse config path parameter into two parts [" + strings.Join(paramArray, ".") + "]  (was expecting 'env.<variable>'")
	}
	envVariable := paramArray[1]

	if cfg.ClientOptions == nil {
		cfg.ClientOptions = &configtypes.ClientOptions{}
	}
	if cfg.ClientOptions.Env == nil {
		cfg.ClientOptions.Env = make(map[string]string)
	}

	cfg.ClientOptions.Env[envVariable] = value
	return nil
}

var initConfigCmd = &cobra.Command{
	Use:               "init",
	Short:             "Initialize config with defaults",
	Long:              "Initialize config with defaults including plugin specific defaults such as default feature flags for all active and installed plugins",
	ValidArgsFunction: noMoreCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Acquire tanzu config lock
		configlib.AcquireTanzuConfigLock()
		defer configlib.ReleaseTanzuConfigLock()

		cfg, err := configlib.GetClientConfigNoLock()
		if err != nil {
			return err
		}
		if cfg.ClientOptions == nil {
			cfg.ClientOptions = &configtypes.ClientOptions{}
		}
		//nolint: staticcheck
		//SA1019: cfg.ClientOptions.CLI is deprecated: CLI has been deprecated and will be removed from future version. use CoreCliOptions (staticcheck)
		if cfg.ClientOptions.CLI == nil {
			cfg.ClientOptions.CLI = &configtypes.CLIOptions{}
		}

		plugins, err := pluginsupplier.GetInstalledPlugins()
		if err != nil {
			return err
		}

		// Add the default featureflags for active plugins based on the currentContext
		// Plugins that are installed but are not active plugin will not be processed here
		// and defaultFeatureFlags will not be configured for those plugins
		for _, desc := range plugins {
			config.AddDefaultFeatureFlagsIfMissing(cfg, desc.DefaultFeatureFlags)
		}

		err = configlib.StoreClientConfig(cfg)
		if err != nil {
			return err
		}

		log.Success("successfully initialized the config")
		return nil
	},
}

var unsetConfigCmd = &cobra.Command{
	Use:               "unset PATH",
	Short:             "Unset config values at the given PATH",
	Long:              "Unset config values at the given PATH. Supported PATH values: [features.global.<feature>, features.<plugin>.<feature>, env.<variable>]",
	ValidArgsFunction: completeUnsetConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("PATH is required")
		}
		if len(args) > 1 {
			return errors.Errorf("only PATH is allowed")
		}

		return unsetConfiguration(args[0])

	},
}

// unsetConfiguration unsets the key-value pair for the given path and removes it
func unsetConfiguration(pathParam string) error {
	// parse the param
	paramArray := strings.Split(pathParam, ".")
	if len(paramArray) < 2 {
		return errors.New("unable to parse config path parameter into parts [" + pathParam + "]  (was expecting 'features.<plugin>.<feature>' or 'env.<env_variable>')")
	}

	configLiteral := paramArray[0]

	switch configLiteral {
	case ConfigLiteralFeatures:
		return unsetFeatures(paramArray)
	case ConfigLiteralEnv:
		return unsetEnvs(paramArray)
	default:
		return errors.New("unsupported config path parameter [" + configLiteral + "] (was expecting 'features.<plugin>.<feature>' or 'env.<env_variable>')")
	}
}

func unsetFeatures(paramArray []string) error {
	if len(paramArray) != 3 {
		return errors.New("unable to parse config path parameter into three parts [" + strings.Join(paramArray, ".") + "]  (was expecting 'features.<plugin>.<feature>'")
	}
	pluginName := paramArray[1]
	featureName := paramArray[2]

	return configlib.DeleteFeature(pluginName, featureName)
}

func unsetEnvs(paramArray []string) error {
	if len(paramArray) != 2 {
		return errors.New("unable to parse config path parameter into two parts [" + strings.Join(paramArray, ".") + "]  (was expecting 'env.<env_variable>'")
	}

	envVariable := paramArray[1]
	return configlib.DeleteEnv(envVariable)
}

// ====================================
// Shell completion functions
// ====================================

func completeSetConfig(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 1 {
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 {
		return cobra.AppendActiveHelp(nil, "You must provide a value as a second argument"),
			cobra.ShellCompDirectiveNoFileComp
	}
	comps := cobra.AppendActiveHelp(nil, "You can modify the below entries, or provide a new one")
	comps = append(comps, completionGetEnvAndFeatures()...)

	return comps, cobra.ShellCompDirectiveNoFileComp
}

func completeUnsetConfig(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}
	return completionGetEnvAndFeatures(), cobra.ShellCompDirectiveNoFileComp
}

func completionGetEnvAndFeatures() []string {
	// Complete all available env.<var> and features.<...> immediately
	// instead of first completing "env." and "features.".
	// This allows doing fuzzy matching with zsh and fish such as:
	//  "tanzu config unset ADD<TAB>" and directly getting the completion
	//  "env.TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY"
	var comps []string
	if envVars, err := configlib.GetAllEnvs(); err == nil {
		for name, value := range envVars {
			comps = append(comps, fmt.Sprintf("%s.%s\tValue: %q", ConfigLiteralEnv, name, value))
		}
	}

	// Retrieve client config node
	cfg, err := configlib.GetClientConfig()
	if err != nil {
		return comps
	}

	if cfg.ClientOptions != nil && cfg.ClientOptions.Features != nil {
		for plugin, features := range cfg.ClientOptions.Features {
			for name, value := range features {
				comps = append(comps, fmt.Sprintf("%s.%s.%s\tValue: %q", ConfigLiteralFeatures, plugin, name, value))
			}
		}
	}

	// Sort to allow for testing
	sort.Strings(comps)

	return comps
}
