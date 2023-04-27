// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

func readOutput(t *testing.T, r io.Reader, c chan<- []byte) {
	data, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}
	c <- data
}

func TestGetCmdForPlugin(t *testing.T) {
	assert := assert.New(t)

	dir, err := os.MkdirTemp("", "tanzu-cli-getcmd")
	assert.Nil(err)
	defer os.RemoveAll(dir)

	path, err := setupFakePlugin(dir, "fakefoo", "")
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

func TestEnvForPlugin(t *testing.T) {
	assert := assert.New(t)

	dir, err := os.MkdirTemp("", "tanzu-cli-getcmd")
	assert.Nil(err)
	defer os.RemoveAll(dir)

	path, err := setupFakePlugin(dir, "fakefoo", "echo $TANZU_BIN")
	assert.Nil(err)

	pi := &PluginInfo{
		Name:             "fakefoo",
		Description:      "Fake foo",
		Group:            plugin.SystemCmdGroup,
		Aliases:          []string{"ff"},
		InstallationPath: path,
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Error(err)
	}
	c := make(chan []byte)
	go readOutput(t, r, c)

	// Set up for our test
	const binaryPath = "/path/to/tanzu/binary"
	stdout := os.Stdout
	arg := os.Args[0]
	defer func() {
		os.Stdout = stdout
		os.Args[0] = arg
	}()
	os.Stdout = w
	os.Args[0] = binaryPath

	err = GetCmdForPlugin(pi).Execute()
	assert.Nil(err)
	w.Close()

	got := <-c
	assert.Equal(binaryPath+"\n", string(got))
}

func TestGetTestCmdForPlugin(t *testing.T) {
	assert := assert.New(t)

	dir, err := os.MkdirTemp("", "tanzu-cli-gettestcmd")
	assert.Nil(err)
	defer os.RemoveAll(dir)

	_, err = setupFakePlugin(dir, "test-fakefoo", "")
	assert.Nil(err)

	path, err := setupFakePlugin(dir, "fakefoo", "")
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

func setupFakePlugin(dir, pluginName, command string) (string, error) {
	filePath := filepath.Join(dir, pluginName)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if command == "" {
		command = "echo hello"
	}
	fmt.Fprintf(f, "#!/bin/bash\n\n%s\n", command)
	return filePath, nil
}
