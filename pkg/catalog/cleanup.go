// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// DeleteIncorrectPluginEntriesFromCatalog deletes the old plugin entries associated with
// 'global' target if a plugin with the same name and different target already exists
// This can happen because of an existing bug in v0.28.x and v0.29.x version of tanzu-cli
// where we allow plugins to be installed when target value is different even if target
// values of “(empty), `global` and `kubernetes` can correspond to same root level command
func DeleteIncorrectPluginEntriesFromCatalog() {
	c, unlock, err := getCatalogCache(true)
	if err != nil {
		return
	}
	defer unlock()

	pluginAssociations := []PluginAssociation{c.StandAlonePlugins}
	for _, spa := range c.ServerPlugins {
		pluginAssociations = append(pluginAssociations, spa)
	}

	for i := range pluginAssociations {
		for _, path := range pluginAssociations[i] {
			pluginInfo, exists := c.IndexByPath[path]
			if !exists {
				continue
			}

			// The "unknown" target was previously used in two scenarios:
			// 1- to represent the global target (>= v0.28 and < v0.90)
			// 2- to represent either the global or kubernetes target (< v0.28)
			// If we have a plugin with the "global" or "k8s" target we should remove any similar plugin using
			// the "unknown" target.
			if pluginInfo.Target == configtypes.TargetGlobal || pluginInfo.Target == configtypes.TargetK8s {
				pluginAssociations[i].Remove(PluginNameTarget(pluginInfo.Name, configtypes.TargetUnknown))
			}
		}
	}

	_ = saveCatalogCache(c)
}
