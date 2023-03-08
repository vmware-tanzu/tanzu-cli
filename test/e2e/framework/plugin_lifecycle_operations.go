// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"
	"strings"

	"encoding/json"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// PluginBasicOps helps to perform the plugin command operations
type PluginBasicOps interface {
	// ListPlugins lists all plugins by running 'tanzu plugin list' command
	ListPlugins() ([]PluginListInfo, error)
	// SearchPlugins searches all plugins for given filter (keyword|regex) by running 'tanzu plugin search' command
	SearchPlugins(filter string) ([]PluginListInfo, error)
	// InstallPlugin installs given plugin
	InstallPlugin(pluginName string) error
	// UninstallPlugin uninstalls given plugin
	UninstallPlugin(pluginName string) error
	// ExecuteSubCommand executes specific plugin sub-command
	ExecuteSubCommand(pluginWithSubCommand string) (string, error)
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

func (po *pluginCmdOps) SearchPlugins(filter string) ([]PluginListInfo, error) {
	searchPluginCmdWithOptions := SearchPluginsCmd
	if len(strings.TrimSpace(filter)) > 0 {
		searchPluginCmdWithOptions = searchPluginCmdWithOptions + " " + strings.TrimSpace(filter)
	}
	searchPluginCmdWithOptions += JSONOutput
	out, stdErr, err := po.cmdExe.Exec(searchPluginCmdWithOptions)

	if err != nil {
		log.Errorf("error while executing plugin search command:'%s', error:'%s' stdErr:'%s'", searchPluginCmdWithOptions, err.Error(), stdErr.String())
		return nil, err
	}
	jsonStr := out.String()
	var list []PluginListInfo
	err = json.Unmarshal([]byte(jsonStr), &list)
	if err != nil {
		log.Errorf("failed to construct node from plugin search output:'%s' error:'%s' ", jsonStr, err.Error())
		return nil, errors.Wrapf(err, "failed to construct json node from search output:'%s'", jsonStr)
	}
	return list, nil
}

func (po *pluginCmdOps) InstallPlugin(pluginName string) error {
	installPluginCmd := fmt.Sprintf(InstallPLuginCmd, pluginName)
	_, stdErr, err := po.cmdExe.Exec(installPluginCmd)
	if err != nil {
		log.Errorf("error while installing the plugin: %s, error: %s stdErr: %s", pluginName, err.Error(), stdErr.String())
	}
	return err
}

func (po *pluginCmdOps) UninstallPlugin(pluginName string) error {
	uninstallPluginCmd := fmt.Sprintf(UninstallPLuginCmd, pluginName)
	_, stdErr, err := po.cmdExe.Exec(uninstallPluginCmd)
	if err != nil {
		log.Errorf("error while uninstalling the plugin: %s, error: %s, stdErr: %s", pluginName, err.Error(), stdErr.String())
	}
	return err
}

func (po *pluginCmdOps) ExecuteSubCommand(pluginWithSubCommand string) (string, error) {
	pluginCmdWithSubCommand := fmt.Sprintf(PluginSubCommand, pluginWithSubCommand)
	stdOut, stdErr, err := po.cmdExe.Exec(pluginCmdWithSubCommand)
	if err != nil {
		log.Errorf("error while running the plugin command: %s, error: %s, stdErr: %s", pluginCmdWithSubCommand, err.Error(), stdErr.String())
		return stdOut.String(), errors.Wrap(err, stdErr.String())
	}
	return stdOut.String(), nil
}
