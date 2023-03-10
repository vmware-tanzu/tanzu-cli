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
	ListPlugins() ([]PluginInfo, error)
	// SearchPlugins searches all plugins for given filter (keyword|regex) by running 'tanzu plugin search' command
	SearchPlugins(filter string) ([]PluginInfo, error)
	// InstallPlugin installs given plugin and flags
	InstallPlugin(pluginName, target, versions string) error
	// DescribePlugin describes given plugin and flags
	DescribePlugin(pluginName, target string) (string, error)
	// UninstallPlugin uninstalls/deletes given plugin
	UninstallPlugin(pluginName, target string) error
	// DeletePlugin deletes/uninstalls given plugin
	DeletePlugin(pluginName, target string) error
	// ExecuteSubCommand executes specific plugin sub-command
	ExecuteSubCommand(pluginWithSubCommand string) (string, error)
	// CleanPlugins executes the plugin clean command to delete all existing plugins
	CleanPlugins() error
}

// PluginSourceOps helps 'plugin source' commands
type PluginSourceOps interface {
	// AddPluginDiscoverySource adds plugin discovery source, and returns stdOut and error info
	AddPluginDiscoverySource(discoveryOpts *DiscoveryOptions) (string, error)

	// UpdatePluginDiscoverySource updates plugin discovery source, and returns stdOut and error info
	UpdatePluginDiscoverySource(discoveryOpts *DiscoveryOptions) (string, error)

	// DeletePluginDiscoverySource removes the plugin discovery source, and returns stdOut and error info
	DeletePluginDiscoverySource(pluginSourceName string) (string, error)

	// ListPluginSources returns all available plugin discovery sources
	ListPluginSources() ([]PluginSourceInfo, error)
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

func (po *pluginCmdOps) UpdatePluginDiscoverySource(discoveryOpts *DiscoveryOptions) (string, error) {
	addCmd := fmt.Sprintf(UpdatePluginSource, discoveryOpts.Name, discoveryOpts.SourceType, discoveryOpts.URI)
	out, _, err := po.cmdExe.Exec(addCmd)
	return out.String(), err
}

func (po *pluginCmdOps) ListPluginSources() ([]PluginSourceInfo, error) {
	stdOut, stdErr, err := po.cmdExe.Exec(ListPluginSources)
	if err != nil {
		log.Errorf("error while executing plugin source list command:'%s', error:'%s' stdErr:'%s' stdOut:'%s'", ListPluginSources, err.Error(), stdErr.String(), stdOut.String())
		return nil, err
	}
	jsonStr := stdOut.String()
	var list []PluginSourceInfo
	err = json.Unmarshal([]byte(jsonStr), &list)
	if err != nil {
		log.Errorf("failed to construct node from plugin source list output:'%s' error:'%s' ", jsonStr, err.Error())
		return nil, errors.Wrapf(err, "failed to construct json node from plugin source list output:'%s'", jsonStr)
	}
	return list, nil
}

func (po *pluginCmdOps) DeletePluginDiscoverySource(pluginSourceName string) (string, error) {
	deleteCmd := fmt.Sprintf(DeletePluginSource, pluginSourceName)
	out, _, err := po.cmdExe.Exec(deleteCmd)
	return out.String(), err
}

func (po *pluginCmdOps) ListPlugins() ([]PluginInfo, error) {
	out, _, err := po.cmdExe.Exec(ListPluginsCmdInJSON)
	if err != nil {
		return nil, err
	}
	jsonStr := out.String()
	var list []PluginInfo
	err = json.Unmarshal([]byte(jsonStr), &list)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct json node from config get output")
	}
	return list, nil
}

func (po *pluginCmdOps) SearchPlugins(filter string) ([]PluginInfo, error) {
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
	var list []PluginInfo
	err = json.Unmarshal([]byte(jsonStr), &list)
	if err != nil {
		log.Errorf("failed to construct node from plugin search output:'%s' error:'%s' ", jsonStr, err.Error())
		return nil, errors.Wrapf(err, "failed to construct json node from search output:'%s'", jsonStr)
	}
	return list, nil
}

func (po *pluginCmdOps) InstallPlugin(pluginName, target, versions string) error {
	installPluginCmd := fmt.Sprintf(InstallPluginCmd, pluginName)
	if len(strings.TrimSpace(target)) > 0 {
		installPluginCmd += " --target " + target
	}
	if len(strings.TrimSpace(versions)) > 0 {
		installPluginCmd += " --version " + versions
	}
	_, stdErr, err := po.cmdExe.Exec(installPluginCmd)
	if err != nil {
		log.Errorf("error while installing the plugin: %s, error: %s stdErr: %s", pluginName, err.Error(), stdErr.String())
	}
	return err
}

func (po *pluginCmdOps) DescribePlugin(pluginName, target string) (string, error) {
	installPluginCmd := fmt.Sprintf(DescribePluginCmd, pluginName)
	if len(strings.TrimSpace(target)) > 0 {
		installPluginCmd += " --target " + target
	}

	stdOut, stdErr, err := po.cmdExe.Exec(installPluginCmd)
	if err != nil {
		log.Errorf("error for plugin describe command: %s, error: %s stdErr: %s", pluginName, err.Error(), stdErr.String())
	}
	return stdOut.String(), err
}

func (po *pluginCmdOps) DeletePlugin(pluginName, target string) error {
	return po.UninstallPlugin(pluginName, target)
}

func (po *pluginCmdOps) UninstallPlugin(pluginName, target string) error {
	uninstallPluginCmd := fmt.Sprintf(UninstallPLuginCmd, pluginName)
	if len(strings.TrimSpace(target)) > 0 {
		uninstallPluginCmd += " --target " + target
	}
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

func (po *pluginCmdOps) CleanPlugins() error {
	_, stdErr, err := po.cmdExe.Exec(CleanPluginsCmd)
	if err != nil {
		log.Errorf("error for plugin clean command: %s, error: %s, stdErr: %s", CleanPluginsCmd, err.Error(), stdErr.String())
	}
	return err
}
