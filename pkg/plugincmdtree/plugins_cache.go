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
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

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

var _ Cache = &cacheImpl{}

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

func (c *cacheImpl) GetTree(rootCmd *cobra.Command, plugin *cli.PluginInfo) (*CommandNode, error) {
	// If the tree does not already exist, we construct it, if it does exist constructAndAddTree is a no-op
	if err := c.constructAndAddTree(rootCmd, plugin); err != nil {
		return nil, err
	}

	pluginCmdTree, exists := c.pluginCommands.CommandTree[plugin.InstallationPath]
	if !exists {
		return nil, fmt.Errorf("failed to get the command tree for plugin '%v:%v' with target %v installed at %v", plugin.Name, plugin.Version, plugin.Target, plugin.InstallationPath)
	}

	return pluginCmdTree, nil
}

// constructAndAddTree uses the 'generate_docs' (default command that plugins support) to get the complete command chains supported.
// However, the plugin docs generated doesn't provide the information regarding the aliases of the command/sub-commands.
// So, this function uses the help command for each sub-command to extract the aliases supported and finally constructs
// the plugin command tree and adds it to cache so that the CLI can extract the command chain by parsing the user input
// against the plugin command tree.
func (c *cacheImpl) constructAndAddTree(rootCmd *cobra.Command, plugin *cli.PluginInfo) error {
	_, exists := c.pluginCommands.CommandTree[plugin.InstallationPath]
	if exists {
		return nil
	}

	pluginCmdTree, err := c.constructPluginCommandTree(rootCmd, plugin)
	if err != nil {
		return errors.Wrapf(err, "failed to generate command tree for plugin %q", plugin.Name)
	}
	c.pluginCommands.CommandTree[plugin.InstallationPath] = pluginCmdTree

	return c.savePluginCommandTree()
}

func (c *cacheImpl) DeletePluginTree(plugin *cli.PluginInfo) error {
	_, exists := c.pluginCommands.CommandTree[plugin.InstallationPath]
	if !exists {
		return nil
	}

	delete(c.pluginCommands.CommandTree, plugin.InstallationPath)

	return c.savePluginCommandTree()
}

func (c *cacheImpl) DeleteTree() error {
	c.pluginCommands.CommandTree = make(map[string]*CommandNode)
	return os.RemoveAll(GetPluginsCommandTreeCachePath())
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

func (c *cacheImpl) constructPluginCommandTree(rootCmd *cobra.Command, plugin *cli.PluginInfo) (*CommandNode, error) {
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
	numTargets := 1
	if plugin.Target == types.TargetK8s {
		// For k8s plugin, we need to generate the command tree for both the k8s level and the root level
		numTargets = 2
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// ignore non mark down files
		if filepath.Ext(file.Name()) != ".md" {
			continue
		}

		// Loop a second time for k8s targets since they are both at the root
		// level and under the k8s target
		for i := 0; i < numTargets; i++ {
			filename := strings.TrimSuffix(file.Name(), ".md")
			cmdNames := strings.Split(filename, "_")
			if i == 0 {
				// Only add the target when on the first loop.
				// If there is a second loop, it is for the root level of the k8s target
				cmdNames = adjustCmdNamesForPluginTarget(cmdNames, plugin)
			}

			var aliasArgs []string
			current := cmdTreeRoot
			for _, cmdName := range cmdNames {
				if current.Subcommands[cmdName] == nil {
					current.Subcommands[cmdName] = NewCommandNode()
				}

				current = current.Subcommands[cmdName]
				if cmdName != "tanzu" {
					// The aliasArgs are used to construct the command we will use to get the help text
					// so we can extract the aliases of command.
					aliasArgs = append(aliasArgs, cmdName)

					if cmdName == string(plugin.Target) {
						if !current.AliasProcessed {
							current.Aliases = getTargetAliases(plugin.Target)
							current.AliasProcessed = true
						}
						continue
					}

					if !current.AliasProcessed {
						// kickoff the goroutine to add the alias to the command

						// Find the command that the CLI has created
						// so that we can read its annotations.
						cmd, _, err := rootCmd.Find(aliasArgs)
						if err != nil {
							return nil, err
						}

						aliasArgsCopy := make([]string, len(aliasArgs))
						copy(aliasArgsCopy, aliasArgs)
						currentCopy := current

						aliasErrGroup.Go(func() error {
							cmdAlias, aliasErr := getPluginCommandAlias(plugin, cmd, aliasArgsCopy)
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
	}
	// Wait for all goroutines to finish or one of them to return an error
	if err := aliasErrGroup.Wait(); err != nil {
		return nil, errors.Wrap(err, "failed to generate command alias")
	}
	if cmdTreeRoot.Subcommands["tanzu"] != nil {
		return cmdTreeRoot.Subcommands["tanzu"], nil
	}

	return nil, nil
}

func getTargetAliases(target types.Target) map[string]struct{} {
	switch target {
	case types.TargetK8s:
		return map[string]struct{}{
			"k8s":        {},
			"kubernetes": {},
		}
	case types.TargetTMC:
		return map[string]struct{}{
			"tmc":             {},
			"mission-control": {},
		}
	case types.TargetOperations:
		return map[string]struct{}{
			"ops":        {},
			"operations": {},
		}
	default:
		log.V(5).Warning("Unexpected target", target)
		return nil
	}
}

// adjustCmdNamesForPluginTarget adjusts the command names to insert the plugin target
// when appropriate.  The cmdNames parameter is the list of command names that were
// extracted from one of the generated docs file; it does not contain the target yet.
func adjustCmdNamesForPluginTarget(cmdNames []string, plugin *cli.PluginInfo) []string {
	// Just the "tanzu" command
	if len(cmdNames) < 2 {
		return cmdNames
	}

	// No changes required since the global target means the plugin
	// is directly at the root level
	if plugin.Target == types.TargetGlobal {
		return cmdNames
	}

	// For remapped commands, we don't add the target
	for _, cmdMap := range plugin.CommandMap {
		// If the cmdNames (excluding the "tanzu" command) starts with the destination command path
		// it means we are dealing with a remapped command and we should not add the target
		index := 1 // Start at 1 to skip the "tanzu" command
		isRemapped := true
		for _, destCmd := range strings.Split(cmdMap.DestinationCommandPath, " ") {
			if strings.TrimSpace(destCmd) == "" {
				continue
			}

			if len(cmdNames) < index+1 || cmdNames[index] != destCmd {
				// Not a remapped command
				isRemapped = false
				break
			}
			index++
		}
		if isRemapped {
			return cmdNames
		}
	}

	// Insert the target as the second element (after "tanzu") in the command names
	return append([]string{cmdNames[0], string(plugin.Target)}, cmdNames[1:]...)
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

func getPluginCommandAlias(plugin *cli.PluginInfo, cmd *cobra.Command, aliasArgs []string) (map[string]struct{}, error) {
	// Drop the the target if there is one since it is not needed when calling the plugin directly
	if len(aliasArgs) > 0 && types.IsValidTarget(aliasArgs[0], false, false) {
		aliasArgs = aliasArgs[1:]
	}

	// Drop the next element of aliasArgs since it is either:
	// - the plugin name, which is not needed when calling the plugin directly
	// - the remapped command name, which will be added back through the command source path annotation
	if len(aliasArgs) > 0 {
		aliasArgs = aliasArgs[1:]
	}

	// Handle any remapped commands by adding the command source path before
	// calling the plugin directly.
	cmdSrcPath := cmd.Annotations[common.AnnotationForCmdSrcPath]
	if cmdSrcPath != "" {
		aliasArgs = append(strings.Split(cmdSrcPath, " "), aliasArgs...)
	}
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
