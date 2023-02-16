// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"

	"encoding/json"

	"github.com/pkg/errors"
)

// PluginBasicOps helps to perform the plugin command operations
type PluginBasicOps interface {
	// ListPlugins lists all plugins by running 'tanzu plugin list' command
	ListPlugins() ([]PluginListInfo, error)
}

// PluginSourceOps helps 'plugin source' commands
type PluginSourceOps interface {
	// AddPluginDiscoverySource adds plugin discovery source, and returns stdOut and error info
	AddPluginDiscoverySource(discoveryOpts *DiscoveryOptions) (string, error)

	// DeletePluginDiscoverySource removes the plugin discovery source, and returns stdOut and error info
	DeletePluginDiscoverySource(discoveryOpts *DiscoveryOptions) (string, error)
}

// PluginCmdOps helps to perform the plugin and its sub-commands lifecycle operations
type PluginCmdOps interface {
	PluginBasicOps
	PluginSourceOps
}

type DiscoveryOptions struct {
	Name       string
	SourceType string
	URI        string
}

type pluginCmdOps struct {
	cmdExe CmdOps
	PluginCmdOps
}

func NewPluginLifecycleOps() PluginCmdOps {
	return &pluginCmdOps{
		cmdExe: NewCmdOps(),
	}
}

func (po *pluginCmdOps) AddPluginDiscoverySource(discoveryOpts *DiscoveryOptions) (string, error) {
	addCmd := fmt.Sprintf(AddPluginSource, discoveryOpts.Name, discoveryOpts.SourceType, discoveryOpts.URI)
	out, _, err := po.cmdExe.Exec(addCmd)
	return out.String(), err
}

func (po *pluginCmdOps) DeletePluginDiscoverySource(discoveryOpts *DiscoveryOptions) (string, error) {
	deleteCmd := fmt.Sprintf(DeletePluginSource, discoveryOpts.Name)
	out, _, err := po.cmdExe.Exec(deleteCmd)
	return out.String(), err
}

func (po *pluginCmdOps) ListPlugins() ([]PluginListInfo, error) {
	out, _, err := po.cmdExe.Exec(ListPluginsCmdInJSON)
	if err != nil {
		return nil, err
	}
	jsonStr := out.String()
	var list []PluginListInfo
	err = json.Unmarshal([]byte(jsonStr), &list)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct json node from config get output")
	}
	return list, nil
}
