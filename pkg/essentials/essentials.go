// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package essentials contain essentials plugin group lifecycle operations
package essentials

import (
	"os"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

// GetEssentialsPluginGroupDetails is a function that retrieves the name and version of the essentials plugin group.
func GetEssentialsPluginGroupDetails() (name, version string) {
	// Set the default name for the essential plugin group.
	name = constants.DefaultCLIEssentialsPluginGroupName

	// Check if the environment variable for the essentials plugin group name is set.
	// If it is, override the default name with the value from the environment variable.
	essentialsPluginGroupName := os.Getenv(constants.TanzuCLIEssentialsPluginGroupName)
	if essentialsPluginGroupName != "" {
		name = essentialsPluginGroupName
	}

	// Check if the environment variable for the essentials plugin group version is set.
	// If it is, set the version to the value from the environment variable.
	essentialsPluginGroupVersion := os.Getenv(constants.TanzuCLIEssentialsPluginGroupVersion)
	if essentialsPluginGroupVersion != "" {
		version = essentialsPluginGroupVersion
	}

	// Return the name and version of the essentials plugin group.
	return name, version
}
