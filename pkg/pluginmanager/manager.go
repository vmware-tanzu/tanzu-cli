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

	"github.com/aunum/log"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	cliv1alpha1 "github.com/vmware-tanzu/tanzu-framework/apis/cli/v1alpha1"
	configapi "github.com/vmware-tanzu/tanzu-plugin-runtime/apis/config/v1alpha1"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	"github.com/vmware-tanzu/tanzu-cli/pkg/artifact"
	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
)

const (
	// exe is an executable file extension
	exe = ".exe"
	// ManifestFileName is the file name for the manifest.
	ManifestFileName = "manifest.yaml"
	// PluginFileName is the file name for the plugin info.
	PluginFileName = "plugin.yaml"
)

var execCommand = exec.Command

type DeletePluginOptions struct {
	Target      cliv1alpha1.Target
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

func discoverPlugins(pd []configapi.PluginDiscovery) ([]discovery.Discovered, error) {
	allPlugins := make([]discovery.Discovered, 0)
	for _, d := range pd {
		discObject, err := discovery.CreateDiscoveryFromV1alpha1(d)
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

// DiscoverStandalonePlugins returns the available standalone plugins
func DiscoverStandalonePlugins() (plugins []discovery.Discovered, err error) {
	cfg, e := configlib.GetClientConfig()
	if e != nil {
		err = errors.Wrapf(e, "unable to get client configuration")
		return
	}

	if cfg == nil || cfg.ClientOptions == nil || cfg.ClientOptions.CLI == nil {
		plugins = []discovery.Discovered{}
		return
	}

	plugins, err = discoverPlugins(cfg.ClientOptions.CLI.DiscoverySources)
	if err != nil {
		return
	}

	for i := range plugins {
		plugins[i].Scope = common.PluginScopeStandalone
		plugins[i].Status = common.PluginStatusNotInstalled
	}
	return
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
		var discoverySources []configapi.PluginDiscovery
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
			case cliv1alpha1.TargetTMC:
				discoveredPlugins[i].Target = cliv1alpha1.TargetTMC
			case cliv1alpha1.TargetK8s:
				discoveredPlugins[i].Target = cliv1alpha1.TargetK8s
			}
		}
		plugins = append(plugins, discoveredPlugins...)
	}
	return plugins, kerrors.NewAggregate(errList)
}

// discoverServerPluginsBasedOnCurrentServer returns the available plugins associated with the given server
func discoverServerPluginsBasedOnCurrentServer() ([]discovery.Discovered, error) {
	var plugins []discovery.Discovered

	server, err := configlib.GetCurrentServer()
	if err != nil || server == nil {
		// If servername is not specified than returning empty list
		// as there are no server plugins that can be discovered
		return plugins, nil
	}
	var discoverySources []configapi.PluginDiscovery
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
		if availablePlugins[i].Target == cliv1alpha1.TargetNone {
			// As we are considering None targeted and k8s target plugin to be treated as same plugins
			// in the case of plugin name conflicts, using `k8s` target to determine the plugin already
			// exists or not.
			// If plugin already exists in the map then combining the installation status for both the plugins
			key := fmt.Sprintf("%s_%s", availablePlugins[i].Name, cliv1alpha1.TargetK8s)
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
			} else if availablePlugins[i].Target == cliv1alpha1.TargetK8s || availablePlugins[i].Scope == common.PluginScopeContext {
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
func DescribePlugin(pluginName string, target cliv1alpha1.Target) (info *cli.PluginInfo, err error) {
	plugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return nil, err
	}
	var matchedPlugins []cli.PluginInfo

	for i := range plugins {
		if plugins[i].Name == pluginName {
			matchedPlugins = append(matchedPlugins, plugins[i])
		}
	}

	if len(matchedPlugins) == 0 {
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

// InstallPlugin installs a plugin from the given repository.
func InstallPlugin(pluginName, version string, target cliv1alpha1.Target) error {
	availablePlugins, err := AvailablePlugins()
	if err != nil {
		return err
	}

	var matchedPlugins []discovery.Discovered
	for i := range availablePlugins {
		if availablePlugins[i].Name == pluginName {
			matchedPlugins = append(matchedPlugins, availablePlugins[i])
		}
	}
	if len(matchedPlugins) == 0 {
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
func UpgradePlugin(pluginName, version string, target cliv1alpha1.Target) error {
	return InstallPlugin(pluginName, version, target)
}

// GetRecommendedVersionOfPlugin returns recommended version of the plugin
func GetRecommendedVersionOfPlugin(pluginName string, target cliv1alpha1.Target) (string, error) {
	availablePlugins, err := AvailablePlugins()
	if err != nil {
		return "", err
	}

	var matchedPlugins []discovery.Discovered
	for i := range availablePlugins {
		if availablePlugins[i].Name == pluginName {
			matchedPlugins = append(matchedPlugins, availablePlugins[i])
		}
	}
	if len(matchedPlugins) == 0 {
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
	if p.Target == cliv1alpha1.TargetNone {
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
	err := verifyPluginPreDownload(p)
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

	var plugin cli.PluginInfo
	var matchedPluginCatalog *catalog.ContextCatalog
	matchedPluginCount := 0

	// Add empty serverName for standalone plugins
	serverNames = append(serverNames, "")

	for _, serverName := range serverNames {
		c, err := catalog.NewContextCatalog(serverName)
		if err != nil {
			continue
		}

		plugins := c.List()
		for i := range plugins {
			if plugins[i].Name == options.PluginName && (options.Target == "" || options.Target == plugins[i].Target) {
				plugin = plugins[i]
				matchedPluginCatalog = c
				matchedPluginCount++
			}
		}
	}

	if matchedPluginCount == 0 {
		return errors.Errorf("unable to find plugin '%v'", options.PluginName)
	}
	if matchedPluginCount > 1 {
		return errors.Errorf("unable to uniquely identify plugin '%v'. Please specify correct Target(kubernetes[k8s]/mission-control[tmc]) of the plugin with `--target` flag", options.PluginName)
	}

	if !options.ForceDelete {
		if err := component.AskForConfirmation(fmt.Sprintf("Deleting Plugin '%s'. Are you sure?", options.PluginName)); err != nil {
			return err
		}
	}
	err = matchedPluginCatalog.Delete(catalog.PluginNameTarget(plugin.Name, plugin.Target))
	if err != nil {
		return fmt.Errorf("plugin %q could not be deleted from cache", options.PluginName)
	}

	// TODO: delete the plugin binary if it is not used by any server

	return nil
}

// SyncPlugins automatically downloads all available plugins to users machine
func SyncPlugins() error {
	log.Info("Checking for required plugins...")
	plugins, err := AvailablePlugins()
	if err != nil {
		return err
	}

	installed := false

	errList := make([]error, 0)
	for idx := range plugins {
		if plugins[idx].Status != common.PluginStatusInstalled {
			installed = true
			err = InstallPlugin(plugins[idx].Name, plugins[idx].RecommendedVersion, plugins[idx].Target)
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
func InstallPluginsFromLocalSource(pluginName, version string, target cliv1alpha1.Target, localPath string, installTestPlugin bool) error {
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
		if pluginName == cli.AllPlugins || availablePlugins[i].Name == pluginName {
			matchedPlugins = append(matchedPlugins, availablePlugins[i])
		}
	}
	if len(matchedPlugins) == 0 {
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

	// Check if the manifest.yaml file exists to see if the directory is legacy structure or not
	if _, err2 := os.Stat(filepath.Join(localPath, ManifestFileName)); errors.Is(err2, os.ErrNotExist) {
		return nil, err
	}

	// As manifest.yaml file exists it assumes in this case the directory is in
	// the legacy structure, and attempt to process it as such
	return discoverPluginsFromLocalSourceWithLegacyDirectoryStructure(localPath)
}

func discoverPluginsFromLocalSource(localPath string) ([]discovery.Discovered, error) {
	// Set default local plugin distro to localpath while installing the plugin
	// from local source. This is done to allow CLI to know the basepath incase the
	// relative path is provided as part of CLIPlugin definition for local discovery
	common.DefaultLocalPluginDistroDir = localPath

	var pds []configapi.PluginDiscovery

	items, err := os.ReadDir(filepath.Join(localPath, "discovery"))
	if err != nil {
		return nil, errors.Wrapf(err, "error while reading local plugin manifest directory")
	}
	for _, item := range items {
		if item.IsDir() {
			pd := configapi.PluginDiscovery{
				Local: &configapi.LocalDiscovery{
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

// discoverPluginsFromLocalSourceWithLegacyDirectoryStructure returns the available plugins
// that are discovered from the provided local path
func discoverPluginsFromLocalSourceWithLegacyDirectoryStructure(localPath string) ([]discovery.Discovered, error) {
	if localPath == "" {
		return nil, nil
	}

	// Get the plugin manifest object from manifest.yaml file
	manifest, err := getPluginManifestResource(filepath.Join(localPath, ManifestFileName))
	if err != nil {
		return nil, err
	}

	var discoveredPlugins []discovery.Discovered

	// Create  discovery.Discovered object for all locally available plugin
	for _, p := range manifest.Plugins {
		if p.Name == common.CoreName {
			continue
		}

		// Get the plugin Info from the plugin.yaml file
		plugin, err := getPluginInfoResource(filepath.Join(localPath, p.Name, PluginFileName))
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal plugin.yaml: %v", err)
		}

		absLocalPath, err := filepath.Abs(localPath)
		if err != nil {
			return nil, err
		}
		// With legacy configuration directory structure creating the pluginBinary path from plugin Info
		// Sample path: cli/<plugin-name>/<plugin-version>/tanzu-<plugin-name>-<os>_<arch>
		// 				cli/login/v0.14.0/tanzu-login-darwin_amd64
		// As mentioned above, we expect the binary for user's OS-ARCH is present and hence creating path accordingly
		pluginBinaryPath := filepath.Join(absLocalPath, p.Name, plugin.Version, fmt.Sprintf("tanzu-%s-%s_%s", p.Name, runtime.GOOS, runtime.GOARCH))
		if cli.BuildArch().IsWindows() {
			pluginBinaryPath += exe
		}
		// Check if the pluginBinary file exists or not
		if _, err := os.Stat(pluginBinaryPath); errors.Is(err, os.ErrNotExist) {
			return nil, errors.Wrapf(err, "unable to find plugin binary for %q", p.Name)
		}

		p := getCLIPluginResourceWithLocalDistroFromPluginInfo(plugin, pluginBinaryPath)

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
func verifyPluginPreDownload(p *discovery.Discovered) error {
	artifactInfo, err := p.Distribution.DescribeArtifact(p.RecommendedVersion, runtime.GOOS, runtime.GOARCH)
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
