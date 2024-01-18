// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugincmdtree

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

const samplePluginToGenerateAliasWithHelpCommand string = `#!/bin/bash

# Dummy Tanzu CLI 'Plugin' to return the help output so that parser can extract aliases

if [ "$1" = "-h" ]; then
  echo "fake plugin help"
  echo "Aliases:"
  echo "  sp"
elif [ "$1" = "foo1" ] && [ "$2" = "-h"  ]; then
    echo "fake foo1 command with aliases"
    echo "Aliases:"
    echo "  foo1, f1"
elif [ "$1" = "bar1" ] && [ "$2" = "-h"  ]; then
    echo "fake bar1 command without aliases"
elif [ "$1" = "foo1" ] && [ "$2" = "foo2" ] && [ "$3" = "-h" ]; then
  echo "fake foo2 command with aliases"
  echo "Aliases:"
  echo "  foo2, f2"
else
  echo "Invalid command."
fi
`

const expectedPluginCmdTree string = `
commandTree:
  ? %s
  : subcommands:
      bar1:
        subcommands: {}
        aliases: {}
      foo1:
        subcommands:
          foo2:
            subcommands: {}
            aliases:
              f2: {}
              foo2: {}
        aliases:
          f1: {}
          foo1: {}
    aliases: {}
`

func Test_RepeatConstructAndAddTree(t *testing.T) {
	for i := 0; i < 10; i++ {
		testConstructAndAddTree(t)
	}
}

func testConstructAndAddTree(t *testing.T) {
	// create the command docs
	tmpCacheDir, err := os.MkdirTemp("", "cache")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpCacheDir)

	tmpCMDDocsDir := filepath.Join(tmpCacheDir, ".docs")
	err = os.MkdirAll(tmpCMDDocsDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(tmpCMDDocsDir)

	// pre-generate the plugin docs for the dummy plugin
	docsFiles := []string{"tanzu.md", "tanzu_sample-plugin.md", "tanzu_sample-plugin_foo1.md", "tanzu_sample-plugin_bar1.md", "tanzu_sample-plugin_foo1_foo2.md"}
	err = createPluginDocs(tmpCMDDocsDir, docsFiles)
	assert.NoError(t, err, "failed to create command docs for testing")

	os.Setenv("TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR", tmpCacheDir)
	defer func() {
		os.Unsetenv("TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR")
	}()

	// setup the plugin
	samplePluginName := "sample-plugin"
	setupDummyPlugin(t, tmpCacheDir, samplePluginName)

	// Initialize the cache
	pct, err := getPluginCommandTree()
	assert.NoError(t, err)
	cache := &cacheImpl{
		pluginCommands: pct,
		pluginDocsGenerator: func(plugin *cli.PluginInfo) error {
			//dummy generator as we are pre-generating the plugin docs
			return nil
		},
	}

	// Create a sample plugin
	plugin := &cli.PluginInfo{
		Name:             "sample-plugin",
		InstallationPath: filepath.Join(tmpCacheDir, samplePluginName),
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}

	// Test constructing the plugin command tree with valid plugin binary path,
	// it should generate the command tree and alias correctly
	err = cache.ConstructAndAddTree(plugin)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(cache.pluginCommands.CommandTree))

	validatePluginCommandTree(t, cache.pluginCommands, fmt.Sprintf(expectedPluginCmdTree, plugin.InstallationPath))

	// Test getting the command tree for a non-existing plugin
	nonExistingPlugin := &cli.PluginInfo{
		Name:             "non-existing-plugin",
		InstallationPath: "/path/to/non-existing-plugin",
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}
	err = cache.ConstructAndAddTree(nonExistingPlugin)
	assert.Error(t, err)
}

func TestCache_GetTree(t *testing.T) {
	// Create a sample plugin
	plugin := &cli.PluginInfo{
		Name:             "sample-plugin",
		InstallationPath: "/path/to/sample-plugin",
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}
	cache := getCacheWithSamplePluginCommandTree(plugin.Name, plugin.InstallationPath)

	expectedCMDTree := cache.pluginCommands.CommandTree[plugin.InstallationPath]
	// Test getting the command tree
	commandTree, err := cache.GetTree(plugin)
	assert.NoError(t, err)
	assert.NotNil(t, commandTree)
	assert.Equal(t, expectedCMDTree, commandTree)

	// Test getting the command tree for a non-existing plugin
	// set the pluginDocsGenerator to the actual docs generator
	cache.pluginDocsGenerator = generatePluginDocs
	nonExistingPlugin := &cli.PluginInfo{
		Name:             "non-existing-plugin",
		InstallationPath: "/path/to/non-existing-plugin",
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}
	nonExistingCommandTree, err := cache.GetTree(nonExistingPlugin)
	assert.Error(t, err)
	assert.Nil(t, nonExistingCommandTree)
}

func TestCache_DeleteTree(t *testing.T) {
	// Create a sample plugin
	plugin := &cli.PluginInfo{
		Name:             "sample-plugin",
		InstallationPath: "/path/to/sample-plugin",
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}
	cache := getCacheWithSamplePluginCommandTree(plugin.Name, plugin.InstallationPath)

	// Test deleting the command tree
	err := cache.DeleteTree(plugin)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(cache.pluginCommands.CommandTree))

	// Test getting the command tree for a non-existing plugin
	nonExistingPlugin := &cli.PluginInfo{
		Name:             "non-existing-plugin",
		InstallationPath: "/path/to/non-existing-plugin",
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}
	err = cache.DeleteTree(nonExistingPlugin)
	assert.NoError(t, err)
}

func getCacheWithSamplePluginCommandTree(_, pluginInstallationPath string) *cacheImpl {
	pluginCMDTree := &CommandNode{
		Subcommands: map[string]*CommandNode{
			"plugin-subcmd1": &CommandNode{
				Subcommands: map[string]*CommandNode{
					"plugin-subcmd2": NewCommandNode(),
				},
				Aliases: map[string]struct{}{
					"pscmd1-alias": {},
				},
			},
		},
	}

	cache := &cacheImpl{
		pluginCommands: &pluginCommandTree{
			CommandTree: map[string]*CommandNode{
				pluginInstallationPath: pluginCMDTree,
			},
		},
		pluginDocsGenerator: func(plugin *cli.PluginInfo) error {
			return nil
		},
	}
	return cache
}

func createPluginDocs(docsDir string, docNames []string) error {
	for _, doc := range docNames {
		docPath := filepath.Join(docsDir, doc)
		file, err := os.Create(docPath)
		if err != nil {
			return err
		}
		defer file.Close()
	}
	return nil
}

func setupDummyPlugin(t *testing.T, dirName, pluginName string) {
	pluginExeFile, err := os.OpenFile(filepath.Join(dirName, pluginName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	assert.NoError(t, err)
	defer pluginExeFile.Close()

	fmt.Fprint(pluginExeFile, samplePluginToGenerateAliasWithHelpCommand)
}
func validatePluginCommandTree(t *testing.T, gotPluginCommandTree *pluginCommandTree, expectedPluginCommandTreeYaml string) {
	// the yaml representation should marshaled and unmarshal to get rid of the "AliasProcessed" field for comparison
	gotPluginCommandTreeBytes, err := yaml.Marshal(gotPluginCommandTree)
	assert.NoError(t, err)
	getPluginCommandTreeUnmarshaled := &pluginCommandTree{}
	err = yaml.Unmarshal(gotPluginCommandTreeBytes, getPluginCommandTreeUnmarshaled)
	assert.NoError(t, err)

	expPluginCmdTree := &pluginCommandTree{}
	err = yaml.Unmarshal([]byte(expectedPluginCommandTreeYaml), expPluginCmdTree)
	assert.NoError(t, err, "failed to unmarshal the expected plugin command tree")
	assert.Equal(t, expPluginCmdTree, getPluginCommandTreeUnmarshaled, "the plugin command tree doesn't match with the expected ")
}
