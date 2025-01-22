// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugincmdtree

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	plugintypes "github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

const samplePluginToGenerateAliasWithHelpCommand string = `#!/bin/bash

# Dummy Tanzu CLI 'Plugin' to return the help output so that parser can extract aliases

if [ "$1" = "-h" ]; then
  echo "fake plugin help"
  echo "Aliases:"
  echo "  %[1]s, %[2]s"
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
elif [ "$1" = "cluster" ] && [ "$2" = "-h"  ]; then
    echo "fake cluster command with aliases"
    echo "Aliases:"
    echo "  cluster, cl"
else
  echo "Invalid command."
fi
`

const remappedCmdName = "cluster"

var pluginName = map[configtypes.Target]string{
	configtypes.TargetGlobal:     "cluster-plugin-global",
	configtypes.TargetK8s:        "cluster-plugin-k8s",
	configtypes.TargetOperations: "cluster-plugin-ops", // Use the remapped command as a prefix to make sure it doesn't affect the command tree
}

var pluginAlias = map[configtypes.Target]string{
	configtypes.TargetGlobal:     "pg",
	configtypes.TargetK8s:        "pk",
	configtypes.TargetOperations: "po",
}

var expectedPluginTree = map[configtypes.Target]string{
	configtypes.TargetGlobal: `
  ? %s
  : subcommands:
      cluster-plugin-global:
        subcommands:
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
        aliases:
          pg: {}
          cluster-plugin-global: {}
      cluster:
        subcommands: {}
        aliases:
          cl: {}
          cluster: {}
    aliases: {}
`,
	configtypes.TargetK8s: `
  ? %s
  : subcommands:
      cluster-plugin-k8s:
        subcommands:
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
        aliases:
          pk: {}
          cluster-plugin-k8s: {}
      kubernetes:
        subcommands:
          cluster-plugin-k8s:
            subcommands:
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
            aliases:
              pk: {}
              cluster-plugin-k8s: {}
        aliases:
          k8s: {}
          kubernetes: {}
      cluster:
        subcommands: {}
        aliases:
          cl: {}
          cluster: {}
    aliases: {}
`,
	configtypes.TargetOperations: `
  ? %s
  : subcommands:
      operations:
        subcommands:
          cluster-plugin-ops:
            subcommands:
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
            aliases:
              po: {}
              cluster-plugin-ops: {}
        aliases:
          ops: {}
          operations: {}
      cluster:
        subcommands: {}
        aliases:
          cl: {}
          cluster: {}
    aliases: {}
`,
}

// Create a test root command which contains different plugin commands we want to test:
// - tanzu
// - tanzu cluster-plugin-global
// - tanzu cluster-plugin-k8s
// - tanzu operations cluster-plugin-ops
// - tanzu kubernetes cluster-plugin-k8s
// - tanzu remapped
var rootCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "tanzu",
	}
	// Add the two targets we will test
	opsTargetCmd := &cobra.Command{Use: "operations"}
	k8sTargetCmd := &cobra.Command{Use: "kubernetes"}
	cmd.AddCommand(opsTargetCmd, k8sTargetCmd)

	// Add the plugins
	// tanzu sample-plugin
	cmd.AddCommand(&cobra.Command{
		Use: pluginName[configtypes.TargetGlobal],
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Annotations: map[string]string{
			common.AnnotationForCmdSrcPath: "",
		},
	})
	// tanzu operations ops-plugin
	opsTargetCmd.AddCommand(&cobra.Command{
		Use: pluginName[configtypes.TargetOperations],
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Annotations: map[string]string{
			common.AnnotationForCmdSrcPath: "",
		},
	})
	// tanzu kuberntes k8s-plugin
	k8sTargetCmd.AddCommand(&cobra.Command{
		Use: pluginName[configtypes.TargetK8s],
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Annotations: map[string]string{
			common.AnnotationForCmdSrcPath: "",
		},
	})
	// k8s plugin are also at the root level
	// tanzu k8s-plugin
	cmd.AddCommand(&cobra.Command{
		Use: pluginName[configtypes.TargetK8s],
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Annotations: map[string]string{
			common.AnnotationForCmdSrcPath: "",
		},
	})
	// And a command from the plugin remapped to the root
	// tanzu remapped
	cmd.AddCommand(&cobra.Command{
		Use: remappedCmdName,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Annotations: map[string]string{
			// This is the source command for the remapped command
			common.AnnotationForCmdSrcPath: remappedCmdName,
		},
	})

	return cmd
}()

func Test_RepeatConstructAndAddTree(t *testing.T) {
	for i := 0; i < 10; i++ {
		testConstructAndAddTreeForPlugin(t, configtypes.TargetGlobal)
	}
}

func Test_ConstructAndAddTreeK8s(t *testing.T) {
	testConstructAndAddTreeForPlugin(t, configtypes.TargetK8s)
}

func Test_ConstructAndAddTreeOps(t *testing.T) {
	testConstructAndAddTreeForPlugin(t, configtypes.TargetOperations)
}

func testConstructAndAddTreeForPlugin(t *testing.T, target configtypes.Target) {
	// create the command docs
	tmpCacheDir, err := os.MkdirTemp("", "cache")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpCacheDir)

	tmpCMDDocsDir := filepath.Join(tmpCacheDir, ".docs")
	err = os.MkdirAll(tmpCMDDocsDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(tmpCMDDocsDir)

	// pre-generate the plugin docs for the dummy plugin
	err = createPluginDocs(tmpCMDDocsDir, target)
	assert.NoError(t, err, "failed to create command docs for testing")

	os.Setenv("TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR", tmpCacheDir)
	defer func() {
		os.Unsetenv("TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR")
	}()

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

	// Create a sample plugin with the specified target and a remapped command
	samplePlugin := &cli.PluginInfo{
		Name:             pluginName[target],
		InstallationPath: filepath.Join(tmpCacheDir, pluginName[target]),
		Target:           target,
		Version:          "1.0.0",
		CommandMap: []plugintypes.CommandMapEntry{
			{
				// This command will be "tanzu remapped"
				SourceCommandPath:      remappedCmdName,
				DestinationCommandPath: remappedCmdName,
			},
		},
	}
	// setup the plugin
	setupDummyPlugin(t, tmpCacheDir, pluginName[target], pluginAlias[target])

	// Test constructing the plugin command tree with valid plugin binary path,
	// it should generate the command tree and alias correctly
	err = cache.constructAndAddTree(rootCmd, samplePlugin)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(cache.pluginCommands.CommandTree))

	validatePluginCommandTree(t, cache.pluginCommands, samplePlugin.InstallationPath, fmt.Sprintf(expectedPluginTree[target], samplePlugin.InstallationPath))

	// Test getting the command tree for a non-existing plugin
	nonExistingPlugin := &cli.PluginInfo{
		Name:             "non-existing-plugin",
		InstallationPath: "/path/to/non-existing-plugin",
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}
	err = cache.constructAndAddTree(rootCmd, nonExistingPlugin)
	assert.Error(t, err)
}

func TestCache_GetTree(t *testing.T) {
	// Create a sample plugin
	plugin := &cli.PluginInfo{
		Name:             pluginName[configtypes.TargetGlobal],
		InstallationPath: "/path/to/sample-plugin",
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}
	cache := getCacheWithSamplePluginCommandTree(plugin.Name, plugin.InstallationPath)

	expectedCMDTree := cache.pluginCommands.CommandTree[plugin.InstallationPath]
	// Test getting the command tree
	commandTree, err := cache.GetTree(rootCmd, plugin)
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
	nonExistingCommandTree, err := cache.GetTree(rootCmd, nonExistingPlugin)
	assert.Error(t, err)
	assert.Nil(t, nonExistingCommandTree)
}

func TestCache_DeletePluginTree(t *testing.T) {
	tmpCacheDir, err := os.MkdirTemp("", "cache")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpCacheDir)

	os.Setenv("TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR", tmpCacheDir)
	defer func() {
		os.Unsetenv("TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR")
	}()

	// Create a sample plugin
	plugin := &cli.PluginInfo{
		Name:             pluginName[configtypes.TargetGlobal],
		InstallationPath: "/path/to/sample-plugin",
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}
	cache := getCacheWithSamplePluginCommandTree(plugin.Name, plugin.InstallationPath)

	// Test deleting the command tree
	err = cache.DeletePluginTree(plugin)
	assert.NoError(t, err)
	// Make sure the cache was updated in memory
	assert.Equal(t, 0, len(cache.pluginCommands.CommandTree))

	// Test getting the command tree for a non-existing plugin
	nonExistingPlugin := &cli.PluginInfo{
		Name:             "non-existing-plugin",
		InstallationPath: "/path/to/non-existing-plugin",
		Target:           configtypes.TargetK8s,
		Version:          "1.0.0",
	}
	err = cache.DeletePluginTree(nonExistingPlugin)
	assert.NoError(t, err)
}

func TestCache_DeleteTree(t *testing.T) {
	tmpCacheDir, err := os.MkdirTemp("", "cache")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpCacheDir)

	os.Setenv("TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR", tmpCacheDir)
	defer func() {
		os.Unsetenv("TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR")
	}()

	// Create a cache file on disk to make sure it will later be removed
	err = os.WriteFile(GetPluginsCommandTreeCachePath(), []byte("commandTree: {}"), 0644)
	assert.NoError(t, err)

	// Create a sample plugin
	plugin := &cli.PluginInfo{
		Name:             pluginName[configtypes.TargetGlobal],
		InstallationPath: "/path/to/sample-plugin",
	}
	cache := getCacheWithSamplePluginCommandTree(plugin.Name, plugin.InstallationPath)

	// Test deleting the entire command tree
	err = cache.DeleteTree()
	assert.NoError(t, err)
	// Make sure the cache was updated in memory
	assert.Equal(t, 0, len(cache.pluginCommands.CommandTree))
	// Make sure the cache file was removed
	_, err = os.Stat(GetPluginsCommandTreeCachePath())
	assert.Error(t, err, "expected the cache file to be removed")
}

func getCacheWithSamplePluginCommandTree(_, pluginInstallationPath string) *cacheImpl {
	pluginCMDTree := &CommandNode{
		Subcommands: map[string]*CommandNode{
			"plugin-subcmd1": {
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

func createPluginDocs(docsDir string, target configtypes.Target) error {
	var targetStr string
	switch target {
	case configtypes.TargetGlobal:
		targetStr = "global"
	case configtypes.TargetK8s:
		targetStr = "k8s"
	case configtypes.TargetOperations:
		targetStr = "ops"
	}

	docNames := []string{
		"tanzu.md",
		"tanzu_cluster-plugin-" + targetStr + ".md",
		"tanzu_cluster-plugin-" + targetStr + "_foo1.md",
		"tanzu_cluster-plugin-" + targetStr + "_bar1.md",
		"tanzu_cluster-plugin-" + targetStr + "_foo1_foo2.md",
		// A remapped command
		"tanzu_" + remappedCmdName + ".md",
	}

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

func setupDummyPlugin(t *testing.T, dirName, pluginName, pluginAlias string) {
	pluginExeFile, err := os.OpenFile(filepath.Join(dirName, pluginName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	assert.NoError(t, err)
	defer pluginExeFile.Close()

	fmt.Fprint(pluginExeFile, fmt.Sprintf(samplePluginToGenerateAliasWithHelpCommand, pluginName, pluginAlias))
}

func validatePluginCommandTree(t *testing.T, gotPluginCommandTree *pluginCommandTree, pluginInstallationPath, expectedPluginCommandTreeYaml string) {
	// the yaml representation should marshaled and unmarshal to get rid of the "AliasProcessed" field for comparison
	gotPluginCommandTreeBytes, err := yaml.Marshal(gotPluginCommandTree.CommandTree[pluginInstallationPath])
	assert.NoError(t, err)
	gotPluginCommandTreeUnmarshaled := &CommandNode{}
	err = yaml.Unmarshal(gotPluginCommandTreeBytes, gotPluginCommandTreeUnmarshaled)
	assert.NoError(t, err)

	expectedPluginCommandTreeYaml = fmt.Sprintf("commandTree:\n%s", expectedPluginCommandTreeYaml)
	expPluginCmdTree := &pluginCommandTree{}
	err = yaml.Unmarshal([]byte(expectedPluginCommandTreeYaml), expPluginCmdTree)
	assert.NoError(t, err, "failed to unmarshal the expected plugin command tree")
	assert.Equal(t, expPluginCmdTree.CommandTree[pluginInstallationPath], gotPluginCommandTreeUnmarshaled, "the plugin command tree doesn't match with the expected ")
}
