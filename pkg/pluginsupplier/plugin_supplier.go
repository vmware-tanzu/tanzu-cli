// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package pluginsupplier provides installed plugins information
package pluginsupplier

import (
	"slices"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// GetInstalledPlugins return the installed plugins
func GetInstalledPlugins() ([]cli.PluginInfo, error) {
	// Migrate context-scoped plugins as standalone plugin if required
	// TODO(anujc): Think on how to invoke this function just once after the newer version
	// of the CLI gets installed as we just need to do this migration once
	catalog.MigrateContextPluginsAsStandaloneIfNeeded()

	// Get all the standalone plugins found in the catalog
	standAloneCatalog, err := catalog.NewContextCatalog("")
	if err != nil {
		return nil, err
	}
	return standAloneCatalog.List(), nil
}

// FilterPluginsByActiveContextType will exclude any plugin with an explicit
// setting of supportedContextType that does not match the type of any active CLI context
// Separating this conditional check so GetInstalledPlugins can
// continue to return all installed plugins regardless of supportedContextType
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

// IsPluginActive returns true if specified plugin is active
func IsPluginActive(pi *cli.PluginInfo) bool {
	if len(pi.SupportedContextType) == 0 {
		return true
	}

	activeContextMap, _ := configlib.GetAllActiveContextsMap()
	for ctxType := range activeContextMap {
		if slices.Contains(pi.SupportedContextType, ctxType) {
			return true
		}
	}
	return false
}

// IsPluginInstalled returns true if plugin is already installed
func IsPluginInstalled(name string, target configtypes.Target, version string) bool {
	// Check if the plugin is already installed, if installed skip the installation of the plugin
	installedPlugins, err := GetInstalledPlugins()
	if err == nil {
		for i := range installedPlugins {
			if installedPlugins[i].Name == name &&
				installedPlugins[i].Target == target &&
				installedPlugins[i].Version == version {
				return true
			}
		}
	}
	return false
}
