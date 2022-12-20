// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

func TestGetCmdForPlugin(t *testing.T) {
	assert := assert.New(t)

	dir, err := os.MkdirTemp("", "tanzu-cli-getcmd")
	assert.Nil(err)
	defer os.RemoveAll(dir)

	path, err := setupFakePlugin(dir, "fakefoo")
	assert.Nil(err)

	pi := &PluginInfo{
		Name:             "fakefoo",
		Description:      "Fake foo",
		Group:            plugin.SystemCmdGroup,
		Aliases:          []string{"ff"},
		InstallationPath: path,
	}
	cmd := GetCmdForPlugin(pi)

	err = cmd.Execute()
	assert.Equal(cmd.Name(), pi.Name)
	assert.Equal(cmd.Short, pi.Description)
	assert.Equal(cmd.Aliases, pi.Aliases)
	assert.Nil(err)
}

func TestGetTestCmdForPlugin(t *testing.T) {
	assert := assert.New(t)

	dir, err := os.MkdirTemp("", "tanzu-cli-gettestcmd")
	assert.Nil(err)
	defer os.RemoveAll(dir)

	_, err = setupFakePlugin(dir, "test-fakefoo")
	assert.Nil(err)

	path, err := setupFakePlugin(dir, "fakefoo")
	assert.Nil(err)

	pi := &PluginInfo{
		Name:             "fakefoo",
		Description:      "Fake foo",
		Group:            plugin.SystemCmdGroup,
		Aliases:          []string{"ff"},
		InstallationPath: path,
	}
	cmd := GetTestCmdForPlugin(pi)

	err = cmd.Execute()
	assert.Equal(cmd.Name(), pi.Name)
	assert.Equal(cmd.Short, pi.Description)
	assert.Nil(err)
}

func setupFakePlugin(dir string, pluginName string) (string, error) {
	filePath := filepath.Join(dir, pluginName)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fmt.Fprintf(f, "#!/bin/bash\n\necho hello\n")
	return filePath, nil
}
