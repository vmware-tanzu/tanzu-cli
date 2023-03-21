// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
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
		serversCmd,
	)
	serversCmd.AddCommand(listServersCmd)
	addDeleteServersCmd()
}

var unattended bool

func addDeleteServersCmd() {
	listServersCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (yaml|json|table)")
	deleteServersCmd.Flags().BoolVarP(&unattended, "yes", "y", false, "Delete the server entry without confirmation")
	serversCmd.AddCommand(deleteServersCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration for the CLI",
	Annotations: map[string]string{
		"group": string(plugin.SystemCmdGroup),
	},
}

var getConfigCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := configlib.GetClientConfig()
		if err != nil {
			return err
		}

		b, err := yaml.Marshal(&cfg)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	},
}

var setConfigCmd = &cobra.Command{
	Use:   "set <path> <value>",
	Short: "Set config values at the given path",
	Long:  "Set config values at the given path. path values: [unstable-versions, cli.edition, features.global.<feature>, features.<plugin>.<feature>, env.<variable>]",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.Errorf("both path and value are required")
		}
		if len(args) > 2 {
			return errors.Errorf("only path and value are allowed")
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
	// special cases:
	// backward compatibility
	if pathParam == "unstable-versions" || pathParam == "cli.unstable-versions" {
		return setUnstableVersions(cfg, value)
	}

	if pathParam == "cli.edition" {
		return setEdition(cfg, value)
	}

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

func setUnstableVersions(cfg *configtypes.ClientConfig, value string) error {
	optionKey := configtypes.VersionSelectorLevel(value)

	switch optionKey {
	case configtypes.AllUnstableVersions,
		configtypes.AlphaUnstableVersions,
		configtypes.ExperimentalUnstableVersions,
		configtypes.NoUnstableVersions:
		cfg.SetUnstableVersionSelector(optionKey) // nolint:staticcheck // Deprecated
	default:
		return fmt.Errorf("unknown unstable-versions setting: %s; should be one of [all, none, alpha, experimental]", optionKey)
	}
	return nil
}

func setEdition(cfg *configtypes.ClientConfig, edition string) error {
	editionOption := configtypes.EditionSelector(edition)

	switch editionOption {
	case configtypes.EditionCommunity, configtypes.EditionStandard:
		cfg.SetEditionSelector(editionOption) // nolint:staticcheck // Deprecated
	default:
		return fmt.Errorf("unknown edition: %s; should be one of [%s, %s]", editionOption, configtypes.EditionStandard, configtypes.EditionCommunity)
	}
	return nil
}

var initConfigCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize config with defaults",
	Long:  "Initialize config with defaults including plugin specific defaults for all active and installed plugins",
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

// Note: Shall be deprecated in a future version. Superseded by 'tanzu context' command.
var serversCmd = &cobra.Command{
	Use:   "server",
	Short: "Configured servers",
}

// Note: Shall be deprecated in a future version. Superseded by 'tanzu context list' command.
var listServersCmd = &cobra.Command{
	Use:   "list",
	Short: "List servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := configlib.GetClientConfig()
		if err != nil {
			return err
		}

		output := component.NewOutputWriter(cmd.OutOrStdout(), outputFormat, "Name", "Type", "Endpoint", "Path", "Context")
		for _, server := range cfg.KnownServers { // nolint:staticcheck // Deprecated
			var endpoint, path, context string
			if server.IsGlobal() { // nolint:staticcheck // Deprecated
				endpoint = server.GlobalOpts.Endpoint
			} else {
				endpoint = server.ManagementClusterOpts.Endpoint
				path = server.ManagementClusterOpts.Path
				context = server.ManagementClusterOpts.Context
			}
			output.AddRow(server.Name, server.Type, endpoint, path, context)
		}
		output.Render()
		return nil
	},
}

// Note: Shall be deprecated in a future version. Superseded by 'tanzu context delete' command.
var deleteServersCmd = &cobra.Command{
	Use:   "delete SERVER_NAME",
	Short: "Delete a server from the config",

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("Server name required. Usage: tanzu config server delete server_name")
		}

		var isAborted error
		if !unattended {
			isAborted = component.AskForConfirmation("Deleting the server entry from the config will remove it from the list of tracked servers. " +
				"You will need to use tanzu login to track this server again. Are you sure you want to continue?")
		}

		if isAborted == nil {
			log.Infof("Deleting entry for cluster %s", args[0])
			serverExists, err := configlib.ServerExists(args[0]) // nolint:staticcheck // Deprecated
			if err != nil {
				return err
			}

			if serverExists {
				err := configlib.RemoveServer(args[0]) // nolint:staticcheck // Deprecated
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("server %s not found in list of known servers", args[0])
			}
		}

		return nil
	},
}

var unsetConfigCmd = &cobra.Command{
	Use:   "unset <path>",
	Short: "Unset config values at the given path",
	Long:  "Unset config values at the given path. path values: [features.global.<feature>, features.<plugin>.<feature>, env.global.<variable>, env.<plugin>.<variable>]",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("path is required")
		}
		if len(args) > 1 {
			return errors.Errorf("only path is allowed")
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
