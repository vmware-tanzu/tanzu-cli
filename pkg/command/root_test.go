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

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	configcli "github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
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
  "completionType": %d,
  "target": "%s"
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
		target            configtypes.Target
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
			target:   configtypes.TargetK8s,
			expected: "dummy",
		},
		{
			test:     "k8s target",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			target:   configtypes.TargetK8s,
			args:     []string{"dummy", "say", "hello"},
			expected: "hello",
		},
		{
			test:     "global target",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			target:   configtypes.TargetGlobal,
			args:     []string{"dummy", "say", "hello"},
			expected: "hello",
		},
		{
			test:     "unknown target (backwards-compatibility)",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			target:   configtypes.TargetUnknown,
			args:     []string{"dummy", "say", "hello"},
			expected: "hello",
		},
		{
			test:            "tmc target",
			plugin:          "dummy",
			version:         "v0.1.0",
			cmdGroup:        plugin.SystemCmdGroup,
			target:          configtypes.TargetTMC,
			args:            []string{"dummy", "say", "hello"},
			expectedFailure: true,
		},
		{
			test:     "run info subcommand",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			target:   configtypes.TargetK8s,
			expected: `{
  "name": "dummy",
  "description": "dummy functionality",
  "version": "v0.1.0",
  "buildSHA": "01234567",
  "group": "System",
  "hidden": false,
  "aliases": [],
  "completionType": 0,
  "target": "kubernetes"
}`,
			args: []string{"dummy", "info"},
		},
		{
			test:            "execute known bad command",
			plugin:          "dummy",
			version:         "v0.1.0",
			cmdGroup:        plugin.SystemCmdGroup,
			target:          configtypes.TargetK8s,
			args:            []string{"dummy", "bad"},
			expectedFailure: true,
			expected:        "bad command failed",
		},
		{
			test:            "invoke missing command",
			plugin:          "dummy",
			version:         "v0.1.0",
			cmdGroup:        plugin.SystemCmdGroup,
			target:          configtypes.TargetK8s,
			args:            []string{"dummy", "missing command"},
			expectedFailure: false,
			expected:        "Available Commands:",
		},
		{
			test:              "when post-install succeeds",
			plugin:            "dummy",
			version:           "v0.1.0",
			cmdGroup:          plugin.SystemCmdGroup,
			target:            configtypes.TargetK8s,
			postInstallResult: 0,
			args:              []string{"dummy", "post-install"},
			expectedFailure:   false,
		},
		{
			test:              "when post-install fails",
			plugin:            "dummy",
			version:           "v0.1.0",
			cmdGroup:          plugin.SystemCmdGroup,
			target:            configtypes.TargetK8s,
			postInstallResult: 1,
			args:              []string{"dummy", "post-install"},
			expectedFailure:   true,
		},
		{
			test:            "all args after command are passed",
			plugin:          "dummy",
			version:         "v0.1.0",
			cmdGroup:        plugin.SystemCmdGroup,
			target:          configtypes.TargetK8s,
			args:            []string{"dummy", "shout", "lots", "of", "things"},
			expected:        "lots of things !!",
			expectedFailure: false,
		},
		{
			test:            "hidden plugin not visible in root command",
			plugin:          "dummy",
			version:         "v0.1.0",
			cmdGroup:        plugin.SystemCmdGroup,
			target:          configtypes.TargetK8s,
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

			err = setupFakePlugin(dir, spec.plugin, spec.version, spec.cmdGroup, completionType, spec.target, spec.postInstallResult, spec.hidden, spec.aliases)
			assert.Nil(err)

			pi := &cli.PluginInfo{
				Name:             spec.plugin,
				Description:      spec.plugin,
				Group:            spec.cmdGroup,
				Aliases:          []string{},
				Hidden:           spec.hidden,
				InstallationPath: filepath.Join(dir, spec.plugin),
				Target:           spec.target,
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

			assert.Equal(spec.expectedFailure, err != nil)
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

func TestEnvVarsSet(t *testing.T) {
	assert := assert.New(t)

	// Create test configuration files
	configFile, _ := os.CreateTemp("", "config")
	os.Setenv("TANZU_CONFIG", configFile.Name())
	defer os.RemoveAll(configFile.Name())
	configFileNG, _ := os.CreateTemp("", "config_ng")
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
	defer os.RemoveAll(configFileNG.Name())

	// Setup default feature flags since we have created new config files
	// TODO(khouzam): This is because AddDefaultFeatureFlagsIfMissing() has already
	// been called in an init() function.  We should fix that in a more generic way.
	c, err := config.GetClientConfigNoLock()
	assert.Nil(err)
	if configcli.AddDefaultFeatureFlagsIfMissing(c, constants.DefaultCliFeatureFlags) {
		_ = config.StoreClientConfig(c)
	}

	rootCmd, err := NewRootCmd()
	assert.Nil(err)

	envVarName := "SOME_TEST_ENV_VAR"
	envVarValue := "SOME_TEST_ENV_VALUE"

	// First check the env var does not exist
	assert.Equal("", os.Getenv(envVarName))

	// Create an environment variable in the CLI
	rootCmd.SetArgs([]string{"config", "set", "env." + envVarName, envVarValue})
	err = rootCmd.Execute()
	assert.Nil(err)

	// Re-initialize the CLI with the config files containing the variable.
	// It is in this call that the CLI creates the OS variables.
	_, err = NewRootCmd()
	assert.Nil(err)
	// Make sure the variable is now set during the call to the CLI
	assert.Equal(envVarValue, os.Getenv(envVarName))

	// Cleanup
	os.Unsetenv("TANZU_CONFIG")
	os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
	os.Unsetenv(envVarName)
}

func setupFakePlugin(dir, pluginName, version string, commandGroup plugin.CmdGroup, completionType uint8, target configtypes.Target, postInstallResult uint8, hidden bool, aliases []string) error {
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

	fmt.Fprintf(f, fakePluginScriptFmtString, pluginName, pluginName, commandGroup, strconv.FormatBool(hidden), aliasesString, completionType, target, postInstallResult, pluginName, pluginName)

	return nil
}
