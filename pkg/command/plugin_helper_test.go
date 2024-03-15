// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	// Import the sqlite driver
	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/assert"

	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
)

var availablePlugins = []plugininventory.PluginIdentifier{
	{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v0.1.0"},
	{Name: "cluster", Target: configtypes.TargetK8s, Version: "v0.0.1"},
	{Name: "feature", Target: configtypes.TargetK8s, Version: "v0.0.2"},
	{Name: "package", Target: configtypes.TargetK8s, Version: "v0.2.0"},
	{Name: "secret", Target: configtypes.TargetK8s, Version: "v0.3.0"},

	{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.2.3"},
	{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.3.0"},
	{Name: "login", Target: configtypes.TargetGlobal, Version: "v1.2.0"},
	{Name: "login", Target: configtypes.TargetGlobal, Version: "v1.2.0-beta.1"},

	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.0.1"},
	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.0.2"},
	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.0.3"},
	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.2.0"},
	{Name: "cluster", Target: configtypes.TargetTMC, Version: "v0.0.5"},
	{Name: "secret", Target: configtypes.TargetK8s, Version: "v0.0.6"},
}

var installedStandalonePlugins = []plugininventory.PluginIdentifier{
	{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v0.1.0"},
	{Name: "secret", Target: configtypes.TargetK8s, Version: "v0.3.0"},
	{Name: "cluster", Target: configtypes.TargetK8s, Version: "v0.0.1"},
	{Name: "feature", Target: configtypes.TargetK8s, Version: "v0.0.2"},

	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.0.1"},
	{Name: "cluster", Target: configtypes.TargetTMC, Version: "v0.0.5"},
}

const createGroupsStmt = `
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'default',
	'v1.1.1',
	'Plugins for TKG',
	'management-cluster',
	'kubernetes',
	'v0.1.0',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'default',
	'v1.1.1',
	'Plugins for TKG',
	'package',
	'kubernetes',
	'v0.2.0',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'default',
	'v1.1.1',
	'Plugins for TKG',
	'secret',
	'kubernetes',
	'v0.3.0',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'default',
	'v1.1.1',
	'Plugins for TKG',
	'isolated-cluster',
	'global',
	'v1.2.3',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'default',
	'v1.1.1',
	'Plugins for TKG',
	'cluster',
	'kubernetes',
	'v1.1.1',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'default',
	'v1.1.1',
	'Plugins for TKG',
	'login',
	'global',
	'v1.2.0',
	'true',
	'false');


INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'default',
	'v2.2.2',
	'Plugins for TKG',
	'isolated-cluster',
	'global',
	'v1.3',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tkg',
	'default',
	'v2.2.2-beta',
	'Plugins for TKG',
	'isolated-cluster',
	'global',
	'v1.3.0',
	'true',
	'false');


INSERT INTO PluginGroups VALUES(
	'vmware',
	'tap',
	'default',
	'v3.3.3',
	'Plugins for TAP',
	'apps',
	'kubernetes',
	'v0.1.0',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tap',
	'default',
	'v3.3.3',
	'Plugins for TAP',
	'package',
	'kubernetes',
	'v0.2.0',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'tap',
	'default',
	'v3.3.3',
	'Plugins for TAP',
	'secret',
	'kubernetes',
	'v0.3.0',
	'true',
	'false');
`

func setupPluginEntries(t *testing.T, db *sql.DB) {
	const digest = "0000000000"

	// Setup DB entries and plugin binaries for all os/architecture combinations
	for _, plugin := range availablePlugins {
		for _, osArch := range cli.AllOSArch {
			uri := fmt.Sprintf("vmware/tkg/%s/%s/%s/%s:%s", osArch.OS(), osArch.Arch(), plugin.Target, plugin.Name, plugin.Version)
			desc := fmt.Sprintf("Plugin %s/%s description", plugin.Name, plugin.Target)

			_, err := db.Exec("INSERT INTO PluginBinaries (PluginName,Target,RecommendedVersion,Version,Hidden,Description,Publisher,Vendor,OS,Architecture,Digest,URI) VALUES(?,?,'',?,'false',?,'test','vmware',?,?,?,?);", plugin.Name, plugin.Target, plugin.Version, desc, osArch.OS(), osArch.Arch(), digest, uri)

			assert.Nil(t, err)
		}
	}
}

func setupTestPluginInventory(t *testing.T) {
	// Create a temporary directory for the plugin inventory DB
	inventoryDir := filepath.Join(
		common.DefaultCacheDir,
		common.PluginInventoryDirName,
		config.DefaultStandaloneDiscoveryName)
	err := os.MkdirAll(inventoryDir, 0755)
	assert.Nil(t, err)

	// Generate a test plugin inventory DB
	dbFile, err := os.Create(filepath.Join(inventoryDir, plugininventory.SQliteDBFileName))
	assert.Nil(t, err)

	// Open DB with the sqlite driver
	db, err := sql.Open("sqlite", dbFile.Name())
	assert.Nil(t, err)
	defer db.Close()

	// Create the table
	_, err = db.Exec(plugininventory.CreateTablesSchema)
	assert.Nil(t, err)

	// Add plugin entries to the DB and create the corresponding binaries
	setupPluginEntries(t, db)

	// Add plugin group entries to the DB
	_, err = db.Exec(createGroupsStmt)
	assert.Nil(t, err)
}

func setupTestPluginCatalog(t *testing.T) {
	// Create catalog for standalone plugins
	cc, err := catalog.NewContextCatalogUpdater("")
	assert.Nil(t, err)
	assert.NotNil(t, cc)

	for _, plugin := range installedStandalonePlugins {
		entry := cli.PluginInfo{
			Name:             plugin.Name,
			Target:           plugin.Target,
			Version:          plugin.Version,
			Description:      fmt.Sprintf("Plugin %s/%s description", plugin.Name, plugin.Target),
			InstallationPath: "/path/" + string(plugin.Target) + "/" + plugin.Name,
		}

		err = cc.Upsert(&entry)
		assert.Nil(t, err)
	}
	cc.Unlock()
}

func setupPluginSourceForTesting(t *testing.T) func() {
	// Setup a temporary configuration
	configFile, err := os.CreateTemp("", "config")
	assert.Nil(t, err)
	os.Setenv("TANZU_CONFIG", configFile.Name())
	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(t, err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

	os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
	os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")

	// Setup a temporary cache directory
	dir, err := os.MkdirTemp("", "cache")
	assert.Nil(t, err)
	common.DefaultCacheDir = dir

	// Set an invalid central repo to make sure we only use the cached DB
	// and don't actually go to the internet when doing completion
	err = configlib.SetCLIDiscoverySource(configtypes.PluginDiscovery{
		OCI: &configtypes.OCIDiscovery{
			Name:  config.DefaultStandaloneDiscoveryName,
			Image: "example.com/tanzu_cli/plugins/plugin-inventory:latest",
		},
	})
	assert.Nil(t, err)

	setupTestPluginInventory(t)

	setupTestPluginCatalog(t)

	return func() {
		os.RemoveAll(dir)
		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
	}
}
