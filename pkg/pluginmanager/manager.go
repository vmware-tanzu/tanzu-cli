// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package pluginmanager is responsible for plugin discovery and installation
package pluginmanager

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	cliv1alpha1 "github.com/vmware-tanzu/tanzu-cli/apis/cli/v1alpha1"
	"github.com/vmware-tanzu/tanzu-cli/pkg/artifact"
	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugincmdtree"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-cli/pkg/telemetry"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const (
	// exe is an executable file extension
	exe = ".exe"
	// ManifestFileName is the file name for the manifest.
	ManifestFileName = "manifest.yaml"
	// PluginManifestFileName is the file name for the manifest.
	PluginManifestFileName = "plugin_manifest.yaml"
	// PluginFileName is the file name for the plugin info.
	PluginFileName = "plugin.yaml"
	// String used to request the user to use the --target flag
	missingTargetStr             = "unable to uniquely identify plugin '%v'. Please specify the target (" + common.TargetList + ") of the plugin using the `--target` flag"
	errorWhileDiscoveringPlugins = "there was an error while discovering plugins, error information: '%v'"
	errorNoDiscoverySourcesFound = "there are no plugin discovery sources available. Please run 'tanzu plugin source init'"
)

var execCommand = exec.Command

type DeletePluginOptions struct {
	Target      configtypes.Target
	PluginName  string
	ForceDelete bool
}

// discoverSpecificPlugins returns all plugins that match the specified criteria from all PluginDiscovery sources,
// along with an aggregated error (if any) that occurred while creating the plugin discovery source or fetching plugins.
func discoverSpecificPlugins(pd []configtypes.PluginDiscovery, options ...discovery.DiscoveryOptions) ([]discovery.Discovered, error) {
	allPlugins := make([]discovery.Discovered, 0)
	errorList := make([]error, 0)
	for _, d := range pd {
		discObject, err := discovery.CreateDiscoveryFromV1alpha1(d, options...)
		if err != nil {
			errorList = append(errorList, errors.Wrapf(err, "unable to create discovery"))
			continue
		}

		plugins, err := discObject.List()
		if err != nil {
			errorList = append(errorList, errors.Wrapf(err, "unable to list plugins from discovery source '%v'", discObject.Name()))
			continue
		}

		allPlugins = append(allPlugins, plugins...)
	}
	return allPlugins, kerrors.NewAggregate(errorList)
}

// discoverSpecificPluginGroups returns all the plugin groups found in the discoveries
func discoverSpecificPluginGroups(pd []configtypes.PluginDiscovery, options ...discovery.DiscoveryOptions) ([]*plugininventory.PluginGroup, error) {
	var allGroups []*plugininventory.PluginGroup
	for _, d := range pd {
		groupDisc, err := discovery.CreateGroupDiscovery(d, options...)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create group discovery")
		}

		groups, err := groupDisc.GetGroups()
		if err != nil {
			log.Warningf("unable to list groups from discovery '%v': %v", groupDisc.Name(), err.Error())
			continue
		}

		if len(groups) > 0 {
			allGroups = append(allGroups, groups...)
		}
	}
	return mergeDuplicateGroups(allGroups), nil
}

// DiscoverStandalonePlugins returns the available standalone plugins
func DiscoverStandalonePlugins(options ...discovery.DiscoveryOptions) ([]discovery.Discovered, error) {
	discoveries, err := getPluginDiscoveries()
	if err != nil {
		return nil, err
	} else if len(discoveries) == 0 {
		return nil, errors.New(errorNoDiscoverySourcesFound)
	}

	plugins, err := discoverSpecificPlugins(discoveries, options...)
	for i := range plugins {
		plugins[i].Scope = common.PluginScopeStandalone
		plugins[i].Status = common.PluginStatusNotInstalled
	}
	return mergeDuplicatePlugins(plugins), err
}

// DiscoverPluginGroups returns the available plugin groups
func DiscoverPluginGroups(options ...discovery.DiscoveryOptions) ([]*plugininventory.PluginGroup, error) {
	discoveries, err := getPluginDiscoveries()
	if err != nil {
		return nil, err
	}
	if len(discoveries) == 0 {
		return nil, errors.New(errorNoDiscoverySourcesFound)
	}

	groups, err := discoverSpecificPluginGroups(discoveries, options...)
	if err != nil {
		return nil, err
	}
	return groups, err
}

// GetAdditionalTestPluginDiscoveries returns an array of plugin discoveries that
// are meant to be used for testing new plugin version.  The comma-separated list of
// such discoveries can be specified through the environment variable
// "TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY".
// Each entry in the variable should be the URI of an OCI image of the DB of the
// discovery in question.
func GetAdditionalTestPluginDiscoveries() []configtypes.PluginDiscovery {
	var testDiscoveries []configtypes.PluginDiscovery
	testDiscoveryImages := config.GetAdditionalTestDiscoveryImages()
	for idx, image := range testDiscoveryImages {
		testDiscoveries = append(testDiscoveries, configtypes.PluginDiscovery{
			OCI: &configtypes.OCIDiscovery{
				Name:  fmt.Sprintf("disc_%d", idx),
				Image: image,
			},
		})
	}
	return testDiscoveries
}

// DiscoverServerPlugins returns the available plugins associated all the active contexts
func DiscoverServerPlugins() ([]discovery.Discovered, error) {
	var plugins []discovery.Discovered
	var errList []error

	currentContextMap, err := configlib.GetAllCurrentContextsMap()
	if err != nil {
		return nil, err
	}
	if len(currentContextMap) == 0 {
		return plugins, nil
	}

	for _, context := range currentContextMap {
		var discoverySources []configtypes.PluginDiscovery
		discoverySources = append(discoverySources, context.DiscoverySources...)
		discoverySources = append(discoverySources, defaultDiscoverySourceBasedOnContext(context)...)
		discoveredPlugins, err := discoverSpecificPlugins(discoverySources)

		// If there is an error while discovering plugins from all of the given plugin sources,
		// append the error to the error list and continue processing the discoveredPlugins,
		// as there may still be plugins that were successfully discovered from some of the discovery sources.
		if err != nil {
			errList = append(errList, err)
		}
		for i := range discoveredPlugins {
			discoveredPlugins[i].Scope = common.PluginScopeContext
			discoveredPlugins[i].Status = common.PluginStatusNotInstalled
			discoveredPlugins[i].ContextName = context.Name

			// Associate Target of the plugin based on the Context Type of the Context
			switch context.Target {
			case configtypes.TargetTMC:
				discoveredPlugins[i].Target = configtypes.TargetTMC
			case configtypes.TargetK8s:
				discoveredPlugins[i].Target = configtypes.TargetK8s
			}
		}
		plugins = append(plugins, discoveredPlugins...)
	}
	return plugins, kerrors.NewAggregate(errList)
}

func mergePluginEntries(plugin1, plugin2 *discovery.Discovered) *discovery.Discovered {
	// Plugins with the same name having `k8s` and `none` targets are also considered the same for
	// backward compatibility reasons, considering, we are adding `k8s` targeted plugins as root level commands.
	if plugin1.Target == configtypes.TargetUnknown {
		plugin1.Target = plugin2.Target
	}

	// Combine the installation status and installedVersion result when combining plugins
	if plugin2.Status == common.PluginStatusInstalled {
		plugin1.Status = common.PluginStatusInstalled
	}
	if plugin2.InstalledVersion != "" {
		plugin1.InstalledVersion = plugin2.InstalledVersion
	}

	// Build a combined Source string
	if plugin1.Source != plugin2.Source {
		plugin1.Source = fmt.Sprintf("%s/%s", plugin1.Source, plugin2.Source)
	}

	// The discovery type could be OCI or Local.
	// When dealing with different discovery types, unset it
	if plugin1.DiscoveryType != plugin2.DiscoveryType {
		plugin1.DiscoveryType = ""
	}

	artifacts1, ok := plugin1.Distribution.(distribution.Artifacts)
	if !ok {
		// This should not happened
		log.Warningf("Plugin '%s' has an unexpected distribution type", plugin1.Name)
		return plugin1
	}

	artifacts2, ok := plugin2.Distribution.(distribution.Artifacts)
	if !ok {
		// This should not happened
		log.Warningf("Plugin '%s' has an unexpected distribution type", plugin2.Name)
		return plugin1
	}

	// For every version in the second plugin, if it doesn't already exist
	// in the first plugin, add it.
	// Also build the new list of supported versions
	for version := range artifacts2 {
		_, exists := artifacts1[version]
		if !exists {
			artifacts1[version] = artifacts2[version]
			plugin1.SupportedVersions = append(plugin1.SupportedVersions, version)
		}
	}
	plugin1.Distribution = artifacts1
	_ = utils.SortVersions(plugin1.SupportedVersions)

	// Set the recommended version to the highest version
	if len(plugin1.SupportedVersions) > 0 {
		plugin1.RecommendedVersion = plugin1.SupportedVersions[len(plugin1.SupportedVersions)-1]
	}

	// Keep the following fields from the first plugin found
	// - Optional
	// - ContextName
	// - Scope
	return plugin1
}

// mergeDuplicatePlugins combines the same plugins to eliminate duplicates by merging the information
// of multiple entries of the same plugin into a single entry.  For example, the Central Repository can
// provide details about a plugin, but an additional test discovery can provide other versions of the
// same plugin.  This function will join all the information into one.
// A plugin is determined by its name-target combination.
// Note that if two versions of the same plugin are found more than once, it will be the first one
// found that will be kept.  The order of the array "plugins" therefore matters.
// This merge operation is deterministic due to the sequence of sources/plugins that we process always
// being the same.
func mergeDuplicatePlugins(plugins []discovery.Discovered) []discovery.Discovered {
	mapOfSelectedPlugins := make(map[string]*discovery.Discovered)
	for i := range plugins {
		target := plugins[i].Target
		if target == configtypes.TargetUnknown {
			// Two plugins with the same name having `k8s` and `none` as targets are also considered the same for
			// backward compatibility reasons. This is because we are adding `k8s`-targeted plugins as root-level commands.
			target = configtypes.TargetK8s
		}

		// If plugin doesn't exist in the map then add the plugin to the map
		// else merge the two entries, giving priority to the first one found
		key := fmt.Sprintf("%s_%s", plugins[i].Name, target)
		dp, exists := mapOfSelectedPlugins[key]
		if !exists {
			mapOfSelectedPlugins[key] = &plugins[i]
		} else {
			mapOfSelectedPlugins[key] = mergePluginEntries(dp, &plugins[i])
		}
	}

	var mergedPlugins []discovery.Discovered
	for key := range mapOfSelectedPlugins {
		mergedPlugins = append(mergedPlugins, *mapOfSelectedPlugins[key])
	}
	return mergedPlugins
}

func mergeGroupEntries(group1, group2 *plugininventory.PluginGroup) *plugininventory.PluginGroup {
	// For every version in the second group, if it doesn't already exist
	// in the first group, add it.
	for version := range group2.Versions {
		_, exists := group1.Versions[version]
		if !exists {
			group1.Versions[version] = group2.Versions[version]
		}
	}

	// Find the latest version
	if len(group1.Versions) > 0 {
		latestVersions := []string{group1.RecommendedVersion, group2.RecommendedVersion}
		_ = utils.SortVersions(latestVersions)

		// Set the recommended version and the description to the ones from the highest version group
		if group2.RecommendedVersion == latestVersions[1] {
			// If it is group2 that has the highest version, replace the RecommendedVersion and Description
			group1.RecommendedVersion = group2.RecommendedVersion
			group1.Description = group2.Description
		}
	}

	return group1
}

// mergeDuplicateGroups combines the same plugin groups to eliminate duplicates by merging the information
// of multiple entries of the same group into a single entry.  For example, the Central Repository can
// provide details about a plugin group, but an additional test discovery can provide other versions of the
// same group.  This function will join all the information into one.
// A group is determined by its vendor-publisher/name combination.
// Note that if two versions of the same group are found more than once, it will be the first one
// found that will be kept.  The order of the array "groups" therefore matters.
// This merge operation is deterministic due to the sequence of sources/groups that we process always
// being the same.
func mergeDuplicateGroups(groups []*plugininventory.PluginGroup) []*plugininventory.PluginGroup {
	mapOfSelectedGroups := make(map[string]*plugininventory.PluginGroup)
	for _, newGroup := range groups {
		// If group doesn't exist in the map then add it.
		// Otherwise merge the two entries, giving priority to the first one found.
		key := plugininventory.PluginGroupToID(newGroup)
		existingGroup, exists := mapOfSelectedGroups[key]
		if !exists {
			mapOfSelectedGroups[key] = newGroup
		} else {
			mapOfSelectedGroups[key] = mergeGroupEntries(existingGroup, newGroup)
		}
	}

	var mergedGroups []*plugininventory.PluginGroup
	for key := range mapOfSelectedGroups {
		mergedGroups = append(mergedGroups, mapOfSelectedGroups[key])
	}
	return mergedGroups
}

func setAvailablePluginsStatus(availablePlugins []discovery.Discovered, installedPlugins []cli.PluginInfo) {
	for i := range installedPlugins {
		for j := range availablePlugins {
			if installedPlugins[i].Name == availablePlugins[j].Name && installedPlugins[i].Target == availablePlugins[j].Target {
				// Match found, Check for update available and update status
				if installedPlugins[i].DiscoveredRecommendedVersion == availablePlugins[j].RecommendedVersion {
					availablePlugins[j].Status = common.PluginStatusInstalled
				} else {
					availablePlugins[j].Status = common.PluginStatusUpdateAvailable
				}
				availablePlugins[j].InstalledVersion = installedPlugins[i].Version
			}
		}
	}
}

// DescribePlugin describes a plugin.
func DescribePlugin(pluginName string, target configtypes.Target) (info *cli.PluginInfo, err error) {
	plugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return nil, err
	}
	var matchedPlugins []cli.PluginInfo
	for i := range plugins {
		if plugins[i].Name == pluginName &&
			(target == configtypes.TargetUnknown || target == plugins[i].Target) {
			matchedPlugins = append(matchedPlugins, plugins[i])
		}
	}

	if len(matchedPlugins) == 0 {
		if target != configtypes.TargetUnknown {
			return nil, errors.Errorf("unable to find plugin '%v' for target '%s'", pluginName, string(target))
		}
		return nil, errors.Errorf("unable to find plugin '%v'", pluginName)
	}

	if len(matchedPlugins) == 1 {
		return &matchedPlugins[0], nil
	}

	for i := range matchedPlugins {
		if matchedPlugins[i].Target == target {
			return &matchedPlugins[i], nil
		}
	}

	return nil, errors.Errorf(missingTargetStr, pluginName)
}

// InitializePlugin initializes the plugin configuration
func InitializePlugin(plugin *cli.PluginInfo) error {
	if plugin == nil {
		return fmt.Errorf("could not get plugin information")
	}

	b, err := execCommand(plugin.InstallationPath, "post-install").CombinedOutput()

	// Note: If user is installing old version of plugin than it is possible that
	// the plugin does not implement post-install command. Ignoring the
	// errors if the command does not exist for a particular plugin.
	if err != nil && !strings.Contains(string(b), "unknown command") {
		log.Warningf("Warning: Failed to initialize plugin '%q' after installation. %v", plugin.Name, string(b))
	}

	return nil
}

// InstallStandalonePlugin installs a plugin by name, version and target as a standalone plugin.
func InstallStandalonePlugin(pluginName, version string, target configtypes.Target) error {
	return installPlugin(pluginName, version, target, "")
}

// InstallPluginFromContext installs a plugin by name, version and target as a context-scope plugin.
func InstallPluginFromContext(pluginName, version string, target configtypes.Target, contextName string) error {
	if contextName == "" {
		log.Warning("Missing context name for a context-scope plugin: %s/%s/%s", pluginName, version, string(target))
	}
	return installPlugin(pluginName, version, target, contextName)
}

// installs a plugin by name, version and target.
// If the contextName is not empty, it implies the plugin is a context-scope plugin, otherwise
// we are installing a standalone plugin.
//
//nolint:gocyclo
func installPlugin(pluginName, version string, target configtypes.Target, contextName string) error {
	discoveries, err := getPluginDiscoveries()
	if err != nil {
		return err
	}
	if len(discoveries) == 0 {
		return errors.New(errorNoDiscoverySourcesFound)
	}
	criteria := &discovery.PluginDiscoveryCriteria{
		Name:    pluginName,
		Target:  target,
		Version: version,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	errorList := make([]error, 0)
	availablePlugins, err := discoverSpecificPlugins(discoveries, discovery.WithPluginDiscoveryCriteria(criteria))
	if err != nil {
		errorList = append(errorList, err)
	}

	if len(availablePlugins) == 0 {
		if target != configtypes.TargetUnknown {
			errorList = append(errorList, errors.Errorf("unable to find plugin '%v' with version '%v' for target '%s'", pluginName, version, string(target)))
			return kerrors.NewAggregate(errorList)
		}
		errorList = append(errorList, errors.Errorf("unable to find plugin '%v' with version '%v'", pluginName, version))
		return kerrors.NewAggregate(errorList)
	}

	// Deal with duplicates from different plugin discovery sources
	availablePlugins = mergeDuplicatePlugins(availablePlugins)

	var matchedPlugins []discovery.Discovered
	for i := range availablePlugins {
		if availablePlugins[i].Name == pluginName &&
			(target == configtypes.TargetUnknown || target == availablePlugins[i].Target) {
			// If the plugin was recommended by a context, lets store that info
			if contextName != "" {
				availablePlugins[i].ContextName = contextName
			}
			matchedPlugins = append(matchedPlugins, availablePlugins[i])
		}
	}
	if len(matchedPlugins) == 0 {
		if target != configtypes.TargetUnknown {
			errorList = append(errorList, errors.Errorf("unable to find plugin '%v' with version '%v' for target '%s'", pluginName, version, string(target)))
			return kerrors.NewAggregate(errorList)
		}
		errorList = append(errorList, errors.Errorf("unable to find plugin '%v' with version '%v'", pluginName, version))
		return kerrors.NewAggregate(errorList)
	}

	if len(matchedPlugins) == 1 {
		// If the version requested was the RecommendedVersion, we should set it explicitly
		if version == cli.VersionLatest {
			version = matchedPlugins[0].RecommendedVersion
		}

		return installOrUpgradePlugin(&matchedPlugins[0], version, false)
	}

	for i := range matchedPlugins {
		if matchedPlugins[i].Target == target {
			// If the version requested was the RecommendedVersion, we should set it explicitly
			if version == cli.VersionLatest {
				version = matchedPlugins[i].RecommendedVersion
			}
			return installOrUpgradePlugin(&matchedPlugins[i], version, false)
		}
	}
	errorList = append(errorList, errors.Errorf(missingTargetStr, pluginName))
	return kerrors.NewAggregate(errorList)
}

// UpgradePlugin upgrades a plugin from the given repository.
func UpgradePlugin(pluginName, version string, target configtypes.Target) error {
	// Upgrade is only triggered from a manual user operation.
	// This means a plugin is installed manually, which means it is installed as a standalone plugin.
	return InstallStandalonePlugin(pluginName, version, target)
}

// InstallPluginsFromGroup installs either the specified plugin or all plugins from the specified group version.
// If the group version is not specified, the latest available version will be used.
// The group identifier including the version used is returned.
func InstallPluginsFromGroup(pluginName, groupIDAndVersion string, options ...PluginManagerOptions) (string, error) { //nolint:gocyclo,funlen
	// Initialize plugin manager options and enable logs by default
	opts := NewPluginManagerOpts()
	for _, option := range options {
		option(opts)
	}

	// Enable or Disable logs
	opts.SetLogMode()
	defer opts.ResetLogMode()

	discoveries, err := getPluginDiscoveries()
	if err != nil {
		return "", err
	}
	if len(discoveries) == 0 {
		return "", errors.New(errorNoDiscoverySourcesFound)
	}

	groupIdentifier := plugininventory.PluginGroupIdentifierFromID(groupIDAndVersion)
	if groupIdentifier == nil {
		return "", fmt.Errorf("could not find group '%s'", groupIDAndVersion)
	}
	if groupIdentifier.Version == "" {
		// If the version is not specified to install from, we use the latest
		groupIdentifier.Version = cli.VersionLatest
	}
	criteria := &discovery.GroupDiscoveryCriteria{
		Vendor:    groupIdentifier.Vendor,
		Publisher: groupIdentifier.Publisher,
		Name:      groupIdentifier.Name,
		Version:   groupIdentifier.Version,
	}

	groups, err := discoverSpecificPluginGroups(discoveries, discovery.WithGroupDiscoveryCriteria(criteria))
	if err != nil {
		return "", err
	}

	if len(groups) == 0 {
		return "", errors.Errorf("unable to find plugin group %v", groupIDAndVersion)
	}

	if len(groups) > 1 {
		log.Warningf("unexpected: group '%s' was found more than once.  Using the first one.", groupIDAndVersion)
	}

	pg := groups[0]
	if groupIdentifier.Version == cli.VersionLatest {
		// If we are installing the latest version, we should set the version we found
		groupIdentifier.Version = pg.RecommendedVersion
		groupIDAndVersion = fmt.Sprintf("%s:%s", groupIDAndVersion, pg.RecommendedVersion)
	}

	numErrors := 0
	numInstalled := 0
	mandatoryPluginsExist := false
	pluginExist := false
	for _, plugin := range pg.Versions[groupIdentifier.Version] {
		if pluginName == cli.AllPlugins || pluginName == plugin.Name {
			pluginExist = true
			if plugin.Mandatory {
				mandatoryPluginsExist = true
				err := InstallStandalonePlugin(plugin.Name, plugin.Version, plugin.Target)
				if err != nil {
					numErrors++
					log.Warningf("unable to install plugin '%s': %v", plugin.Name, err.Error())
				} else {
					numInstalled++
				}
			}
		}
	}

	if !pluginExist {
		return groupIDAndVersion, fmt.Errorf("plugin '%s' is not part of the group '%s'", pluginName, groupIDAndVersion)
	}

	if !mandatoryPluginsExist {
		if pluginName == cli.AllPlugins {
			return groupIDAndVersion, fmt.Errorf("plugin group '%s' has no mandatory plugins to install", groupIDAndVersion)
		}
		return groupIDAndVersion, fmt.Errorf("plugin '%s' from group '%s' is not mandatory to install", pluginName, groupIDAndVersion)
	}

	if numErrors > 0 {
		return groupIDAndVersion, fmt.Errorf("could not install %d plugin(s) from group '%s'", numErrors, groupIDAndVersion)
	}

	if numInstalled == 0 {
		return groupIDAndVersion, fmt.Errorf("plugin '%s' is not part of the group '%s'", pluginName, groupIDAndVersion)
	}

	return groupIDAndVersion, nil
}

func installOrUpgradePlugin(p *discovery.Discovered, version string, installTestPlugin bool) error {
	if p.Target == configtypes.TargetUnknown {
		log.Infof("Installing plugin '%v:%v'", p.Name, version)
	} else {
		log.Infof("Installing plugin '%v:%v' with target '%v'", p.Name, version, p.Target)
	}

	var plugin *cli.PluginInfo
	if !installTestPlugin {
		// If we need to install the test plugin we know we are doing a local
		// installation.  In that case, we don't use the cache as the binary is
		// already local to the machine.
		plugin = getPluginFromCache(p, version)
	}
	if plugin == nil {
		binary, err := fetchAndVerifyPlugin(p, version)
		if err != nil {
			return err
		}

		plugin, err = installAndDescribePlugin(p, version, binary)
		if err != nil {
			return err
		}
	} else {
		log.Infof("Plugin binary for '%v:%v' found in cache", p.Name, version)
	}

	if installTestPlugin {
		if err := doInstallTestPlugin(p, plugin.InstallationPath, version); err != nil {
			return err
		}
	}

	return updatePluginInfoAndInitializePlugin(p, plugin)
}

func getPluginFromCache(p *discovery.Discovered, version string) *cli.PluginInfo {
	pluginArtifact, err := p.Distribution.DescribeArtifact(version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil
	}

	// TODO(khouzam): We should not be checking the presence of the binary directly here,
	// as it bypasses the plugin catalog abstraction.  Instead, we should ask the plugin
	// catalog to know if the plugin binary is present already.
	pluginFileName := fmt.Sprintf("%s_%s_%s", version, pluginArtifact.Digest, p.Target)
	pluginPath := filepath.Join(common.DefaultPluginRoot, p.Name, pluginFileName)

	if cli.BuildArch().IsWindows() {
		pluginPath += exe
	}
	if _, err := os.Stat(pluginPath); err != nil {
		return nil
	}

	plugin, err := describePlugin(p, pluginPath)
	if err != nil {
		return nil
	}
	return plugin
}

func fetchAndVerifyPlugin(p *discovery.Discovered, version string) ([]byte, error) {
	// verify plugin before download
	err := verifyPluginPreDownload(p, version)
	if err != nil {
		return nil, errors.Wrapf(err, "%q plugin pre-download verification failed", p.Name)
	}

	b, err := p.Distribution.Fetch(version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch the plugin metadata for plugin %q", p.Name)
	}

	// verify plugin after download but before installation
	d, err := p.Distribution.GetDigest(version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}
	err = verifyPluginPostDownload(p, d, b)
	if err != nil {
		return nil, errors.Wrapf(err, "%q plugin post-download verification failed", p.Name)
	}
	return b, nil
}

func installAndDescribePlugin(p *discovery.Discovered, version string, binary []byte) (*cli.PluginInfo, error) {
	pluginFileName := fmt.Sprintf("%s_%x_%s", version, sha256.Sum256(binary), p.Target)
	pluginPath := filepath.Join(common.DefaultPluginRoot, p.Name, pluginFileName)

	if err := os.MkdirAll(filepath.Dir(pluginPath), os.ModePerm); err != nil {
		return nil, err
	}

	if cli.BuildArch().IsWindows() {
		pluginPath += exe
	}

	if err := os.WriteFile(pluginPath, binary, 0755); err != nil {
		return nil, errors.Wrap(err, "could not write file")
	}

	return describePlugin(p, pluginPath)
}

func describePlugin(p *discovery.Discovered, pluginPath string) (*cli.PluginInfo, error) {
	bytesInfo, err := execCommand(pluginPath, "info").Output()
	if err != nil {
		return nil, errors.Wrapf(err, "could not describe plugin %q", p.Name)
	}

	var plugin cli.PluginInfo
	if err = json.Unmarshal(bytesInfo, &plugin); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal plugin %q description", p.Name)
	}
	plugin.InstallationPath = pluginPath
	plugin.Discovery = p.Source
	plugin.DiscoveredRecommendedVersion = p.RecommendedVersion
	plugin.Target = p.Target
	plugin.Scope = p.Scope
	if plugin.Version == p.RecommendedVersion {
		plugin.Status = common.PluginStatusInstalled
	} else {
		plugin.Status = common.PluginStatusUpdateAvailable
	}
	return &plugin, nil
}

func doInstallTestPlugin(p *discovery.Discovered, pluginPath, version string) error {
	log.Infof("Installing test plugin for '%v:%v'", p.Name, version)
	binary, err := p.Distribution.FetchTest(version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		if os.Getenv("TZ_ENFORCE_TEST_PLUGIN") == "1" {
			return errors.Wrapf(err, "unable to install test plugin for '%v:%v'", p.Name, version)
		}
		log.Infof("  ... skipped: %s", err.Error())
		return nil
	}
	testPluginPath := cli.TestPluginPathFromPluginPath(pluginPath)

	err = os.WriteFile(testPluginPath, binary, 0755)
	if err != nil {
		return errors.Wrap(err, "error while saving test plugin binary")
	}
	return nil
}

func updatePluginInfoAndInitializePlugin(p *discovery.Discovered, plugin *cli.PluginInfo) error {
	c, err := catalog.NewContextCatalogUpdater(p.ContextName)
	if err != nil {
		return err
	}
	if err := c.Upsert(plugin); err != nil {
		log.Info("Plugin Info could not be updated in cache")
	}

	// We are not using defer `c.Unlock()` to release the lock here because we want to unlock the lock as soon as possible
	// Using `defer` here will release the lock after `InitializePlugin`, `ConfigureDefaultFeatureFlagsIfMissing`,
	// `addPluginToCommandTreeCache` invocations which is not what we want.
	c.Unlock()

	if err := InitializePlugin(plugin); err != nil {
		log.Infof("could not initialize plugin after installing: %v", err.Error())
	}
	if err := config.ConfigureDefaultFeatureFlagsIfMissing(plugin.DefaultFeatureFlags); err != nil {
		log.Infof("could not configure default featureflags for the plugin: %v", err.Error())
	}
	// add plugin to the plugin command tree cache for telemetry to consume later for plugin command chain parsing
	addPluginToCommandTreeCache(plugin)
	return nil
}

// addPluginToCommandTreeCache would construct and add the plugin command tree to the command tree cache
// which would be consumed by telemetry for plugin command chain parsing
func addPluginToCommandTreeCache(plugin *cli.PluginInfo) {
	// update the plugin command tree cache
	ctr, err := plugincmdtree.NewCache()
	if err != nil {
		telemetry.LogError(err, "")
		return
	}
	err = ctr.ConstructAndAddTree(plugin)
	if err != nil {
		telemetry.LogError(err, "")
	}
}

// deletePluginFromCommandTreeCache deletes the plugin command tree from the command tree cache
// which would be consumed by telemetry
func deletePluginFromCommandTreeCache(plugin *cli.PluginInfo) {
	// delete the plugin command tree from the plugin command tree cache
	ctr, err := plugincmdtree.NewCache()
	if err != nil {
		telemetry.LogError(err, "")
		return
	}
	err = ctr.DeleteTree(plugin)
	if err != nil {
		telemetry.LogError(err, "")
	}
}

// DeletePlugin deletes a plugin.
func DeletePlugin(options DeletePluginOptions) error {
	serverNames, err := configlib.GetAllCurrentContextsList()
	if err != nil {
		return err
	}

	var matchedCatalogNames []string
	var matchedPlugins []cli.PluginInfo

	// Add empty serverName for standalone plugins
	serverNames = append(serverNames, "")

	for _, serverName := range serverNames {
		c, err := catalog.NewContextCatalog(serverName)
		if err != nil {
			continue
		}

		plugins := c.List()
		for i := range plugins {
			if plugins[i].Name == options.PluginName &&
				(options.Target == configtypes.TargetUnknown || options.Target == plugins[i].Target) {
				matchedPlugins = append(matchedPlugins, plugins[i])
				matchedCatalogNames = append(matchedCatalogNames, serverName)
			}
		}
	}

	if len(matchedPlugins) == 0 {
		if options.Target != configtypes.TargetUnknown {
			return errors.Errorf("unable to find plugin '%v' for target '%s'", options.PluginName, string(options.Target))
		}
		return errors.Errorf("unable to find plugin '%v'", options.PluginName)
	}

	// It is possible that the catalog contains two entries for a name/target combination:
	// a context-scope installation and a standalone installation.  We need to delete both in this case.
	// If all matched plugins are from the same target, this is when we can still delete them all.
	uniqueTarget := matchedPlugins[0].Target
	for i := range matchedPlugins {
		if matchedPlugins[i].Target != uniqueTarget {
			return errors.Errorf(missingTargetStr, options.PluginName)
		}
	}

	if !options.ForceDelete {
		if err := component.AskForConfirmation(
			fmt.Sprintf("Deleting Plugin '%s' for target '%s'. Are you sure?",
				options.PluginName, string(uniqueTarget))); err != nil {
			return err
		}
	}
	// Delete the plugins from the command tree cache which would be consumed by telemetry
	for i := range matchedPlugins {
		deletePluginFromCommandTreeCache(&matchedPlugins[i])
	}
	// Delete all plugins that match since they are all from the same target
	return doDeletePluginFromCatalog(options.PluginName, uniqueTarget, matchedCatalogNames)

	// TODO: delete the plugin binary if it is not used by any server
}

func doDeletePluginFromCatalog(pluginName string, target configtypes.Target, catalogNames []string) error {
	for _, n := range catalogNames {
		// We must create one catalog at a time to be able to delete a plugin.
		// If we re-use the catalogs created above, when we delete the plugin
		// in one catalog, the next catalog will put it back since that catalog
		// was created before the plugin was deleted.
		c, err := catalog.NewContextCatalogUpdater(n)
		if err != nil {
			continue
		}

		err = c.Delete(catalog.PluginNameTarget(pluginName, target))
		c.Unlock()
		if err != nil {
			return fmt.Errorf("plugin %q could not be deleted from cache", pluginName)
		}
	}

	return nil
}

// SyncPlugins will install the plugins required by the current contexts.
// If the central-repo is disabled, all discovered plugins will be installed.
func SyncPlugins() error {
	log.Info("Checking for required plugins...")
	errList := make([]error, 0)
	// We no longer sync standalone plugins.
	// With a centralized approach to discovering plugins, synchronizing
	// standalone plugins would install ALL plugins available for ALL
	// products.
	// Instead, we only synchronize any plugins that are specifically specified
	// by the contexts.
	//
	// Note: to install all plugins for a specific product, plugin groups will
	// need to be used.
	plugins, err := DiscoverServerPlugins()
	if err != nil {
		errList = append(errList, err)
	}
	if installedPlugins, err := pluginsupplier.GetInstalledServerPlugins(); err == nil {
		setAvailablePluginsStatus(plugins, installedPlugins)
	}

	installed := false
	for idx := range plugins {
		if plugins[idx].Status == common.PluginStatusNotInstalled || plugins[idx].Status == common.PluginStatusUpdateAvailable {
			installed = true
			p := plugins[idx]
			err = InstallPluginFromContext(p.Name, p.RecommendedVersion, p.Target, p.ContextName)
			if err != nil {
				errList = append(errList, err)
			}
		}
	}
	err = kerrors.NewAggregate(errList)
	if err != nil {
		return err
	}

	if !installed {
		log.Info("All required plugins are already installed and up-to-date")
	} else {
		log.Info("Successfully installed all required plugins")
	}
	return nil
}

// InstallPluginsFromLocalSource installs plugin from local source directory
//
//nolint:gocyclo
func InstallPluginsFromLocalSource(pluginName, version string, target configtypes.Target, localPath string, installTestPlugin bool) error {
	// Set default local plugin distro to local-path as while installing the plugin
	// from local source we should take t
	common.DefaultLocalPluginDistroDir = localPath

	availablePlugins, err := DiscoverPluginsFromLocalSource(localPath)
	if err != nil {
		return errors.Wrap(err, "unable to discover plugins")
	}

	var errList []error

	var matchedPlugins []discovery.Discovered
	for i := range availablePlugins {
		if (pluginName == cli.AllPlugins || availablePlugins[i].Name == pluginName) &&
			(target == configtypes.TargetUnknown || target == availablePlugins[i].Target) {
			matchedPlugins = append(matchedPlugins, availablePlugins[i])
		}
	}
	if len(matchedPlugins) == 0 {
		if pluginName == cli.AllPlugins {
			if target != configtypes.TargetUnknown {
				return errors.Errorf("unable to find any plugins for target '%s'", string(target))
			}
			return errors.Errorf("unable to find any plugins at the specified location")
		}

		if target != configtypes.TargetUnknown {
			return errors.Errorf("unable to find plugin '%v' with version '%v' for target '%s'", pluginName, version, string(target))
		}
		return errors.Errorf("unable to find plugin '%v' with version '%v'", pluginName, version)
	}

	if len(matchedPlugins) == 1 {
		return installOrUpgradePlugin(&matchedPlugins[0], FindVersion(matchedPlugins[0].RecommendedVersion, version), installTestPlugin)
	}

	for i := range matchedPlugins {
		// Install all plugins otherwise include all matching plugins
		if pluginName == cli.AllPlugins || matchedPlugins[i].Target == target {
			err = installOrUpgradePlugin(&matchedPlugins[i], FindVersion(matchedPlugins[i].RecommendedVersion, version), installTestPlugin)
			if err != nil {
				errList = append(errList, err)
			}
		}
	}

	err = kerrors.NewAggregate(errList)
	if err != nil {
		return err
	}

	return nil
}

// DiscoverPluginsFromLocalSource returns the available plugins that are discovered from the provided local path
func DiscoverPluginsFromLocalSource(localPath string) ([]discovery.Discovered, error) {
	if localPath == "" {
		return nil, nil
	}

	plugins, err := discoverPluginsFromLocalSource(localPath)
	// If no error then return the discovered plugins
	if err == nil {
		return plugins, nil
	}

	// Check if the manifest.yaml or plugin_manifest.yaml file exists to see if the directory can be used or not
	_, err2 := os.Stat(filepath.Join(localPath, ManifestFileName))
	_, err3 := os.Stat(filepath.Join(localPath, PluginManifestFileName))
	if errors.Is(err2, os.ErrNotExist) && errors.Is(err3, os.ErrNotExist) {
		return nil, err
	}

	// As manifest.yaml or plugin_manifest.yaml file exists it assumes in this case the directory is supported
	// and attempt to process it as such
	return discoverPluginsFromLocalSourceBasedOnManifestFile(localPath)
}

func discoverPluginsFromLocalSource(localPath string) ([]discovery.Discovered, error) {
	// Set default local plugin distro to localpath while installing the plugin
	// from local source. This is done to allow CLI to know the basepath incase the
	// relative path is provided as part of CLIPlugin definition for local discovery
	common.DefaultLocalPluginDistroDir = localPath

	var pds []configtypes.PluginDiscovery

	items, err := os.ReadDir(filepath.Join(localPath, "discovery"))
	if err != nil {
		return nil, errors.Wrapf(err, "error while reading local plugin manifest directory")
	}
	for _, item := range items {
		if item.IsDir() {
			pd := configtypes.PluginDiscovery{
				Local: &configtypes.LocalDiscovery{
					Name: "",
					Path: filepath.Join(localPath, "discovery", item.Name()),
				},
			}
			pds = append(pds, pd)
		}
	}

	plugins, err := discoverSpecificPlugins(pds)
	if err != nil {
		log.Warningf(errorWhileDiscoveringPlugins, err.Error())
	}

	for i := range plugins {
		plugins[i].Scope = common.PluginScopeStandalone
		plugins[i].Status = common.PluginStatusNotInstalled
		plugins[i].DiscoveryType = common.DiscoveryTypeLocal
	}
	return plugins, nil
}

// Clean deletes all plugins and tests.
func Clean() error {
	errorList := make([]error, 0)

	// Clean the plugin catalog
	if err := catalog.CleanCatalogCache(); err != nil {
		errorList = append(errorList, errors.Wrapf(err, "Failed to clean the catalog cache"))
	}

	// Clean plugin inventory cache
	pluginDataDir := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName)
	if err := os.RemoveAll(pluginDataDir); err != nil {
		errorList = append(errorList, errors.Wrapf(err, "Failed to clean the plugin inventory cache"))
	}

	// Remove all plugin binaries
	if err := os.RemoveAll(common.DefaultPluginRoot); err != nil {
		errorList = append(errorList, errors.Wrapf(err, "Failed to clean the plugin binaries"))
	}

	// Remove the plugin command tree registry cache
	commandTreeDir := filepath.Dir(plugincmdtree.GetPluginsCommandTreeCachePath())
	if err := os.RemoveAll(commandTreeDir); err != nil {
		errorList = append(errorList, errors.Wrapf(err, "Failed to clean the plugin command tree cache"))
	}

	return kerrors.NewAggregate(errorList)
}

// getCLIPluginResourceWithLocalDistroFromPluginInfo return cliv1alpha1.CLIPlugin resource from the pluginInfo
// Note: This function generates cliv1alpha1.CLIPlugin which contains only single local distribution type artifact for
// OS-ARCH where user is running the cli
// This function is only used to create CLIPlugin resource for local plugin installation with legacy directory structure
func getCLIPluginResourceWithLocalDistroFromPluginInfo(plugin *cli.PluginInfo, pluginBinaryPath string) cliv1alpha1.CLIPlugin {
	return cliv1alpha1.CLIPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: plugin.Name,
		},
		Spec: cliv1alpha1.CLIPluginSpec{
			Description:        plugin.Description,
			RecommendedVersion: plugin.Version,
			Target:             plugin.Target,
			Artifacts: map[string]cliv1alpha1.ArtifactList{
				plugin.Version: []cliv1alpha1.Artifact{
					{
						URI:  pluginBinaryPath,
						Type: common.DistributionTypeLocal,
						OS:   runtime.GOOS,
						Arch: runtime.GOARCH,
					},
				},
			},
		},
	}
}

// discoverPluginsFromLocalSourceBasedOnManifestFile returns the available plugins
// that are discovered from the provided local path
func discoverPluginsFromLocalSourceBasedOnManifestFile(localPath string) ([]discovery.Discovered, error) {
	if localPath == "" {
		return nil, nil
	}

	var manifest *cli.Manifest
	// Get the plugin manifest object from manifest.yaml file
	manifest1, err1 := getPluginManifestResource(filepath.Join(localPath, ManifestFileName))
	manifest2, err2 := getPluginManifestResource(filepath.Join(localPath, PluginManifestFileName))

	if err2 == nil {
		manifest = manifest2
	} else if err1 == nil {
		manifest = manifest1
	} else {
		return nil, kerrors.NewAggregate([]error{err1, err2})
	}

	var discoveredPlugins []discovery.Discovered

	// Create  discovery.Discovered object for all locally available plugin
	for _, p := range manifest.Plugins {
		if p.Name == common.CoreName {
			continue
		}

		pluginInfo := cli.PluginInfo{
			Name:        p.Name,
			Description: p.Description,
			Target:      configtypes.Target(p.Target),
		}

		if len(p.Versions) != 0 {
			pluginInfo.Version = p.Versions[0]
		}

		if pluginInfo.Version == "" {
			// Get the plugin version from the plugin.yaml file
			plugin, err := getPluginInfoResource(filepath.Join(localPath, p.Name, PluginFileName))
			if err != nil {
				return nil, fmt.Errorf("could not unmarshal plugin.yaml: %v", err)
			}
			pluginInfo.Version = plugin.Version
		}

		absLocalPath, err := filepath.Abs(localPath)
		if err != nil {
			return nil, err
		}
		// With legacy configuration directory structure creating the pluginBinary path from plugin Info
		// Sample path: cli/[target]/<plugin-name>/<plugin-version>/tanzu-<plugin-name>-<os>_<arch>
		// 				cli/[target]/v0.14.0/tanzu-login-darwin_amd64
		// As mentioned above, we expect the binary for user's OS-ARCH is present and hence creating path accordingly
		pluginBinaryPath := filepath.Join(absLocalPath, string(pluginInfo.Target), p.Name, pluginInfo.Version, fmt.Sprintf("tanzu-%s-%s_%s", p.Name, runtime.GOOS, runtime.GOARCH))
		if cli.BuildArch().IsWindows() {
			pluginBinaryPath += exe
		}
		// Check if the pluginBinary file exists or not
		if _, err := os.Stat(pluginBinaryPath); errors.Is(err, os.ErrNotExist) {
			return nil, errors.Wrapf(err, "unable to find plugin binary for %q", p.Name)
		}

		p := getCLIPluginResourceWithLocalDistroFromPluginInfo(&pluginInfo, pluginBinaryPath)

		// Create  discovery.Discovered resource from CLIPlugin resource
		dp, err := discovery.DiscoveredFromK8sV1alpha1(&p)
		if err != nil {
			return nil, err
		}
		dp.DiscoveryType = common.DiscoveryTypeLocal
		dp.Scope = common.PluginScopeStandalone
		dp.Status = common.PluginStatusNotInstalled

		discoveredPlugins = append(discoveredPlugins, dp)
	}

	return discoveredPlugins, nil
}

// getPluginManifestResource returns cli.Manifest resource by reading manifest file
func getPluginManifestResource(manifestFilePath string) (*cli.Manifest, error) {
	b, err := os.ReadFile(manifestFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not find %s file: %v", filepath.Base(manifestFilePath), err)
	}

	var manifest cli.Manifest
	err = yaml.Unmarshal(b, &manifest)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal %s: %v", filepath.Base(manifestFilePath), err)
	}
	return &manifest, nil
}

// getPluginInfoResource returns cliapi.PluginInfo resource by reading plugin file
func getPluginInfoResource(pluginFilePath string) (*cli.PluginInfo, error) {
	b, err := os.ReadFile(pluginFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not find %s file: %v", filepath.Base(pluginFilePath), err)
	}

	var plugin cli.PluginInfo
	err = yaml.Unmarshal(b, &plugin)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal %s: %v", filepath.Base(pluginFilePath), err)
	}
	return &plugin, nil
}

// verifyPluginPreDownload verifies that the plugin distribution repo is trusted
// and returns error if the verification fails.
func verifyPluginPreDownload(p *discovery.Discovered, version string) error {
	artifactInfo, err := p.Distribution.DescribeArtifact(version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	if artifactInfo.Image != "" {
		return verifyRegistry(artifactInfo.Image)
	}
	if artifactInfo.URI != "" {
		return verifyArtifactLocation(artifactInfo.URI)
	}
	return errors.Errorf("no download information available for artifact \"%s:%s:%s:%s\"", p.Name, p.RecommendedVersion, runtime.GOOS, runtime.GOARCH)
}

// verifyRegistry verifies the authenticity of the registry from where cli is
// trying to download the plugins by comparing it with the list of trusted registries
func verifyRegistry(image string) error {
	trustedRegistries := config.GetTrustedRegistries()
	for _, tr := range trustedRegistries {
		// Verify fullname of the registry has trusted registry fullname as the prefix
		if tr != "" && strings.HasPrefix(image, tr) {
			return nil
		}
	}
	return errors.Errorf("untrusted registry detected with image %q. Allowed registries are %v", image, trustedRegistries)
}

// verifyArtifactLocation verifies the artifact location from where the cli is
// trying to download the plugins by comparing it with the list of trusted locations
func verifyArtifactLocation(uri string) error {
	art, err := artifact.NewURIArtifact(uri)
	if err != nil {
		return err
	}

	switch art.(type) {
	case *artifact.LocalArtifact:
		// trust local artifacts implicitly
		return nil

	default:
		trustedLocations := config.GetTrustedArtifactLocations()
		for _, tl := range trustedLocations {
			// Verify that the URI has a trusted location as the prefix
			if tl != "" && strings.HasPrefix(uri, tl) {
				return nil
			}
		}
		return errors.Errorf("untrusted artifact location detected with URI %q. Allowed locations are %v", uri, trustedLocations)
	}
}

// verifyPluginPostDownload compares the source digest of the plugin against the
// SHA256 hash of the downloaded binary to ensure that the binary was not altered
// during transit.
func verifyPluginPostDownload(p *discovery.Discovered, srcDigest string, b []byte) error {
	if srcDigest == "" {
		// Skip if the Distribution repo does not have the source digest.
		return nil
	}

	d := sha256.Sum256(b)
	actDigest := fmt.Sprintf("%x", d)
	if actDigest != srcDigest {
		return errors.Errorf("plugin %q has been corrupted during download. source digest: %s, actual digest: %s", p.Name, srcDigest, actDigest)
	}

	return nil
}

func FindVersion(recommendedPluginVersion, requestedVersion string) string {
	if requestedVersion == "" || requestedVersion == cli.VersionLatest {
		return recommendedPluginVersion
	}
	return requestedVersion
}

// getPluginDiscoveries returns the plugin discoveries found in the configuration file.
func getPluginDiscoveries() ([]configtypes.PluginDiscovery, error) {
	// Look for testing discoveries.  Those should be stored and searched AFTER the central repo.
	testDiscoveries := GetAdditionalTestPluginDiscoveries()

	// The configured discoveries should be searched BEFORE the test discoveries.
	// For example, if the staging central repo is added as a test discovery, it
	// may contain older versions of a plugin that is now published to the production
	// central repo; we therefore need to search the test discoveries last.
	discoverySources, _ := configlib.GetCLIDiscoverySources()
	return append(discoverySources, testDiscoveries...), nil
}

// IsPluginsFromPluginGroupInstalled checks if all plugins from a specific group are installed and if a new version is available.
// This function uses cache data to verify rather than fetching the inventory image
func IsPluginsFromPluginGroupInstalled(name, version string, options ...PluginManagerOptions) (bool, bool, error) {
	// Initialize plugin manager options and enable logs by default
	opts := NewPluginManagerOpts()
	for _, option := range options {
		option(opts)
	}
	// Enable or Disable logs
	opts.SetLogMode()
	defer opts.ResetLogMode()

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
	groups, err := DiscoverPluginGroups(discovery.WithGroupDiscoveryCriteria(discoveryCriteria), discovery.WithUseLocalCacheOnly())
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
