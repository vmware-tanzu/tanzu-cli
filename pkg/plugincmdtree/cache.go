// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugincmdtree provides functionality for constructing and maintaining the plugin command trees
package plugincmdtree

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// Cache is the local cache for storing and accessing
// command trees of different plugins
//
//go:generate counterfeiter -o ../fakes/plugin_cmd_tree_cache_fake.go --fake-name CommandTreeCache . Cache
type Cache interface {
	// GetTree returns the plugin command tree
	// If the plugin command tree doesn't exist, it constructs and adds the command tree to the cache
	// and then returns the plugin command tree, otherwise it returns an error
	GetTree(rootCmd *cobra.Command, plugin *cli.PluginInfo) (*CommandNode, error)
	// DeletePluginTree deletes the plugin command tree from the cache
	DeletePluginTree(plugin *cli.PluginInfo) error
	// DeleteTree deletes the entire command tree from the cache
	DeleteTree() error
}

type CommandNode struct {
	Subcommands    map[string]*CommandNode `yaml:"subcommands" json:"subcommands"`
	Aliases        map[string]struct{}     `yaml:"aliases" json:"aliases"`
	AliasProcessed bool                    `yaml:"-" json:"-"`
}

func NewCommandNode() *CommandNode {
	return &CommandNode{
		Subcommands: make(map[string]*CommandNode),
		Aliases:     make(map[string]struct{}),
	}
}
