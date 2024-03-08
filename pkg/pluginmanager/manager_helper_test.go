// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	// Import the sqlite driver
	_ "modernc.org/sqlite"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"

	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

var testPlugins = []plugininventory.PluginIdentifier{
	{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v1.6.0"},
	{Name: "cluster", Target: configtypes.TargetK8s, Version: "v1.6.0"},
	{Name: "myplugin", Target: configtypes.TargetK8s, Version: "v1.6.0"},
	{Name: "feature", Target: configtypes.TargetK8s, Version: "v0.2.0"},
	{Name: "pluginwitharmdarwin", Target: configtypes.TargetK8s, Version: "v2.0.0"},

	{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.2.3"},
	{Name: "isolated-cluster", Target: configtypes.TargetGlobal, Version: "v1.3.0"},
	{Name: "login", Target: configtypes.TargetGlobal, Version: "v0.2.0"},
	{Name: "login", Target: configtypes.TargetGlobal, Version: "v0.2.0-beta.1"},
	{Name: "login", Target: configtypes.TargetGlobal, Version: "v0.20.0"},

	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.0.1"},
	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.0.2"},
	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.0.3"},
	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.2.0"},
	{Name: "cluster", Target: configtypes.TargetTMC, Version: "v0.2.0"},
	{Name: "myplugin", Target: configtypes.TargetTMC, Version: "v0.2.0"},
	{Name: "pluginwitharmwindows", Target: configtypes.TargetTMC, Version: "v4.0.0"},
}

var testPluginsNoARM64 = []plugininventory.PluginIdentifier{
	{Name: "pluginnoarmdarwin", Target: configtypes.TargetK8s, Version: "v1.0.0"},
	{Name: "pluginnoarmwindows", Target: configtypes.TargetTMC, Version: "v3.0.0"},
}

var installedStandalonePlugins = []plugininventory.PluginIdentifier{
	{Name: "management-cluster", Target: configtypes.TargetK8s, Version: "v0.1.0"},
	{Name: "cluster", Target: configtypes.TargetK8s, Version: "v0.0.1"},
	{Name: "feature", Target: configtypes.TargetK8s, Version: "v0.0.2"},
	{Name: "secret", Target: configtypes.TargetK8s, Version: "v0.3.0"},

	{Name: "management-cluster", Target: configtypes.TargetTMC, Version: "v0.0.1"},
	{Name: "cluster", Target: configtypes.TargetTMC, Version: "v0.0.5"},
}

const createGroupsStmt = `
INSERT INTO PluginGroups VALUES(
	'vmware',
	'test',
	'default',
	'v1.6.0',
	'Description for vmware-test/default:v1.6.0',
	'management-cluster',
	'kubernetes',
	'v1.6.0',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'test',
	'default',
	'v1.6.0',
	'Description for vmware-test/default:v1.6.0',
	'feature',
	'kubernetes',
	'v0.2.0',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'test',
	'default',
	'v1.6.0',
	'Description for vmware-test/default:v1.6.0',
	'myplugin',
	'kubernetes',
	'v1.6.0',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'test',
	'default',
	'v1.6.0',
	'Description for vmware-test/default:v1.6.0',
	'isolated-cluster',
	'global',
	'v1.2.3',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'test',
	'default',
	'v1.6.0',
	'Description for vmware-test/default:v1.6.0',
	'cluster',
	'kubernetes',
	'v1.6.0',
	'false',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'test',
	'default',
	'v2.1.0',
	'Description for vmware-test/default:v2.1.0',
	'isolated-cluster',
	'global',
	'v1.3',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'test',
	'default',
	'v2.1.0-beta',
	'Description for vmware-test/default:v2.1.0-beta',
	'isolated-cluster',
	'global',
	'v1.3.0',
	'true',
	'false');
INSERT INTO PluginGroups VALUES(
	'vmware',
	'test',
	'default',
	'v2.2.0',
	'Description for vmware-test/default:v2.2.0',
	'isolated-cluster',
	'global',
	'v1',
	'true',
	'false');
`
const (
	digestForAMD64 = "0000000000"
	digestForARM64 = "1111111111"
)

func findDiscoveredPlugin(discovered []discovery.Discovered, pluginName string, target configtypes.Target) *discovery.Discovered {
	for i := range discovered {
		if pluginName == discovered[i].Name && target == discovered[i].Target {
			return &discovered[i]
		}
	}
	return nil
}

func findPluginInfo(pd []cli.PluginInfo, pluginName string, target configtypes.Target) *cli.PluginInfo {
	for i := range pd {
		if pluginName == pd[i].Name && target == pd[i].Target {
			return &pd[i]
		}
	}
	return nil
}

func findGroupVersion(allGroups []*plugininventory.PluginGroup, id string) bool {
	groupID := plugininventory.PluginGroupIdentifierFromID(id)
	for _, g := range allGroups {
		if g.Publisher == groupID.Publisher &&
			g.Vendor == groupID.Vendor &&
			g.Name == groupID.Name {
			for v := range g.Versions {
				if v == groupID.Version {
					return true
				}
			}
		}
	}
	return false
}

func setupPluginBinaryInCache(name, version string, target configtypes.Target, arch cli.Arch, digest string) {
	dir := filepath.Join(common.DefaultPluginRoot, name)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatal(err, "unable to create temporary directory for plugin binary")
	}

	pluginBinary := filepath.Join(dir, fmt.Sprintf("%s_%s_%s", version, digest, target))
	if arch.IsWindows() {
		pluginBinary += exe
	}

	f, err := os.OpenFile(pluginBinary, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatal(err, "unable to create temporary plugin binary")
	}
	defer f.Close()

	_, err = fmt.Fprintf(f,
		`{"name":"%s",
"description":"Test plugin",
"target":"%s",
"version": "%s",
"buildSHA":"c2dbd15",
"digest":"%s",
"group":"Run",
"docURL":"",
"completionType":0,
"aliases":["test"],
"installationPath":"",
"discovery":"",
"scope":"",
"status":""}`, name, target, version, digest)

	if err != nil {
		log.Fatal(err, fmt.Sprintf("Error while generating plugin binary %s", pluginBinary))
	}
}

func createPluginEntry(db *sql.DB, plugin plugininventory.PluginIdentifier, arch cli.Arch, digest string) {
	uri := fmt.Sprintf("vmware/test/%s/%s/%s/%s:%s", arch.OS(), arch.Arch(), plugin.Target, plugin.Name, plugin.Version)
	desc := fmt.Sprintf("Plugin %s description", plugin.Name)

	_, err := db.Exec("INSERT INTO PluginBinaries (PluginName,Target,RecommendedVersion,Version,Hidden,Description,Publisher,Vendor,OS,Architecture,Digest,URI) VALUES(?,?,'',?,'false',?,'test','vmware',?,?,?,?);", plugin.Name, plugin.Target, plugin.Version, desc, arch.OS(), arch.Arch(), digest, uri)

	if err != nil {
		log.Fatal(err, fmt.Sprintf("failed to create %s:%s for target %s for testing", plugin.Name, plugin.Version, plugin.Target))
	}
}

func setupPluginEntriesAndBinaries(db *sql.DB) {
	// Setup DB entries and plugin binaries for all os/architecture combinations
	for _, plugin := range testPlugins {
		for _, osArch := range cli.AllOSArch {
			digest := digestForAMD64
			if osArch.Arch() == cli.DarwinARM64.Arch() {
				digest = digestForARM64
			}
			createPluginEntry(db, plugin, osArch, digest)
			setupPluginBinaryInCache(plugin.Name, plugin.Version, plugin.Target, osArch, digest)
		}
	}

	// Setup DB entries and plugin binaries but skip ARM64
	digest := digestForAMD64
	for _, plugin := range testPluginsNoARM64 {
		for _, osArch := range cli.AllOSArch {
			if osArch.Arch() != cli.DarwinARM64.Arch() {
				createPluginEntry(db, plugin, osArch, digest)
				setupPluginBinaryInCache(plugin.Name, plugin.Version, plugin.Target, osArch, digest)
			}
		}
	}
}

func setupTestPluginInventory() {
	// Create a temporary directory for the plugin inventory DB
	inventoryDir := filepath.Join(
		common.DefaultCacheDir,
		common.PluginInventoryDirName,
		config.DefaultStandaloneDiscoveryName)
	err := os.MkdirAll(inventoryDir, 0755)
	if err != nil {
		log.Fatal(err, "unable to create temporary directory for plugin inventory")
	}

	// Generate a test plugin inventory DB
	dbFile, err := os.Create(filepath.Join(inventoryDir, plugininventory.SQliteDBFileName))
	if err != nil {
		log.Fatal(err, "unable to create temporary file for plugin inventory")
	}
	// Open DB with the sqlite driver
	db, err := sql.Open("sqlite", dbFile.Name())
	if err != nil {
		log.Fatal(err, "unable to open create temporary plugin inventory DB")
	}
	defer db.Close()

	// Create the table
	_, err = db.Exec(plugininventory.CreateTablesSchema)
	if err != nil {
		log.Fatal(err, "failed to create DB table for testing")
	}

	// Add plugin entries to the DB and create the corresponding binaries
	setupPluginEntriesAndBinaries(db)

	// Add plugin group entries to the DB
	_, err = db.Exec(createGroupsStmt)
	if err != nil {
		log.Fatal(err, "failed to create plugin groups for testing")
	}
}

func setupTestPluginCatalog() {
	// Create catalog for standalone plugins
	cc, err := catalog.NewContextCatalogUpdater("")
	if err != nil {
		log.Fatal(err, "unable to create catalog updater")
	}

	for _, plugin := range installedStandalonePlugins {
		entry := cli.PluginInfo{
			Name:             plugin.Name,
			Target:           plugin.Target,
			Version:          plugin.Version,
			Description:      fmt.Sprintf("Plugin %s/%s description", plugin.Name, plugin.Target),
			InstallationPath: "/path/" + string(plugin.Target) + "/" + plugin.Name,
		}

		err = cc.Upsert(&entry)
		if err != nil {
			log.Fatal(err, "unable to insert into catalog")
		}
	}
	cc.Unlock()
}

func setupPluginSourceForTesting() func() {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		log.Fatal(err, "unable to create temporary directory")
	}

	common.DefaultPluginRoot = filepath.Join(tmpDir, "plugin-root")

	// Setup the two temporary configuration files
	configFile := filepath.Join(tmpDir, "tanzu_config.yaml")
	configNextGenFile := filepath.Join(tmpDir, "tanzu_config_ng.yaml")
	os.Setenv("TANZU_CONFIG", configFile)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configNextGenFile)

	// Setup both test configuration files
	err = copy.Copy(filepath.Join("test", "config.yaml"), configFile)
	if err != nil {
		log.Fatal(err, "Error while coping tanzu config file for testing")
	}

	err = copy.Copy(filepath.Join("test", "config-ng2.yaml"), configNextGenFile)
	if err != nil {
		log.Fatal(err, "Error while coping tanzu config next gen file for testing")
	}

	// Setup a temporary cache directory
	common.DefaultCacheDir = filepath.Join(tmpDir, "cache")

	common.DefaultLocalPluginDistroDir = filepath.Join(tmpDir, "distro")
	err = copy.Copy(filepath.Join("test", "local"), common.DefaultLocalPluginDistroDir)
	if err != nil {
		log.Fatal(err, "Error while setting local distro for testing")
	}

	setupTestPluginInventory()
	os.Setenv("TEST_TANZU_CLI_USE_DB_CACHE_ONLY", "1")

	return func() {
		os.RemoveAll(tmpDir)
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TEST_TANZU_CLI_USE_DB_CACHE_ONLY")
	}
}

func setupLocalDistroForTesting() func() {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		log.Fatal(err, "unable to create temporary directory")
	}

	tmpHomeDir, err := os.MkdirTemp(os.TempDir(), "home")
	if err != nil {
		log.Fatal(err, "unable to create temporary home directory")
	}

	config.DefaultStandaloneDiscoveryType = "local"
	config.DefaultStandaloneDiscoveryLocalPath = "default"

	common.DefaultPluginRoot = filepath.Join(tmpDir, "plugin-root")
	common.DefaultLocalPluginDistroDir = filepath.Join(tmpDir, "distro")
	common.DefaultCacheDir = filepath.Join(tmpDir, "cache")

	configFile := filepath.Join(tmpDir, "tanzu_config.yaml")
	configNextGenFile := filepath.Join(tmpDir, "tanzu_config_ng.yaml")
	os.Setenv("TANZU_CONFIG", configFile)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configNextGenFile)
	os.Setenv("HOME", tmpHomeDir)

	err = copy.Copy(filepath.Join("test", "local"), common.DefaultLocalPluginDistroDir)
	if err != nil {
		log.Fatal(err, "Error while setting local distro for testing")
	}

	err = copy.Copy(filepath.Join("test", "config.yaml"), configFile)
	if err != nil {
		log.Fatal(err, "Error while coping tanzu config file for testing")
	}

	err = copy.Copy(filepath.Join("test", "config-ng.yaml"), configNextGenFile)
	if err != nil {
		log.Fatal(err, "Error while coping tanzu config next gen file for testing")
	}

	err = configlib.SetFeature("global", "context-target-v2", "true")
	if err != nil {
		log.Fatal(err, "Error while coping tanzu config file for testing")
	}

	return func() {
		os.RemoveAll(tmpDir)
	}
}

func mockInstallPlugin(assert *assert.Assertions, name, version string, target configtypes.Target) {
	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	err := InstallStandalonePlugin(name, version, target)
	assert.Nil(err)
}

// Reference: https://jamiethompson.me/posts/Unit-Testing-Exec-Command-In-Golang/
func fakeInfoExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...) //nolint:gosec
	tc := "FILE_PATH=" + command
	home := "HOME=" + os.Getenv("HOME")
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", tc, home}
	return cmd
}
