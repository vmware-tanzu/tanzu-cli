// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

func TestPluginList(t *testing.T) {
	tests := []struct {
		test            string
		plugins         []string
		versions        []string
		targets         []configtypes.Target
		args            []string
		expected        string
		expectedFailure bool
	}{
		{
			test:            "no '--local' option",
			args:            []string{"plugin", "list", "--local", "someDirectory"},
			expectedFailure: true,
			expected:        "unknown flag: --local",
		},
		{
			test:            "With empty config file(no discovery sources added) and no plugins installed",
			plugins:         []string{},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION TARGET VERSION STATUS",
		},
		{
			test:            "With empty config file(no discovery sources added) and when one additional plugin installed",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			targets:         []configtypes.Target{configtypes.TargetK8s},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION TARGET VERSION STATUS foo some foo description kubernetes v0.1.0 installed",
		},
		{
			test:            "With empty config file(no discovery sources added) and when more than one plugin is installed",
			plugins:         []string{"foo", "bar"},
			versions:        []string{"v0.1.0", "v0.2.0"},
			targets:         []configtypes.Target{configtypes.TargetTMC, configtypes.TargetK8s},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION TARGET VERSION STATUS bar some bar description kubernetes v0.2.0 installed foo some foo description mission-control v0.1.0 installed",
		},
		{
			test:            "when json output is requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			targets:         []configtypes.Target{configtypes.TargetK8s},
			args:            []string{"plugin", "list", "-o", "json"},
			expectedFailure: false,
			expected:        `[ { "context": "", "description": "some foo description", "name": "foo", "status": "installed", "target": "kubernetes", "version": "v0.1.0" } ]`,
		},
		{
			test:            "when yaml output is requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			targets:         []configtypes.Target{configtypes.TargetK8s},
			args:            []string{"plugin", "list", "-o", "yaml"},
			expectedFailure: false,
			expected:        `- context: "" description: some foo description name: foo status: installed target: kubernetes version: v0.1.0`,
		},
		{
			test:            "plugin describe json output requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			targets:         []configtypes.Target{configtypes.TargetK8s},
			args:            []string{"plugin", "describe", "foo", "-o", "json"},
			expectedFailure: false,
			expected:        `[ { "description": "some foo description", "installationpath": "%v", "name": "foo", "status": "installed", "target": "kubernetes", "version": "v0.1.0" } ]`,
		},
	}

	for _, spec := range tests {
		tkgConfigFile, _ := os.CreateTemp("", "config")
		os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())
		defer os.RemoveAll(tkgConfigFile.Name())

		tkgConfigFileNG, _ := os.CreateTemp("", "config_ng")
		os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
		defer os.RemoveAll(tkgConfigFileNG.Name())

		dir, err := os.MkdirTemp("", "tanzu-cli-root-cmd")
		assert.Nil(t, err)
		defer os.RemoveAll(dir)
		os.Setenv("TEST_CUSTOM_CATALOG_CACHE_DIR", dir)
		os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
		os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")

		// Always turn on the context feature
		featureArray := strings.Split(constants.FeatureContextCommand, ".")
		err = config.SetFeature(featureArray[1], featureArray[2], "true")
		assert.Nil(t, err)

		var completionType uint8
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)
			cc, err := catalog.NewContextCatalogUpdater("")
			assert.Nil(err)
			pluginInstallationPath := ""
			for i, pluginName := range spec.plugins {
				err = setupFakePlugin(dir, pluginName, spec.versions[i], plugin.SystemCmdGroup, completionType, spec.targets[i], 1, false, []string{pluginName[:2]})
				assert.Nil(err)
				pluginInstallationPath = filepath.Join(common.DefaultPluginRoot, pluginName)
				pi := &cli.PluginInfo{
					Name:             pluginName,
					Description:      fmt.Sprintf("some %s description", pluginName),
					Group:            plugin.SystemCmdGroup,
					Aliases:          []string{pluginName[:2]},
					Version:          spec.versions[i],
					InstallationPath: pluginInstallationPath,
					Status:           common.PluginStatusInstalled,
					Target:           spec.targets[i],
				}
				assert.Nil(err)
				err = cc.Upsert(pi)
				assert.Nil(err)
			}
			cc.Unlock()

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)
			b := bytes.NewBufferString("")
			rootCmd.SetOut(b)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)

			if spec.expected != "" {
				if spec.expectedFailure {
					assert.Contains(err.Error(), spec.expected)
				} else {
					got, err := io.ReadAll(b)
					assert.Nil(err)

					if strings.Contains(spec.expected, "installationpath") {
						spec.expected = fmt.Sprintf(spec.expected, pluginInstallationPath)
					}

					// whitespace-agnostic match
					assert.Contains(strings.Join(strings.Fields(string(got)), " "), spec.expected)
				}
			}
		})
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
	}
}

func TestDeletePlugin(t *testing.T) {
	tests := []struct {
		test             string
		plugins          []string
		versions         []string
		targets          []configtypes.Target
		args             []string
		expectedErrorMsg string
		expectedFailure  bool
	}{
		{
			test:             "delete an uninstalled plugin",
			plugins:          []string{},
			versions:         []string{"v0.1.0"},
			args:             []string{"plugin", "delete", "foo", "-y"},
			expectedFailure:  true,
			expectedErrorMsg: "unable to find plugin 'foo'",
		},
		{
			test:            "delete an installed plugin",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			targets:         []configtypes.Target{configtypes.TargetK8s},
			args:            []string{"plugin", "delete", "foo", "-y"},
			expectedFailure: false,
		},
	}

	for _, spec := range tests {
		dir, err := os.MkdirTemp("", "tanzu-cli-root-cmd")
		assert.Nil(t, err)
		defer os.RemoveAll(dir)
		os.Setenv("TEST_CUSTOM_CATALOG_CACHE_DIR", dir)
		os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
		os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")
		var completionType uint8
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)
			cc, err := catalog.NewContextCatalogUpdater("")
			assert.Nil(err)

			for i, pluginName := range spec.plugins {
				err = setupFakePlugin(dir, pluginName, spec.versions[i], plugin.SystemCmdGroup, completionType, spec.targets[i], 1, false, []string{pluginName[:2]})
				assert.Nil(err)
				pi := &cli.PluginInfo{
					Name:             pluginName,
					Description:      fmt.Sprintf("some %s description", pluginName),
					Group:            plugin.SystemCmdGroup,
					Aliases:          []string{pluginName[:2]},
					Version:          spec.versions[i],
					InstallationPath: filepath.Join(common.DefaultPluginRoot, pluginName),
				}
				assert.Nil(err)
				err = cc.Upsert(pi)
				assert.Nil(err)
			}
			cc.Unlock()
			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)
			if spec.expectedErrorMsg != "" {
				assert.Contains(err.Error(), spec.expectedErrorMsg)
			}
			if !spec.expectedFailure {
				pi, exists := cc.Get(spec.args[2])
				assert.Equal(exists, true)
				_, err := os.Stat(pi.InstallationPath)
				assert.Equal(os.IsNotExist(err), true)
			}
		})
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
	}
}

func TestInstallPlugin(t *testing.T) {
	tests := []struct {
		test             string
		args             []string
		expectedErrorMsg string
		expectedFailure  bool
	}{
		{
			test:             "need plugin name if no group",
			args:             []string{"plugin", "install"},
			expectedFailure:  true,
			expectedErrorMsg: "missing plugin name as an argument",
		},
		{
			test:             "no 'all' option",
			args:             []string{"plugin", "install", "all"},
			expectedFailure:  true,
			expectedErrorMsg: "the 'all' argument can only be used with the '--group' flag",
		},
		{
			test:             "invalid target",
			args:             []string{"plugin", "install", "--target", "invalid", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: invalidTargetMsg,
		},
		{
			test:             "no --group and --local-source together",
			args:             []string{"plugin", "install", "--group", "testgroup", "--local-source", "./", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: "if any flags in the group [group local-source] are set none of the others can be",
		},
		{
			test:             "no --group and --target together",
			args:             []string{"plugin", "install", "--group", "testgroup", "--target", "k8s", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: "if any flags in the group [group target] are set none of the others can be",
		},
		{
			test:             "no --group and --version together",
			args:             []string{"plugin", "install", "--group", "testgroup", "--version", "v1.1.1", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: "if any flags in the group [group version] are set none of the others can be",
		},
	}

	assert := assert.New(t)

	tkgConfigFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

	tkgConfigFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
	os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
	os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")

	featureArray := strings.Split(constants.FeatureContextCommand, ".")
	err = config.SetFeature(featureArray[1], featureArray[2], "true")
	assert.Nil(err)

	defer func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
	}()

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)
			if spec.expectedErrorMsg != "" {
				assert.Contains(err.Error(), spec.expectedErrorMsg)
			}
		})
	}
}

func TestUpgradePlugin(t *testing.T) {
	tests := []struct {
		test             string
		args             []string
		expectedErrorMsg string
		expectedFailure  bool
	}{
		{
			test:             "invalid target",
			args:             []string{"plugin", "upgrade", "--target", "invalid", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: invalidTargetMsg,
		},
	}

	assert := assert.New(t)

	tkgConfigFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

	tkgConfigFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
	os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
	os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")

	featureArray := strings.Split(constants.FeatureContextCommand, ".")
	err = config.SetFeature(featureArray[1], featureArray[2], "true")
	assert.Nil(err)

	defer func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
	}()

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)
			if spec.expectedErrorMsg != "" {
				assert.Contains(err.Error(), spec.expectedErrorMsg)
			}
		})
	}
}
