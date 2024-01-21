// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
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
		dir, err := os.MkdirTemp("", "tanzu-cli-root-cmd")
		assert.Nil(t, err)
		defer os.RemoveAll(dir)
		os.Setenv("TEST_CUSTOM_CATALOG_CACHE_DIR", dir)

		// Setup a temporary configuration.  This means no
		// plugin source will be available which will prevent
		// trying to install the essential plugins.
		// This speeds up the test.
		configFile, _ := os.CreateTemp("", "config")
		os.Setenv("TANZU_CONFIG", configFile.Name())
		defer os.RemoveAll(configFile.Name())

		configFileNG, _ := os.CreateTemp("", "config_ng")
		os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		defer os.RemoveAll(configFileNG.Name())

		os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
		os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")

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
				InstallationPath: filepath.Join(dir, fmt.Sprintf("%s_%s", spec.plugin, string(spec.target))),
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
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
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
			unexpected: []string{"Target", "kubernetes", "mission-control"},
		},
		{
			test:                   "top run for only k8s empty",
			args:                   []string{},
			installedPluginTargets: []configtypes.Target{configtypes.TargetTMC},
			expected:               []string{"Target", "mission-control"},
			unexpected:             []string{"kubernetes"},
		},
		{
			test:                   "top run for only tmc empty",
			args:                   []string{},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s},
			expected:               []string{"Target", "kubernetes"},
			unexpected:             []string{"mission-control"},
		},
		{
			test:                   "top run for no empty targets",
			args:                   []string{},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s, configtypes.TargetTMC},
			expected:               []string{"Target", "kubernetes", "mission-control"},
		},
		{
			test:       "top help for all empty targets",
			args:       []string{"-h"},
			unexpected: []string{"Target", "kubernetes", "mission-control"},
		},
		{
			test:                   "top help for only k8s empty",
			args:                   []string{"-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetTMC},
			expected:               []string{"Target", "mission-control"},
			unexpected:             []string{"kubernetes"},
		},
		{
			test:                   "top help for only tmc empty",
			args:                   []string{"-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s},
			expected:               []string{"Target", "kubernetes"},
			unexpected:             []string{"mission-control"},
		},
		{
			test:                   "top help for no empty targets",
			args:                   []string{"-h"},
			installedPluginTargets: []configtypes.Target{configtypes.TargetK8s, configtypes.TargetTMC},
			expected:               []string{"Target", "kubernetes", "mission-control"},
		},
		// ========================
		// Tests for the k8s target
		// ========================
		{
			test: "help for k8s target run when empty",
			args: []string{"k8s"},
			expected: []string{
				"Note: No plugins are currently installed for the \"kubernetes\" target",
				"Tanzu CLI plugins that target a Kubernetes cluster",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for k8s target help when empty",
			args: []string{"k8s", "-h"},
			expected: []string{
				"Tanzu CLI plugins that target a Kubernetes cluster",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for k8s target invalid when empty",
			args: []string{"k8s", "invalid"},
			expected: []string{
				"Note: No plugins are currently installed for the \"kubernetes\" target",
				"Tanzu CLI plugins that target a Kubernetes cluster",
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
				"Tanzu CLI plugins that target a Kubernetes cluster",
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
				"Tanzu CLI plugins that target a Kubernetes cluster",
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
				"Tanzu CLI plugins that target a Kubernetes cluster",
				"Note: No plugins are currently installed for the \"kubernetes\" target",
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
				"Tanzu CLI plugins that target a Kubernetes cluster",
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
				"Note: No plugins are currently installed for the \"mission-control\" target",
				"Tanzu CLI plugins that target a Tanzu Mission Control endpoint",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for tmc target help when empty",
			args: []string{"tmc", "-h"},
			expected: []string{
				"Tanzu CLI plugins that target a Tanzu Mission Control endpoint",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
		{
			test: "help for tmc target invalid when empty",
			args: []string{"tmc", "invalid"},
			expected: []string{
				"Note: No plugins are currently installed for the \"mission-control\" target",
				"Tanzu CLI plugins that target a Tanzu Mission Control endpoint",
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
				"Tanzu CLI plugins that target a Tanzu Mission Control endpoint",
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
				"Tanzu CLI plugins that target a Tanzu Mission Control endpoint",
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
				"Tanzu CLI plugins that target a Tanzu Mission Control endpoint",
				"Note: No plugins are currently installed for the \"mission-control\" target",
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
				"Tanzu CLI plugins that target a Tanzu Mission Control endpoint",
			},
			unexpected: []string{
				"Available command groups",
			},
		},
	}

	for _, spec := range tests {
		dir, err := os.MkdirTemp("", "tanzu-cli-root-cmd")
		assert.Nil(t, err)
		defer os.RemoveAll(dir)
		os.Setenv("TEST_CUSTOM_CATALOG_CACHE_DIR", dir)

		// Setup a temporary configuration.  This means no
		// plugin source will be available which will prevent
		// trying to install the essential plugins.
		// This speeds up the test.
		configFile, _ := os.CreateTemp("", "config")
		os.Setenv("TANZU_CONFIG", configFile.Name())
		defer os.RemoveAll(configFile.Name())

		configFileNG, _ := os.CreateTemp("", "config_ng")
		os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		defer os.RemoveAll(configFileNG.Name())

		os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
		os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")

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

			assert.Nil(err)

			for _, target := range spec.installedPluginTargets {
				pluginName := "dummy"
				pi := &cli.PluginInfo{
					Name:             pluginName,
					Description:      pluginName,
					Group:            plugin.SystemCmdGroup,
					Version:          "v0.1.0",
					Aliases:          []string{},
					Hidden:           false,
					InstallationPath: filepath.Join(dir, fmt.Sprintf("%s_%s", pluginName, string(target))),
					Target:           target,
				}
				err = setupFakePlugin(dir, pi.Name, pi.Version, pi.Group, 0, pi.Target, 0, pi.Hidden, pi.Aliases)
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
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
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
	os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
	os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")
	defer os.RemoveAll(configFileNG.Name())

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

	// Cleanup
	os.Unsetenv("TANZU_CONFIG")
	os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
	os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
	os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
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
