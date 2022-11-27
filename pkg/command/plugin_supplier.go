// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides commands
package command

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aunum/log"
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
)

// FlatDirPluginSupplier is a naive supplier of plugins based on expecting a
// flat list of plugin binaries within a single directory.
// TODO(vuil): To be superceded by a plugin LCM component.
type FlatDirPluginSupplier struct {
	pluginDir string
}

// GetInstalledPlugins returns plugins for the supplier.
func (s *FlatDirPluginSupplier) GetInstalledPlugins() ([]*cli.PluginInfo, error) {
	pluginDir := s.pluginDir
	if pluginDir == "" {
		pluginDir = common.DefaultPluginRoot
	}

	plugins := make([]*cli.PluginInfo, 0)
	infos, err := os.ReadDir(pluginDir)
	if err != nil {
		log.Debug("Unable to find installed plugins")
		return plugins, nil
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		pluginName := info.Name()
		pluginPath := filepath.Join(pluginDir, pluginName)

		pi, err := getPluginInfo(pluginPath)
		if err != nil {
			log.Debug("Unable to get plugin info for %s: %v\n", pluginName, err)
			pi = &cli.PluginInfo{
				Name:             pluginName,
				InstallationPath: pluginPath,
			}
		}
		plugins = append(plugins, pi)
	}

	return plugins, nil
}

// getPluginInfo builds and return a PluginInfo from output of the "<pluginpath> info" command
func getPluginInfo(pluginPath string) (*cli.PluginInfo, error) {
	bytesInfo, err := exec.Command(pluginPath, "info").Output()
	if err != nil {
		return nil, errors.Wrapf(err, "could not obtain plugin description for %s", pluginPath)
	}

	var pi cli.PluginInfo
	pi.InstallationPath = pluginPath

	var descriptor plugin.PluginDescriptor
	if err = json.Unmarshal(bytesInfo, &descriptor); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal plugin descriptor for %s", pluginPath)
	}

	pi.Name = descriptor.Name
	pi.Description = descriptor.Description
	pi.Version = descriptor.Version
	pi.Group = descriptor.Group
	pi.Aliases = descriptor.Aliases
	pi.Hidden = descriptor.Hidden

	return &pi, nil
}
