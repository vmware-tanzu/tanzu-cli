// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"fmt"
	"os"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// getEssentialsPluginGroupDetails is a function that retrieves the name and version of the essentials plugin group.
func getEssentialsPluginGroupDetails() (name, version string) {
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

// InstallEssentialPluginGroups is a function that installs or upgrades the essential plugin groups.
func InstallEssentialPluginGroups() (string, error) {
	// Retrieve the name and version of the essential plugin group.
	name, version := getEssentialsPluginGroupDetails()

	// Check if the plugins from the plugin group are installed, and if an update is available.
	installed, updateAvailable, err := IsPluginsFromPluginGroupInstalled(name, version)
	// If there's an error, return it with additional context.
	if err != nil {
		return "", fmt.Errorf("failed to check if plugins from group are installed: %w", err)
	}

	// If the plugins are already installed and no update is available, return with no error.
	if installed && !updateAvailable {
		return "", nil
	}

	// If an update is available, log a message indicating that the essential plugin groups are being upgraded.
	// If the plugins are not installed, log a message indicating that the essential plugin groups are being installed.
	actionMessage := constants.InstallEssentialPluginGroupsMsg
	if updateAvailable {
		actionMessage = constants.UpgradeEssentialPluginGroupsMsg
	}

	log.Info(actionMessage)

	// Print an empty line
	fmt.Println()

	// Attempt to install or upgrade the essential plugin group.
	_, err = installEssentialPluginGroup(name, version)

	// If there's an error during installation or upgrade, return it with additional context.
	if err != nil {
		return "", fmt.Errorf("failed to install or upgrade essential plugin group: %w", err)
	}

	// If the installation or upgrade is successful, return with no error.
	return "", nil
}

// installEssentialPluginGroup is a function that installs the essential plugin group.
func installEssentialPluginGroup(name, version string) (string, error) {
	pluginGroupNameWithVersion := name

	// Combine the name and version into a single string.
	if version != "" {
		pluginGroupNameWithVersion = fmt.Sprintf("%v:%v", pluginGroupNameWithVersion, version)
	}

	// Disable logs.
	log.QuietMode(true)
	// Ensure that logs are re-enabled when we're done.
	defer log.QuietMode(false)

	// Attempt to install the plugins from the group.
	groupWithVersion, err := InstallPluginsFromGroup("all", pluginGroupNameWithVersion)

	// If there's an error during installation, return it with additional context.
	if err != nil {
		return "", fmt.Errorf("failed to install plugins from group: %w", err)
	}

	// If the installation is successful, return the group with version.
	return groupWithVersion, nil
}
