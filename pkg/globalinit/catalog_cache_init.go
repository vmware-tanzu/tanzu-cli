// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package globalinit

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/lastversion"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// This global initializer checks if the last executed CLI version is < 1.3.0.
// If so it refreshes the plugin catalog to:
// 1- migrate context-scoped plugins as standalone plugins
// 2- make sure any remapping data is in the catalog

func init() {
	RegisterInitializer("Plugin Info Catalog Initializer", triggerForPreCommandRemapping, updatePluginCatalog)
}

func triggerForPreCommandRemapping() bool {
	// If the last executed CLI version is < 1.3.0, we need to refresh the plugin catalog.
	return lastversion.GetLastExecutedCLIVersion() == lastversion.OlderThan1_3_0
}

// updatePluginCatalog does the following catalog udpates:
// 1- Migrate context-scoped plugins as standalone plugins
// 2- Reads the info from each installed plugin and updates the plugin catalog
//
//	with the latest info.  We need to do this to make sure the command re-mapping
//	data is in the cache.
func updatePluginCatalog(outStream io.Writer) error {
	catalog.MigrateContextPluginsAsStandaloneIfNeeded()

	return refreshPluginsForRemapping(outStream)
}

// refreshPluginsForRemapping reads the info from each installed plugin
// and updates the plugin catalog with the latest info.
// We need to do this to make sure the command re-mapping data is in the cache.
func refreshPluginsForRemapping(outStream io.Writer) error {
	plugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return err
	}
	if len(plugins) == 0 {
		return nil
	}
	sort.Sort(cli.PluginInfoSorter(plugins))

	fmt.Fprintf(outStream, "Refreshing the %d installed plugins...\n", len(plugins))
	log.SetStderr(outStream)
	for i := range plugins {
		log.V(7).Infof("Refreshing plugin %s...", plugins[i].Name)

		if pInfo, err := refreshPluginInfo(&plugins[i], plugins[i].InstallationPath); err == nil {
			// Create a new catalog for each plugin in case a plugin tries to access the
			// catalog.  For example, the "test" plugin does access the catalog.
			c, err := catalog.NewContextCatalogUpdater("")
			if err != nil {
				log.V(7).Infof("Error creating catalog for plugin %s...", plugins[i].Name)
			} else {
				err = c.Upsert(pInfo)
				if err != nil {
					log.V(7).Infof("Error refreshing plugin %s...", plugins[i].Name)
				}
			}
			c.Unlock()
		} else {
			log.V(7).Infof("Error getting info for plugin %s...", plugins[i].Name)
		}
	}
	return nil
}

func refreshPluginInfo(plugin *cli.PluginInfo, pluginPath string) (*cli.PluginInfo, error) {
	bytesInfo, err := exec.Command(pluginPath, "info").Output()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the info for plugin %q", plugin.Name)
	}

	var newInfo cli.PluginInfo
	if err = json.Unmarshal(bytesInfo, &newInfo); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal plugin %q info", plugin.Name)
	}

	// Update the plugin info with the new info that older CLIs were not aware of.
	plugin.InvokedAs = newInfo.InvokedAs
	plugin.SupportedContextType = newInfo.SupportedContextType
	plugin.CommandMap = newInfo.CommandMap

	return plugin, nil
}
