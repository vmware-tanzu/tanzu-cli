// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

func TestFlatDirPluginSupplier(t *testing.T) {
	assert := assert.New(t)
	dir, err := os.MkdirTemp("", "tanzu-cli-flatdir-supplier")
	assert.Nil(err)
	defer os.RemoveAll(dir)

	var completionType, postInstallResult uint8
	err = setupFakePlugin(dir, "flatfoo", "v0.1.0", plugin.SystemCmdGroup, completionType, postInstallResult, false, []string{"ff", "ffoo"})
	assert.Nil(err)

	supplier := FlatDirPluginSupplier{pluginDir: dir}
	pi, err := supplier.GetInstalledPlugins()
	assert.Equal(len(pi), 1)
	assert.Nil(err)

	expected := cli.PluginInfo{
		Name:             "flatfoo",
		Description:      "flatfoo functionality",
		Group:            plugin.SystemCmdGroup,
		Hidden:           false,
		Aliases:          []string{"ff", "ffoo"},
		Version:          "v0.1.0",
		InstallationPath: filepath.Join(dir, "flatfoo"),
	}
	assert.Equal(expected, *pi[0])
}
