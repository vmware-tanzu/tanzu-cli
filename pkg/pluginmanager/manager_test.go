// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
)

var expectedDiscoveredContextPlugins = []discovery.Discovered{
	{
		Name:               "cluster",
		RecommendedVersion: "v1.6.0",
		Scope:              common.PluginScopeContext,
		ContextName:        "mgmt",
		Target:             configtypes.TargetK8s,
	},
	{
		Name:               "cluster",
		RecommendedVersion: "v0.2.0",
		Scope:              common.PluginScopeContext,
		ContextName:        "tmc-fake",
		Target:             configtypes.TargetTMC,
	},
	{
		Name:               "management-cluster",
		RecommendedVersion: "v0.2.0",
		Scope:              common.PluginScopeContext,
		ContextName:        "tmc-fake",
		Target:             configtypes.TargetTMC,
	},
}
var expectedDiscoveredStandalonePlugins = []discovery.Discovered{
	{
		Name:               "management-cluster",
		Description:        "Plugin management-cluster description",
		RecommendedVersion: "v1.6.0",
		SupportedVersions:  []string{"v1.6.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetK8s,
	},
	{
		Name:               "cluster",
		Description:        "Plugin cluster description",
		RecommendedVersion: "v1.6.0",
		SupportedVersions:  []string{"v1.6.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetK8s,
	},
	{
		Name:               "myplugin",
		Description:        "Plugin myplugin description",
		RecommendedVersion: "v1.6.0",
		SupportedVersions:  []string{"v1.6.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetK8s,
	},
	{
		Name:               "feature",
		Description:        "Plugin feature description",
		RecommendedVersion: "v0.2.0",
		SupportedVersions:  []string{"v0.2.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetK8s,
	},
	{
		Name:               "isolated-cluster",
		Description:        "Plugin isolated-cluster description",
		RecommendedVersion: "v1.3.0",
		SupportedVersions:  []string{"v1.2.3", "v1.3.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetGlobal,
	},
	{
		Name:               "login",
		Description:        "Plugin login description",
		RecommendedVersion: "v0.20.0",
		SupportedVersions:  []string{"v0.2.0-beta.1", "v0.2.0", "v0.20.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetGlobal,
	},
	{
		Name:               "management-cluster",
		Description:        "Plugin management-cluster description",
		RecommendedVersion: "v0.2.0",
		SupportedVersions:  []string{"v0.0.1", "v0.0.2", "v0.0.3", "v0.2.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetTMC,
	},
	{
		Name:               "cluster",
		Description:        "Plugin cluster description",
		RecommendedVersion: "v0.2.0",
		SupportedVersions:  []string{"v0.2.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetTMC,
	},
	{
		Name:               "myplugin",
		Description:        "Plugin myplugin description",
		RecommendedVersion: "v0.2.0",
		SupportedVersions:  []string{"v0.2.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetTMC,
	},
	{
		Name:               "pluginnoarmdarwin",
		Description:        "Plugin pluginnoarmdarwin description",
		RecommendedVersion: "v1.0.0",
		SupportedVersions:  []string{"v1.0.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetK8s,
	},
	{
		Name:               "pluginwitharmdarwin",
		Description:        "Plugin pluginwitharmdarwin description",
		RecommendedVersion: "v2.0.0",
		SupportedVersions:  []string{"v2.0.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetK8s,
	}, {
		Name:               "pluginnoarmwindows",
		Description:        "Plugin pluginnoarmwindows description",
		RecommendedVersion: "v3.0.0",
		SupportedVersions:  []string{"v3.0.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetTMC,
	},
	{
		Name:               "pluginwitharmwindows",
		Description:        "Plugin pluginwitharmwindows description",
		RecommendedVersion: "v4.0.0",
		SupportedVersions:  []string{"v4.0.0"},
		Scope:              common.PluginScopeStandalone,
		ContextName:        "",
		Target:             configtypes.TargetTMC,
	},
}

var expectedDiscoveredGroups = []string{"vmware-test/default:v1.6.0", "vmware-test/default:v2.1.0"}

const (
	testGroupName    = "vmware-test/default"
	testGroupVersion = "v1.6.0"
)

func Test_DiscoverStandalonePlugins(t *testing.T) {
	assertions := assert.New(t)

	defer setupPluginSourceForTesting()()

	standalonePlugins, err := DiscoverStandalonePlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedDiscoveredStandalonePlugins), len(standalonePlugins))

	for i := range expectedDiscoveredStandalonePlugins {
		p := findDiscoveredPlugin(standalonePlugins, expectedDiscoveredStandalonePlugins[i].Name, expectedDiscoveredStandalonePlugins[i].Target)
		assertions.NotNil(p)
		assertions.Equal(expectedDiscoveredStandalonePlugins[i].Description, p.Description)
		assertions.Equal(expectedDiscoveredStandalonePlugins[i].RecommendedVersion, p.RecommendedVersion)
		assertions.Equal(expectedDiscoveredStandalonePlugins[i].SupportedVersions, p.SupportedVersions)
		assertions.Equal(expectedDiscoveredStandalonePlugins[i].Scope, p.Scope)
		assertions.Equal(expectedDiscoveredStandalonePlugins[i].ContextName, p.ContextName)
	}
}

func Test_DiscoverServerPlugins(t *testing.T) {
	assertions := assert.New(t)

	defer setupPluginSourceForTesting()()

	serverPlugins, err := DiscoverServerPlugins()
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to list plugins from discovery source 'default-mgmt': Failed to load Kubeconfig file")
	assertions.Equal(len(expectedDiscoveredContextPlugins), len(serverPlugins))

	for i := range expectedDiscoveredContextPlugins {
		p := findDiscoveredPlugin(serverPlugins, expectedDiscoveredContextPlugins[i].Name, expectedDiscoveredContextPlugins[i].Target)
		assertions.NotNil(p)
		assertions.Equal(expectedDiscoveredContextPlugins[i].RecommendedVersion, p.RecommendedVersion)
		assertions.Equal(expectedDiscoveredContextPlugins[i].Scope, p.Scope)
		assertions.Equal(expectedDiscoveredContextPlugins[i].ContextName, p.ContextName)
	}
}

func Test_DiscoverPluginGroups(t *testing.T) {
	assertions := assert.New(t)

	defer setupPluginSourceForTesting()()

	groups, err := DiscoverPluginGroups()
	assertions.Nil(err)

	for _, id := range expectedDiscoveredGroups {
		found := findGroupVersion(groups, id)
		assertions.True(found, fmt.Sprintf("unable to find group %s", id))
	}
}

func Test_InstallStandalonePlugin(t *testing.T) {
	assertions := assert.New(t)

	defer setupPluginSourceForTesting()()
	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	// Try installing nonexistent plugin
	err := InstallStandalonePlugin("not-exists", "v0.2.0", configtypes.TargetUnknown)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'not-exists'")

	// Install login (standalone) plugin with just vMajor.Minor.Patch as version
	// Make sure it does not install other available plugins like (v0.20.0 or v0.2.0-beta.1)
	// and installs specified v0.2.0
	err = InstallStandalonePlugin("login", "v0.2.0", configtypes.TargetUnknown)
	assertions.Nil(err)
	// Verify installed plugin
	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(1, len(installedPlugins))
	assertions.Equal("login", installedPlugins[0].Name)

	// Install login (standalone) plugin with just vMajor(v0) as version
	// Make sure it installs latest version available plugins v0.20.0
	// among available versions (v0.2.0, v0.2.0-beta.1, v0.20.0)
	err = InstallStandalonePlugin("login", "v0", configtypes.TargetUnknown)
	assertions.Nil(err)
	// Verify installed plugin
	installedPlugins, err = pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(1, len(installedPlugins))
	assertions.Equal("login", installedPlugins[0].Name)
	assertions.Equal("v0.20.0", installedPlugins[0].Version)

	// Install login (standalone) plugin with just vMajor.Minor (v0.2) as version
	// Make sure it does not install other available plugins like (v0.20.0 or v0.2.0-beta.1)
	// and installs v0.2.0
	err = InstallStandalonePlugin("login", "v0.2", configtypes.TargetUnknown)
	assertions.Nil(err)
	// Verify installed plugin
	installedPlugins, err = pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(1, len(installedPlugins))
	assertions.Equal("login", installedPlugins[0].Name)
	assertions.Equal("v0.2.0", installedPlugins[0].Version)

	// Try installing myplugin plugin with no context-type and no specific version
	err = InstallStandalonePlugin("myplugin", cli.VersionLatest, configtypes.TargetUnknown)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), fmt.Sprintf(missingTargetStr, "myplugin"))

	// Try installing myplugin plugin with context-type=tmc with no specific version
	err = InstallStandalonePlugin("myplugin", cli.VersionLatest, configtypes.TargetTMC)
	assertions.Nil(err)

	// Try installing myplugin plugin through context-type=k8s with incorrect version
	err = InstallStandalonePlugin("myplugin", "v1.0.0", configtypes.TargetK8s)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'myplugin' matching version 'v1.0.0'")

	// Try installing myplugin plugin through context-type=k8s with the correct version
	err = InstallStandalonePlugin("myplugin", "v1.6.0", configtypes.TargetK8s)
	assertions.Nil(err)

	// Try installing management-cluster plugin
	err = InstallStandalonePlugin("management-cluster", "v1.6.0", configtypes.TargetK8s)
	assertions.Nil(err)

	// Try installing the feature plugin which is targeted for k8s but requesting the TMC target
	err = InstallStandalonePlugin("feature", "v0.2.0", configtypes.TargetTMC)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'feature' matching version 'v0.2.0' for target 'mission-control'")

	// When on Darwin ARM64, try installing a plugin that is only available for Darwin AMD64
	// and see that it still gets installed (it will use AMD64)
	//
	// First make the CLI believe we are running on Darwin ARM64.  We need this for when
	// the unit tests are run on Linux for example.
	realArch := cli.BuildArch()
	cli.SetArch(cli.DarwinARM64)
	err = InstallStandalonePlugin("pluginnoarmdarwin", "v1.0.0", configtypes.TargetUnknown)
	assertions.Nil(err)
	// Make sure that after the plugin is installed (using AMD64), the arch is back to ARM64
	assertions.Equal(cli.DarwinARM64, cli.BuildArch())
	// Now reset to the real machine architecture
	cli.SetArch(realArch)

	// When on Darwin ARM64, try installing a plugin that IS available for Darwin ARM64
	// and make sure it is the ARM64 one that gets installed (not AMD64)
	//
	// First make the CLI believe we are running on Darwin ARM64.  We need this for when
	// the unit tests are run on Linux for example.
	cli.SetArch(cli.DarwinARM64)
	err = InstallStandalonePlugin("pluginwitharmdarwin", "v2.0.0", configtypes.TargetUnknown)
	assertions.Nil(err)
	// Make sure that after the plugin is installed (using AMD64), the arch is back to ARM64
	assertions.Equal(cli.DarwinARM64, cli.BuildArch())
	// Now reset to the real machine architecture
	cli.SetArch(realArch)

	// When on Windows ARM64, try installing a plugin that is only available for Windows AMD64
	// and see that it still gets installed (it will use AMD64)
	//
	// First make the CLI believe we are running on Windows ARM64.  We need this for when
	// the unit tests are run on Linux for example.
	realArch = cli.BuildArch()
	cli.SetArch(cli.WinARM64)
	err = InstallStandalonePlugin("pluginnoarmwindows", "v3.0.0", configtypes.TargetUnknown)
	assertions.Nil(err)
	// Make sure that after the plugin is installed (using AMD64), the arch is back to ARM64
	assertions.Equal(cli.WinARM64, cli.BuildArch())
	// Now reset to the real machine architecture
	cli.SetArch(realArch)

	// When on Windows ARM64, try installing a plugin that IS available for Windows ARM64
	// and make sure it is the ARM64 one that gets installed (not AMD64)
	//
	// First make the CLI believe we are running on Windows ARM64.  We need this for when
	// the unit tests are run on Linux for example.
	cli.SetArch(cli.WinARM64)
	err = InstallStandalonePlugin("pluginwitharmwindows", "v4.0.0", configtypes.TargetUnknown)
	assertions.Nil(err)
	// Make sure that after the plugin is installed (using AMD64), the arch is back to ARM64
	assertions.Equal(cli.WinARM64, cli.BuildArch())
	// Now reset to the real machine architecture
	cli.SetArch(realArch)

	expectedInstalledStandalonePlugins := []cli.PluginInfo{
		{
			Name:    "login",
			Version: "v0.2.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetGlobal,
		},
		{
			Name:    "management-cluster",
			Version: "v1.6.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetK8s,
		},
		{
			Name:    "myplugin",
			Version: "v1.6.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetK8s,
		},
		{
			Name:    "myplugin",
			Version: "v0.2.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetTMC,
		},
		{
			Name:    "pluginnoarmdarwin",
			Version: "v1.0.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetK8s,
		},
		{
			Name:    "pluginwitharmdarwin",
			Version: "v2.0.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetK8s,
		},
		{
			Name:    "pluginnoarmwindows",
			Version: "v3.0.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetTMC,
		},
		{
			Name:    "pluginwitharmwindows",
			Version: "v4.0.0",
			Scope:   common.PluginScopeStandalone,
			Target:  configtypes.TargetTMC,
		},
	}

	// Verify installed plugins
	installedStandalonePlugins, err := pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(len(expectedInstalledStandalonePlugins), len(installedStandalonePlugins))

	for i := 0; i < len(expectedInstalledStandalonePlugins); i++ {
		pd := findPluginInfo(installedStandalonePlugins, expectedInstalledStandalonePlugins[i].Name, expectedInstalledStandalonePlugins[i].Target)
		assertions.NotNil(pd)
		assertions.Equal(expectedInstalledStandalonePlugins[i].Version, pd.Version)

		if strings.HasPrefix(pd.Name, "pluginnoarm") {
			// Make sure these plugins are always installed as AMD64
			assertions.Equal(digestForAMD64, pd.Digest)
		}

		if strings.HasPrefix(pd.Name, "pluginwitharm") {
			// Make sure these plugins are always installed as ARM64
			assertions.Equal(digestForARM64, pd.Digest)
		}
	}
}

func Test_InstallPluginsFromGroup(t *testing.T) {
	assertions := assert.New(t)

	defer setupPluginSourceForTesting()()
	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	// Install the management-cluster plugin from a group:version
	groupID := testGroupName + ":" + testGroupVersion
	fullGroupID, err := InstallPluginsFromGroup("management-cluster", groupID)
	assertions.Nil(err)
	assertions.Equal(groupID, fullGroupID)

	installedStandalonePlugins, err := pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(1, len(installedStandalonePlugins))
	pd := findPluginInfo(installedStandalonePlugins, "management-cluster", configtypes.TargetK8s)
	assertions.NotNil(pd)
	assertions.Equal("v1.6.0", pd.Version)

	// Install the isolated-cluster from the latest group
	// The latest plugin group is `vmware-test/default:v2.2.0` which contains
	// `isolated-cluster` plugin version as `v1`. So it should install the
	// latest minor/patch of v1 for isolated-cluster which is `v1.3.0`
	groupID = testGroupName
	fullGroupID, err = InstallPluginsFromGroup("isolated-cluster", groupID)
	assertions.Nil(err)
	assertions.Equal(groupID+":v2.2.0", fullGroupID)

	installedStandalonePlugins, err = pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(2, len(installedStandalonePlugins))
	pd = findPluginInfo(installedStandalonePlugins, "management-cluster", configtypes.TargetK8s)
	assertions.NotNil(pd)
	assertions.Equal("v1.6.0", pd.Version)
	pd = findPluginInfo(installedStandalonePlugins, "isolated-cluster", configtypes.TargetGlobal)
	assertions.NotNil(pd)
	assertions.Equal("v1.3.0", pd.Version)

	// Install all plugins from a group:version
	// Note that this should replace isolated-cluster:v1.3.0 with its v1.2.3 version
	groupID = testGroupName + ":" + testGroupVersion
	fullGroupID, err = InstallPluginsFromGroup(cli.AllPlugins, groupID)
	assertions.Nil(err)
	assertions.Equal(groupID, fullGroupID)

	installedStandalonePlugins, err = pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(4, len(installedStandalonePlugins))
	pd = findPluginInfo(installedStandalonePlugins, "management-cluster", configtypes.TargetK8s)
	assertions.NotNil(pd)
	assertions.Equal("v1.6.0", pd.Version)
	pd = findPluginInfo(installedStandalonePlugins, "isolated-cluster", configtypes.TargetGlobal)
	assertions.NotNil(pd)
	assertions.Equal("v1.2.3", pd.Version)
	pd = findPluginInfo(installedStandalonePlugins, "myplugin", configtypes.TargetK8s)
	assertions.NotNil(pd)
	assertions.Equal("v1.6.0", pd.Version)
	pd = findPluginInfo(installedStandalonePlugins, "feature", configtypes.TargetK8s)
	assertions.NotNil(pd)
	assertions.Equal("v0.2.0", pd.Version)

	// Install all plugins from the plugin group `vmware-test/default:v2.1`
	// This should install latest patch of `v1.3` for isolated cluster plugin
	// based on the plugin-group `vmware-test/default:v2.1.0`
	groupID = testGroupName
	fullGroupID, err = InstallPluginsFromGroup("isolated-cluster", groupID+":v2.1")
	assertions.Nil(err)
	assertions.Equal(groupID+":v2.1.0", fullGroupID)

	installedStandalonePlugins, err = pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(4, len(installedStandalonePlugins))
	pd = findPluginInfo(installedStandalonePlugins, "management-cluster", configtypes.TargetK8s)
	assertions.NotNil(pd)
	assertions.Equal("v1.6.0", pd.Version)
	pd = findPluginInfo(installedStandalonePlugins, "isolated-cluster", configtypes.TargetGlobal)
	assertions.NotNil(pd)
	assertions.Equal("v1.3.0", pd.Version)
	pd = findPluginInfo(installedStandalonePlugins, "myplugin", configtypes.TargetK8s)
	assertions.NotNil(pd)
	assertions.Equal("v1.6.0", pd.Version)
	pd = findPluginInfo(installedStandalonePlugins, "feature", configtypes.TargetK8s)
	assertions.NotNil(pd)
	assertions.Equal("v0.2.0", pd.Version)
}

func Test_InstallPluginsFromGroupErrors(t *testing.T) {
	assertions := assert.New(t)

	defer setupPluginSourceForTesting()()
	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	// make sure a poorly formatted group is properly handled
	groupID := "invalid"
	_, err := InstallPluginsFromGroup("cluster", groupID)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "could not find group")

	groupID = "invalid/withslash"
	_, err = InstallPluginsFromGroup("cluster", groupID)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "could not find group")

	groupID = "vendor-publisher/"
	_, err = InstallPluginsFromGroup("cluster", groupID)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "could not find group")

	groupID = "vendor-/name"
	_, err = InstallPluginsFromGroup("cluster", groupID)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "could not find group")

	groupID = "-publisher/name"
	_, err = InstallPluginsFromGroup("cluster", groupID)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "could not find group")

	groupID = testGroupName
	fullGroupID, err := InstallPluginsFromGroup("cluster", groupID)
	assertions.NotNil(err)
	assertions.Equal(groupID+":v2.2.0", fullGroupID)
	assertions.Contains(err.Error(), fmt.Sprintf("plugin 'cluster' is not part of the group '%s'", fullGroupID))

	groupID = testGroupName + ":" + testGroupVersion
	fullGroupID, err = InstallPluginsFromGroup("cluster", groupID)
	assertions.NotNil(err)
	assertions.Equal(groupID, fullGroupID)
	assertions.Contains(err.Error(), fmt.Sprintf("plugin 'cluster' from group '%s' is not mandatory to install", fullGroupID))
}

func Test_InstallPlugin_InstalledPlugins_From_LocalSource(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistroForTesting()()

	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	currentDirAbsPath, _ := filepath.Abs(".")
	localPluginSourceDir := filepath.Join(currentDirAbsPath, "test", "local")

	// Try installing nonexistent plugin
	err := InstallPluginsFromLocalSource("not-exists", "v0.2.0", configtypes.TargetUnknown, localPluginSourceDir, false)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'not-exists'")

	// Try installing the feature plugin which is targeted for k8s but requesting the TMC target
	err = InstallPluginsFromLocalSource("feature", "v0.2.0", configtypes.TargetTMC, localPluginSourceDir, false)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'feature' matching version 'v0.2.0' for target 'mission-control'")

	// Install login from local source directory
	err = InstallPluginsFromLocalSource("login", "v0.2.0", configtypes.TargetUnknown, localPluginSourceDir, false)
	assertions.Nil(err)
	// Verify installed plugin
	installedStandalonePlugins, err := pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(1, len(installedStandalonePlugins))
	assertions.Equal("login", installedStandalonePlugins[0].Name)

	// Try installing cluster plugin from local source directory
	err = InstallPluginsFromLocalSource("cluster", "v0.2.0", configtypes.TargetTMC, localPluginSourceDir, false)
	assertions.Nil(err)
	installedStandalonePlugins, err = pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(2, len(installedStandalonePlugins))

	// Try installing a plugin from incorrect local path
	err = InstallPluginsFromLocalSource("cluster", "v0.2.0", configtypes.TargetTMC, "fakepath", false)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "no such file or directory")
}

func Test_DescribePlugin(t *testing.T) {
	assertions := assert.New(t)

	defer setupLocalDistroForTesting()()

	// Try to describe plugin when plugin is not installed
	_, err := DescribePlugin("login", configtypes.TargetUnknown)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'login'")

	// Install login (standalone) package
	mockInstallPlugin(assertions, "login", "v0.2.0", configtypes.TargetUnknown)

	// Try to describe plugin when plugin after installing plugin
	pd, err := DescribePlugin("login", configtypes.TargetUnknown)
	assertions.Nil(err)
	assertions.Equal("login", pd.Name)
	assertions.Equal("v0.2.0", pd.Version)

	// Try to describe plugin when plugin is not installed
	_, err = DescribePlugin("cluster", configtypes.TargetTMC)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'cluster'")

	// Install cluster package
	mockInstallPlugin(assertions, "myplugin", "v0.2.0", configtypes.TargetTMC)

	// Try to describe plugin when plugin after installing plugin
	pd, err = DescribePlugin("myplugin", configtypes.TargetTMC)
	assertions.Nil(err)
	assertions.Equal("myplugin", pd.Name)
	assertions.Equal("v0.2.0", pd.Version)

	// Install the feature plugin for k8s
	mockInstallPlugin(assertions, "feature", "v0.2.0", configtypes.TargetK8s)
	// Try to describe the feature plugin but requesting the TMC target
	_, err = DescribePlugin("feature", configtypes.TargetTMC)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'feature' for target 'mission-control'")
}

func checkPluginIsInstalled(name string, target configtypes.Target) bool {
	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	if err == nil {
		for i := range installedPlugins {
			if installedPlugins[i].Name == name &&
				installedPlugins[i].Target == target {
				return true
			}
		}
	}
	return false
}

func Test_DeletePlugin(t *testing.T) {
	assertions := assert.New(t)

	defer setupPluginSourceForTesting()()

	setupTestPluginCatalog()

	// Try to delete plugin when plugin is not installed without a target
	err := DeletePlugin(DeletePluginOptions{PluginName: "invalid", Target: configtypes.TargetUnknown, ForceDelete: true})
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'invalid'")
	assertions.NotContains(err.Error(), "for target")

	// Try to delete plugin when plugin is not installed with a target
	err = DeletePlugin(DeletePluginOptions{PluginName: "invalid", Target: configtypes.TargetTMC, ForceDelete: true})
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'invalid' for target 'mission-control'")

	// Try to Delete plugin present for two targets without specifying the target
	assertions.True(checkPluginIsInstalled("cluster", configtypes.TargetK8s))
	assertions.True(checkPluginIsInstalled("cluster", configtypes.TargetTMC))
	err = DeletePlugin(DeletePluginOptions{PluginName: "cluster", Target: configtypes.TargetUnknown, ForceDelete: true})
	assertions.NotNil(err)
	assertions.Contains(err.Error(), fmt.Sprintf(missingTargetStr, "cluster"))

	// Try to Delete one of the two plugins specifying the target
	err = DeletePlugin(DeletePluginOptions{PluginName: "cluster", Target: configtypes.TargetTMC, ForceDelete: true})
	assertions.Nil(err)
	assertions.False(checkPluginIsInstalled("cluster", configtypes.TargetTMC))
	assertions.True(checkPluginIsInstalled("cluster", configtypes.TargetK8s))

	// Try to Delete plugin without specifying the target now that the other is deleted
	err = DeletePlugin(DeletePluginOptions{PluginName: "cluster", Target: "", ForceDelete: true})
	assertions.Nil(err)
	assertions.False(checkPluginIsInstalled("cluster", configtypes.TargetTMC))
	assertions.False(checkPluginIsInstalled("cluster", configtypes.TargetK8s))

	// Try to delete a plugin with the wrong target
	assertions.False(checkPluginIsInstalled("secret", configtypes.TargetTMC))
	err = DeletePlugin(DeletePluginOptions{PluginName: "secret", Target: configtypes.TargetTMC, ForceDelete: true})
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find plugin 'secret' for target 'mission-control'")
	assertions.True(checkPluginIsInstalled("secret", configtypes.TargetK8s))

	// Delete all plugins for the k8s target
	assertions.True(checkPluginIsInstalled("secret", configtypes.TargetK8s))
	assertions.True(checkPluginIsInstalled("management-cluster", configtypes.TargetK8s))
	assertions.True(checkPluginIsInstalled("management-cluster", configtypes.TargetTMC))
	err = DeletePlugin(DeletePluginOptions{PluginName: "all", Target: configtypes.TargetK8s, ForceDelete: true})
	assertions.Nil(err)
	assertions.False(checkPluginIsInstalled("secret", configtypes.TargetK8s))
	assertions.False(checkPluginIsInstalled("management-cluster", configtypes.TargetK8s))
	assertions.True(checkPluginIsInstalled("management-cluster", configtypes.TargetTMC))

	// Delete all remaining plugins without specifying a target
	err = DeletePlugin(DeletePluginOptions{PluginName: "all", Target: configtypes.TargetUnknown, ForceDelete: true})
	assertions.Nil(err)
	assertions.False(checkPluginIsInstalled("management-cluster", configtypes.TargetTMC))

	// There should be no more plugins installed
	installedPlugins, err := pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Zero(len(installedPlugins))

	// Try to delete all plugins again
	err = DeletePlugin(DeletePluginOptions{PluginName: "all", Target: configtypes.TargetUnknown, ForceDelete: true})
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find any installed plugins")
	assertions.NotContains(err.Error(), "for target")

	// Try to delete all plugins for a specific target again
	err = DeletePlugin(DeletePluginOptions{PluginName: "all", Target: configtypes.TargetK8s, ForceDelete: true})
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "unable to find any installed plugins for target 'kubernetes'")
}

func Test_SyncPlugins(t *testing.T) {
	assertions := assert.New(t)

	defer setupPluginSourceForTesting()()
	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	// Get the server plugins (they are not installed yet)
	serverPlugins, err := DiscoverServerPlugins()
	assertions.NotNil(err)
	// There is an error for the kubernetes discovery since we don't have a cluster
	// but other server plugins will be found, so we use those
	assertions.Contains(err.Error(), `Failed to load Kubeconfig file from "config"`)
	assertions.Equal(len(expectedDiscoveredContextPlugins), len(serverPlugins))

	for _, edp := range expectedDiscoveredContextPlugins {
		p := findDiscoveredPlugin(serverPlugins, edp.Name, edp.Target)
		assertions.NotNil(p)
		assertions.Equal(common.PluginStatusNotInstalled, p.Status)
	}

	// Sync all available plugins
	err = SyncPlugins()
	assertions.NotNil(err)
	// There is an error for the kubernetes discovery since we don't have a cluster
	// but other server plugins will be found, so we use those
	assertions.Contains(err.Error(), `Failed to load Kubeconfig file from "config"`)
}

func Test_setAvailablePluginsStatus(t *testing.T) {
	assertions := assert.New(t)

	availablePlugins := []discovery.Discovered{{Name: "fake1", DiscoveryType: "oci", RecommendedVersion: "v1.0.0", Status: common.PluginStatusNotInstalled, Target: configtypes.TargetK8s}}
	installedPlugin := []cli.PluginInfo{{Name: "fake2", Version: "v2.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v2.0.0", Target: configtypes.TargetUnknown}}

	// If installed plugin is not part of available(discovered) plugins then
	// installed version == ""
	// status  == not installed
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v1.0.0", availablePlugins[0].RecommendedVersion)
	assertions.Equal("", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusNotInstalled, availablePlugins[0].Status)

	// If installed plugin is not part of available(discovered) plugins because of the Target mismatch
	installedPlugin = []cli.PluginInfo{{Name: "fake1", Version: "v1.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v1.0.0", Target: configtypes.TargetUnknown}}
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v1.0.0", availablePlugins[0].RecommendedVersion)
	assertions.Equal("", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusNotInstalled, availablePlugins[0].Status)

	// If installed plugin is part of available(discovered) plugins and provided available plugin is already installed
	installedPlugin = []cli.PluginInfo{{Name: "fake1", Version: "v1.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v1.0.0", Target: configtypes.TargetK8s}}
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v1.0.0", availablePlugins[0].RecommendedVersion)
	assertions.Equal("v1.0.0", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusInstalled, availablePlugins[0].Status)

	// If installed plugin is part of available(discovered) plugins but recommended discovered version is different than the one installed
	// then available plugin status should show 'update available'
	availablePlugins = []discovery.Discovered{{Name: "fake1", DiscoveryType: "oci", RecommendedVersion: "v8.0.0-latest", Status: common.PluginStatusNotInstalled}}
	installedPlugin = []cli.PluginInfo{{Name: "fake1", Version: "v1.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v1.0.0"}}
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v8.0.0-latest", availablePlugins[0].RecommendedVersion)
	assertions.Equal("v1.0.0", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusUpdateAvailable, availablePlugins[0].Status)

	// If installed plugin is part of available(discovered) plugins but recommended discovered version is same as the recommended discovered version
	// for the installed plugin(stored as part of catalog cache) then available plugin status should show 'installed'
	availablePlugins = []discovery.Discovered{{Name: "fake1", DiscoveryType: "oci", RecommendedVersion: "v8.0.0-latest", Status: common.PluginStatusNotInstalled}}
	installedPlugin = []cli.PluginInfo{{Name: "fake1", Version: "v1.0.0", Discovery: "local", DiscoveredRecommendedVersion: "v8.0.0-latest"}}
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v8.0.0-latest", availablePlugins[0].RecommendedVersion)
	assertions.Equal("v1.0.0", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusInstalled, availablePlugins[0].Status)

	// If installed plugin is part of available(discovered) plugins and versions installed is different from discovered version
	// it should be reflected in RecommendedVersion as well as InstalledVersion and status should be `update available`
	availablePlugins[0].Status = common.PluginStatusNotInstalled
	availablePlugins[0].RecommendedVersion = "v3.0.0"
	setAvailablePluginsStatus(availablePlugins, installedPlugin)
	assertions.Equal(len(availablePlugins), 1)
	assertions.Equal("fake1", availablePlugins[0].Name)
	assertions.Equal("v3.0.0", availablePlugins[0].RecommendedVersion)
	assertions.Equal("v1.0.0", availablePlugins[0].InstalledVersion)
	assertions.Equal(common.PluginStatusUpdateAvailable, availablePlugins[0].Status)
}

func Test_DiscoverPluginsFromLocalSourceBasedOnManifestFile(t *testing.T) {
	assertions := assert.New(t)

	// When passing directory structure where manifest.yaml and plugin_manifest.yaml both files are missing
	_, err := discoverPluginsFromLocalSourceBasedOnManifestFile(filepath.Join("test", "local"))
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "could not find manifest.yaml file")
	assertions.Contains(err.Error(), "could not find plugin_manifest.yaml file")

	// When passing directory structure which contains manifest.yaml file
	discoveredPlugins, err := discoverPluginsFromLocalSourceBasedOnManifestFile(filepath.Join("test", "legacy"))
	assertions.Nil(err)
	assertions.Equal(2, len(discoveredPlugins))

	assertions.Equal("foo", discoveredPlugins[0].Name)
	assertions.Equal("Foo plugin", discoveredPlugins[0].Description)
	assertions.Equal("v0.12.0", discoveredPlugins[0].RecommendedVersion)
	assertions.Equal(common.PluginScopeStandalone, discoveredPlugins[0].Scope)
	assertions.Equal(configtypes.TargetUnknown, discoveredPlugins[0].Target)

	assertions.Equal("bar", discoveredPlugins[1].Name)
	assertions.Equal("Bar plugin", discoveredPlugins[1].Description)
	assertions.Equal("v0.10.0", discoveredPlugins[1].RecommendedVersion)
	assertions.Equal(common.PluginScopeStandalone, discoveredPlugins[1].Scope)
	assertions.Equal(configtypes.TargetUnknown, discoveredPlugins[1].Target)

	// When passing directory structure which contains plugin_manifest.yaml file
	discoveredPlugins, err = discoverPluginsFromLocalSourceBasedOnManifestFile(filepath.Join("test", "artifacts1"))
	assertions.Nil(err)
	assertions.Equal(2, len(discoveredPlugins))

	assertions.Equal("foo", discoveredPlugins[0].Name)
	assertions.Equal("Foo plugin", discoveredPlugins[0].Description)
	assertions.Equal("v0.12.0", discoveredPlugins[0].RecommendedVersion)
	assertions.Equal(common.PluginScopeStandalone, discoveredPlugins[0].Scope)
	assertions.Equal(configtypes.TargetK8s, discoveredPlugins[0].Target)

	assertions.Equal("bar", discoveredPlugins[1].Name)
	assertions.Equal("Bar plugin", discoveredPlugins[1].Description)
	assertions.Equal("v0.10.0", discoveredPlugins[1].RecommendedVersion)
	assertions.Equal(common.PluginScopeStandalone, discoveredPlugins[1].Scope)
	assertions.Equal(configtypes.TargetGlobal, discoveredPlugins[1].Target)
}

func Test_InstallPluginsFromLocalSourceWithLegacyDirectoryStructure(t *testing.T) {
	assertions := assert.New(t)

	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	// Using generic InstallPluginsFromLocalSource to test the legacy directory install
	// When passing legacy directory structure which contains manifest.yaml file
	err := InstallPluginsFromLocalSource("all", "", configtypes.TargetUnknown, filepath.Join("test", "legacy"), false)
	assertions.Nil(err)

	// Verify installed plugin
	installedStandalonePlugins, err := pluginsupplier.GetInstalledPlugins()
	assertions.Nil(err)
	assertions.Equal(2, len(installedStandalonePlugins))
	assertions.ElementsMatch([]string{"bar", "foo"}, []string{installedStandalonePlugins[0].Name, installedStandalonePlugins[1].Name})
}

func Test_VerifyRegistry(t *testing.T) {
	assertions := assert.New(t)

	var err error

	testImage := "fake.repo.com/image:v1.0.0"
	err = configureAndTestVerifyRegistry(testImage, "", "", "")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com", "", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/image", "", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/foo", "", "")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "", "fake.repo.com", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "fake.repo.com/image", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "fake.repo.com/foo", "")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com/image")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com/foo")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com", "", "fake.repo.com/foo")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "fake.repo.com", "fake.repo.com/foo")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com", "fake.repo.com", "fake.repo.com/foo")
	assertions.Nil(err)

	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com.private.com", "", "")
	assertions.NotNil(err)
	err = configureAndTestVerifyRegistry(testImage, "private.fake.repo.com", "", "")
	assertions.NotNil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/image/foo", "", "")
	assertions.NotNil(err)

	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com.private.com,private.fake.repo.com")
	assertions.NotNil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com,private.fake.repo.com")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "private.fake.repo.com,fake.repo.com")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "", "", "fake.repo.com/image,fake.repo.com")
	assertions.Nil(err)

	testImage = "fake1.repo.com/image:v1.0.0"
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/image", "", "")
	assertions.NotNil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake.repo.com/image,fake1.repo.com/image", "", "")
	assertions.Nil(err)
	err = configureAndTestVerifyRegistry(testImage, "fake1.repo.com/image", "", "")
	assertions.Nil(err)
}

func configureAndTestVerifyRegistry(testImage, defaultRegistry, customImageRepository, allowedRegistries string) error {
	config.DefaultAllowedPluginRepositories = defaultRegistry
	os.Setenv(constants.ConfigVariableCustomImageRepository, customImageRepository)
	os.Setenv(constants.AllowedRegistries, allowedRegistries)

	err := verifyRegistry(testImage)

	config.DefaultAllowedPluginRepositories = ""
	os.Setenv(constants.ConfigVariableCustomImageRepository, "")
	os.Setenv(constants.AllowedRegistries, "")
	return err
}

func TestVerifyArtifactLocation(t *testing.T) {
	tcs := []struct {
		name   string
		uri    string
		errStr string
	}{
		{
			name: "trusted location",
			uri:  "https://tmc-cli.s3-us-west-2.amazonaws.com/plugins/artifacts",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyArtifactLocation(tc.uri)
			if tc.errStr != "" {
				assert.EqualError(t, err, tc.errStr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyPluginPostDownload(t *testing.T) {
	tcs := []struct {
		name string
		p    *discovery.Discovered
		d    string
		path string
		err  string
	}{
		{
			name: "success - no source digest",
			p:    &discovery.Discovered{Name: "login"},
			path: "test/local/distribution/v0.2.0/tanzu-login",
		},
		{
			name: "success - with source digest",
			p:    &discovery.Discovered{Name: "login"},
			d:    "e109197e3e4ed9f13065596367f1fd0992df43717c7098324da4a00cb8b81c36",
			path: "test/local/distribution/v0.2.0/tanzu-login",
		},
		{
			name: "failure - digest mismatch",
			p:    &discovery.Discovered{Name: "login"},
			d:    "f3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			path: "test/local/distribution/v0.2.0/tanzu-login",
			err:  "plugin \"login\" has been corrupted during download. source digest: f3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855, actual digest: e109197e3e4ed9f13065596367f1fd0992df43717c7098324da4a00cb8b81c36",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			b, err := os.ReadFile(tc.path)
			assert.NoError(t, err)

			err = verifyPluginPostDownload(tc.p, tc.d, b)
			if tc.err != "" {
				assert.EqualError(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}
	filePath := os.Getenv("FILE_PATH")
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read plugin\n")
		os.Exit(2)
	}
	fmt.Fprint(os.Stdout, string(bytes))
}

func TestGetAdditionalTestPluginDiscoveries(t *testing.T) {
	assertions := assert.New(t)

	// Start with no additional discoveries
	err := os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, "")
	assertions.Nil(err)

	discoveries := GetAdditionalTestPluginDiscoveries()
	assertions.Nil(err)
	assertions.Equal(0, len(discoveries))

	// Set a single additional discovery
	expectedDiscovery := "localhost:9876/my/discovery/image:v0"
	err = os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, expectedDiscovery)
	assertions.Nil(err)

	discoveries = GetAdditionalTestPluginDiscoveries()
	assertions.Nil(err)
	assertions.Equal(1, len(discoveries))
	assertions.Equal(expectedDiscovery, discoveries[0].OCI.Image)

	// Set multiple additional discoveries
	expectedDiscoveries := []string{
		"localhost:9876/my/discovery/image:v1",
		"localhost:9876/my/discovery/image:v3",
		"localhost:9876/my/discovery/image:v2",
		"localhost:9876/my/discovery/image:v4",
	}
	// Use different spacing between discoveries
	err = os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting,
		expectedDiscoveries[0]+","+expectedDiscoveries[1]+"   ,"+expectedDiscoveries[2]+"  ,  "+expectedDiscoveries[3])
	assertions.Nil(err)

	discoveries = GetAdditionalTestPluginDiscoveries()
	assertions.Equal(len(expectedDiscoveries), len(discoveries))
	assertions.Equal(expectedDiscoveries[0], discoveries[0].OCI.Image)
	assertions.Equal(expectedDiscoveries[1], discoveries[1].OCI.Image)
	assertions.Equal(expectedDiscoveries[2], discoveries[2].OCI.Image)
	assertions.Equal(expectedDiscoveries[3], discoveries[3].OCI.Image)

	os.Unsetenv(constants.ConfigVariableAdditionalDiscoveryForTesting)
}

func TestGetPluginDiscoveries(t *testing.T) {
	assertions := assert.New(t)

	// Setup 2 local discoveries
	defer setupLocalDistroForTesting()()

	// Start with no additional discoveries
	err := os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, "")
	assertions.Nil(err)

	discoveries, err := getPluginDiscoveries()
	assertions.Nil(err)
	assertions.Equal(2, len(discoveries))
	assertions.Equal("default-local", discoveries[0].Local.Name)
	assertions.Equal("fake", discoveries[1].Local.Name)

	// Set a single test discovery
	expectedTestDiscovery := "localhost:9876/my/discovery/image:v10"
	err = os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, expectedTestDiscovery)
	assertions.Nil(err)

	discoveries, err = getPluginDiscoveries()
	assertions.Nil(err)
	assertions.Equal(3, len(discoveries))
	// The test discovery must be last
	assertions.Equal("default-local", discoveries[0].Local.Name)
	assertions.Equal("fake", discoveries[1].Local.Name)
	assertions.Equal(expectedTestDiscovery, discoveries[2].OCI.Image)

	// Set multiple additional discoveries
	expectedTestDiscoveries := []string{
		"localhost:9876/my/discovery/image:v11",
		"localhost:9876/my/discovery/image:v13",
		"localhost:9876/my/discovery/image:v12",
		"localhost:9876/my/discovery/image:v14",
	}
	// Use different spacing between discoveries
	err = os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting,
		expectedTestDiscoveries[0]+","+expectedTestDiscoveries[1]+"   ,"+expectedTestDiscoveries[2]+"  ,  "+expectedTestDiscoveries[3])
	assertions.Nil(err)

	discoveries, err = getPluginDiscoveries()
	assertions.Nil(err)
	assertions.Equal(len(expectedTestDiscoveries)+2, len(discoveries))
	// The test discoveries in order but after the configured discoveries
	assertions.Equal("default-local", discoveries[0].Local.Name)
	assertions.Equal("fake", discoveries[1].Local.Name)
	assertions.Equal(expectedTestDiscoveries[0], discoveries[2].OCI.Image)
	assertions.Equal(expectedTestDiscoveries[1], discoveries[3].OCI.Image)
	assertions.Equal(expectedTestDiscoveries[2], discoveries[4].OCI.Image)
	assertions.Equal(expectedTestDiscoveries[3], discoveries[5].OCI.Image)

	os.Unsetenv(constants.ConfigVariableAdditionalDiscoveryForTesting)
}

func TestMergeDuplicatePlugins(t *testing.T) {
	assertions := assert.New(t)

	preMergePlugins := []discovery.Discovered{
		{
			Name:               "myplugin",
			Target:             configtypes.TargetK8s,
			Description:        "First description",
			RecommendedVersion: "v2.2.2",
			InstalledVersion:   "",
			SupportedVersions:  []string{"v2.2.2"},
			Distribution: distribution.Artifacts{
				"v2.2.2": []distribution.Artifact{
					{
						Image:  "localhost:9876/my/discovery/darwin_amd64:v2.2.2",
						Digest: "digest222damd",
						OS:     "darwin",
						Arch:   "amd64",
					},
					{
						Image:  "localhost:9876/my/discovery/darwin_arm64:v2.2.2",
						Digest: "digest222darm",
						OS:     "darwin",
						Arch:   "arm64",
					},
					{
						Image:  "localhost:9876/my/discovery/linux_amd64:v2.2.2",
						Digest: "digest222lamd",
						OS:     "linux",
						Arch:   "amd64",
					},
				},
			},
			Optional:      true,
			Scope:         common.PluginScopeStandalone,
			Source:        "discovery1",
			ContextName:   "ctx1",
			DiscoveryType: common.DiscoveryTypeOCI,
			Status:        common.PluginStatusNotInstalled,
		},
		{
			Name:               "myplugin",
			Target:             configtypes.TargetK8s,
			Description:        "Second description",
			RecommendedVersion: "v3.3.3",
			InstalledVersion:   "v0.1.0",
			SupportedVersions:  []string{"v0.1.0", "v3.3.3"},
			Distribution: distribution.Artifacts{
				"v0.1.0": []distribution.Artifact{
					{
						Image:  "localhost:9876/my/discovery/darwin_amd64:v0.1.0",
						Digest: "digest010damd",
						OS:     "darwin",
						Arch:   "amd64",
					},
					{
						Image:  "localhost:9876/my/discovery/darwin_arm64:v0.1.0",
						Digest: "digest010damd",
						OS:     "darwin",
						Arch:   "arm64",
					},
					{
						Image:  "localhost:9876/my/discovery/linux_amd64:v0.1.0",
						Digest: "digest010lamd",
						OS:     "linux",
						Arch:   "amd64",
					},
				},
				"v3.3.3": []distribution.Artifact{
					{
						Image:  "localhost:9876/my/discovery/darwin_amd64:v3.3.3",
						Digest: "digest333damd",
						OS:     "darwin",
						Arch:   "amd64",
					},
					{
						Image:  "localhost:9876/my/discovery/darwin_arm64:v3.3.3",
						Digest: "digest333darm",
						OS:     "darwin",
						Arch:   "arm64",
					},
					{
						Image:  "localhost:9876/my/discovery/linux_amd64:v3.3.3",
						Digest: "digest333lamd",
						OS:     "linux",
						Arch:   "amd64",
					},
				},
			},
			Optional:      false,
			Scope:         common.PluginScopeStandalone,
			Source:        "discovery2",
			ContextName:   "ctx2",
			DiscoveryType: common.DiscoveryTypeLocal,
			Status:        common.PluginStatusInstalled,
		},
	}

	expectedPlugin := discovery.Discovered{
		Name:               "myplugin",
		Target:             configtypes.TargetK8s,
		Description:        "First description",
		RecommendedVersion: "v3.3.3",
		InstalledVersion:   "v0.1.0",
		SupportedVersions:  []string{"v0.1.0", "v2.2.2", "v3.3.3"},
		Distribution: distribution.Artifacts{
			"v0.1.0": []distribution.Artifact{
				{
					Image:  "localhost:9876/my/discovery/darwin_amd64:v0.1.0",
					Digest: "digest010damd",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "localhost:9876/my/discovery/darwin_arm64:v0.1.0",
					Digest: "digest010damd",
					OS:     "darwin",
					Arch:   "arm64",
				},
				{
					Image:  "localhost:9876/my/discovery/linux_amd64:v0.1.0",
					Digest: "digest010lamd",
					OS:     "linux",
					Arch:   "amd64",
				},
			},
			"v2.2.2": []distribution.Artifact{
				{
					Image:  "localhost:9876/my/discovery/darwin_amd64:v2.2.2",
					Digest: "digest222damd",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "localhost:9876/my/discovery/darwin_arm64:v2.2.2",
					Digest: "digest222darm",
					OS:     "darwin",
					Arch:   "arm64",
				},
				{
					Image:  "localhost:9876/my/discovery/linux_amd64:v2.2.2",
					Digest: "digest222lamd",
					OS:     "linux",
					Arch:   "amd64",
				},
			},
			"v3.3.3": []distribution.Artifact{
				{
					Image:  "localhost:9876/my/discovery/darwin_amd64:v3.3.3",
					Digest: "digest333damd",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "localhost:9876/my/discovery/darwin_arm64:v3.3.3",
					Digest: "digest333darm",
					OS:     "darwin",
					Arch:   "arm64",
				},
				{
					Image:  "localhost:9876/my/discovery/linux_amd64:v3.3.3",
					Digest: "digest333lamd",
					OS:     "linux",
					Arch:   "amd64",
				},
			},
		},
		Optional:      true,
		Scope:         common.PluginScopeStandalone,
		Source:        "discovery1/discovery2",
		ContextName:   "ctx1",
		DiscoveryType: "",
		Status:        common.PluginStatusInstalled,
	}

	mergedPlugins := mergeDuplicatePlugins(preMergePlugins)
	assertions.Equal(1, len(mergedPlugins))
	assertions.Equal(expectedPlugin, mergedPlugins[0])
}

func TestMergeDuplicatePluginsWithReplacedVersion(t *testing.T) {
	assertions := assert.New(t)

	preMergePlugins := []discovery.Discovered{
		{
			Name:               "myplugin",
			Target:             configtypes.TargetK8s,
			Description:        "First description",
			RecommendedVersion: "v1.1.1",
			InstalledVersion:   "",
			SupportedVersions:  []string{"v1.1.1"},
			Distribution: distribution.Artifacts{
				"v1.1.1": []distribution.Artifact{
					{
						Image:  "localhost:9876/my/discovery/darwin_amd64:v2.2.2",
						Digest: "digest222damd",
						OS:     "darwin",
						Arch:   "amd64",
					},
					{
						Image:  "localhost:9876/my/discovery/darwin_arm64:v2.2.2",
						Digest: "digest222darm",
						OS:     "darwin",
						Arch:   "arm64",
					},
					{
						Image:  "localhost:9876/my/discovery/linux_amd64:v2.2.2",
						Digest: "digest222lamd",
						OS:     "linux",
						Arch:   "amd64",
					},
				},
			},
			Optional:      true,
			Scope:         common.PluginScopeStandalone,
			Source:        "discovery1",
			ContextName:   "ctx1",
			DiscoveryType: common.DiscoveryTypeOCI,
			Status:        common.PluginStatusNotInstalled,
		},
		{
			Name:               "myplugin",
			Target:             configtypes.TargetK8s,
			Description:        "Second description",
			RecommendedVersion: "v1.1.1",
			InstalledVersion:   "v1.1.1",
			SupportedVersions:  []string{"v0.1.0", "v1.1.1"},
			Distribution: distribution.Artifacts{
				"v0.1.0": []distribution.Artifact{
					{
						Image:  "localhost:9876/my/discovery/darwin_amd64:v0.1.0",
						Digest: "digest010damd",
						OS:     "darwin",
						Arch:   "amd64",
					},
					{
						Image:  "localhost:9876/my/discovery/darwin_arm64:v0.1.0",
						Digest: "digest010damd",
						OS:     "darwin",
						Arch:   "arm64",
					},
					{
						Image:  "localhost:9876/my/discovery/linux_amd64:v0.1.0",
						Digest: "digest010lamd",
						OS:     "linux",
						Arch:   "amd64",
					},
				},
				"v1.1.1": []distribution.Artifact{
					{
						Image:  "localhost:9876/my/discovery/darwin_amd64:v1.1.1",
						Digest: "digest111damd",
						OS:     "darwin",
						Arch:   "amd64",
					},
					{
						Image:  "localhost:9876/my/discovery/darwin_arm64:v1.1.1",
						Digest: "digest111damd",
						OS:     "darwin",
						Arch:   "arm64",
					},
					{
						Image:  "localhost:9876/my/discovery/linux_amd64:v1.1.1",
						Digest: "digest111lamd",
						OS:     "linux",
						Arch:   "amd64",
					},
				},
			},
			Optional:      false,
			Scope:         common.PluginScopeStandalone,
			Source:        "discovery2",
			ContextName:   "ctx2",
			DiscoveryType: common.DiscoveryTypeLocal,
			Status:        common.PluginStatusInstalled,
		},
	}

	expectedPlugin := discovery.Discovered{
		Name:               "myplugin",
		Target:             configtypes.TargetK8s,
		Description:        "First description",
		RecommendedVersion: "v1.1.1",
		InstalledVersion:   "v1.1.1",
		SupportedVersions:  []string{"v0.1.0", "v1.1.1"},
		Distribution: distribution.Artifacts{
			"v0.1.0": []distribution.Artifact{
				{
					Image:  "localhost:9876/my/discovery/darwin_amd64:v0.1.0",
					Digest: "digest010damd",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "localhost:9876/my/discovery/darwin_arm64:v0.1.0",
					Digest: "digest010damd",
					OS:     "darwin",
					Arch:   "arm64",
				},
				{
					Image:  "localhost:9876/my/discovery/linux_amd64:v0.1.0",
					Digest: "digest010lamd",
					OS:     "linux",
					Arch:   "amd64",
				},
			},
			"v1.1.1": []distribution.Artifact{
				{
					Image:  "localhost:9876/my/discovery/darwin_amd64:v2.2.2",
					Digest: "digest222damd",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					Image:  "localhost:9876/my/discovery/darwin_arm64:v2.2.2",
					Digest: "digest222darm",
					OS:     "darwin",
					Arch:   "arm64",
				},
				{
					Image:  "localhost:9876/my/discovery/linux_amd64:v2.2.2",
					Digest: "digest222lamd",
					OS:     "linux",
					Arch:   "amd64",
				},
			},
		},
		Optional:      true,
		Scope:         common.PluginScopeStandalone,
		Source:        "discovery1/discovery2",
		ContextName:   "ctx1",
		DiscoveryType: "",
		Status:        common.PluginStatusInstalled,
	}

	mergedPlugins := mergeDuplicatePlugins(preMergePlugins)
	assertions.Equal(1, len(mergedPlugins))
	assertions.Equal(expectedPlugin, mergedPlugins[0])
}

func TestMergeDuplicateGroups(t *testing.T) {
	assertions := assert.New(t)

	preMergeGroups := []*plugininventory.PluginGroup{
		{
			Name:        "default",
			Vendor:      "fakevendor",
			Publisher:   "fakepublisher",
			Description: "Description for fakevendor-fakepublisher/default:v2.0.0",
			Hidden:      false,
			Versions: map[string][]*plugininventory.PluginGroupPluginEntry{
				"v2.0.0": {
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v2.2.2"},
						Mandatory:        true,
					},
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.1.1"},
						Mandatory:        true,
					},
				},
				"v1.0.0": {
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v1.0.0"},
						Mandatory:        true,
					},
				},
			},
		},
		{
			Name:        "default",
			Vendor:      "fakevendor",
			Publisher:   "fakepublisher",
			Description: "Description for fakevendor-fakepublisher/default:v3.0.0",
			Hidden:      false,
			Versions: map[string][]*plugininventory.PluginGroupPluginEntry{
				"v3.0.0": {
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v3.3.3"},
						Mandatory:        true,
					},
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.2.3"},
						Mandatory:        true,
					},
				},
			},
		},
	}

	expectedGroup := &plugininventory.PluginGroup{
		Name:        "default",
		Vendor:      "fakevendor",
		Publisher:   "fakepublisher",
		Description: "Description for fakevendor-fakepublisher/default:v3.0.0",
		Hidden:      false,
		Versions: map[string][]*plugininventory.PluginGroupPluginEntry{
			"v2.0.0": {
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v2.2.2"},
					Mandatory:        true,
				},
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.1.1"},
					Mandatory:        true,
				},
			},
			"v1.0.0": {
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v1.0.0"},
					Mandatory:        true,
				},
			},
			"v3.0.0": {
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v3.3.3"},
					Mandatory:        true,
				},
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.2.3"},
					Mandatory:        true,
				},
			},
		},
	}

	mergedGroup := mergeDuplicateGroups(preMergeGroups)
	assertions.Equal(1, len(mergedGroup))
	assertions.Equal(expectedGroup, mergedGroup[0])
}

func TestMergeDuplicateGroupsWithReplacedVersion(t *testing.T) {
	assertions := assert.New(t)

	preMergeGroups := []*plugininventory.PluginGroup{
		{
			Name:        "default",
			Vendor:      "fakevendor",
			Publisher:   "fakepublisher",
			Description: "Description for fakevendor-fakepublisher/default:v2.0.0",
			Hidden:      false,
			Versions: map[string][]*plugininventory.PluginGroupPluginEntry{
				"v2.0.0": {
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v2.2.2"},
						Mandatory:        true,
					},
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.1.1"},
						Mandatory:        true,
					},
				},
				"v1.0.0": {
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v1.0.0"},
						Mandatory:        true,
					},
				},
			},
		},
		{
			Name:        "default",
			Vendor:      "fakevendor",
			Publisher:   "fakepublisher",
			Description: "Description for fakevendor-fakepublisher/default:v3.0.0",
			Hidden:      false,
			Versions: map[string][]*plugininventory.PluginGroupPluginEntry{
				// Same version as the other group.  This version should be ignored while we keep the one from the other group.
				"v2.0.0": {
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v3.3.3"},
						Mandatory:        true,
					},
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.2.3"},
						Mandatory:        true,
					},
				},
				"v3.0.0": {
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v3.3.3"},
						Mandatory:        true,
					},
					{
						PluginIdentifier: plugininventory.PluginIdentifier{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.2.3"},
						Mandatory:        true,
					},
				},
			},
		},
	}

	expectedGroup := &plugininventory.PluginGroup{
		Name:        "default",
		Vendor:      "fakevendor",
		Publisher:   "fakepublisher",
		Description: "Description for fakevendor-fakepublisher/default:v3.0.0",
		Hidden:      false,
		Versions: map[string][]*plugininventory.PluginGroupPluginEntry{
			"v2.0.0": {
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v2.2.2"},
					Mandatory:        true,
				},
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.1.1"},
					Mandatory:        true,
				},
			},
			"v1.0.0": {
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v1.0.0"},
					Mandatory:        true,
				},
			},
			"v3.0.0": {
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v3.3.3"},
					Mandatory:        true,
				},
				{
					PluginIdentifier: plugininventory.PluginIdentifier{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.2.3"},
					Mandatory:        true,
				},
			},
		},
	}

	mergedGroup := mergeDuplicateGroups(preMergeGroups)
	assertions.Equal(1, len(mergedGroup))
	assertions.Equal(expectedGroup, mergedGroup[0])
}

func TestClean(t *testing.T) {
	assertions := assert.New(t)

	// Create a fake cache directory
	cacheDir, err := os.MkdirTemp("", "test-cache")
	assertions.Nil(err)
	common.DefaultCacheDir = cacheDir
	defer os.RemoveAll(cacheDir)

	pluginDir, err := os.MkdirTemp("", "test-plugins")
	assertions.Nil(err)
	common.DefaultPluginRoot = pluginDir
	defer os.RemoveAll(pluginDir)

	// Add a catalog file to the cache
	catalogFile := filepath.Join(common.DefaultCacheDir, "catalog.yaml")
	_, err = os.Create(catalogFile)
	assertions.Nil(err)

	// Add a couple of plugin inventory directories with files inside
	inventoryDir := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName)
	err = os.Mkdir(inventoryDir, 0777)
	assertions.Nil(err)
	for _, dirName := range []string{config.DefaultStandaloneDiscoveryName, "disc_0", "disc_1"} {
		dir := filepath.Join(inventoryDir, dirName)
		err = os.Mkdir(dir, 0777)
		assertions.Nil(err)

		_, err = os.Create(filepath.Join(dir, plugininventory.SQliteDBFileName))
		assertions.Nil(err)

		_, err = os.Create(filepath.Join(
			dir,
			"digest.f7603fe167bfa39c23dc06dda1336cb5a3a0c86ecfc919612ebc7620fca4026a"))
		assertions.Nil(err)
	}

	// Add a couple of fake plugin directories with a binary
	for _, dirName := range []string{"builder", "cluster"} {
		dir := filepath.Join(common.DefaultPluginRoot, dirName)
		err = os.Mkdir(dir, 0777)
		assertions.Nil(err)

		_, err = os.Create(filepath.Join(dir, "v9.9.9_d5ae2d7b3bf9416e569fdf77c45d69ae21e793939ad5c42c97b0c83cc46cf55d_global"))
		assertions.Nil(err)
	}

	// Now call ask the pluginmanager to clean those up
	err = Clean()
	assertions.Nil(err)

	// Verify everything is gone
	_, err = os.Stat(catalogFile)
	assertions.True(errors.Is(err, os.ErrNotExist))
	_, err = os.Stat(inventoryDir)
	assertions.True(errors.Is(err, os.ErrNotExist))
	_, err = os.Stat(common.DefaultPluginRoot)
	assertions.True(errors.Is(err, os.ErrNotExist))
}

func TestRemoveOldPluginsWhenDuplicates(t *testing.T) {
	tests := []struct {
		name   string
		input  []discovery.Discovered
		output []discovery.Discovered
	}{
		{
			name: "Test1",
			input: []discovery.Discovered{
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.1.0"},
				{Name: "bar", Target: configtypes.TargetK8s, RecommendedVersion: "v1.1.0"},
				{Name: "baz", Target: configtypes.TargetK8s, RecommendedVersion: "v1.2.0"},
			},
			output: []discovery.Discovered{
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.1.0"},
				{Name: "bar", Target: configtypes.TargetK8s, RecommendedVersion: "v1.1.0"},
				{Name: "baz", Target: configtypes.TargetK8s, RecommendedVersion: "v1.2.0"},
			},
		},
		{
			name: "Test2",
			input: []discovery.Discovered{
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.1.0"},
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.1.0"},
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.2.0"},
			},
			output: []discovery.Discovered{
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.2.0"},
			},
		},
		{
			name: "Test3",
			input: []discovery.Discovered{
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.1.0"},
				{Name: "bar", Target: configtypes.TargetTMC, RecommendedVersion: "v1.1.0"},
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.2.0"},
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.2.0"},
			},
			output: []discovery.Discovered{
				{Name: "bar", Target: configtypes.TargetTMC, RecommendedVersion: "v1.1.0"},
				{Name: "foo", Target: configtypes.TargetK8s, RecommendedVersion: "v1.2.0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeOldPluginsWhenDuplicates(tt.input)
			if !reflect.DeepEqual(got, tt.output) {
				t.Errorf("Testcase: %v, RemoveOldPluginsWhenDuplicates() = %v, want %v", tt.name, got, tt.output)
			}
		})
	}
}
