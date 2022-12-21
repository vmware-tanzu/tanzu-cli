// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

const (
	fakePluginScriptFmtString string = `#!/bin/bash

# Minimally Viable Dummy Tanzu CLI 'Plugin'

info() {
   cat << EOF
{
  "name": "%s",
  "description": "%s functionality",
  "version": "v0.1.0",
  "buildSHA": "01234567",
  "group": "%s",
  "hidden": %s,
  "aliases": %s,
  "completionType": %d
}
EOF
  exit 0
}

say() { shift && echo $@; }
shout() { shift && echo $@ "!!"; }
post-install() { exit %d; }
bad() { echo "bad command failed"; exit 1; }

case "$1" in
    info)  $1 "$@";;
    say)   $1 "$@";;
    shout) $1 "$@";;
    bad)   $1 "$@";;
    post-install)  $1 "$@";;
    *) cat << EOF
%s functionality

Usage:
  tanzu %s [command]

Available Commands:
  say     Say a phrase
  shout   Shout a phrase
  bad     (non-working)
EOF
       exit 0
       ;;
esac
`
)

func TestExecute(t *testing.T) {
	assert := assert.New(t)
	err := Execute()
	assert.Nil(err)
}

type TestPluginSupplier struct {
	pluginInfos []*cli.PluginInfo
}

func (s *TestPluginSupplier) GetInstalledPlugins() ([]*cli.PluginInfo, error) {
	return s.pluginInfos, nil
}

func TestRootCmdWithNoAdditionalPlugins(t *testing.T) {
	assert := assert.New(t)
	rootCmd, err := NewRootCmd()
	assert.Nil(err)
	err = rootCmd.Execute()
	assert.Nil(err)
}

func TestSubcommandNonexistent(t *testing.T) {
	assert := assert.New(t)
	rootCmd, err := NewRootCmd()
	assert.Nil(err)
	rootCmd.SetArgs([]string{"nonexistent", "say", "hello"})
	err = rootCmd.Execute()
	assert.NotNil(err)
}

func TestSubcommands(t *testing.T) {
	tests := []struct {
		test              string
		plugin            string
		version           string
		cmdGroup          plugin.CmdGroup
		postInstallResult uint8
		hidden            bool
		aliases           []string
		args              []string
		expected          string
		unexpected        string
		expectedFailure   bool
	}{
		{
			test:     "no arg test",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			expected: "dummy",
		},
		{
			test:     "run info subcommand",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			expected: `{
  "name": "dummy",
  "description": "dummy functionality",
  "version": "v0.1.0",
  "buildSHA": "01234567",
  "group": "System",
  "hidden": false,
  "aliases": [],
  "completionType": 0
}`,
			args: []string{"dummy", "info"},
		},
		{
			test:            "execute known bad command",
			plugin:          "dummy",
			version:         "v0.1.0",
			cmdGroup:        plugin.SystemCmdGroup,
			args:            []string{"dummy", "bad"},
			expectedFailure: true,
			expected:        "bad command failed",
		},
		{
			test:            "invoke missing command",
			plugin:          "dummy",
			version:         "v0.1.0",
			cmdGroup:        plugin.SystemCmdGroup,
			args:            []string{"dummy", "missing command"},
			expectedFailure: false,
			expected:        "Available Commands:",
		},
		{
			test:              "when post-install succeeds",
			plugin:            "dummy",
			version:           "v0.1.0",
			cmdGroup:          plugin.SystemCmdGroup,
			postInstallResult: 0,
			args:              []string{"dummy", "post-install"},
			expectedFailure:   false,
		},
		{
			test:              "when post-install fails",
			plugin:            "dummy",
			version:           "v0.1.0",
			cmdGroup:          plugin.SystemCmdGroup,
			postInstallResult: 1,
			args:              []string{"dummy", "post-install"},
			expectedFailure:   true,
		},
		{
			test:            "all args after command are passed",
			plugin:          "dummy",
			version:         "v0.1.0",
			cmdGroup:        plugin.SystemCmdGroup,
			args:            []string{"dummy", "shout", "lots", "of", "things"},
			expected:        "lots of things !!",
			expectedFailure: false,
		},
		{
			test:            "hidden plugin not visible in root command",
			plugin:          "dummy",
			version:         "v0.1.0",
			cmdGroup:        plugin.SystemCmdGroup,
			hidden:          true,
			args:            []string{},
			expectedFailure: false,
			unexpected:      "dummy",
		},
	}

	for _, spec := range tests {
		dir, err := os.MkdirTemp("", "tanzu-cli-root-cmd")
		assert.Nil(t, err)
		defer os.RemoveAll(dir)
		os.Setenv("TEST_CUSTOM_CATALOG_CACHE_DIR", dir)
		var completionType uint8
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			r, w, err := os.Pipe()
			if err != nil {
				t.Error(err)
			}
			c := make(chan []byte)
			go readOutput(t, r, c)

			// Set up for our test
			stdout := os.Stdout
			stderr := os.Stderr
			defer func() {
				os.Stdout = stdout
				os.Stderr = stderr
			}()
			os.Stdout = w
			os.Stderr = w

			err = setupFakePlugin(dir, spec.plugin, spec.version, spec.cmdGroup, completionType, spec.postInstallResult, spec.hidden, spec.aliases)
			assert.Nil(err)

			pi := &cli.PluginInfo{
				Name:             spec.plugin,
				Description:      spec.plugin,
				Group:            spec.cmdGroup,
				Aliases:          []string{},
				Hidden:           spec.hidden,
				InstallationPath: filepath.Join(dir, spec.plugin),
			}
			cc, err := catalog.NewContextCatalog("")
			assert.Nil(err)
			err = cc.Upsert(pi)
			assert.Nil(err)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()

			w.Close()
			got := <-c

			assert.Equal(err != nil, spec.expectedFailure)
			if spec.expected != "" {
				assert.Contains(string(got), spec.expected)
			}
			if spec.unexpected != "" {
				assert.NotContains(string(got), spec.unexpected)
			}
		})
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
	}
}

func setupFakePlugin(dir, pluginName, version string, commandGroup plugin.CmdGroup, completionType uint8, postInstallResult uint8, hidden bool, aliases []string) error {
	filePath := filepath.Join(dir, pluginName)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	var aliasesString string
	if len(aliases) > 0 {
		b, _ := json.Marshal(aliases)
		aliasesString = string(b)
	} else {
		aliasesString = "[]"
	}

	fmt.Fprintf(f, fakePluginScriptFmtString, pluginName, pluginName, commandGroup, strconv.FormatBool(hidden), aliasesString, completionType, postInstallResult, pluginName, pluginName)

	return nil
}
