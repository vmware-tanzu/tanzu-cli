// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

func PopulateDefaultCentralDiscovery(force bool) error {
	discoverySources, _ := configlib.GetCLIDiscoverySources()

	// Add the default central plugin discovery if it is not there.
	// If len(discoverySources)==0, we don't add the central discovery;
	// this allows a user to delete the default central discovery and not
	// have the CLI add it again.  A user can then use "plugin source init"
	// to add the default discovery again.
	if force || discoverySources == nil {
		defaultDiscovery := configtypes.PluginDiscovery{
			OCI: &configtypes.OCIDiscovery{
				Name:  DefaultStandaloneDiscoveryName,
				Image: constants.TanzuCLIDefaultCentralPluginDiscoveryImage,
			},
		}
		return configlib.SetCLIDiscoverySource(defaultDiscovery)
	}
	return nil
}
