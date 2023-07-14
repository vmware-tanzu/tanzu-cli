// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompletionPluginBundle(t *testing.T) {
	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		// ============================
		// tanzu plugin download-bundle
		// ============================
		{
			test: "no completion after the download-bundle command",
			args: []string{"__complete", "plugin", "download-bundle", ""},
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "--\n:6\n",
		},
		{
			test: "no completion after the download-bundle command with --to-tar",
			args: []string{"__complete", "plugin", "download-bundle", "--to-tar", "plugin.tar", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "no completion after the download-bundle command with --dry-run",
			args: []string{"__complete", "plugin", "download-bundle", "--dry-run", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "no completion after the download-bundle command with --image",
			args: []string{"__complete", "plugin", "download-bundle", "--image", "example.com/image:latest", ""},
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "--\n:6\n",
		},
		{
			test: "no completion for the --image flag value of the download-bundle command",
			args: []string{"__complete", "plugin", "download-bundle", "--image", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "file completion for the --to-tar flag value of the download-bundle command",
			args: []string{"__complete", "plugin", "download-bundle", "--to-tar", ""},
			// ":0" is the value of the ShellCompDirectiveDefault
			expected: ":0\n",
		},
		{
			test: "completion for the --group flag value for the group name part of the download-bundle command",
			args: []string{"__complete", "plugin", "download-bundle", "--group", ""},
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "vmware-tap/default\tPlugins for TAP\n" +
				"vmware-tkg/default\tPlugins for TKG\n" +
				":6\n",
		},
		{
			test: "completion for the --group flag value for the version part of the download-bundle command",
			args: []string{"__complete", "plugin", "download-bundle", "--group", "vmware-tkg/default:"},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "vmware-tkg/default:v1.1.1\n" +
				"vmware-tkg/default:v2.2.2\n" +
				"vmware-tkg/default:v2.2.2-beta\n" +
				":4\n",
		},
		// TODO(khouzam): Fix this
		// {
		// 	test: "completion for the --group flag value for the group name part of the download-bundle command with --image",
		// 	args: []string{"__complete", "plugin", "download-bundle", "--group", ""},
		// 	// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
		// 	expected: "image groups\n" +
		// 		":6\n",
		// },
		// {
		// 	test: "completion for the --group flag value for the version part of the download-bundle command with --image",
		// 	args: []string{"__complete", "plugin", "download-bundle", "--group", "vmware-tkg/default:"},
		// 	// ":4" is the value of the ShellCompDirectiveNoFileComp
		// 	expected: "image groups\n" +
		// 		":4\n",
		// },
		// ============================
		// tanzu plugin upload-bundle
		// ============================
		{
			test: "file completion for the --tar flag value of the upload-bundle command",
			args: []string{"__complete", "plugin", "upload-bundle", "--tar", ""},
			// ":0" is the value of the ShellCompDirectiveDefault
			expected: ":0\n",
		},
		{
			test: "completion for the --group flag value for the version part of the download-bundle command",
			args: []string{"__complete", "plugin", "upload-bundle", "--tar", "plugin.tar", "--to-repo", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "flag completion after the upload-bundle command when no flags are present",
			args: []string{"__complete", "plugin", "upload-bundle", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "--tar\tsource tar file\n" +
				"--to-repo\tdestination repository for publishing plugins\n" +
				":4\n",
		},
		{
			test: "flag completion after the upload-bundle command when one flag is present",
			args: []string{"__complete", "plugin", "upload-bundle", "--tar", "plugin.tar", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "--to-repo\tdestination repository for publishing plugins\n" +
				":4\n",
		},
		{
			test: "no completion after the upload-bundle command when all flags are present",
			args: []string{"__complete", "plugin", "upload-bundle", "--tar", "plugin.tar", "--to-repo", "repo", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
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
