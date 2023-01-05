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
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

func TestPluginList(t *testing.T) {
	tests := []struct {
		test            string
		plugins         []string
		versions        []string
		args            []string
		expected        string
		expectedFailure bool
	}{
		{
			test:            "With empty config file(no discovery sources added) and no plugins installed",
			plugins:         []string{},
			versions:        []string{"v0.1.0"},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION SCOPE DISCOVERY VERSION STATUS",
		},
		{
			test:            "With empty config file(no discovery sources added) and when one additional plugin installed",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION SCOPE DISCOVERY VERSION STATUS foo some foo description Standalone v0.1.0 installed",
		},
		{
			test:            "With empty config file(no discovery sources added) and when more than one plugin is installed",
			plugins:         []string{"foo", "bar"},
			versions:        []string{"v0.1.0", "v0.2.0"},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION SCOPE DISCOVERY VERSION STATUS bar some bar description Standalone v0.2.0 installed foo some foo description Standalone v0.1.0 installed",
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
	}

	for _, spec := range tests {
		tkgConfigFile, _ := os.CreateTemp("", "config")
		os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())
		defer os.RemoveAll(tkgConfigFile.Name())

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

			b := bytes.NewBufferString("")
			rootCmd.SetOut(b)

			err = rootCmd.Execute()
			assert.Nil(err)

			got, err := io.ReadAll(b)

			assert.Equal(err != nil, spec.expectedFailure)

			if spec.expected != "" {
				// whitespace-agnostic match
				assert.Contains(strings.Join(strings.Fields(string(got)), " "), spec.expected)
			}
		})
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
		os.Unsetenv("TANZU_CONFIG")
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
