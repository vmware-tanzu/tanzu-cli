// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package pluginsupplier provides installed plugins information
package pluginsupplier

import (
	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
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
	standAloneCatalog, err := catalog.NewContextCatalog("")
	if err != nil {
		return nil, err
	}
	plugins := standAloneCatalog.List()

	// Any server plugin installed takes precedence over the same plugin
	// installed as standalone.  We therefore remove those standalone
	// plugins from the list.
	plugins, err = removeInstalledServerPlugins(plugins)
	if err != nil {
		return nil, err
	}
	return plugins, nil
}

// GetInstalledServerPlugins returns the installed server plugins.
func GetInstalledServerPlugins() ([]cli.PluginInfo, error) {
	serverNames, err := configlib.GetAllCurrentContextsList()
	if err != nil {
		return nil, err
	}

	var serverPlugins []cli.PluginInfo
	for _, serverName := range serverNames {
		if serverName != "" {
			serverCatalog, err := catalog.NewContextCatalog(serverName)
			if err != nil {
				return nil, err
			}
			serverPlugins = append(serverPlugins, serverCatalog.List()...)
		}
	}

	return serverPlugins, nil
}

// Remove any installed standalone plugin if it is also installed as a server plugin.
func removeInstalledServerPlugins(standalone []cli.PluginInfo) ([]cli.PluginInfo, error) {
	serverPlugins, err := GetInstalledServerPlugins()
	if err != nil {
		return nil, err
	}

	var installedStandalone []cli.PluginInfo
	for i := range standalone {
		found := false
		for j := range serverPlugins {
			if standalone[i].Name == serverPlugins[j].Name && standalone[i].Target == serverPlugins[j].Target {
				found = true
				break
			}
		}
		if !found {
			installedStandalone = append(installedStandalone, standalone[i])
		}
	}
	return installedStandalone, nil
}
