// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
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
			test:            "when no plugins are are added",
			plugins:         []string{},
			versions:        []string{"v0.1.0"},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION VERSION STATUS",
		},
		{
			test:            "when one additional plugin is added",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION VERSION STATUS foo some foo description v0.1.0 installed",
		},
		{
			test:            "when more than one plugin is added",
			plugins:         []string{"foo", "bar"},
			versions:        []string{"v0.1.0", "v0.2.0"},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION VERSION STATUS bar some bar description v0.2.0 installed foo some foo description v0.1.0 installed",
		},
		{
			test:            "when json output is requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			args:            []string{"plugin", "list", "-o", "json"},
			expectedFailure: false,
			expected:        `[ { "description": "some foo description", "name": "foo", "status": "installed", "version": "v0.1.0" } ]`,
		},
		{
			test:            "when yaml output is requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			args:            []string{"plugin", "list", "-o", "yaml"},
			expectedFailure: false,
			expected:        `- description: some foo description name: foo status: installed version: v0.1.0`,
		},
	}

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			plugins := make([]*cli.PluginInfo, 0)
			for i, pluginName := range spec.plugins {
				pi := &cli.PluginInfo{
					Name:             pluginName,
					Description:      fmt.Sprintf("some %s description", pluginName),
					Group:            plugin.SystemCmdGroup,
					Aliases:          []string{pluginName[:2]},
					Version:          spec.versions[i],
					InstallationPath: filepath.Join(common.DefaultPluginRoot, pluginName),
				}
				plugins = append(plugins, pi)
			}

			rootCmd, err := NewRootCmd(&TestPluginSupplier{pluginInfos: plugins})
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
	}
}
