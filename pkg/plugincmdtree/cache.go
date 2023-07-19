// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugincmdtree provides functionality for constructing and maintaining the plugin command trees
package plugincmdtree

import "github.com/vmware-tanzu/tanzu-cli/pkg/cli"

// Cache is the local cache for storing and accessing
// command trees of different plugins
//
//go:generate counterfeiter -o ../fakes/plugin_cmd_tree_cache_fake.go --fake-name CommandTreeCache . Cache
type Cache interface {
	// GetTree returns the plugin command tree
	// If the plugin command tree doesn't exist, it constructs and adds the command tree to the cache
	// and then returns the plugin command tree, otherwise it returns error
	GetTree(plugin *cli.PluginInfo) (*CommandNode, error)
	// ConstructAndAddTree constructs and adds the plugin command tree to the cache
	// If the plugin command tree already exists, it returns success immediately
	ConstructAndAddTree(plugin *cli.PluginInfo) error
	// DeleteTree deletes the plugin command tree from the cache
	DeleteTree(plugin *cli.PluginInfo) error
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
