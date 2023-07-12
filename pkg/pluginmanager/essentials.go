// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"fmt"
	"os"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
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

// InstallEssentialPluginGroups is a function that installs or upgrades the essential plugin groups.
func InstallEssentialPluginGroups() (string, error) {
	// Retrieve the name and version of the essential plugin group.
	name, version := GetEssentialsPluginGroupDetails()

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

// IsPluginsFromPluginGroupInstalled checks if all plugins from a specific group are installed and if a new version is available.
func IsPluginsFromPluginGroupInstalled(name, version string) (bool, bool, error) {
	// Disable logs.
	log.QuietMode(true)
	// Ensure that logs are re-enabled when we're done.
	defer log.QuietMode(false)

	// Retrieve the list of currently installed plugins.
	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		// If there's an error, wrap it with additional context and return.
		return false, false, fmt.Errorf("failed to get installed plugins: %w", err)
	}

	// Append the version to the plugin group name if it's not empty.
	pluginGroupName := name
	if version != "" {
		pluginGroupName = fmt.Sprintf("%v:%v", pluginGroupName, version)
	}

	// Parse the plugin group identifier from the name.
	groupIdentifier := plugininventory.PluginGroupIdentifierFromID(pluginGroupName)
	if groupIdentifier == nil {
		// If the identifier is nil, return an error.
		return false, false, fmt.Errorf("incorrect plugin-group %q specified", pluginGroupName)
	}

	// If no version is specified, use the latest version.
	if groupIdentifier.Version == "" {
		groupIdentifier.Version = cli.VersionLatest
	}

	// Create the discovery criteria for the plugin group.
	discoveryCriteria := &discovery.GroupDiscoveryCriteria{
		Vendor:    groupIdentifier.Vendor,
		Publisher: groupIdentifier.Publisher,
		Name:      groupIdentifier.Name,
		Version:   groupIdentifier.Version,
	}
	// Discover the plugin groups that match the criteria.
	groups, err := DiscoverPluginGroups(discoveryCriteria, discovery.WithForceCache())
	if err != nil {
		// If there's an error, wrap it with additional context and return.
		return false, false, fmt.Errorf("failed to discover plugin groups: %w", err)
	}
	if len(groups) == 0 {
		// If no groups are found, return an error.
		return false, false, fmt.Errorf("plugin-group %q cannot be found", pluginGroupName)
	}

	// Get the first group from the list.
	group := groups[0]

	// Create a list of mandatory plugins from the group.
	var pluginsOfGroup []*plugininventory.PluginGroupPluginEntry
	for _, plugin := range group.Versions[group.RecommendedVersion] {
		if plugin.Mandatory {
			pluginsOfGroup = append(pluginsOfGroup, plugin)
		}
	}

	// Check if all plugins from the group are installed and if a new version is available.
	return isAllPluginsFromGroupInstalled(pluginsOfGroup, installedPlugins), isNewPluginVersionAvailable(pluginsOfGroup, installedPlugins), nil
}

// isAllPluginsFromGroupInstalled checks if all plugins from a specific group are installed.
func isAllPluginsFromGroupInstalled(plugins []*plugininventory.PluginGroupPluginEntry, installedPlugins []cli.PluginInfo) bool {
	// Create a map to store the installed plugins.
	installedPluginsMap := make(map[string]*cli.PluginInfo)
	for i := range installedPlugins {
		// Use the plugin name, target, and version as the key.
		installedPlugin := installedPlugins[i]
		key := utils.GenerateKey(installedPlugin.Name, string(installedPlugin.Target), installedPlugin.Version)
		installedPluginsMap[key] = &installedPlugin
	}

	// Iterate over each plugin in the group.
	for _, plugin := range plugins {
		// Check if the current plugin is installed.
		key := utils.GenerateKey(plugin.Name, string(plugin.Target), plugin.Version)
		if _, ok := installedPluginsMap[key]; !ok {
			// If the plugin is not installed, return false.
			return false
		}
	}
	// If all plugins are installed, return true.
	return true
}

// isNewPluginVersionAvailable checks if a new version of any plugin from a specific group is available.
func isNewPluginVersionAvailable(plugins []*plugininventory.PluginGroupPluginEntry, installedPlugins []cli.PluginInfo) bool {
	// Create a map of installed plugins for faster lookup
	installedPluginsMap := make(map[string]*cli.PluginInfo)
	for i := range installedPlugins {
		installedPlugin := installedPlugins[i]
		key := utils.GenerateKey(installedPlugin.Name, string(installedPlugin.Target))
		installedPluginsMap[key] = &installedPlugin
	}

	for _, plugin := range plugins {
		key := utils.GenerateKey(plugin.Name, string(plugin.Target))
		installedPlugin, ok := installedPluginsMap[key]
		if !ok {
			continue // Plugin not installed, skip to next plugin
		}

		// Check if a new version of the plugin is available.
		if utils.IsNewVersion(plugin.Version, installedPlugin.Version) {
			return true // New version available
		}
	}

	return false // No new version available
}
