// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// PluginBasicOps helps to perform the plugin command operations
type PluginBasicOps interface {
	// ListPlugins lists all plugins by running 'tanzu plugin list' command
	ListPlugins() ([]*PluginInfo, error)
	// ListInstalledPlugins lists all installed plugins
	ListInstalledPlugins() ([]*PluginInfo, error)
	// ListPluginsForGivenContext lists all plugins for a given context and either installed only or all
	ListPluginsForGivenContext(context string, installedOnly bool) ([]*PluginInfo, error)
	// SearchPlugins searches all plugins for given filter (keyword|regex) by running 'tanzu plugin search' command
	SearchPlugins(filter string) ([]*PluginInfo, error)
	// InstallPlugin installs given plugin and flags
	InstallPlugin(pluginName, target, versions string) error
	// Sync performs sync operation
	Sync() (string, error)
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
	ListPluginSources() ([]*PluginSourceInfo, error)
}

type PluginGroupOps interface {
	// SearchPluginGroups performs plugin group search
	// input: flagsWithValues - flags and values if any
	SearchPluginGroups(flagsWithValues string) ([]*PluginGroup, error)

	// InstallPluginsFromGroup a plugin or all plugins from the given plugin group
	InstallPluginsFromGroup(pluginNameORAll, groupName string) error
}

// PluginCmdOps helps to perform the plugin and its sub-commands lifecycle operations
type PluginCmdOps interface {
	PluginBasicOps
	PluginSourceOps
	PluginGroupOps
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

func (po *pluginCmdOps) ListPluginSources() ([]*PluginSourceInfo, error) {
	return ExecuteCmdAndBuildJSONOutput[PluginSourceInfo](po.cmdExe, ListPluginSourcesWithJSONOutputFlag)
}

func (po *pluginCmdOps) DeletePluginDiscoverySource(pluginSourceName string) (string, error) {
	deleteCmd := fmt.Sprintf(DeletePluginSource, pluginSourceName)
	out, stdErr, err := po.cmdExe.Exec(deleteCmd)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, deleteCmd, err.Error(), stdErr.String(), out.String())
	}
	return out.String(), err
}

func (po *pluginCmdOps) ListPlugins() ([]*PluginInfo, error) {
	return ExecuteCmdAndBuildJSONOutput[PluginInfo](po.cmdExe, ListPluginsCmdWithJSONOutputFlag)
}

func (po *pluginCmdOps) ListInstalledPlugins() ([]*PluginInfo, error) {
	plugins, err := ExecuteCmdAndBuildJSONOutput[PluginInfo](po.cmdExe, ListPluginsCmdWithJSONOutputFlag)
	installedPlugins := make([]*PluginInfo, 0)
	for i := range plugins {
		if plugins[i].Status == Installed {
			installedPlugins = append(installedPlugins, plugins[i])
		}
	}
	return installedPlugins, err
}

func (po *pluginCmdOps) ListPluginsForGivenContext(context string, installedOnly bool) ([]*PluginInfo, error) {
	plugins, err := ExecuteCmdAndBuildJSONOutput[PluginInfo](po.cmdExe, ListPluginsCmdWithJSONOutputFlag)
	contextSpecificPlugins := make([]*PluginInfo, 0)
	for i := range plugins {
		if plugins[i].Context == context {
			if installedOnly {
				if plugins[i].Status == Installed {
					contextSpecificPlugins = append(contextSpecificPlugins, plugins[i])
				}
			} else {
				contextSpecificPlugins = append(contextSpecificPlugins, plugins[i])
			}
		}
	}
	return contextSpecificPlugins, err
}

func (po *pluginCmdOps) Sync() (string, error) {
	out, stdErr, err := po.cmdExe.Exec(pluginSyncCmd)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, pluginSyncCmd, err.Error(), stdErr.String(), out.String())
	}
	return out.String(), err
}

func (po *pluginCmdOps) SearchPlugins(filter string) ([]*PluginInfo, error) {
	searchPluginCmdWithOptions := SearchPluginsCmd
	if len(strings.TrimSpace(filter)) > 0 {
		searchPluginCmdWithOptions = searchPluginCmdWithOptions + " " + strings.TrimSpace(filter)
	}
	result, err := ExecuteCmdAndBuildJSONOutput[PluginSearch](po.cmdExe, searchPluginCmdWithOptions+JSONOutput)
	if err != nil {
		return nil, err
	}
	// Convert from PluginSearch to PluginInfo
	var plugins []*PluginInfo
	for _, p := range result {
		plugins = append(plugins, &PluginInfo{
			Name:        p.Name,
			Description: p.Description,
			Target:      p.Target,
			Version:     p.Latest,
		})
	}
	return plugins, nil
}

func (po *pluginCmdOps) SearchPluginGroups(flagsWithValues string) ([]*PluginGroup, error) {
	searchPluginGroupCmdWithOptions := SearchPluginGroupsCmd
	if len(strings.TrimSpace(flagsWithValues)) > 0 {
		searchPluginGroupCmdWithOptions = searchPluginGroupCmdWithOptions + " " + strings.TrimSpace(flagsWithValues)
	}
	return ExecuteCmdAndBuildJSONOutput[PluginGroup](po.cmdExe, searchPluginGroupCmdWithOptions+JSONOutput)
}

func (po *pluginCmdOps) InstallPlugin(pluginName, target, versions string) error {
	installPluginCmd := fmt.Sprintf(InstallPluginCmd, pluginName)
	if len(strings.TrimSpace(target)) > 0 {
		installPluginCmd += " --target " + target
	}
	if len(strings.TrimSpace(versions)) > 0 {
		installPluginCmd += " --version " + versions
	}
	out, stdErr, err := po.cmdExe.Exec(installPluginCmd)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, installPluginCmd, err.Error(), stdErr.String(), out.String())
	}
	return err
}

func (po *pluginCmdOps) InstallPluginsFromGroup(pluginNameORAll, groupName string) error {
	var installPluginCmd string
	if len(pluginNameORAll) > 0 {
		installPluginCmd = fmt.Sprintf(InstallPluginFromGroupCmd, pluginNameORAll, groupName)
	} else {
		installPluginCmd = fmt.Sprintf(InstallAllPluginsFromGroupCmd, groupName)
	}
	out, stdErr, err := po.cmdExe.Exec(installPluginCmd)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, installPluginCmd, err.Error(), stdErr.String(), out.String())
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
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, installPluginCmd, err.Error(), stdErr.String(), stdOut.String())
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
	out, stdErr, err := po.cmdExe.Exec(uninstallPluginCmd)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, uninstallPluginCmd, err.Error(), stdErr.String(), out.String())
	}
	return err
}

func (po *pluginCmdOps) ExecuteSubCommand(pluginWithSubCommand string) (string, error) {
	pluginCmdWithSubCommand := fmt.Sprintf(PluginSubCommand, pluginWithSubCommand)
	stdOut, stdErr, err := po.cmdExe.Exec(pluginCmdWithSubCommand)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, pluginCmdWithSubCommand, err.Error(), stdErr.String(), stdOut.String())
		return stdOut.String(), errors.Wrap(err, stdErr.String())
	}
	return stdOut.String(), nil
}

func (po *pluginCmdOps) CleanPlugins() error {
	out, stdErr, err := po.cmdExe.Exec(CleanPluginsCmd)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, CleanPluginsCmd, err.Error(), stdErr.String(), out.String())
	}
	return err
}
