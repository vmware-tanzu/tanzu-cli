// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugincmdtree

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
)

const pluginsCommandTreeDir = "plugins_command_tree"

type pluginCommandTree struct {
	CommandTree map[string]*CommandNode `yaml:"commandTree" json:"commandTree"`
}

func getPluginsCommandTreeCacheDir() string {
	// NOTE: TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR is only for test purpose
	customCommandTreeCacheDirForTest := os.Getenv("TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR")
	if customCommandTreeCacheDirForTest != "" {
		return customCommandTreeCacheDirForTest
	}
	return filepath.Join(common.DefaultCacheDir, pluginsCommandTreeDir)
}
func GetPluginsCommandTreeCachePath() string {
	return filepath.Join(getPluginsCommandTreeCacheDir(), "command_tree.yaml")
}

func getPluginsDocsCachePath() string {
	return filepath.Join(getPluginsCommandTreeCacheDir(), ".docs")
}

type cacheImpl struct {
	pluginCommands      *pluginCommandTree
	pluginDocsGenerator func(plugin *cli.PluginInfo) error
}

// NewCache create a cache for plugin command tree
func NewCache() (Cache, error) {
	pct, err := getPluginCommandTree()
	if err != nil {
		return nil, err
	}
	// Cache Implementation uses the 'generate_docs' (default command that plugins support) to construct the complete command chains supported.
	// However, the plugin docs generated doesn't provide the information regarding the aliases of the command/sub-commands.
	// So, it would use the help command for each sub-command to extract the aliases supported and finally construct
	// the plugin command tree and adds it to cache so that telemetry client(collector) can extract the command chain by parsing the user input
	// against the plugin command tree.
	// Note: A possible future enhancement would be to teach plugins through plugin-runtime library to provide the plugin command tree with
	// more exhaustive details(ex: flags(also whether they are maskable), sub-commands, aliases, positions arguments etc.)
	return &cacheImpl{
		pluginCommands:      pct,
		pluginDocsGenerator: generatePluginDocs,
	}, nil
}

func (c *cacheImpl) GetTree(plugin *cli.PluginInfo) (*CommandNode, error) {
	// This is just a safety net to generate the command tree if we missed/failed to generate the plugin command tree during plugin install
	// If the plugin command tree exists, then ConstructAndAddTree is a no-op
	if err := c.ConstructAndAddTree(plugin); err != nil {
		return nil, err
	}

	pluginCmdTree, exists := c.pluginCommands.CommandTree[plugin.InstallationPath]
	if !exists {
		return nil, fmt.Errorf("failed to get the command tree for plugin '%v:%v' with target %v installled at %v", plugin.Name, plugin.Version, plugin.Target, plugin.InstallationPath)
	}

	return pluginCmdTree, nil
}

// ConstructAndAddTree uses the 'generate_docs' (default command that plugins support) to get the complete command chains supported.
// However, the plugin docs generated doesn't provide the information regarding the aliases of the command/sub-commands.
// So, it would use the help command for each sub-command to extract the aliases supported and finally construct
// the plugin command tree and adds it to cache so that CLI can extract the command chain by parsing the user input
// against the plugin command tree.
func (c *cacheImpl) ConstructAndAddTree(plugin *cli.PluginInfo) error {
	_, exists := c.pluginCommands.CommandTree[plugin.InstallationPath]
	if exists {
		return nil
	}

	pluginCmdTree, err := c.constructPluginCommandTree(plugin)
	if err != nil {
		return errors.Wrapf(err, "failed to generate command tree for plugin %q", plugin.Name)
	}
	c.pluginCommands.CommandTree[plugin.InstallationPath] = pluginCmdTree

	return c.savePluginCommandTree()
}
func (c *cacheImpl) DeleteTree(plugin *cli.PluginInfo) error {
	_, exists := c.pluginCommands.CommandTree[plugin.InstallationPath]
	if !exists {
		return nil
	}

	delete(c.pluginCommands.CommandTree, plugin.InstallationPath)

	return c.savePluginCommandTree()
}

func (c *cacheImpl) savePluginCommandTree() error {
	data, err := yaml.Marshal(c.pluginCommands)
	if err != nil {
		return errors.Wrap(err, "failed to marshal plugin command tree")
	}
	err = os.WriteFile(GetPluginsCommandTreeCachePath(), data, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write the command tree to %q file ", GetPluginsCommandTreeCachePath())
	}
	return nil
}

func (c *cacheImpl) constructPluginCommandTree(plugin *cli.PluginInfo) (*CommandNode, error) {
	if err := c.pluginDocsGenerator(plugin); err != nil {
		return nil, errors.Wrapf(err, "failed to generate docs for the plugin %q", plugin.Name)
	}
	// construct the command tree
	cmdTreeRoot := NewCommandNode()

	docsDir := getPluginsDocsCachePath()
	files, err := os.ReadDir(docsDir)
	if err != nil {
		return nil, errors.Wrapf(err, "error while reading local plugin command tree directory")
	}
	var aliasErrGroup errgroup.Group
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// ignore non mark down files
		if filepath.Ext(file.Name()) != ".md" {
			continue
		}

		filename := strings.TrimSuffix(file.Name(), ".md")
		cmdNames := strings.Split(filename, "_")

		var aliasArgs []string
		current := cmdTreeRoot
		for _, cmdName := range cmdNames {
			if current.Subcommands[cmdName] == nil {
				current.Subcommands[cmdName] = NewCommandNode()
			}

			current = current.Subcommands[cmdName]
			if cmdName != "tanzu" && cmdName != plugin.Name {
				aliasArgs = append(aliasArgs, cmdName)
				aliasArgsCopy := make([]string, len(aliasArgs))
				copy(aliasArgsCopy, aliasArgs)
				currentCopy := current
				if !currentCopy.AliasProcessed {
					// kickoff the goroutine to add the alias to the command
					aliasErrGroup.Go(func() error {
						cmdAlias, aliasErr := getPluginCommandAlias(plugin, aliasArgsCopy)
						if aliasErr != nil {
							return aliasErr
						}
						currentCopy.Aliases = cmdAlias
						return nil
					})
					currentCopy.AliasProcessed = true
				}
			}
		}
	}
	// Wait for all goroutines to finish or one of them to return an error
	if err := aliasErrGroup.Wait(); err != nil {
		return nil, errors.Wrap(err, "failed to generate command alias")
	}
	if cmdTreeRoot.Subcommands["tanzu"] != nil && cmdTreeRoot.Subcommands["tanzu"].Subcommands[plugin.Name] != nil {
		return cmdTreeRoot.Subcommands["tanzu"].Subcommands[plugin.Name], nil
	}

	return nil, nil
}

func getPluginCommandTree() (*pluginCommandTree, error) {
	b, err := os.ReadFile(GetPluginsCommandTreeCachePath())
	if err != nil {
		return &pluginCommandTree{
			CommandTree: make(map[string]*CommandNode),
		}, nil
	}

	var ctr pluginCommandTree
	err = yaml.Unmarshal(b, &ctr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the  plugin command tree ")
	}

	return &ctr, nil
}

func generatePluginDocs(plugin *cli.PluginInfo) error {
	docsDir := getPluginsDocsCachePath()
	_ = os.RemoveAll(docsDir)
	_ = os.MkdirAll(docsDir, 0755)

	args := []string{"generate-docs", "--docs-dir", docsDir}

	runner := cli.NewRunner(plugin.Name, plugin.InstallationPath, args)
	ctx := context.Background()
	if _, _, err := runner.RunOutput(ctx); err != nil {
		return err
	}
	return nil
}

func getPluginCommandAlias(plugin *cli.PluginInfo, aliasArgs []string) (map[string]struct{}, error) {
	aliasArgs = append(aliasArgs, "-h")
	runner := cli.NewRunner(plugin.Name, plugin.InstallationPath, aliasArgs)
	ctx := context.Background()
	stdout, _, err := runner.RunOutput(ctx)
	if err != nil {
		return nil, err
	}

	aMap := make(map[string]struct{})
	aliases := extractAliases(stdout)
	for _, alias := range aliases {
		aMap[strings.TrimSpace(alias)] = struct{}{}
	}

	return aMap, nil
}

func extractAliases(output string) []string {
	re := regexp.MustCompile(`Aliases:\s*(.*)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		aliases := strings.Split(matches[1], ",")
		return aliases
	}
	return nil
}
