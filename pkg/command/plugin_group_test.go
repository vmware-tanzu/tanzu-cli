// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "vmware-tkg/default:v1.1.1\n" +
				"vmware-tkg/default:v2.2.2\n" +
				"vmware-tkg/default:v2.2.2-beta\n" +
				":4\n",
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
