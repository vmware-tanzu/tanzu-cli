// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/csp"
	"github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo"
	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/globalinit"
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
show-invoke-context() { echo "args = ($@), context is (${TANZU_CLI_INVOKED_GROUP}):(${TANZU_CLI_INVOKED_COMMAND}):(${TANZU_CLI_COMMAND_MAPPED_FROM})"; exit 0; }

case "$1" in
    info)  $1 "$@";;
    say)   $1 "$@";;
    shout) $1 "$@";;
    bad)   $1 "$@";;
    show-invoke-context) $1 "$@";;
    post-install)  $1 "$@";;
    *) cat << EOF
%s functionality

Usage:
  tanzu %s [command]

Available Commands:
  say     Say a phrase
  shout   Shout a phrase
  bad     (non-working)
  show-invoke-context Shows invocation details
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

// testCLIEnvironment captures information needed to reset the top level "target" commands,
// clean up config and unset envvars
type testCLIEnvironment struct {
	cacheDir     string
	configFile   string
	configFileNG string
	dataStore    string
	envVars      []string
}

// helper to set up a clean environment to create a root CLI command
func setupTestCLIEnvironment(t *testing.T) testCLIEnvironment {
	dir, err := os.MkdirTemp("", "tanzu-cli-root-cmd")
	assert.Nil(t, err)
	os.Setenv("TEST_CUSTOM_CATALOG_CACHE_DIR", dir)

	// Setup a temporary configuration.  This means no
	// plugin source will be available which will prevent
	// trying to install the essential plugins.
	// This speeds up the test.
	configFile, _ := os.CreateTemp("", "config")
	os.Setenv("TANZU_CONFIG", configFile.Name())

	configFileNG, _ := os.CreateTemp("", "config_ng")
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

	os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
	os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")

	tmpDataStoreFile, _ := os.CreateTemp("", "data-store.yaml")
	os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDataStoreFile.Name())

	// Start each test with the defaults of the target commands
	// and reset the help flag in case it was set
	tmcCmd.ResetCommands()
	tmcCmd.Hidden = false
	helpFlag := tmcCmd.Flags().Lookup("help")
	if helpFlag != nil {
		_ = helpFlag.Value.Set("false")
	}
	k8sCmd.ResetCommands()
	k8sCmd.Hidden = false
	helpFlag = k8sCmd.Flags().Lookup("help")
	if helpFlag != nil {
		_ = helpFlag.Value.Set("false")
	}
	opsCmd.ResetCommands()
	opsCmd.Hidden = false
	helpFlag = opsCmd.Flags().Lookup("help")
	if helpFlag != nil {
		_ = helpFlag.Value.Set("false")
	}

	return testCLIEnvironment{
		cacheDir:     dir,
		configFile:   configFile.Name(),
		configFileNG: configFileNG.Name(),
		dataStore:    tmpDataStoreFile.Name(),

		envVars: []string{
			"TEST_CUSTOM_CATALOG_CACHE_DIR",
			"TANZU_CONFIG",
			"TANZU_CONFIG_NEXT_GEN",
			"TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER",
			"TANZU_CLI_EULA_PROMPT_ANSWER",
			"TEST_CUSTOM_DATA_STORE_FILE",
		},
	}
}

// helper to clean up CLI environment
func tearDownTestCLIEnvironment(env testCLIEnvironment) {
	os.RemoveAll(env.cacheDir)
	os.RemoveAll(env.configFile)
	os.RemoveAll(env.configFileNG)
	os.RemoveAll(env.dataStore)

	for _, envVar := range env.envVars {
		os.Unsetenv(envVar)
	}
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
			test:     "k8s plugin at root level",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			target:   configtypes.TargetK8s,
			args:     []string{"dummy", "say", "hello"},
			expected: "hello",
		},
		{
			test:     "k8s target",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			target:   configtypes.TargetK8s,
			args:     []string{"k8s", "dummy", "say", "hello"},
			expected: "hello",
		},
		{
			test:     "kubernetes target",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			target:   configtypes.TargetK8s,
			args:     []string{"kubernetes", "dummy", "say", "hello"},
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
			test:     "tmc target",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			target:   configtypes.TargetTMC,
			args:     []string{"tmc", "dummy", "say", "hello"},
			expected: "hello",
		},
		{
			test:     "mission-control target",
			plugin:   "dummy",
			version:  "v0.1.0",
			cmdGroup: plugin.SystemCmdGroup,
			target:   configtypes.TargetTMC,
			args:     []string{"mission-control", "dummy", "say", "hello"},
			expected: "hello",
		},
		{
			test:            "tmc target not at root",
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
		env := setupTestCLIEnvironment(t)
		defer tearDownTestCLIEnvironment(env)

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

			err = setupFakePlugin(env.cacheDir, spec.plugin, spec.version, spec.cmdGroup, completionType, spec.target, spec.postInstallResult, spec.hidden, spec.aliases)
			assert.Nil(err)

			pi := &cli.PluginInfo{
				Name:             spec.plugin,
				Description:      spec.plugin,
				Group:            spec.cmdGroup,
				Aliases:          []string{},
				Hidden:           spec.hidden,
				InstallationPath: filepath.Join(env.cacheDir, fmt.Sprintf("%s_%s", spec.plugin, string(spec.target))),
				Target:           spec.target,
			}
			cc, err := catalog.NewContextCatalogUpdater("")
			assert.Nil(err)
			err = cc.Upsert(pi)
			cc.Unlock()
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
	}
}

func TestTargetCommands(t *testing.T) {
	tests := []struct {
		test                   string
		installedPluginTargets []configtypes.Target
		args                   []string
		expected               []string
		unexpected             []string
		expectedFailure        bool
	}{
		// ===================================
		// Tests for the top-level help output
		// ===================================
		{
			test:       "top run for all empty targets",
			args:       []string{},
			unexpected: []string{"Target", "kubernetes", "mission-control", "operations"},
		},
		{
			test:                   "top run for only ops empty",
			args:                   []string{},
			installedPluginTargets: []configtypes.Target{configtypes.TargetTMC},
			expected:               []string{"Target", "mission-control"},
			unexpected:             []string{"kubernetes", "operations"},
		},
		{
			test:                   "top run for only tmc empty",
			args:                   []string{},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s, configtypes.TargetOperations},
			expected:               []string{"Target", "operations"},
			unexpected:             []string{"mission-control", "kubernetes"},
		},
		{
			test:                   "top run for no empty targets",
			args:                   []string{},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s, configtypes.TargetTMC, configtypes.TargetOperations},
			expected:               []string{"Target", "operations", "mission-control"},
			unexpected:             []string{"kubernetes"},
		},
		{
			test:       "top help for all empty targets",
			args:       []string{"-h"},
			unexpected: []string{"Target", "kubernetes", "mission-control", "operations"},
		},
		{
			test:                   "top help for only ops empty",
			args:                   []string{"-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetTMC},
			expected:               []string{"Target", "mission-control"},
			unexpected:             []string{"kubernetes", "operations"},
		},
		{
			test:                   "top help for only tmc empty",
			args:                   []string{"-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s, configtypes.TargetOperations},
			expected:               []string{"Target", "operations"},
			unexpected:             []string{"mission-control", "kubernetes"},
		},
		{
			test:                   "top help for no empty targets",
			args:                   []string{"-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s, configtypes.TargetTMC, configtypes.TargetOperations},
			expected:               []string{"Target", "operations", "mission-control"},
			unexpected:             []string{"kubernetes"},
		},
		// ========================
		// Tests for the k8s target
		// ========================
		{
			test: "help for k8s target run when empty",
			args: []string{"k8s"},
			expected: []string{
				"Command \"kubernetes\" is deprecated",
				"Note: No plugins are currently installed for \"kubernetes\"",
				"Commands that interact with a Kubernetes endpoint",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for k8s target help when empty",
			args: []string{"k8s", "-h"},
			expected: []string{
				"Command \"kubernetes\" is deprecated",
				"Commands that interact with a Kubernetes endpoint",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for k8s target invalid when empty",
			args: []string{"k8s", "invalid"},
			expected: []string{
				"Command \"kubernetes\" is deprecated",
				"Note: No plugins are currently installed for \"kubernetes\"",
				"Commands that interact with a Kubernetes endpoint",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test:                   "help for k8s target run when not empty",
			args:                   []string{"k8s"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s},
			expected: []string{
				"Command \"kubernetes\" is deprecated",
				"Commands that interact with a Kubernetes endpoint",
				"Available command groups:\n\n  System\n    dummy                   dummy \n\nFlags:",
			},
			unexpected: []string{
				"Note: No plugins are currently installed",
			},
		},
		{
			test:                   "help for k8s target help when not empty",
			args:                   []string{"k8s", "-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s},
			expected: []string{
				"Command \"kubernetes\" is deprecated",
				"Commands that interact with a Kubernetes endpoint",
				"Available command groups:\n\n  System\n    dummy                   dummy \n\nFlags:",
			},
			unexpected: []string{
				"Note: No plugins are currently installed",
			},
		},
		{
			test:                   "help for k8s target run when empty but tmc not empty",
			args:                   []string{"k8s"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetTMC},
			expected: []string{
				"Command \"kubernetes\" is deprecated",
				"Commands that interact with a Kubernetes endpoint",
				"Note: No plugins are currently installed for \"kubernetes\"",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test:                   "help for k8s target help when empty but tmc not empty",
			args:                   []string{"k8s", "-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetTMC},
			expected: []string{
				"Commands that interact with a Kubernetes endpoint",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		// ========================
		// Tests for the tmc target
		// ========================
		{
			test: "help for tmc target run when empty",
			args: []string{"tmc"},
			expected: []string{
				"Note: No plugins are currently installed for \"mission-control\"",
				"Commands that provide functionality for Tanzu Mission Control",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for tmc target help when empty",
			args: []string{"tmc", "-h"},
			expected: []string{
				"Commands that provide functionality for Tanzu Mission Control",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for tmc target invalid when empty",
			args: []string{"tmc", "invalid"},
			expected: []string{
				"Note: No plugins are currently installed for \"mission-control\"",
				"Commands that provide functionality for Tanzu Mission Control",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test:                   "help for tmc target run when not empty",
			args:                   []string{"tmc"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetTMC},
			expected: []string{
				"Commands that provide functionality for Tanzu Mission Control",
				"Available command groups:\n\n  System\n    dummy                   dummy \n\nFlags:",
			},
			unexpected: []string{
				"Note: No plugins are currently installed",
			},
		},
		{
			test:                   "help for tmc target help when not empty",
			args:                   []string{"tmc", "-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetTMC},
			expected: []string{
				"Commands that provide functionality for Tanzu Mission Control",
				"Available command groups:\n\n  System\n    dummy                   dummy \n\nFlags:",
			},
			unexpected: []string{
				"Note: No plugins are currently installed",
			},
		},
		{
			test:                   "help for tmc target run when empty but k8s not empty",
			args:                   []string{"tmc"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s},
			expected: []string{
				"Commands that provide functionality for Tanzu Mission Control",
				"Note: No plugins are currently installed for \"mission-control\"",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test:                   "help for tmc target help when empty but k8s not empty",
			args:                   []string{"tmc", "-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s},
			expected: []string{
				"Commands that provide functionality for Tanzu Mission Control",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		// ========================
		// Tests for the ops target
		// ========================
		{
			test: "help for ops target run when empty",
			args: []string{"ops"},
			expected: []string{
				"Note: No plugins are currently installed for \"operations\"",
				"Commands that support Kubernetes operations for Tanzu Platform for Kubernetes",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for ops target help when empty",
			args: []string{"ops", "-h"},
			expected: []string{
				"Commands that support Kubernetes operations for Tanzu Platform for Kubernetes",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for ops target invalid when empty",
			args: []string{"ops", "invalid"},
			expected: []string{
				"Note: No plugins are currently installed for \"operations\"",
				"Commands that support Kubernetes operations for Tanzu Platform for Kubernetes",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test:                   "help for ops target run when not empty",
			args:                   []string{"ops"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetOperations},
			expected: []string{
				"Commands that support Kubernetes operations for Tanzu Platform for Kubernetes",
				"Available command groups:\n\n  System\n    dummy                   dummy \n\nFlags:",
			},
			unexpected: []string{
				"Note: No plugins are currently installed",
			},
		},
		{
			test:                   "help for ops target help when not empty",
			args:                   []string{"ops", "-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetOperations},
			expected: []string{
				"Commands that support Kubernetes operations for Tanzu Platform for Kubernetes",
				"Available command groups:\n\n  System\n    dummy                   dummy \n\nFlags:",
			},
			unexpected: []string{
				"Note: No plugins are currently installed",
			},
		},
		{
			test:                   "help for ops target run when empty but k8s not empty",
			args:                   []string{"ops"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s},
			expected: []string{
				"Commands that support Kubernetes operations for Tanzu Platform for Kubernetes",
				"Note: No plugins are currently installed for \"operations\"",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test:                   "help for ops target help when empty but k8s not empty",
			args:                   []string{"ops", "-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s},
			expected: []string{
				"Commands that support Kubernetes operations for Tanzu Platform for Kubernetes",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
	}

	for _, spec := range tests {
		env := setupTestCLIEnvironment(t)
		defer tearDownTestCLIEnvironment(env)

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

			for _, target := range spec.installedPluginTargets {
				pluginName := "dummy"
				pi := &cli.PluginInfo{
					Name:             pluginName,
					Description:      pluginName,
					Group:            plugin.SystemCmdGroup,
					Version:          "v0.1.0",
					Aliases:          []string{},
					Hidden:           false,
					InstallationPath: filepath.Join(env.cacheDir, fmt.Sprintf("%s_%s", pluginName, string(target))),
					Target:           target,
				}
				err = setupFakePlugin(env.cacheDir, pi.Name, pi.Version, pi.Group, 0, pi.Target, 0, pi.Hidden, pi.Aliases)
				assert.Nil(err)

				cc, err := catalog.NewContextCatalogUpdater("")
				assert.Nil(err)
				err = cc.Upsert(pi)
				cc.Unlock()
				assert.Nil(err)
			}

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()

			w.Close()
			got := <-c

			assert.Equal(spec.expectedFailure, err != nil)
			for _, expected := range spec.expected {
				assert.Contains(string(got), expected)
			}
			for _, unexpected := range spec.unexpected {
				assert.NotContains(string(got), unexpected)
			}
		})
	}
}

func TestGlobalInit(t *testing.T) {
	tests := []struct {
		test         string
		args         []string
		trigger      bool
		initExpected bool
	}{
		{
			test:         "global init for a command",
			args:         []string{"plugin", "list"},
			trigger:      true,
			initExpected: true,
		},
		{
			test:         "no global init for a command",
			args:         []string{"plugin", "list"},
			trigger:      false,
			initExpected: false,
		},
		{
			test:         "no global init for version",
			args:         []string{"version"},
			trigger:      true,
			initExpected: false,
		},
		{
			test:         "no global init for completion",
			args:         []string{"completion", "bash"},
			trigger:      true,
			initExpected: false,
		},
		{
			test:         "no global init for __complete",
			args:         []string{"__complete", ""},
			trigger:      true,
			initExpected: false,
		},
	}

	const initString = "INITIALIZER CALLED"
	for _, spec := range tests {
		env := setupTestCLIEnvironment(t)
		defer tearDownTestCLIEnvironment(env)

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

			// Setup the specified initializer trigger
			globalinit.RegisterInitializer(
				"initializer",
				func() bool {
					return spec.trigger
				},
				func(io.Writer) error {
					fmt.Println(initString)
					return nil
				},
			)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Nil(err)

			w.Close()
			got := <-c

			if spec.initExpected {
				assert.Contains(string(got), initString)
			} else {
				assert.NotContains(string(got), initString)
				assert.NotContains(string(got), "The initialization encountered an error")
			}
		})
	}
}

func TestSetLastVersion(t *testing.T) {
	tests := []struct {
		test            string
		version         string
		expectedVersion string
	}{
		{
			test:            "set version to v1.2.3",
			version:         "v1.2.3",
			expectedVersion: "v1.2.3",
		},
	}

	originalVersion := buildinfo.Version
	defer func() {
		buildinfo.Version = originalVersion
	}()

	for _, spec := range tests {
		env := setupTestCLIEnvironment(t)
		defer tearDownTestCLIEnvironment(env)

		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			buildinfo.Version = spec.version

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			// Execute any command to trigger the version update
			rootCmd.SetArgs([]string{"plugin", "list"})
			err = rootCmd.Execute()
			assert.Nil(err)

			// Read the data store file and make sure it contains the last executed version
			b, err := os.ReadFile(env.dataStore)
			assert.Nil(err)
			assert.Contains(string(b), spec.expectedVersion)
		})
	}
}

func TestEnvVarsSet(t *testing.T) {
	env := setupTestCLIEnvironment(t)
	defer tearDownTestCLIEnvironment(env)

	assert := assert.New(t)

	err := config.ConfigureFeatureFlags(constants.DefaultCliFeatureFlags)
	assert.Nil(err)

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

	os.Unsetenv(envVarName)
}

func setupFakePlugin(dir, pluginName, _ string, commandGroup plugin.CmdGroup, completionType uint8, target configtypes.Target, postInstallResult uint8, hidden bool, aliases []string) error {
	filePath := filepath.Join(dir, fmt.Sprintf("%s_%s", pluginName, string(target)))

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

func TestCompletionShortHelpInActiveHelp(t *testing.T) {
	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		{
			test: "short help as active help at level 1",
			args: []string{"__complete", "plugin", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "clean\tClean the plugins\n" +
				"describe\tDescribe a plugin\n" +
				"download-bundle\tDownload plugin bundle to the local system\n" +
				"group\tManage plugin-groups\n" +
				"install\tInstall a plugin\n" +
				"list\tList installed plugins\n" +
				"search\tSearch for available plugins\n" +
				"source\tManage plugin discovery sources\n" +
				"sync\tInstalls all plugins recommended by the active contexts\n" +
				"uninstall\tUninstall a plugin\n" +
				"upgrade\tUpgrade a plugin\n" +
				"upload-bundle\tUpload plugin bundle to a repository\n" +
				"_activeHelp_ Command help: Manage CLI plugins\n" +
				":4\n",
		},
		{
			test: "short help as active help at level 2",
			args: []string{"__complete", "plugin", "search", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ Command help: Search for available plugins\n" +
				"_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
	}

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Nil(err)

			assert.Equal(spec.expected, out.String())
		})
	}

	os.Unsetenv("TANZU_ACTIVE_HELP")
}

type fakePluginRemapAttributes struct {
	name                 string
	description          string
	target               configtypes.Target
	cmdGroup             plugin.CmdGroup
	supportedContextType []configtypes.ContextType
	invokedAs            []string
	aliases              []string
	commandMap           []plugin.CommandMapEntry
}

func setupFakePluginInfo(p fakePluginRemapAttributes, pluginDir string) *cli.PluginInfo {
	description := fmt.Sprintf("%s commands", p.name)
	if p.description != "" {
		description = p.description
	}
	cmdGroup := plugin.SystemCmdGroup
	if p.cmdGroup != "" {
		cmdGroup = p.cmdGroup
	}

	pi := &cli.PluginInfo{
		Name:                 p.name,
		Description:          description,
		Group:                cmdGroup,
		Version:              "v0.1.0",
		Aliases:              []string{},
		Hidden:               false,
		InstallationPath:     filepath.Join(pluginDir, fmt.Sprintf("%s_%s", p.name, string(p.target))),
		Target:               p.target,
		InvokedAs:            []string{},
		SupportedContextType: []configtypes.ContextType{},
		CommandMap:           p.commandMap,
	}

	if len(p.invokedAs) != 0 {
		pi.InvokedAs = p.invokedAs
	}
	if len(p.supportedContextType) != 0 {
		pi.SupportedContextType = p.supportedContextType
	}
	if len(p.aliases) != 0 {
		pi.Aliases = p.aliases
	}
	return pi
}

// Tests behavior of command tree with commands remapped at plugin and command level
func TestCommandRemapping(t *testing.T) {
	tests := []struct {
		test                    string
		pluginVariants          []fakePluginRemapAttributes
		args                    []string
		expected                []string
		unexpected              []string
		expectedFailure         bool
		activeContextType       configtypes.ContextType
		allowActiveContextCheck bool
	}{
		{
			test: "one unmapped k8s plugin",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy",
					target: configtypes.TargetK8s,
				},
			},
			args:     []string{"dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "one mapped k8s plugin (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:      "dummy2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"dummy"},
				},
			},
			args:     []string{"dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "once mapped, command using original plugin name is not accessible (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:      "dummy2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"dummy"},
				},
			},
			args:            []string{"dummy2"},
			expectedFailure: true,
		},
		{
			test: "two plugins share aliases shows duplicate warning (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:      "dummy2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"dummy3"},
					aliases:   []string{"dum"},
				},
			},
			args:     []string{"dummy3", "say", "hello"},
			expected: []string{"hello", "the alias dum is duplicated across plugins: dummy, dummy3"},
		},
		{
			test: "plugin replaces another if former maps to an alias of the latter (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "bars",
					target:  configtypes.TargetK8s,
					aliases: []string{"bar"},
				},
				{
					name:      "bar2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"bar"},
				},
			},
			args:       []string{""},
			expected:   []string{"bar2 commands"},
			unexpected: []string{"bars commands"},
		},
		{
			test: "two plugins sharing aliases does not show duplicate warning if one replaces another (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:      "dummy2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"dummy"},
					aliases:   []string{"dum"},
				},
			},
			args:       []string{"dummy", "say", "hello"},
			expected:   []string{"hello"},
			unexpected: []string{"duplicated across plugins"},
		},
		{
			test: "map to deeper subcommand is valid as long as parent exists (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:      "dummy2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"dummy", "kubernetes dummy"},
				},
			},
			args:     []string{"kubernetes", "dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "k8s-target plugins also maps to kubernetes command group (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:      "dummy2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"dummy"},
				},
			},
			args:     []string{"kubernetes", "dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "k8s-target plugins remapping in kubernetes command group also remap top-level command group (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:      "dummy2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"kubernetes dummy"},
				},
			},
			args:     []string{"dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "map to deeper subcommand is invalid if parent missing (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:      "dummy2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"dummy", "nonexistentprefix dummy"},
				},
			},
			args:            []string{"nonexistentprefix", "dummy"},
			expectedFailure: true,
		},
		{
			test: "one mapped tmc plugin (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:      "dummy2",
					target:    configtypes.TargetTMC,
					invokedAs: []string{"dummy", "mission-control dummy"},
				},
			},
			args:     []string{"mission-control", "dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "two plugins remapped to same location will warn (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:      "dummy2",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"dummy"},
					aliases:   []string{"dum"},
				},
				{
					name:      "dummy3",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"dummy"},
					aliases:   []string{"dum"},
				},
			},
			args:     []string{"dummy", "say", "hello"},
			expected: []string{"hello", "Warning, multiple command groups are being remapped to the same command names : \"dummy, dummy\""},
		},
		{
			// because there is nothing that the mapping would override...
			test: "mapped plugin with supportedContextType, command is not hidden at target when no active context (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:                 "dummy2",
					target:               configtypes.TargetK8s,
					invokedAs:            []string{"dummy"},
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTanzu},
				},
			},
			args:                    []string{"kubernetes"},
			expected:                []string{"dummy2 commands"},
			allowActiveContextCheck: true,
		},
		{
			test: "mapping plugins when conditional on active contexts (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:                 "dummy2",
					target:               configtypes.TargetK8s,
					invokedAs:            []string{"dummy"},
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTanzu},
				},
			},
			activeContextType: configtypes.ContextTypeTanzu,
			args:              []string{},
			expected:          []string{"dummy2 commands"},
			unexpected:        []string{"dummy commands"},
		},
		{
			test: "no mapping if active context not one of supportContextType list (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:                 "dummy2",
					target:               configtypes.TargetK8s,
					invokedAs:            []string{"dummy"},
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTanzu},
				},
			},
			activeContextType:       configtypes.ContextTypeK8s,
			allowActiveContextCheck: true,
			args:                    []string{},
			expected:                []string{"dummy commands"},
			unexpected:              []string{"dummy2 commands"},
		},
		{
			test: "nesting plugin within another plugin is not supported (invokedAs)",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:      "deeper",
					target:    configtypes.TargetK8s,
					invokedAs: []string{"kubernetes dummy deeper"},
				},
			},
			args:     []string{"kubernetes", "dummy", "deeper", "say", "hello"},
			expected: []string{"Remap of plugin into command tree (dummy) associated with another plugin is not supported"},
		},
		// ---------- test mapping with CommandMapEntry directives
		{
			test: "one mapped k8s plugin",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			args:     []string{"dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "once mapped, command using original plugin name is not accessible",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			args:            []string{"dummy2"},
			expectedFailure: true,
		},
		{
			test: "two plugins share aliases shows duplicate warning",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:    "dummy2",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy3",
						},
					},
				},
			},
			args:     []string{"dummy3", "say", "hello"},
			expected: []string{"hello", "the alias dum is duplicated across plugins: dummy, dummy3"},
		},
		{
			test: "plugin replaces another if former maps to an alias of the latter",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "bars",
					target:  configtypes.TargetK8s,
					aliases: []string{"bar"},
				},
				{
					name:   "bar2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "bar",
						},
					},
				},
			},
			args:       []string{""},
			expected:   []string{"bar2 commands"},
			unexpected: []string{"bars commands"},
		},
		{
			test: "two plugins sharing aliases does not show duplicate warning if one replaces another",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:    "dummy2",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			args:       []string{"dummy", "say", "hello"},
			expected:   []string{"hello"},
			unexpected: []string{"duplicated across plugins"},
		},
		{
			test: "two plugins sharing aliases does not show duplicate warning if one is explicitly removed",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:    "dummy2",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy3",
							Overrides:              "dummy",
						},
					},
				},
			},
			args:       []string{"dummy3", "say", "hello"},
			expected:   []string{"hello"},
			unexpected: []string{"duplicated across plugins"},
		},
		{
			test: "create deeper subcommand via mapping is valid as long as parent exists",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
						{
							DestinationCommandPath: "operations dummy",
						},
					},
				},
			},
			args:     []string{"operations", "dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "k8s-target plugins also maps to kubernetes command group",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			args:     []string{"kubernetes", "dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "k8s-target plugins remapping in kubernetes command group also remap top-level command group",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "kubernetes dummy",
						},
					},
				},
			},
			args:     []string{"dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "map to deeper subcommand is invalid if parent missing",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "nonexistentprefix dummy",
						},
					},
				},
			},
			args:            []string{"nonexistentprefix", "dummy"},
			expectedFailure: true,
		},
		{
			test: "one mapped tmc plugin",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy2",
					target: configtypes.TargetTMC,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "mission-control dummy",
						},
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			args:     []string{"mission-control", "dummy", "say", "hello"},
			expected: []string{"hello"},
		},
		{
			test: "two plugins remapped to same location will warn",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy2",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
				{
					name:    "dummy3",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			args:     []string{"dummy", "say", "hello"},
			expected: []string{"hello", "Warning, multiple command groups are being remapped to the same command names : \"dummy, dummy\""},
		},
		{
			test: "mapped plugin with supportedContextType, command is not hidden even with no active context type",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:                 "dummy2",
					target:               configtypes.TargetK8s,
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTanzu},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			args:       []string{"kubernetes"},
			expected:   []string{"kubernetes", "dummy2 commands"},
			unexpected: []string{"No plugins are currently installed for \"kubernetes\""},
		},
		// Only works for the Kubernetes target which is now always hidden
		// {
		// 	test: "mapped plugin is only one for target, top level should show target",
		// 	pluginVariants: []fakePluginRemapAttributes{
		// 		{
		// 			name:    "dummy2",
		// 			target:  configtypes.TargetOperations,
		// 			aliases: []string{"dum"},
		// 			commandMap: []plugin.CommandMapEntry{
		// 				{
		// 					DestinationCommandPath: "dummy",
		// 				},
		// 			},
		// 		},
		// 	},
		// 	args:     []string{""},
		// 	expected: []string{"operations"},
		// },
		{
			test: "mapping ok with no active context as long as it does not override any existing",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:                 "dummy2",
					target:               configtypes.TargetTMC,
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTMC},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "mission-control dummy",
						},
					},
				},
			},
			args:       []string{"mission-control"},
			expected:   []string{"dummy2 commands"},
			unexpected: []string{"No plugins are currently installed for \"mission-control\""},
		},
		{
			test: "mapping plugin ok when found active context type",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:                 "dummy2",
					target:               configtypes.TargetK8s,
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTanzu},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			activeContextType:       configtypes.ContextTypeTanzu,
			allowActiveContextCheck: true,
			args:                    []string{},
			expected:                []string{"dummy2 commands"},
			unexpected:              []string{"dummy commands"},
		},
		{
			test: "mapping ok when feature flag not true, regardless of active context",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:                 "dummy2",
					target:               configtypes.TargetK8s,
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTanzu},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			activeContextType:       configtypes.ContextTypeK8s,
			allowActiveContextCheck: false,
			args:                    []string{},
			expected:                []string{"dummy2 commands"},
			unexpected:              []string{"dummy commands"},
		},
		{
			test: "when feature flag set, no mapping if active context not one of supportContextType list",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:                 "dummy2",
					target:               configtypes.TargetK8s,
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTanzu},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			activeContextType:       configtypes.ContextTypeK8s,
			allowActiveContextCheck: true,
			args:                    []string{},
			expected:                []string{"dummy commands"},
			unexpected:              []string{"dummy2 commands"},
		},
		{
			test: "no mapping of entry if active context type is not among its requiredContextType list",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:    "dummy2",
					target:  configtypes.TargetGlobal,
					aliases: []string{"dum"},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "conditionally-mapped",
							RequiredContextType:    []configtypes.ContextType{configtypes.ContextTypeTanzu},
						},
						{
							DestinationCommandPath: "always-mapped",
						},
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			activeContextType: configtypes.ContextTypeK8s,
			args:              []string{},
			expected:          []string{"always-mapped", "dummy2 commands"},
			unexpected:        []string{"conditionally-mapped", "dummy commands"},
		},
		{
			test: "apply mapping of entry if active context type is among its requiredContextType list",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:    "dummy2",
					target:  configtypes.TargetGlobal,
					aliases: []string{"dum"},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "conditionally-mapped",
							RequiredContextType:    []configtypes.ContextType{configtypes.ContextTypeTanzu},
						},
						{
							DestinationCommandPath: "always-mapped",
						},
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			activeContextType: configtypes.ContextTypeTanzu,
			args:              []string{},
			expected:          []string{"conditionally-mapped", "always-mapped", "dummy2 commands"},
			unexpected:        []string{"dummy commands"},
		},
		{
			test: "supportedContextType has no bearing on effect of RequiredContextType",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:                 "dummy2",
					target:               configtypes.TargetGlobal,
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTanzu},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "conditionally-mapped",
							RequiredContextType:    []configtypes.ContextType{configtypes.ContextTypeTanzu},
						},
						{
							DestinationCommandPath: "always-mapped",
						},
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			activeContextType: configtypes.ContextTypeK8s,
			args:              []string{},
			expected:          []string{"always-mapped", "dummy2 commands"},
			unexpected:        []string{"conditionally-mapped", "dummy commands"},
		},
		{
			test: "when feature flag is set supportedContextType takes precedence over RequiredContextType",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:                 "dummy2",
					target:               configtypes.TargetGlobal,
					aliases:              []string{"dum"},
					supportedContextType: []configtypes.ContextType{configtypes.ContextTypeTanzu},
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "conditionally-mapped",
							RequiredContextType:    []configtypes.ContextType{configtypes.ContextTypeK8s},
						},
						{
							DestinationCommandPath: "not-always-mapped",
						},
						{
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			activeContextType:       configtypes.ContextTypeK8s,
			allowActiveContextCheck: true,
			args:                    []string{},
			expected:                []string{"dummy commands"},
			unexpected:              []string{"conditionally-mapped", "not-always-mapped", "dummy2 commands"},
		},
		{
			test: "nesting plugin within another plugin is not supported",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:   "deeper",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "kubernetes dummy deeper",
						},
					},
				},
			},
			args:     []string{"kubernetes", "dummy", "deeper", "say", "hello"},
			expected: []string{"Remap of plugin into command tree (dummy) associated with another plugin is not supported"},
		},
		// --- Command level mapping tests
		{
			test: "command-level: plugin command is mapped to top level appears at top level",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "topshout",
							SourceCommandPath:      "shout",
							Description:            "extracted shout command",
						},
					},
				},
			},
			args:     []string{},
			expected: []string{"topshout", "extracted shout command"},
		},
		{
			test: "command-level: plugin command mapped to top level fails when not invoked with destination command name",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "topshout",
							SourceCommandPath:      "shout",
							Description:            "extracted shout command",
						},
					},
				},
			},
			args:            []string{"shout", "hello"},
			expectedFailure: true,
		},
		{
			test: "command-level: plugin command is mapped to top level fails when not using destination command",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "topshout",
							SourceCommandPath:      "shout",
							Description:            "extracted shout command",
						},
					},
				},
			},
			args:     []string{"topshout", "hello"},
			expected: []string{"hello !!"},
		},
		{
			test: "command-level: invocation details available when plugin command is mapped to top level",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "invoke-info",
							SourceCommandPath:      "show-invoke-context",
							Description:            "Shows invocation details",
						},
					},
				},
			},
			args:     []string{"invoke-info", "arg1", "arg2"},
			expected: []string{"args = (show-invoke-context arg1 arg2), context is ():(invoke-info):(show-invoke-context)"},
		},
		{
			test: "command-level: invocation details available when plugin command is mapped to top level and invoked via a target",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "invoke-info",
							SourceCommandPath:      "show-invoke-context",
							Description:            "Shows invocation details",
						},
					},
				},
			},
			args:     []string{"kubernetes", "invoke-info", "arg1", "arg2"},
			expected: []string{"args = (show-invoke-context arg1 arg2), context is (kubernetes):(invoke-info):(show-invoke-context)"},
		},
		{
			test: "command-level: invocation details available when plugin command is mapped to deeper level",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "plugin source invoke-info",
							SourceCommandPath:      "show-invoke-context",
							Description:            "Shows invocation details",
						},
					},
				},
			},
			args:     []string{"plugin", "source", "invoke-info", "arg1", "arg2"},
			expected: []string{"args = (show-invoke-context arg1 arg2), context is (plugin source):(invoke-info):(show-invoke-context)"},
		},
		{
			test: "command-level: subcommand replaces another top-level command group",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:   "dummy2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
							SourceCommandPath:      "show-invoke-context",
							Description:            "Shows invocation details",
						},
					},
				},
			},
			args:     []string{"dummy", "arg1", "arg2"},
			expected: []string{"args = (show-invoke-context arg1 arg2), context is ():(dummy):(show-invoke-context)"},
		},
		{
			test: "command-level: subcommand replaces another subcommand of another plugin group is not allowed",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:   "dummy2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy say",
							SourceCommandPath:      "show-invoke-context",
							Description:            "Shows invocation details",
						},
					},
				},
			},
			args:     []string{"dummy", "arg1", "arg2"},
			expected: []string{"Remap of plugin into command tree (dummy) associated with another plugin is not supported"},
		},
		{
			test: "command-level: subcommand map entry's aliases are used",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:    "dummy",
					target:  configtypes.TargetK8s,
					aliases: []string{"dum"},
				},
				{
					name:   "dummy2",
					target: configtypes.TargetK8s,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "dummy",
							SourceCommandPath:      "show-invoke-context",
							Description:            "Shows invocation details",
							Aliases:                []string{"sic"},
						},
					},
				},
			},
			args:     []string{"sic", "arg1", "arg2"},
			expected: []string{"args = (show-invoke-context arg1 arg2), context is ():(dummy):(show-invoke-context)"},
		},
		{
			test:     "when nothing under platform-engineering command group",
			args:     []string{"tpe"},
			expected: []string{"No plugins are currently installed for \"platform-engineering\""},
		},
		{
			test: "mapping ok with under command platform-engineering command group",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy2",
					target: configtypes.TargetGlobal,
					commandMap: []plugin.CommandMapEntry{
						{
							DestinationCommandPath: "platform-engineering dummy",
						},
					},
				},
			},
			args:       []string{"tpe"},
			expected:   []string{"dummy2 commands"},
			unexpected: []string{"No plugins are currently installed for \"platform-engineering\""},
		},
		// --- Command level mapping tests for the "tanzu help" command
		{
			test: "help command for one unmapped plugin",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy",
					target: configtypes.TargetGlobal,
				},
			},
			args: []string{"help", "dummy"},
			// Help output for the dummy plugin
			expected: []string{"dummy functionality\n\nUsage:\n  tanzu dummy [command]"},
		},
		{
			test: "help command for one mapped plugin",
			pluginVariants: []fakePluginRemapAttributes{
				{
					name:   "dummy2",
					target: configtypes.TargetGlobal,
					commandMap: []plugin.CommandMapEntry{
						{
							SourceCommandPath:      "show-invoke-context",
							DestinationCommandPath: "dummy",
						},
					},
				},
			},
			args: []string{"help", "dummy"},
			// Output for the show-invoke-context command which prints the passed arguments
			// and environment.  So we confirm that the show-invoke-context was called, passed -h
			// and received the remapping environment.
			expected: []string{"args = (show-invoke-context -h), context is ():(dummy):(show-invoke-context)"},
		},
	}

	for _, spec := range tests {
		env := setupTestCLIEnvironment(t)
		defer tearDownTestCLIEnvironment(env)

		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			configErr := config.ConfigureFeatureFlags(map[string]bool{
				"features.global.plugin-override-on-active-context-type": spec.allowActiveContextCheck}, config.SkipIfExists())
			assert.Nil(configErr)

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

			if spec.activeContextType != "" {
				err = config.SetContext(&configtypes.Context{Name: "test-context", ContextType: spec.activeContextType}, true)
				assert.Nil(err)
			}

			for _, p := range spec.pluginVariants {
				pi := setupFakePluginInfo(p, env.cacheDir)
				err = setupFakePlugin(env.cacheDir, pi.Name, pi.Version, pi.Group, 0, pi.Target, 0, pi.Hidden, pi.Aliases)

				assert.Nil(err)

				cc, err := catalog.NewContextCatalogUpdater("")
				assert.Nil(err)
				err = cc.Upsert(pi)
				cc.Unlock()
				assert.Nil(err)
			}

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			// To be able to test the "tanzu help" command, we need to set the os.Args
			// instead of using the cobra command's SetArgs method.
			// rootCmd.SetArgs(spec.args)
			// See getHelpArguments() in pkg/cli/plugin_cmd.go
			os.Args = append([]string{"tanzu"}, spec.args...)

			os.Unsetenv("TANZU_CLI_INVOKED_GROUP")
			os.Unsetenv("TANZU_CLI_INVOKED_COMMAND")
			os.Unsetenv("TANZU_CLI_COMMAND_MAPPED_FROM")
			err = rootCmd.Execute()

			w.Close()
			got := <-c

			assert.Equal(spec.expectedFailure, err != nil)
			for _, expected := range spec.expected {
				assert.Contains(string(got), expected)
			}
			for _, unexpected := range spec.unexpected {
				assert.NotContains(string(got), unexpected)
			}
		})
	}
}

func TestUpdateConfigWithTanzuCSPIssuer(t *testing.T) {
	env := setupTestCLIEnvironment(t)
	defer tearDownTestCLIEnvironment(env)

	// Set up mocks for the centralConfigIssuerUpdateFlagGetter and cliContextUpdateStatusGetter functions
	centralConfigIssuerUpdateFlagGetter := func() bool {
		return true
	}
	cliContextUpdateStatusGetter := func(string, interface{}) error {
		return nil
	}

	// Test TMC context with the VCSP staging Issuer, is updated to TCSP staging issuer, and refresh token and expiration time are unchanged
	missionContextWithVCSPStagingIssuer := createFakeContextWithAuthData("mission-context-with-vcsp-staging-issuer", csp.StgIssuer,
		"refresh-token-should-not-be-modified", common.APITokenType, configtypes.ContextTypeTMC)
	err := config.SetContext(missionContextWithVCSPStagingIssuer, false)
	assert.NoError(t, err)
	updateConfigWithTanzuCSPIssuer(centralConfigIssuerUpdateFlagGetter, cliContextUpdateStatusGetter)
	gotCtx, err := config.GetContext(missionContextWithVCSPStagingIssuer.Name)
	assert.NoError(t, err)
	assert.Equal(t, csp.StgIssuerTCSP, gotCtx.GlobalOpts.Auth.Issuer)
	assert.Equal(t, missionContextWithVCSPStagingIssuer.GlobalOpts.Auth.RefreshToken, gotCtx.GlobalOpts.Auth.RefreshToken)
	assert.Equal(t, missionContextWithVCSPStagingIssuer.GlobalOpts.Auth.Expiration.Local(), gotCtx.GlobalOpts.Auth.Expiration.Local())

	// Test TMC context with the VCSP prod Issuer, is updated to TCSP prod issuer and refresh token and expiration time are unchanged
	missionContextWithVCSPProdIssuer := createFakeContextWithAuthData("mission-context-with-vcsp-prod-issuer", csp.ProdIssuer,
		"refresh-token-should-not-be-modified", common.APITokenType, configtypes.ContextTypeTMC)
	err = config.SetContext(missionContextWithVCSPProdIssuer, false)
	assert.NoError(t, err)
	updateConfigWithTanzuCSPIssuer(centralConfigIssuerUpdateFlagGetter, cliContextUpdateStatusGetter)

	gotCtx, err = config.GetContext(missionContextWithVCSPProdIssuer.Name)
	assert.NoError(t, err)
	assert.Equal(t, csp.ProdIssuerTCSP, gotCtx.GlobalOpts.Auth.Issuer)
	assert.Equal(t, missionContextWithVCSPProdIssuer.GlobalOpts.Auth.RefreshToken, gotCtx.GlobalOpts.Auth.RefreshToken)
	assert.Equal(t, missionContextWithVCSPProdIssuer.GlobalOpts.Auth.Expiration.Local(), gotCtx.GlobalOpts.Auth.Expiration.Local())

	// Test Tanzu context with the VCSP staging Issuer, is updated to TCSP staging issuer
	// and refresh token and expiration time are unchanged as the token is API token
	tanzuContextWithVCSPStagingIssuer := createFakeContextWithAuthData("tanzu-context-with-tcsp-prod-issuer", csp.StgIssuer,
		"refresh-token-to-be-modified", common.APITokenType, configtypes.ContextTypeTanzu)
	err = config.SetContext(tanzuContextWithVCSPStagingIssuer, false)
	assert.NoError(t, err)
	updateConfigWithTanzuCSPIssuer(centralConfigIssuerUpdateFlagGetter, cliContextUpdateStatusGetter)
	gotCtx, err = config.GetContext(tanzuContextWithVCSPStagingIssuer.Name)
	assert.NoError(t, err)
	assert.Equal(t, csp.StgIssuerTCSP, gotCtx.GlobalOpts.Auth.Issuer)
	assert.Equal(t, tanzuContextWithVCSPStagingIssuer.GlobalOpts.Auth.RefreshToken, gotCtx.GlobalOpts.Auth.RefreshToken)
	assert.Equal(t, tanzuContextWithVCSPStagingIssuer.GlobalOpts.Auth.Expiration.Local(), gotCtx.GlobalOpts.Auth.Expiration.Local())

	// Test Tanzu context with the VCSP prod Issuer, is updated to TCSP prod issuer
	// and refresh token and expiration time are unchanged as the token is API token
	tanzuContextWithVCSPProdIssuer := createFakeContextWithAuthData("tanzu-context-with-tcsp-prod-issuer", csp.ProdIssuer,
		"refresh-token-to-be-modified", common.APITokenType, configtypes.ContextTypeTanzu)
	err = config.SetContext(tanzuContextWithVCSPProdIssuer, false)
	assert.NoError(t, err)
	updateConfigWithTanzuCSPIssuer(centralConfigIssuerUpdateFlagGetter, cliContextUpdateStatusGetter)
	gotCtx, err = config.GetContext(tanzuContextWithVCSPProdIssuer.Name)
	assert.NoError(t, err)
	assert.Equal(t, csp.ProdIssuerTCSP, gotCtx.GlobalOpts.Auth.Issuer)
	assert.Equal(t, tanzuContextWithVCSPProdIssuer.GlobalOpts.Auth.RefreshToken, gotCtx.GlobalOpts.Auth.RefreshToken)
	assert.Equal(t, tanzuContextWithVCSPProdIssuer.GlobalOpts.Auth.Expiration.Local(), gotCtx.GlobalOpts.Auth.Expiration.Local())

	// Test Tanzu context with the VCSP prod Issuer, is updated to TCSP prod issuer
	// and refresh token and expiration time are invalidated as the token is id-token(interactive login token)
	tanzuContextWithVCSPProdIssuerWithIDTokenType := createFakeContextWithAuthData("tanzu-context-with-tcsp-prod-issuer-with-IDToken",
		csp.ProdIssuer, "refresh-token-to-be-modified", common.IDTokenType, configtypes.ContextTypeTanzu)
	err = config.SetContext(tanzuContextWithVCSPProdIssuerWithIDTokenType, false)
	assert.NoError(t, err)
	updateConfigWithTanzuCSPIssuer(centralConfigIssuerUpdateFlagGetter, cliContextUpdateStatusGetter)
	gotCtx, err = config.GetContext(tanzuContextWithVCSPProdIssuerWithIDTokenType.Name)
	assert.NoError(t, err)
	assert.Equal(t, csp.ProdIssuerTCSP, gotCtx.GlobalOpts.Auth.Issuer)
	assert.Equal(t, "Invalid", gotCtx.GlobalOpts.Auth.RefreshToken)
	assert.True(t, gotCtx.GlobalOpts.Auth.Expiration.Before(time.Now().Local()))

	// Test Tanzu context with the TCSP prod Issuer is unchanged
	tanzuContextWithTCSPProdIssuerWithIDTokenType := createFakeContextWithAuthData("tanzu-context-with-tcsp-prod-issuer-with-IDToken",
		csp.ProdIssuerTCSP, "refresh-token-to-be-modified", common.IDTokenType, configtypes.ContextTypeTanzu)
	err = config.SetContext(tanzuContextWithTCSPProdIssuerWithIDTokenType, false)
	assert.NoError(t, err)
	updateConfigWithTanzuCSPIssuer(centralConfigIssuerUpdateFlagGetter, cliContextUpdateStatusGetter)
	gotCtx, err = config.GetContext(tanzuContextWithTCSPProdIssuerWithIDTokenType.Name)
	assert.NoError(t, err)
	assert.Equal(t, tanzuContextWithTCSPProdIssuerWithIDTokenType.GlobalOpts.Auth.Issuer, gotCtx.GlobalOpts.Auth.Issuer)
	assert.Equal(t, tanzuContextWithTCSPProdIssuerWithIDTokenType.GlobalOpts.Auth.RefreshToken, gotCtx.GlobalOpts.Auth.RefreshToken)
	assert.Equal(t, tanzuContextWithTCSPProdIssuerWithIDTokenType.GlobalOpts.Auth.Expiration.Local(), gotCtx.GlobalOpts.Auth.Expiration.Local())
}

func createFakeContextWithAuthData(ctxName, issuer, refreshToken, tokenType string, ctxType configtypes.ContextType) *configtypes.Context {
	return &configtypes.Context{
		Name:        ctxName,
		ContextType: ctxType,
		GlobalOpts: &configtypes.GlobalServer{
			Auth: configtypes.GlobalServerAuth{
				Issuer:       issuer,
				UserName:     "test-user-name",
				AccessToken:  "access-token",
				RefreshToken: refreshToken,
				Expiration:   time.Now().Local().Add(5 * time.Second),
				Type:         tokenType,
			},
		},
	}
}
