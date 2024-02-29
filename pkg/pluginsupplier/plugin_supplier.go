// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package pluginsupplier provides installed plugins information
package pluginsupplier

import (
	"os"
	"slices"
	"strconv"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// GetInstalledPlugins return the installed plugins( both standalone and server plugins )
func GetInstalledPlugins() ([]cli.PluginInfo, error) {
	plugins, err := GetInstalledServerPlugins()
	if err != nil {
		return nil, err
	}
	standalonePlugins, err := GetInstalledStandalonePlugins()
	if err != nil {
		return nil, err
	}
	plugins = append(plugins, standalonePlugins...)

	return plugins, nil
}

// GetInstalledStandalonePlugins returns the installed standalone plugins.
func GetInstalledStandalonePlugins() ([]cli.PluginInfo, error) {
	standalonePlugins, _, err := getInstalledStandaloneAndServerPlugins()
	if err != nil {
		return nil, err
	}
	return standalonePlugins, nil
}

// GetInstalledServerPlugins returns the installed server plugins.
func GetInstalledServerPlugins() ([]cli.PluginInfo, error) {
	_, serverPlugins, err := getInstalledStandaloneAndServerPlugins()
	if err != nil {
		return nil, err
	}
	return serverPlugins, nil
}

// FilterPluginsByActiveContextType will exclude any plugin with an explicit
// setting of supportedContextType that does not match the type of any active CLI context
// Separating this conditional check so GetInstalledStandalonePlugins can
// continue to return all standalone plugins regardless of supportedContextType
func FilterPluginsByActiveContextType(plugins []cli.PluginInfo) (result []cli.PluginInfo, err error) {
	activeContextMap, err := configlib.GetAllActiveContextsMap()
	if err != nil {
		return nil, err
	}
	for _, p := range plugins { //nolint:gocritic
		if len(p.SupportedContextType) == 0 {
			result = append(result, p)
		} else {
			for ctxType := range activeContextMap {
				if slices.Contains(p.SupportedContextType, ctxType) {
					result = append(result, p)
					break
				}
			}
		}
	}

	return result, nil
}

// IsStandalonePluginInstalled returns true if standalone plugin is already installed
func IsStandalonePluginInstalled(name string, target configtypes.Target, version string) bool {
	// Check if the standalone plugin is already installed, if installed skip the installation of the plugin
	installedStandalonePlugins, err := GetInstalledStandalonePlugins()
	if err == nil {
		for i := range installedStandalonePlugins {
			if installedStandalonePlugins[i].Name == name &&
				installedStandalonePlugins[i].Target == target &&
				installedStandalonePlugins[i].Version == version {
				return true
			}
		}
	}
	return false
}

func getInstalledStandaloneAndServerPlugins() (standalonePlugins, serverPlugins []cli.PluginInfo, err error) {
	// Get all the standalone plugins found in the catalog
	standAloneCatalog, err := catalog.NewContextCatalog("")
	if err != nil {
		return nil, nil, err
	}
	standalonePlugins = standAloneCatalog.List()

	// Get all the server plugins found in the catalog
	serverNames, err := configlib.GetAllActiveContextsList()
	if err != nil {
		return nil, nil, err
	}
	for _, serverName := range serverNames {
		if serverName != "" {
			serverCatalog, err := catalog.NewContextCatalog(serverName)
			if err != nil {
				return nil, nil, err
			}
			serverPlugins = append(serverPlugins, serverCatalog.List()...)
		}
	}

	// If the same plugin is present both as standalone and
	// as a server plugin we need to select which one to use
	// based on the TANZU_CLI_STANDALONE_OVER_CONTEXT_PLUGINS variable
	standalonePlugins, serverPlugins = filterIdenticalStandaloneAndServerPlugins(standalonePlugins, serverPlugins)
	return standalonePlugins, serverPlugins, nil
}

// Remove an installed standalone plugin if it is also installed as a server plugin,
// or vice versa if the TANZU_CLI_STANDALONE_OVER_CONTEXT_PLUGINS variable is enabled
func filterIdenticalStandaloneAndServerPlugins(standalonePlugins, serverPlugins []cli.PluginInfo) (installedStandalone, installedServer []cli.PluginInfo) {
	standaloneOverServerPlugins, _ := strconv.ParseBool(os.Getenv(constants.ConfigVariableStandaloneOverContextPlugins))

	if !standaloneOverServerPlugins {
		installedServer = serverPlugins

		for i := range standalonePlugins {
			found := false
			for j := range serverPlugins {
				if standalonePlugins[i].Name == serverPlugins[j].Name && standalonePlugins[i].Target == serverPlugins[j].Target {
					found = true
					break
				}
			}
			if !found {
				// No server plugin of the same name/target so we keep the standalone plugin
				installedStandalone = append(installedStandalone, standalonePlugins[i])
			}
		}
	} else {
		installedStandalone = standalonePlugins

		for i := range serverPlugins {
			found := false
			for j := range standalonePlugins {
				if serverPlugins[i].Name == standalonePlugins[j].Name && serverPlugins[i].Target == standalonePlugins[j].Target {
					found = true
					break
				}
			}
			if !found {
				// No standalone plugin of the same name/target so we keep the server plugin
				installedServer = append(installedServer, serverPlugins[i])
			}
		}
	}

	return installedStandalone, installedServer
}
