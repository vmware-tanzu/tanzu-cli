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
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

func TestPluginList(t *testing.T) {
	tests := []struct {
		test               string
		centralRepoFeature bool
		plugins            []string
		versions           []string
		args               []string
		expected           string
		expectedFailure    bool
	}{
		{
			test:            "With empty config file(no discovery sources added) and no plugins installed",
			plugins:         []string{},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION TARGET DISCOVERY VERSION STATUS",
		},
		{
			test:            "With empty config file(no discovery sources added) and when one additional plugin installed",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION TARGET DISCOVERY VERSION STATUS foo some foo description v0.1.0 installed",
		},
		{
			test:            "With empty config file(no discovery sources added) and when more than one plugin is installed",
			plugins:         []string{"foo", "bar"},
			versions:        []string{"v0.1.0", "v0.2.0"},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION TARGET DISCOVERY VERSION STATUS bar some bar description v0.2.0 installed foo some foo description v0.1.0 installed",
		},
		{
			test:            "when json output is requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			args:            []string{"plugin", "list", "-o", "json"},
			expectedFailure: false,
			expected:        `[ { "description": "some foo description", "discovery": "", "name": "foo", "scope": "Standalone", "status": "installed", "version": "v0.1.0" } ]`,
		},
		{
			test:            "when yaml output is requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			args:            []string{"plugin", "list", "-o", "yaml"},
			expectedFailure: false,
			expected:        `- description: some foo description discovery: "" name: foo scope: Standalone status: installed version: v0.1.0`,
		},
		{
			test:               "no '--local' option with central repo",
			centralRepoFeature: true,
			args:               []string{"plugin", "list", "--local", "someDirectory"},
			expectedFailure:    true,
			expected:           "the '--local' flag does not apply to this command. Please use 'tanzu plugin search --local'",
		},
		{
			test:               "With empty config file(no discovery sources added) and no plugins installed with central repo",
			centralRepoFeature: true,
			plugins:            []string{},
			args:               []string{"plugin", "list"},
			expectedFailure:    false,
			expected:           "NAME DESCRIPTION TARGET VERSION STATUS",
		},
		{
			test:               "With empty config file(no discovery sources added) and when one additional plugin installed with central repo",
			centralRepoFeature: true,
			plugins:            []string{"foo"},
			versions:           []string{"v0.1.0"},
			args:               []string{"plugin", "list"},
			expectedFailure:    false,
			expected:           "NAME DESCRIPTION TARGET VERSION STATUS foo some foo description v0.1.0 installed",
		},
		{
			test:               "With empty config file(no discovery sources added) and when more than one plugin is installed with central repo",
			centralRepoFeature: true,
			plugins:            []string{"foo", "bar"},
			versions:           []string{"v0.1.0", "v0.2.0"},
			args:               []string{"plugin", "list"},
			expectedFailure:    false,
			expected:           "NAME DESCRIPTION TARGET VERSION STATUS bar some bar description v0.2.0 installed foo some foo description v0.1.0 installed",
		},
		{
			test:               "when json output is requested with central repo",
			centralRepoFeature: true,
			plugins:            []string{"foo"},
			versions:           []string{"v0.1.0"},
			args:               []string{"plugin", "list", "-o", "json"},
			expectedFailure:    false,
			expected:           `[ { "description": "some foo description", "name": "foo", "status": "installed", "target": "", "version": "v0.1.0" } ]`,
		},
		{
			test:               "when yaml output is requested with central repo",
			centralRepoFeature: true,
			plugins:            []string{"foo"},
			versions:           []string{"v0.1.0"},
			args:               []string{"plugin", "list", "-o", "yaml"},
			expectedFailure:    false,
			expected:           `- description: some foo description name: foo status: installed target: "" version: v0.1.0`,
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

		// Always turn on the context feature
		featureArray := strings.Split(constants.FeatureContextCommand, ".")
		err = config.SetFeature(featureArray[1], featureArray[2], "true")
		assert.Nil(t, err)

		// Set the Central Repository feature
		if spec.centralRepoFeature {
			featureArray := strings.Split(constants.FeatureCentralRepository, ".")
			err := config.SetFeature(featureArray[1], featureArray[2], "true")
			assert.Nil(t, err)
		}

		var completionType uint8
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)
			cc, err := catalog.NewContextCatalog("")
			assert.Nil(err)
			for i, pluginName := range spec.plugins {
				err = setupFakePlugin(dir, pluginName, spec.versions[i], plugin.SystemCmdGroup, completionType, 1, false, []string{pluginName[:2]})
				assert.Nil(err)
				pi := &cli.PluginInfo{
					Name:             pluginName,
					Description:      fmt.Sprintf("some %s description", pluginName),
					Group:            plugin.SystemCmdGroup,
					Aliases:          []string{pluginName[:2]},
					Version:          spec.versions[i],
					InstallationPath: filepath.Join(common.DefaultPluginRoot, pluginName),
					Status:           common.PluginStatusInstalled,
				}
				assert.Nil(err)
				err = cc.Upsert(pi)
				assert.Nil(err)
			}

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

					// whitespace-agnostic match
					assert.Contains(strings.Join(strings.Fields(string(got)), " "), spec.expected)
				}
			}
		})
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
	}
}

func TestDeletePlugin(t *testing.T) {
	tests := []struct {
		test             string
		plugins          []string
		versions         []string
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
			args:            []string{"plugin", "delete", "foo", "-y"},
			expectedFailure: false,
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
			cc, err := catalog.NewContextCatalog("")
			assert.Nil(err)

			for i, pluginName := range spec.plugins {
				err = setupFakePlugin(dir, pluginName, spec.versions[i], plugin.SystemCmdGroup, completionType, 1, false, []string{pluginName[:2]})
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
	}
}

func TestInstallPlugin(t *testing.T) {
	tests := []struct {
		test               string
		centralRepoFeature string
		args               []string
		expectedErrorMsg   string
		expectedFailure    bool
	}{
		{
			test:               "no 'all' option with central repo",
			centralRepoFeature: "true",
			args:               []string{"plugin", "install", "all"},
			expectedFailure:    true,
			expectedErrorMsg:   "the 'all' argument can only be used with the --group or --local flags",
		},
		{
			test:               "invalid target",
			centralRepoFeature: "true",
			args:               []string{"plugin", "install", "--target", "invalid", "myplugin"},
			expectedFailure:    true,
			expectedErrorMsg:   "invalid target specified. Please specify correct value of `--target` or `-t` flag from 'kubernetes/k8s/mission-control/tmc'",
		},
	}

	assert := assert.New(t)

	tkgConfigFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

	tkgConfigFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())

	featureArray := strings.Split(constants.FeatureContextCommand, ".")
	err = config.SetFeature(featureArray[1], featureArray[2], "true")
	assert.Nil(err)

	defer func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
	}()

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			// Set the Central Repository feature
			featureArray := strings.Split(constants.FeatureCentralRepository, ".")
			err := config.SetFeature(featureArray[1], featureArray[2], spec.centralRepoFeature)
			assert.Nil(err)

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
		test               string
		centralRepoFeature string
		args               []string
		expectedErrorMsg   string
		expectedFailure    bool
	}{
		{
			test:               "invalid target",
			centralRepoFeature: "true",
			args:               []string{"plugin", "upgrade", "--target", "invalid", "myplugin"},
			expectedFailure:    true,
			expectedErrorMsg:   "invalid target specified. Please specify correct value of `--target` or `-t` flag from 'kubernetes/k8s/mission-control/tmc'",
		},
	}

	assert := assert.New(t)

	tkgConfigFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

	tkgConfigFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())

	featureArray := strings.Split(constants.FeatureContextCommand, ".")
	err = config.SetFeature(featureArray[1], featureArray[2], "true")
	assert.Nil(err)

	defer func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
	}()

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			// Set the Central Repository feature
			featureArray := strings.Split(constants.FeatureCentralRepository, ".")
			err := config.SetFeature(featureArray[1], featureArray[2], spec.centralRepoFeature)
			assert.Nil(err)

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
