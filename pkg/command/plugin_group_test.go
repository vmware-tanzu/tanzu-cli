// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginGroupSearch(t *testing.T) {
	tests := []struct {
		test            string
		args            []string
		expected        string
		expectedFailure bool
	}{
		{
			test:            "search for all groups",
			args:            []string{"plugin", "group", "search"},
			expectedFailure: false,
			expected:        "GROUP DESCRIPTION LATEST vmware-tap/default Plugins for TAP v3.3.3 vmware-tkg/default Plugins for TKG v2.2.2",
		},
		{
			test:            "search for group with --name",
			args:            []string{"plugin", "group", "search", "--name", "vmware-tap/default"},
			expectedFailure: false,
			expected:        "GROUP DESCRIPTION LATEST vmware-tap/default Plugins for TAP v3.3.3",
		},
		{
			test:            "search for invalid group with --name",
			args:            []string{"plugin", "group", "search", "--name", "invalid"},
			expectedFailure: true,
			expected:        `incorrect plugin-group "invalid" specified`,
		},
		{
			test:            "search for group with --show-details",
			args:            []string{"plugin", "group", "search", "--show-details"},
			expectedFailure: false,
			expected:        "name: vmware-tap/default description: Plugins for TAP latest: v3.3.3 versions: - v3.3.3 name: vmware-tkg/default description: Plugins for TKG latest: v2.2.2 versions: - v1.1.1 - v2.2.2-beta - v2.2.2",
		},
		{
			test:            "search for group with --show-details and --name",
			args:            []string{"plugin", "group", "search", "--show-details", "--name", "vmware-tap/default"},
			expectedFailure: false,
			expected:        "name: vmware-tap/default description: Plugins for TAP latest: v3.3.3 versions: - v3.3.3",
		},
		{
			test:            "search for all groups with json",
			args:            []string{"plugin", "group", "search", "-o", "json"},
			expectedFailure: false,
			expected:        "[ { \"description\": \"Plugins for TAP\", \"group\": \"vmware-tap/default\", \"latest\": \"v3.3.3\" }, { \"description\": \"Plugins for TKG\", \"group\": \"vmware-tkg/default\", \"latest\": \"v2.2.2\" } ]",
		},
		{
			test:            "search for group with --name with json",
			args:            []string{"plugin", "group", "search", "--name", "vmware-tap/default", "-o", "json"},
			expectedFailure: false,
			expected:        "[ { \"description\": \"Plugins for TAP\", \"group\": \"vmware-tap/default\", \"latest\": \"v3.3.3\" } ]",
		},
		{
			test:            "search for group with --show-details with json",
			args:            []string{"plugin", "group", "search", "--show-details", "-o", "json"},
			expectedFailure: false,
			expected:        "[ { \"Name\": \"vmware-tap/default\", \"Description\": \"Plugins for TAP\", \"Latest\": \"v3.3.3\", \"Versions\": [ \"v3.3.3\" ] }, { \"Name\": \"vmware-tkg/default\", \"Description\": \"Plugins for TKG\", \"Latest\": \"v2.2.2\", \"Versions\": [ \"v1.1.1\", \"v2.2.2-beta\", \"v2.2.2\" ] } ]",
		},
		{
			test:            "search for group with --show-details and --name with json",
			args:            []string{"plugin", "group", "search", "--show-details", "--name", "vmware-tap/default", "-o", "json"},
			expectedFailure: false,
			expected:        "[ { \"Name\": \"vmware-tap/default\", \"Description\": \"Plugins for TAP\", \"Latest\": \"v3.3.3\", \"Versions\": [ \"v3.3.3\" ] } ]",
		},
	}

	// Setup a plugin source and a set of installed plugins
	defer setupPluginSourceForTesting(t)()

	// For these tests, we force using the cache.
	// Normal behavior of the CLI verifies the cache validity
	// which we don't want for unit tests.
	os.Setenv("TEST_TANZU_CLI_USE_DB_CACHE_ONLY", "1")

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)
			if spec.expected != "" {
				if spec.expectedFailure {
					assert.Equal(spec.expected, err.Error())
				} else {
					// whitespace-agnostic match
					assert.Equal(spec.expected, strings.Join(strings.Fields(out.String()), " "))
				}
			}
		})
	}

	os.Unsetenv("TEST_TANZU_CLI_USE_DB_CACHE_ONLY")
}

func TestPluginGroupGet(t *testing.T) {
	tests := []struct {
		test            string
		args            []string
		expected        string
		expectedFailure bool
	}{
		{
			test:            "get a plugin group",
			args:            []string{"plugin", "group", "get", "vmware-tkg/default"},
			expectedFailure: false,
			expected:        "Plugins in Group: vmware-tkg/default:v2.2.2 NAME TARGET VERSION isolated-cluster global v1.3",
		},
		{
			test:            "get a plugin group with version",
			args:            []string{"plugin", "group", "get", "vmware-tkg/default:v1.1.1"},
			expectedFailure: false,
			expected:        "Plugins in Group: vmware-tkg/default:v1.1.1 NAME TARGET VERSION isolated-cluster global v1.2.3 login global v1.2.0 management-cluster kubernetes v0.1.0 package kubernetes v0.2.0 secret kubernetes v0.3.0",
		},
		{
			test:            "get a plugin group in json",
			args:            []string{"plugin", "group", "get", "vmware-tkg/default", "-o", "json"},
			expectedFailure: false,
			expected:        "[ { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v2.2.2\", \"pluginname\": \"isolated-cluster\", \"plugintarget\": \"global\", \"pluginversion\": \"v1.3\" } ]",
		},
		{
			test:            "get a plugin group with --all with no context-scoped",
			args:            []string{"plugin", "group", "get", "vmware-tkg/default", "--all"},
			expectedFailure: false,
			expected:        "Plugins in Group: vmware-tkg/default:v2.2.2 Standalone Plugins NAME TARGET VERSION isolated-cluster global v1.3 [i] The standalone plugins in this plugin group are installed when the 'tanzu plugin install --group vmware-tkg/default' command is invoked. Contextual Plugins NAME TARGET VERSION [i] The contextual plugins in this plugin group are automatically installed, and only available for use, when a Tanzu context which supports them is created or activated/used.",
		},
		{
			test:            "get a plugin group with --all with context-scoped",
			args:            []string{"plugin", "group", "get", "vmware-tkg/default:v1.1.1", "--all"},
			expectedFailure: false,
			expected:        "Plugins in Group: vmware-tkg/default:v1.1.1 Standalone Plugins NAME TARGET VERSION isolated-cluster global v1.2.3 login global v1.2.0 management-cluster kubernetes v0.1.0 package kubernetes v0.2.0 secret kubernetes v0.3.0 [i] The standalone plugins in this plugin group are installed when the 'tanzu plugin install --group vmware-tkg/default:v1.1.1' command is invoked. Contextual Plugins NAME TARGET VERSION cluster kubernetes v1.1.1 [i] The contextual plugins in this plugin group are automatically installed, and only available for use, when a Tanzu context which supports them is created or activated/used.",
		},
		{
			test:            "get a plugin group in json with --all with no context-scoped",
			args:            []string{"plugin", "group", "get", "vmware-tkg/default", "-o", "json", "--all"},
			expectedFailure: false,
			expected:        "[ { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v2.2.2\", \"pluginname\": \"isolated-cluster\", \"plugintarget\": \"global\", \"pluginversion\": \"v1.3\" } ]",
		},
		{
			test:            "get a plugin group in json with --all with context-scoped",
			args:            []string{"plugin", "group", "get", "vmware-tkg/default:v1.1.1", "-o", "json", "--all"},
			expectedFailure: false,
			expected:        "[ { \"context-scoped\": true, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"cluster\", \"plugintarget\": \"kubernetes\", \"pluginversion\": \"v1.1.1\" }, { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"isolated-cluster\", \"plugintarget\": \"global\", \"pluginversion\": \"v1.2.3\" }, { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"login\", \"plugintarget\": \"global\", \"pluginversion\": \"v1.2.0\" }, { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"management-cluster\", \"plugintarget\": \"kubernetes\", \"pluginversion\": \"v0.1.0\" }, { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"package\", \"plugintarget\": \"kubernetes\", \"pluginversion\": \"v0.2.0\" }, { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"secret\", \"plugintarget\": \"kubernetes\", \"pluginversion\": \"v0.3.0\" } ]",
		},
		{
			test:            "get a plugin group with version in json",
			args:            []string{"plugin", "group", "get", "vmware-tkg/default:v1.1.1", "-o", "json"},
			expectedFailure: false,
			expected:        "[ { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"isolated-cluster\", \"plugintarget\": \"global\", \"pluginversion\": \"v1.2.3\" }, { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"login\", \"plugintarget\": \"global\", \"pluginversion\": \"v1.2.0\" }, { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"management-cluster\", \"plugintarget\": \"kubernetes\", \"pluginversion\": \"v0.1.0\" }, { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"package\", \"plugintarget\": \"kubernetes\", \"pluginversion\": \"v0.2.0\" }, { \"context-scoped\": false, \"group\": \"vmware-tkg/default:v1.1.1\", \"pluginname\": \"secret\", \"plugintarget\": \"kubernetes\", \"pluginversion\": \"v0.3.0\" } ]",
		},
		{
			test:            "get an invalid plugin group",
			args:            []string{"plugin", "group", "get", "invalid"},
			expectedFailure: true,
			expected:        "incorrect plugin-group \"invalid\" specified",
		},
		{
			test:            "get a plugin group with an invalid version",
			args:            []string{"plugin", "group", "get", "vmware-tkg/default:v0.888.0"},
			expectedFailure: true,
			expected:        "plugin-group \"vmware-tkg/default:v0.888.0\" cannot be found",
		},
	}

	// Setup a plugin source and a set of installed plugins
	defer setupPluginSourceForTesting(t)()

	// For these tests, we force using the cache.
	// Normal behavior of the CLI verifies the cache validity
	// which we don't want for unit tests.
	os.Setenv("TEST_TANZU_CLI_USE_DB_CACHE_ONLY", "1")

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)
			if spec.expected != "" {
				if spec.expectedFailure {
					assert.Equal(spec.expected, err.Error())
				} else {
					// whitespace-agnostic match
					assert.Equal(spec.expected, strings.Join(strings.Fields(out.String()), " "))
				}
			}
		})
	}

	os.Unsetenv("TEST_TANZU_CLI_USE_DB_CACHE_ONLY")
}

func TestCompletionPluginGroup(t *testing.T) {
	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		{
			test: "no completion after the group search command",
			args: []string{"__complete", "plugin", "group", "search", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "completion for the --output flag value of the group search command",
			args: []string{"__complete", "plugin", "group", "search", "--output", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForOutputFlag + ":4\n",
		},
		{
			test: "completion for the --name flag value of the group search command",
			args: []string{"__complete", "plugin", "group", "search", "--name", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "vmware-tap/default\tPlugins for TAP\n" +
				"vmware-tkg/default\tPlugins for TKG\n" +
				":4\n",
		},
		{
			test: "completion for the group name part of the group get command",
			args: []string{"__complete", "plugin", "group", "get", ""},
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "vmware-tap/default\tPlugins for TAP\n" +
				"vmware-tkg/default\tPlugins for TKG\n" +
				":6\n",
		},
		{
			test: "completion for the version name part of the group get command",
			args: []string{"__complete", "plugin", "group", "get", "vmware-tkg/default:"},
			// ":36" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveKeepOrder
			expected: "vmware-tkg/default:v2.2.2\n" +
				"vmware-tkg/default:v2.2.2-beta\n" +
				"vmware-tkg/default:v1.1.1\n" +
				":36\n",
		},
		{
			test: "no completion after the first arg of the group get command",
			args: []string{"__complete", "plugin", "group", "get", "vmware-tkg/default", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "completion for the --output flag value for the group get command",
			args: []string{"__complete", "plugin", "group", "get", "--output", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForOutputFlag + ":4\n",
		},
	}

	// Setup a plugin source and a set of installed plugins
	defer setupPluginSourceForTesting(t)()

	// Do NOT force using the cache with TEST_TANZU_CLI_USE_DB_CACHE_ONLY
	// because we need to test that the shell completion code itself
	// forces the use of the cache.

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

			resetPluginCommandFlags()
		})
	}
}
