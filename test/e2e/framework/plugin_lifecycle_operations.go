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
	// ListPlugins execytes 'tanzu plugin list' command and
	// returns the output, stdOut, stdErr and error
	ListPlugins(opts ...E2EOption) ([]*PluginInfo, string, string, error)
	// ListInstalledPlugins lists all installed plugins
	ListInstalledPlugins(opts ...E2EOption) ([]*PluginInfo, error)
	// ListRecommendedPluginsFromActiveContext lists all recommended plugins for the active context
	ListRecommendedPluginsFromActiveContext(installedOnly bool, opts ...E2EOption) ([]*PluginInfo, error)
	// SearchPlugins searches all plugins for given filter (keyword|regex) by running 'tanzu plugin search' command
	SearchPlugins(filter string, opts ...E2EOption) ([]*PluginInfo, string, string, error)
	// InstallPlugin installs given plugin and flags
	InstallPlugin(pluginName, target, versions string, opts ...E2EOption) (stdOut, stdErr string, err error)
	// Sync performs sync operation and returns stdOut, stdErr and error
	Sync(opts ...E2EOption) (string, string, error)
	// DescribePlugin describes given plugin and flags, returns the plugin description as PluginDescribe
	DescribePlugin(pluginName, target string, opts ...E2EOption) ([]*PluginDescribe, error)
	// DescribePluginLegacy describes given plugin and flags, returns plugin description in string format
	DescribePluginLegacy(pluginName, target string, opts ...E2EOption) (string, error)
	// UninstallPlugin uninstalls/deletes given plugin
	UninstallPlugin(pluginName, target string, opts ...E2EOption) error
	// DeletePlugin deletes/uninstalls given plugin
	DeletePlugin(pluginName, target string, opts ...E2EOption) error
	// ExecuteSubCommand executes specific plugin sub-command
	ExecuteSubCommand(pluginWithSubCommand string, opts ...E2EOption) (string, error)
	// CleanPlugins executes the plugin clean command to delete all existing plugins
	CleanPlugins(opts ...E2EOption) error
	// RunPluginCmd runs plugin command with provided options
	RunPluginCmd(options string, opts ...E2EOption) (string, string, error)
}

// PluginSourceOps helps 'plugin source' commands
type PluginSourceOps interface {
	// UpdatePluginDiscoverySource updates plugin discovery source, and returns stdOut and error info
	UpdatePluginDiscoverySource(discoveryOpts *DiscoveryOptions, opts ...E2EOption) (string, error)

	// DeletePluginDiscoverySource removes the plugin discovery source, and returns stdOut and error info
	DeletePluginDiscoverySource(pluginSourceName string, opts ...E2EOption) (string, error)

	// ListPluginSources returns all available plugin discovery sources
	ListPluginSources(opts ...E2EOption) ([]*PluginSourceInfo, error)

	// InitPluginDiscoverySource initializes the plugin source to its default value, and returns stdOut and error info
	InitPluginDiscoverySource(opts ...E2EOption) (string, error)
}

type PluginGroupOps interface {
	// SearchPluginGroups performs plugin group search
	// input: flagsWithValues - flags and values if any
	SearchPluginGroups(flagsWithValues string, opts ...E2EOption) ([]*PluginGroup, error)

	// GetPluginGroup performs plugin group get
	// input: flagsWithValues - flags and values if any
	GetPluginGroup(groupName string, flagsWithValues string, opts ...E2EOption) ([]*PluginGroupGet, error)

	// InstallPluginsFromGroup a plugin or all plugins from the given plugin group
	InstallPluginsFromGroup(pluginNameORAll, groupName string, opts ...E2EOption) (stdout string, stdErr string, err error)
}

type PluginDownloadAndUploadOps interface {
	// DownloadPluginBundle downloads the plugin inventory and plugin bundles to local tar file
	DownloadPluginBundle(image string, groups []string, toTar string, opts ...E2EOption) error

	// UploadPluginBundle performs the uploading plugin bundle to the remote repository
	// Based on the remote repository status, it setups a new discovery source endpoint
	// or merges the additional plugins in the bundle to the existing discovery source
	UploadPluginBundle(toRepo string, tar string, opts ...E2EOption) error
}

// PluginCmdOps helps to perform the plugin and its sub-commands lifecycle operations
type PluginCmdOps interface {
	PluginBasicOps
	PluginSourceOps
	PluginGroupOps
	PluginDownloadAndUploadOps
}

type DiscoveryOptions struct {
	Name       string
	SourceType string
	URI        string
}

type pluginCmdOps struct {
	cmdExe CmdOps
}

func NewPluginLifecycleOps() PluginCmdOps {
	return &pluginCmdOps{
		cmdExe: NewCmdOps(),
	}
}

func (po *pluginCmdOps) UpdatePluginDiscoverySource(discoveryOpts *DiscoveryOptions, opts ...E2EOption) (string, error) {
	updateCmd := fmt.Sprintf(UpdatePluginSource, "%s", discoveryOpts.Name, discoveryOpts.URI)
	out, _, err := po.cmdExe.TanzuCmdExec(updateCmd, opts...)
	return out.String(), err
}

func (po *pluginCmdOps) ListPluginSources(opts ...E2EOption) ([]*PluginSourceInfo, error) {
	output, _, _, err := ExecuteCmdAndBuildJSONOutput[PluginSourceInfo](po.cmdExe, ListPluginSourcesWithJSONOutputFlag, opts...)
	return output, err
}

func (po *pluginCmdOps) DeletePluginDiscoverySource(pluginSourceName string, opts ...E2EOption) (string, error) {
	deleteCmd := fmt.Sprintf(DeletePluginSource, "%s", pluginSourceName)
	out, stdErr, err := po.cmdExe.TanzuCmdExec(deleteCmd, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, deleteCmd, err.Error(), stdErr.String(), out.String())
	}
	return out.String(), err
}

func (po *pluginCmdOps) InitPluginDiscoverySource(opts ...E2EOption) (string, error) {
	initCmd := fmt.Sprintf(InitPluginDiscoverySource, "%s")
	out, stdErr, err := po.cmdExe.TanzuCmdExec(initCmd, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, initCmd, err.Error(), stdErr.String(), out.String())
	}
	return out.String(), err
}

func (po *pluginCmdOps) ListPlugins(opts ...E2EOption) (output []*PluginInfo, stdout string, stdErr string, err error) {
	output, stdOut, stdErr, err := ExecuteCmdAndBuildJSONOutput[PluginInfo](po.cmdExe, ListPluginsCmdWithJSONOutputFlag, opts...)
	return output, stdOut, stdErr, err
}

func (po *pluginCmdOps) ListInstalledPlugins(opts ...E2EOption) ([]*PluginInfo, error) {
	plugins, _, _, err := ExecuteCmdAndBuildJSONOutput[PluginInfo](po.cmdExe, ListPluginsCmdWithJSONOutputFlag, opts...)
	installedPlugins := make([]*PluginInfo, 0)
	for i := range plugins {
		if plugins[i].Status == Installed {
			installedPlugins = append(installedPlugins, plugins[i])
		}
	}
	return installedPlugins, err
}

func (po *pluginCmdOps) ListRecommendedPluginsFromActiveContext(installedOnly bool, opts ...E2EOption) ([]*PluginInfo, error) {
	plugins, _, _, err := ExecuteCmdAndBuildJSONOutput[PluginInfo](po.cmdExe, ListPluginsCmdWithJSONOutputFlag, opts...)
	recommendedPlugins := make([]*PluginInfo, 0)
	for i := range plugins {
		if plugins[i].Recommended != "" {
			if installedOnly {
				if plugins[i].Status == Installed || plugins[i].Status == UpdateAvailable || plugins[i].Status == RecommendUpdate {
					recommendedPlugins = append(recommendedPlugins, plugins[i])
				}
			} else {
				recommendedPlugins = append(recommendedPlugins, plugins[i])
			}
		}
	}
	return recommendedPlugins, err
}

func (po *pluginCmdOps) Sync(opts ...E2EOption) (string, string, error) {
	out, stdErr, err := po.cmdExe.TanzuCmdExec(pluginSyncCmd, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, pluginSyncCmd, err.Error(), stdErr.String(), out.String())
	}
	return out.String(), stdErr.String(), err
}

func (po *pluginCmdOps) SearchPlugins(filter string, opts ...E2EOption) (plugins []*PluginInfo, stdOutStr, stdErrStr string, err error) {
	searchPluginCmdWithOptions := SearchPluginsCmd
	if len(strings.TrimSpace(filter)) > 0 {
		searchPluginCmdWithOptions = searchPluginCmdWithOptions + " " + strings.TrimSpace(filter)
	}
	result, stdOutStr, stdErrStr, err := ExecuteCmdAndBuildJSONOutput[PluginSearch](po.cmdExe, searchPluginCmdWithOptions+JSONOutput, opts...)
	if err != nil {
		return plugins, stdOutStr, stdErrStr, err
	}
	// Convert from PluginSearch to PluginInfo
	for _, p := range result {
		plugins = append(plugins, &PluginInfo{
			Name:        p.Name,
			Description: p.Description,
			Target:      p.Target,
			Version:     p.Latest,
		})
	}
	return plugins, stdOutStr, stdErrStr, err
}

func (po *pluginCmdOps) SearchPluginGroups(flagsWithValues string, opts ...E2EOption) ([]*PluginGroup, error) {
	searchPluginGroupCmdWithOptions := SearchPluginGroupsCmd
	if len(strings.TrimSpace(flagsWithValues)) > 0 {
		searchPluginGroupCmdWithOptions = searchPluginGroupCmdWithOptions + " " + strings.TrimSpace(flagsWithValues)
	}
	list, _, _, err := ExecuteCmdAndBuildJSONOutput[PluginGroup](po.cmdExe, searchPluginGroupCmdWithOptions+JSONOutput, opts...)
	return list, err
}

func (po *pluginCmdOps) GetPluginGroup(groupName string, flagsWithValues string, opts ...E2EOption) ([]*PluginGroupGet, error) {
	getPluginGroupCmdWithOptions := fmt.Sprintf(GetPluginGroupCmd, "%s", groupName)
	if len(strings.TrimSpace(flagsWithValues)) > 0 {
		getPluginGroupCmdWithOptions = getPluginGroupCmdWithOptions + " " + strings.TrimSpace(flagsWithValues)
	}
	pluginList, _, _, err := ExecuteCmdAndBuildJSONOutput[PluginGroupGet](po.cmdExe, getPluginGroupCmdWithOptions+JSONOutput, opts...)
	return pluginList, err
}

func (po *pluginCmdOps) InstallPlugin(pluginName, target, versions string, opts ...E2EOption) (stdout, stdErr string, err error) {
	installPluginCmd := fmt.Sprintf(InstallPluginCmd, "%s", pluginName)
	if len(strings.TrimSpace(target)) > 0 {
		installPluginCmd += " --target " + target
	}
	if len(strings.TrimSpace(versions)) > 0 {
		installPluginCmd += " --version " + versions
	}
	stdOutBuff, stdErrBuff, err := po.cmdExe.TanzuCmdExec(installPluginCmd, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, installPluginCmd, err.Error(), stdOutBuff.String(), stdErrBuff.String())
	}
	return stdOutBuff.String(), stdErrBuff.String(), err
}

func (po *pluginCmdOps) InstallPluginsFromGroup(pluginNameORAll, groupName string, opts ...E2EOption) (stdout string, stdErr string, err error) {
	var installPluginCmd string
	if len(pluginNameORAll) > 0 {
		installPluginCmd = fmt.Sprintf(InstallPluginFromGroupCmd, "%s", pluginNameORAll, groupName)
	} else {
		installPluginCmd = fmt.Sprintf(InstallAllPluginsFromGroupCmd, "%s", groupName)
	}
	out, stdErrBuffer, err := po.cmdExe.TanzuCmdExec(installPluginCmd, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, installPluginCmd, err.Error(), stdErrBuffer.String(), out.String())
	}
	return out.String(), stdErrBuffer.String(), err
}

func (po *pluginCmdOps) DescribePlugin(pluginName, target string, opts ...E2EOption) ([]*PluginDescribe, error) {
	pluginDescCmd := fmt.Sprintf(DescribePluginCmd, "%s", pluginName)
	if len(strings.TrimSpace(target)) > 0 {
		pluginDescCmd += " --target " + target
	}
	list, _, _, err := ExecuteCmdAndBuildJSONOutput[PluginDescribe](po.cmdExe, pluginDescCmd, opts...)
	return list, err
}

func (po *pluginCmdOps) DescribePluginLegacy(pluginName, target string, opts ...E2EOption) (string, error) {
	installPluginCmd := fmt.Sprintf(DescribePluginCmd, "%s", pluginName)
	if len(strings.TrimSpace(target)) > 0 {
		installPluginCmd += " --target " + target
	}

	stdOut, stdErr, err := po.cmdExe.TanzuCmdExec(installPluginCmd, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, installPluginCmd, err.Error(), stdErr.String(), stdOut.String())
	}
	return stdOut.String(), err
}

func (po *pluginCmdOps) DeletePlugin(pluginName, target string, opts ...E2EOption) error {
	return po.UninstallPlugin(pluginName, target, opts...)
}

func (po *pluginCmdOps) UninstallPlugin(pluginName, target string, opts ...E2EOption) error {
	uninstallPluginCmd := fmt.Sprintf(UninstallPLuginCmd, "%s", pluginName)
	if len(strings.TrimSpace(target)) > 0 {
		uninstallPluginCmd += " --target " + target
	}
	out, stdErr, err := po.cmdExe.TanzuCmdExec(uninstallPluginCmd, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, uninstallPluginCmd, err.Error(), stdErr.String(), out.String())
	}
	return err
}

func (po *pluginCmdOps) ExecuteSubCommand(pluginWithSubCommand string, opts ...E2EOption) (string, error) {
	pluginCmdWithSubCommand := fmt.Sprintf(PluginSubCommand, "%s", pluginWithSubCommand)
	stdOut, stdErr, err := po.cmdExe.TanzuCmdExec(pluginCmdWithSubCommand, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, pluginCmdWithSubCommand, err.Error(), stdErr.String(), stdOut.String())
		return stdOut.String(), errors.Wrap(err, stdErr.String())
	}
	return stdOut.String(), nil
}

func (po *pluginCmdOps) CleanPlugins(opts ...E2EOption) error {
	out, stdErr, err := po.cmdExe.TanzuCmdExec(CleanPluginsCmd, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, CleanPluginsCmd, err.Error(), stdErr.String(), out.String())
	}
	return err
}

func (po *pluginCmdOps) RunPluginCmd(options string, opts ...E2EOption) (string, string, error) {
	cmd := PluginCmdWithOptions
	if options != "" {
		cmd += options
	}
	stdOut, stdErr, err := po.cmdExe.TanzuCmdExec(cmd, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, CleanPluginsCmd, err.Error(), stdErr.String(), stdOut.String())
		return stdOut.String(), stdErr.String(), nil
	}
	return stdOut.String(), stdErr.String(), nil
}

func (po *pluginCmdOps) DownloadPluginBundle(image string, groups []string, toTar string, opts ...E2EOption) error {
	downloadPluginBundle := PluginDownloadBundleCmd
	if len(strings.TrimSpace(image)) > 0 {
		downloadPluginBundle += " --image " + image
	}
	if len(groups) > 0 {
		downloadPluginBundle += " --group " + strings.Join(groups, ",")
	}
	downloadPluginBundle += " --to-tar " + strings.TrimSpace(toTar)

	out, stdErr, err := po.cmdExe.TanzuCmdExec(downloadPluginBundle, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, downloadPluginBundle, err.Error(), stdErr.String(), out.String())
	}
	return err
}

func (po *pluginCmdOps) UploadPluginBundle(toRepo string, tar string, opts ...E2EOption) error {
	uploadPluginBundle := PluginUploadBundleCmd
	uploadPluginBundle += " --to-repo " + strings.TrimSpace(toRepo)
	uploadPluginBundle += " --tar " + strings.TrimSpace(tar)
	out, stdErr, err := po.cmdExe.TanzuCmdExec(uploadPluginBundle, opts...)
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, uploadPluginBundle, err.Error(), stdErr.String(), out.String())
	}
	return err
}
