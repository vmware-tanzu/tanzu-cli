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
