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
	"go.uber.org/multierr"
	"golang.org/x/mod/semver"
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
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
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
)

var execCommand = exec.Command

type DeletePluginOptions struct {
	Target      configtypes.Target
	PluginName  string
	ForceDelete bool
}

// ValidatePlugin validates the plugin info.
func ValidatePlugin(p *cli.PluginInfo) (err error) {
	// skip builder plugin for bootstrapping
	if p.Name == "builder" {
		return nil
	}
	if p.Name == "" {
		err = multierr.Append(err, errors.New("plugin name cannot be empty"))
	}
	if p.Version == "" {
		err = multierr.Append(err, fmt.Errorf("plugin %q version cannot be empty", p.Name))
	}
	if !semver.IsValid(p.Version) && p.Version != "dev" {
		err = multierr.Append(err, fmt.Errorf("version %q %q is not a valid semantic version", p.Name, p.Version))
	}
	if p.Description == "" {
		err = multierr.Append(err, fmt.Errorf("plugin %q description cannot be empty", p.Name))
	}
	if p.Group == "" {
		err = multierr.Append(err, fmt.Errorf("plugin %q group cannot be empty", p.Name))
	}
	return
}

func discoverPlugins(pd []configtypes.PluginDiscovery) ([]discovery.Discovered, error) {
	return discoverSpecificPlugins(pd, nil)
}

// discoverSpecificPlugins returns the available plugin matching the criteria, if the criteria is not nil.
func discoverSpecificPlugins(pd []configtypes.PluginDiscovery, criteria *discovery.PluginDiscoveryCriteria) ([]discovery.Discovered, error) {
	allPlugins := make([]discovery.Discovered, 0)
	for _, d := range pd {
		discObject, err := discovery.CreateDiscoveryFromV1alpha1(d, criteria)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create discovery")
		}

		plugins, err := discObject.List()
		if err != nil {
			log.Warningf("unable to list plugin from discovery '%v': %v", discObject.Name(), err.Error())
			continue
		}

		allPlugins = append(allPlugins, plugins...)
	}
	return allPlugins, nil
}

// discoverPluginGroups returns all the plugin groups found in the discoveries
func discoverPluginGroups(pd []configtypes.PluginDiscovery) ([]*discovery.DiscoveredPluginGroups, error) {
	var allDiscovered []*discovery.DiscoveredPluginGroups
	for _, d := range pd {
		discObject, err := discovery.CreateDiscoveryFromV1alpha1(d, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create discovery")
		}

		groupDisc, ok := discObject.(discovery.GroupDiscovery)
		if !ok {
			// This discovery does not support plugin groups
			continue
		}

		groups, err := groupDisc.GetAllGroups()
		if err != nil {
			log.Warningf("unable to list groups from discovery '%v': %v", discObject.Name(), err.Error())
			continue
		}

		allDiscovered = append(
			allDiscovered,
			&discovery.DiscoveredPluginGroups{
				Source: discObject.Name(),
				Groups: groups,
			})
	}
	return allDiscovered, nil
}

// discoverPluginGroup returns the one matching plugin group found in the discoveries
func discoverPluginGroup(pd []configtypes.PluginDiscovery, groupID string) (*plugininventory.PluginGroup, error) {
	groupsByDiscovery, err := discoverPluginGroups(pd)
	if err != nil {
		return nil, err
	}

	var matchingDiscoveries []string
	var matchingGroup *plugininventory.PluginGroup
	for _, discAndGroups := range groupsByDiscovery {
		for _, group := range discAndGroups.Groups {
			id := fmt.Sprintf("%s-%s/%s", group.Vendor, group.Publisher, group.Name)
			if id == groupID {
				// Found the group.
				if matchingGroup == nil {
					// Store the first matching group found
					matchingGroup = group
				}
				matchingDiscoveries = append(matchingDiscoveries, discAndGroups.Source)
			}
		}
	}

	if len(matchingDiscoveries) > 1 {
		log.Warningf("group '%s' was found in multiple discoveries: %v.  Using the first one.", groupID, matchingDiscoveries)
	}

	return matchingGroup, nil
}

// DiscoverStandalonePlugins returns the available standalone plugins
func DiscoverStandalonePlugins() ([]discovery.Discovered, error) {
	discoveries, err := getPluginDiscoveries()
	if err != nil {
		return nil, err
	}

	plugins, err := discoverPlugins(discoveries)
	if err != nil {
		return plugins, err
	}

	for i := range plugins {
		plugins[i].Scope = common.PluginScopeStandalone
		plugins[i].Status = common.PluginStatusNotInstalled
	}
	return plugins, err
}

// DiscoverPluginGroups returns the available plugin groups
func DiscoverPluginGroups() ([]*discovery.DiscoveredPluginGroups, error) {
	discoveries, err := getPluginDiscoveries()
	if err != nil {
		return nil, err
	}

	return discoverPluginGroups(discoveries)
}

// getPreReleasePluginDiscovery
// For our pre-releases we use an environment variable to point to the
// repository of plugins.  This is because the configuration
// cfg.ClientOptions.CLI.DiscoverySources
// is read by older CLIs so we don't want to modify it.
// TODO(khouzam): remove before 1.0
func getPreReleasePluginDiscovery() ([]configtypes.PluginDiscovery, error) {
	centralRepoTestImage := os.Getenv(constants.ConfigVariablePreReleasePluginRepoImage)
	if centralRepoTestImage == "" {
		// Don't set a default value.  This test repo URI is not meant to be public.
		return nil, fmt.Errorf("you must set the environment variable %s to the URI of the image of the plugin repository.  Please see the documentation", constants.ConfigVariablePreReleasePluginRepoImage)
	} else if centralRepoTestImage == constants.ConfigVariablePreReleasePluginRepoImageBypass {
		return nil, nil
	}
	return []configtypes.PluginDiscovery{
		{
			OCI: &configtypes.OCIDiscovery{
				Name:  "default_pre_release",
				Image: centralRepoTestImage,
			},
		}}, nil
}

// getAdditionalTestPluginDiscoveries returns an array of plugin discoveries that
// are meant to be used for testing new plugin version.  The comma-separated list of
// such discoveries can be specified through the environment variable
// "ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY".
// Each entry in the variable should be the URI of an OCI image of the DB of the
// discovery in question.
func getAdditionalTestPluginDiscoveries() []configtypes.PluginDiscovery {
	var testDiscoveries []configtypes.PluginDiscovery
	testDiscoveryImages := strings.Split(os.Getenv(constants.ConfigVariableAdditionalDiscoveryForTesting), ",")
	count := 0
	for _, image := range testDiscoveryImages {
		image = strings.TrimSpace(image)
		if image != "" {
			testDiscoveries = append(testDiscoveries, configtypes.PluginDiscovery{
				OCI: &configtypes.OCIDiscovery{
					Name:  fmt.Sprintf("test_%d", count),
					Image: image,
				},
			})
			count++
		}
	}
	return testDiscoveries
}

// DiscoverServerPlugins returns the available plugins associated all the active contexts
func DiscoverServerPlugins() ([]discovery.Discovered, error) {
	// If the context and target feature is enabled, discover plugins from all currentContexts
	// Else discover plugin based on current Server
	if configlib.IsFeatureActivated(constants.FeatureContextCommand) {
		return discoverServerPluginsBasedOnAllCurrentContexts()
	}
	return discoverServerPluginsBasedOnCurrentServer()
}

func discoverServerPluginsBasedOnAllCurrentContexts() ([]discovery.Discovered, error) {
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
		discoveredPlugins, err := discoverPlugins(discoverySources)
		if err != nil {
			errList = append(errList, err)
			continue
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

// discoverServerPluginsBasedOnCurrentServer returns the available plugins associated with the given server
func discoverServerPluginsBasedOnCurrentServer() ([]discovery.Discovered, error) {
	var plugins []discovery.Discovered

	server, err := configlib.GetCurrentServer() // nolint:staticcheck // Deprecated
	if err != nil || server == nil {
		// If servername is not specified than returning empty list
		// as there are no server plugins that can be discovered
		return plugins, nil
	}
	var discoverySources []configtypes.PluginDiscovery
	discoverySources = append(discoverySources, server.DiscoverySources...)
	discoverySources = append(discoverySources, defaultDiscoverySourceBasedOnServer(server)...)

	plugins, err = discoverPlugins(discoverySources)
	if err != nil {
		return plugins, err
	}
	for i := range plugins {
		plugins[i].Scope = common.PluginScopeContext
		plugins[i].Status = common.PluginStatusNotInstalled
	}
	return plugins, nil
}

// DiscoverPlugins returns all the discovered plugins including standalone and context-scoped plugins
// Context scoped plugin discovery happens for all active contexts
func DiscoverPlugins() ([]discovery.Discovered, []discovery.Discovered) {
	serverPlugins, err := DiscoverServerPlugins()
	if err != nil {
		log.Warningf("unable to discover server plugins, %v", err.Error())
	}

	standalonePlugins, err := DiscoverStandalonePlugins()
	if err != nil {
		log.Warningf("unable to discover standalone plugins, %v", err.Error())
	}

	// TODO(anuj): Remove duplicate plugins with server plugins getting higher priority
	return serverPlugins, standalonePlugins
}

// AvailablePlugins returns the list of available plugins including discovered and installed plugins.
// Plugin discovery happens for all active contexts
func AvailablePlugins() ([]discovery.Discovered, error) {
	discoveredServerPlugins, discoveredStandalonePlugins := DiscoverPlugins()
	return availablePlugins(discoveredServerPlugins, discoveredStandalonePlugins)
}

// AvailablePluginsFromLocalSource returns the list of available plugins from local source
func AvailablePluginsFromLocalSource(localPath string) ([]discovery.Discovered, error) {
	localStandalonePlugins, err := DiscoverPluginsFromLocalSource(localPath)
	if err != nil {
		log.Warningf("Unable to discover standalone plugins from local source, %v", err.Error())
	}
	return availablePlugins([]discovery.Discovered{}, localStandalonePlugins)
}

func availablePlugins(discoveredServerPlugins, discoveredStandalonePlugins []discovery.Discovered) ([]discovery.Discovered, error) {
	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return nil, err
	}

	availablePlugins := availablePluginsFromStandaloneAndServerPlugins(discoveredServerPlugins, discoveredStandalonePlugins)
	setAvailablePluginsStatus(availablePlugins, installedPlugins)

	installedStandalonePlugins, err := pluginsupplier.GetInstalledStandalonePlugins()
	if err != nil {
		return nil, err
	}
	installedButNotDiscoveredPlugins := getInstalledButNotDiscoveredStandalonePlugins(availablePlugins, installedStandalonePlugins)
	availablePlugins = append(availablePlugins, installedButNotDiscoveredPlugins...)

	availablePlugins = combineDuplicatePlugins(availablePlugins)

	return availablePlugins, nil
}

// combineDuplicatePlugins combines same plugins to eliminate duplicates
// When there is a plugin name conflicts and target of both the plugins are same, remove duplicate one.
// In addition to above, plugin with same name having `k8s` and `none` target are also considered same for
// backward compatibility reasons. Considering, we are adding `k8s` targeted plugins as root level commands.
//
// E.g. A plugin 'foo' getting discovered/installed with `<none>` target and a plugin `foo` getting discovered
// with `k8s` discovery (having `k8s` target) should be treated as same plugin.
// This function takes this case into consideration and removes `<none>` targeted plugin for above the mentioned scenario.
func combineDuplicatePlugins(availablePlugins []discovery.Discovered) []discovery.Discovered {
	mapOfSelectedPlugins := make(map[string]discovery.Discovered)

	// TODO: If there are multiple discovered (but not installed) plugins of the same name then we will
	// always end up keeping one deterministically (due to the sequence of sources/plugins that we process),
	// but we should merge the result and show the combined result for duplicate plugins
	combinePluginInstallationStatus := func(plugin1, plugin2 discovery.Discovered) discovery.Discovered {
		// Combine the installation status and installedVersion result when combining plugins
		if plugin2.Status == common.PluginStatusInstalled {
			plugin1.Status = common.PluginStatusInstalled
		}
		if plugin2.InstalledVersion != "" {
			plugin1.InstalledVersion = plugin2.InstalledVersion
		}
		return plugin1
	}

	for i := range availablePlugins {
		if availablePlugins[i].Target == configtypes.TargetUnknown {
			// As we are considering None targeted and k8s target plugin to be treated as same plugins
			// in the case of plugin name conflicts, using `k8s` target to determine the plugin already
			// exists or not.
			// If plugin already exists in the map then combining the installation status for both the plugins
			key := fmt.Sprintf("%s_%s", availablePlugins[i].Name, configtypes.TargetK8s)
			dp, exists := mapOfSelectedPlugins[key]
			if !exists {
				mapOfSelectedPlugins[key] = availablePlugins[i]
			} else {
				mapOfSelectedPlugins[key] = combinePluginInstallationStatus(dp, availablePlugins[i])
			}
		} else {
			// If plugin doesn't exist in the map then add the plugin to the map
			// else combine the installation status for both the plugins
			key := fmt.Sprintf("%s_%s", availablePlugins[i].Name, availablePlugins[i].Target)
			dp, exists := mapOfSelectedPlugins[key]
			if !exists {
				mapOfSelectedPlugins[key] = availablePlugins[i]
			} else if availablePlugins[i].Target == configtypes.TargetK8s || availablePlugins[i].Scope == common.PluginScopeContext {
				mapOfSelectedPlugins[key] = combinePluginInstallationStatus(availablePlugins[i], dp)
			}
		}
	}

	var selectedPlugins []discovery.Discovered
	for key := range mapOfSelectedPlugins {
		selectedPlugins = append(selectedPlugins, mapOfSelectedPlugins[key])
	}

	return selectedPlugins
}

func mergePluginEntries(plugin1, plugin2 *discovery.Discovered) *discovery.Discovered {
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

	// Keep the following fields from the first plugin found
	// - RecommendedVersion
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

func getInstalledButNotDiscoveredStandalonePlugins(availablePlugins []discovery.Discovered, installedPlugins []cli.PluginInfo) []discovery.Discovered {
	var newPlugins []discovery.Discovered
	for i := range installedPlugins {
		found := false
		for j := range availablePlugins {
			if installedPlugins[i].Name == availablePlugins[j].Name && installedPlugins[i].Target == availablePlugins[j].Target {
				found = true
				// If plugin is installed but marked as not installed as part of availablePlugins list
				// mark the plugin as installed
				// This is possible if user has used --local mode to install the plugin which is also
				// getting discovered from the configured discovery sources
				if availablePlugins[j].Status == common.PluginStatusNotInstalled {
					availablePlugins[j].Status = common.PluginStatusInstalled
				}
			}
		}
		if !found {
			p := DiscoveredFromPlugininfo(&installedPlugins[i])
			p.Scope = common.PluginScopeStandalone
			p.Status = common.PluginStatusInstalled
			p.InstalledVersion = installedPlugins[i].Version
			newPlugins = append(newPlugins, p)
		}
	}
	return newPlugins
}

// DiscoveredFromPlugininfo returns discovered plugin object from k8sV1alpha1
func DiscoveredFromPlugininfo(p *cli.PluginInfo) discovery.Discovered {
	dp := discovery.Discovered{
		Name:               p.Name,
		Description:        p.Description,
		RecommendedVersion: p.Version,
		Source:             p.Discovery,
		SupportedVersions:  []string{p.Version},
		Target:             p.Target,
	}
	return dp
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

func availablePluginsFromStandaloneAndServerPlugins(discoveredServerPlugins, discoveredStandalonePlugins []discovery.Discovered) []discovery.Discovered {
	availablePlugins := discoveredServerPlugins

	// Check whether the default standalone discovery type is local or not
	isLocalStandaloneDiscovery := config.GetDefaultStandaloneDiscoveryType() == common.DiscoveryTypeLocal

	for i := range discoveredStandalonePlugins {
		matchIndex := pluginIndexForName(availablePlugins, &discoveredStandalonePlugins[i])

		// Add the standalone plugin to available plugins if it doesn't exist in the serverPlugins list
		// OR
		// Current standalone discovery or plugin discovered is of type 'local'
		// We are overriding the discovered plugins that we got from server in case of 'local' discovery type
		// to allow developers to use the plugins that are built locally and not returned from the server
		// This local discovery is only used for development purpose and should not be used for production
		if matchIndex < 0 {
			availablePlugins = append(availablePlugins, discoveredStandalonePlugins[i])
			continue
		}
		if isLocalStandaloneDiscovery || discoveredStandalonePlugins[i].DiscoveryType == common.DiscoveryTypeLocal { // matchIndex >= 0 is guaranteed here
			availablePlugins[matchIndex] = discoveredStandalonePlugins[i]
		}
	}
	return availablePlugins
}

func pluginIndexForName(availablePlugins []discovery.Discovered, p *discovery.Discovered) int {
	for j := range availablePlugins {
		if p != nil && p.Name == availablePlugins[j].Name && p.Target == availablePlugins[j].Target {
			return j
		}
	}
	return -1 // haven't found a match
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

	return nil, errors.Errorf("unable to uniquely identify plugin '%v'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag", pluginName)
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
// nolint: gocyclo
func installPlugin(pluginName, version string, target configtypes.Target, contextName string) error {
	if configlib.IsFeatureActivated(constants.FeatureDisableCentralRepositoryForTesting) {
		// The legacy installation can figure out if the plugin is from a context
		// because it searches all contexts for plugins.  So, we don't need to pass on that parameter.
		return legacyPluginInstall(pluginName, version, target)
	}

	discoveries, err := getPluginDiscoveries()
	if err != nil || len(discoveries) == 0 {
		return err
	}
	availablePlugins, err := discoverSpecificPlugins(discoveries, &discovery.PluginDiscoveryCriteria{
		Name:    pluginName,
		Target:  target,
		Version: version,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	})
	if err != nil {
		return err
	}

	if len(availablePlugins) == 0 {
		if target != configtypes.TargetUnknown {
			return errors.Errorf("unable to find plugin '%v' for target '%s'", pluginName, string(target))
		}
		return errors.Errorf("unable to find plugin '%v'", pluginName)
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
			return errors.Errorf("unable to find plugin '%v' for target '%s'", pluginName, string(target))
		}
		return errors.Errorf("unable to find plugin '%v'", pluginName)
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

	return errors.Errorf("unable to uniquely identify plugin '%v'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag", pluginName)
}

// legacyInstallPlugin installs a plugin by name, version and target.
// This function is only used without the Central Repository feature.
func legacyPluginInstall(pluginName, version string, target configtypes.Target) error {
	availablePlugins, err := AvailablePlugins()
	if err != nil {
		return err
	}

	var matchedPlugins []discovery.Discovered
	for i := range availablePlugins {
		if availablePlugins[i].Name == pluginName &&
			(target == configtypes.TargetUnknown || target == availablePlugins[i].Target) {
			matchedPlugins = append(matchedPlugins, availablePlugins[i])
		}
	}
	if len(matchedPlugins) == 0 {
		if target != configtypes.TargetUnknown {
			return errors.Errorf("unable to find plugin '%v' for target '%s'", pluginName, string(target))
		}
		return errors.Errorf("unable to find plugin '%v'", pluginName)
	}

	if len(matchedPlugins) == 1 {
		return installOrUpgradePlugin(&matchedPlugins[0], version, false)
	}

	for i := range matchedPlugins {
		if matchedPlugins[i].Target == target {
			return installOrUpgradePlugin(&matchedPlugins[i], version, false)
		}
	}

	return errors.Errorf("unable to uniquely identify plugin '%v'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag", pluginName)
}

// UpgradePlugin upgrades a plugin from the given repository.
func UpgradePlugin(pluginName, version string, target configtypes.Target) error {
	// Upgrade is only triggered from a manual user operation.
	// This means a plugin is installed manually, which means it is installed as a standalone plugin.
	return InstallStandalonePlugin(pluginName, version, target)
}

// InstallPluginsFromGroup installs either the specified plugin or all plugins from the named group
func InstallPluginsFromGroup(pluginName, groupID string) error {
	discoveries, err := getPluginDiscoveries()
	if err != nil || len(discoveries) == 0 {
		return err
	}

	group, err := discoverPluginGroup(discoveries, groupID)
	if err != nil {
		return err
	}

	if group == nil {
		return fmt.Errorf("could not find group '%s'", groupID)
	}

	numErrors := 0
	numInstalled := 0
	for _, plugin := range group.Plugins {
		if pluginName == cli.AllPlugins || pluginName == plugin.Name {
			err = InstallStandalonePlugin(plugin.Name, plugin.Version, plugin.Target)
			if err != nil {
				numErrors++
				log.Warningf("unable to install plugin '%s': %v", plugin.Name, err.Error())
			} else {
				numInstalled++
			}
		}
	}

	if numErrors > 0 {
		return fmt.Errorf("could not install %d plugin(s) from group '%s'", numErrors, groupID)
	}
	if numInstalled == 0 {
		return fmt.Errorf("plugin '%s' is not part of the group '%s'", pluginName, groupID)
	}
	return nil
}

// GetRecommendedVersionOfPlugin returns recommended version of the plugin
func GetRecommendedVersionOfPlugin(pluginName string, target configtypes.Target) (string, error) {
	availablePlugins, err := AvailablePlugins()
	if err != nil {
		return "", err
	}

	var matchedPlugins []discovery.Discovered
	for i := range availablePlugins {
		if availablePlugins[i].Name == pluginName &&
			(target == configtypes.TargetUnknown || target == availablePlugins[i].Target) {
			matchedPlugins = append(matchedPlugins, availablePlugins[i])
		}
	}
	if len(matchedPlugins) == 0 {
		if target != configtypes.TargetUnknown {
			return "", errors.Errorf("unable to find plugin '%v' for target '%s'", pluginName, string(target))
		}
		return "", errors.Errorf("unable to find plugin '%v'", pluginName)
	}

	if len(matchedPlugins) == 1 {
		return matchedPlugins[0].RecommendedVersion, nil
	}

	for i := range matchedPlugins {
		if matchedPlugins[i].Target == target {
			return matchedPlugins[i].RecommendedVersion, nil
		}
	}
	return "", errors.Errorf("unable to uniquely identify plugin '%v'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag", pluginName)
}

func installOrUpgradePlugin(p *discovery.Discovered, version string, installTestPlugin bool) error {
	if p.Target == configtypes.TargetUnknown {
		log.Infof("Installing plugin '%v:%v'", p.Name, version)
	} else {
		log.Infof("Installing plugin '%v:%v' with target '%v'", p.Name, version, p.Target)
	}

	binary, err := fetchAndVerifyPlugin(p, version)
	if err != nil {
		return err
	}

	plugin, err := installAndDescribePlugin(p, version, binary)
	if err != nil {
		return err
	}

	if installTestPlugin {
		if err := doInstallTestPlugin(p, plugin.InstallationPath, version); err != nil {
			return err
		}
	}

	return updatePluginInfoAndInitializePlugin(p, plugin)
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
	c, err := catalog.NewContextCatalog(p.ContextName)
	if err != nil {
		return err
	}
	if err := c.Upsert(plugin); err != nil {
		log.Info("Plugin Info could not be updated in cache")
	}
	if err := InitializePlugin(plugin); err != nil {
		log.Infof("could not initialize plugin after installing: %v", err.Error())
	}
	if err := config.ConfigureDefaultFeatureFlagsIfMissing(plugin.DefaultFeatureFlags); err != nil {
		log.Infof("could not configure default featureflags for the plugin: %v", err.Error())
	}
	return nil
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
			return errors.Errorf("unable to uniquely identify plugin '%v'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag", options.PluginName)
		}
	}

	if !options.ForceDelete {
		if err := component.AskForConfirmation(
			fmt.Sprintf("Deleting Plugin '%s' for target '%s'. Are you sure?",
				options.PluginName, string(uniqueTarget))); err != nil {
			return err
		}
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
		c, err := catalog.NewContextCatalog(n)
		if err != nil {
			continue
		}

		err = c.Delete(catalog.PluginNameTarget(pluginName, target))
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
	var plugins []discovery.Discovered
	var err error
	if !configlib.IsFeatureActivated(constants.FeatureDisableCentralRepositoryForTesting) {
		// We no longer sync standalone plugins.
		// With a centralized approach to discovering plugins, synchronizing
		// standalone plugins would install ALL plugins available for ALL
		// products.
		// Instead, we only synchronize any plugins that are specifically specified
		// by the contexts.
		//
		// Note: to install all plugins for a specific product, plugin groups will
		// need to be used.
		plugins, err = DiscoverServerPlugins()
		if err != nil {
			return err
		}
		if installedPlugins, err := pluginsupplier.GetInstalledServerPlugins(); err == nil {
			setAvailablePluginsStatus(plugins, installedPlugins)
		}
	} else {
		plugins, err = AvailablePlugins()
		if err != nil {
			return err
		}
	}

	installed := false

	errList := make([]error, 0)
	for idx := range plugins {
		if plugins[idx].Status == common.PluginStatusNotInstalled {
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
// nolint: gocyclo
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
			return errors.Errorf("unable to find plugin '%v' for target '%s'", pluginName, string(target))
		}
		return errors.Errorf("unable to find plugin '%v'", pluginName)
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

	plugins, err := discoverPlugins(pds)
	if err != nil {
		return nil, err
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
	if err := catalog.CleanCatalogCache(); err != nil {
		return errors.Errorf("Failed to clean the catalog cache %v", err)
	}
	return os.RemoveAll(common.DefaultPluginRoot)
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
	var testDiscoveries []configtypes.PluginDiscovery
	if !configlib.IsFeatureActivated(constants.FeatureDisableCentralRepositoryForTesting) {
		// Look for testing discoveries.  Those should be stored and searched AFTER the central repo.
		testDiscoveries = getAdditionalTestPluginDiscoveries()

		// Look for the pre-release Central Repository discovery
		pd, err := getPreReleasePluginDiscovery()
		if err != nil {
			return testDiscoveries, err
		}
		// If pd is nil without an error, we bypass the prerelease discovery
		// and fallback to the normal plugin source configuration.
		if pd != nil {
			// The central repository discovery MUST be searched first
			// so we insert before the test discoveries
			return append(pd, testDiscoveries...), nil
		}
	}

	// Look for configured plugin discovery sources
	cfg, err := configlib.GetClientConfig()
	if err != nil {
		return testDiscoveries, errors.Wrapf(err, "unable to get client configuration")
	}

	if cfg == nil || cfg.ClientOptions == nil || cfg.ClientOptions.CLI == nil {
		return testDiscoveries, nil
	}
	// The configured discoveries should be searched BEFORE the test discoveries.
	// For example, if the staging central repo is added as a test discovery, it
	// may contain older versions of a plugin that is now published to the production
	// central repo; we therefore need to search the test discoveries last.
	return append(cfg.ClientOptions.CLI.DiscoverySources, testDiscoveries...), nil
}
