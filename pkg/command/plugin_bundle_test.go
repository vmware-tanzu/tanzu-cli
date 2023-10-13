// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

func TestCompletionPluginBundle(t *testing.T) {
	var downloadImageCalled bool

	tests := []struct {
		test                  string
		args                  []string
		expected              string
		imageMustBeDownloaded bool
	}{
		// ============================
		// tanzu plugin download-bundle
		// ============================
		{
			test: "completion of flags after the download-bundle command",
			args: []string{"__complete", "plugin", "download-bundle", ""},
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "--\n" +
				":6\n",
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
			test: "completion of flags after the download-bundle command with --image",
			args: []string{"__complete", "plugin", "download-bundle", "--image", "example.com/image:latest", ""},
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "--\n" +
				":6\n",
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
			test:                  "completion for the --group flag value for the group name part of the download-bundle command",
			args:                  []string{"__complete", "plugin", "download-bundle", "--group", ""},
			imageMustBeDownloaded: true,
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "vmware-tap/default\tPlugins for TAP\n" +
				"vmware-tkg/default\tPlugins for TKG\n" +
				":6\n",
		},
		{
			test:                  "completion for the --group flag value for the version part of the download-bundle command",
			args:                  []string{"__complete", "plugin", "download-bundle", "--group", "vmware-tkg/default:"},
			imageMustBeDownloaded: true,
			// ":36" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveKeepOrder
			expected: "vmware-tkg/default:v2.2.2\n" +
				"vmware-tkg/default:v2.2.2-beta\n" +
				"vmware-tkg/default:v1.1.1\n" +
				":36\n",
		},
		{
			test:                  "completion for the --group flag value for the group name part of the download-bundle command with --image",
			args:                  []string{"__complete", "plugin", "download-bundle", "--image", "example.com/image:latest", "--group", ""},
			imageMustBeDownloaded: true,
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "vmware-tap/default\tPlugins for TAP\n" +
				"vmware-tkg/default\tPlugins for TKG\n" +
				":6\n",
		},
		{
			test:                  "completion for the --group flag value for the version part of the download-bundle command with --image",
			args:                  []string{"__complete", "plugin", "download-bundle", "--image", "example.com/image:latest", "--group", "vmware-tkg/default:"},
			imageMustBeDownloaded: true,
			// ":36" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveKeepOrder
			expected: "vmware-tkg/default:v2.2.2\n" +
				"vmware-tkg/default:v2.2.2-beta\n" +
				"vmware-tkg/default:v1.1.1\n" +
				":36\n",
		},
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

	// Provide a test image processor to avoid actually downloading DB images
	fakeImageOperations := &fakes.ImageOperationsImpl{}

	// Copy the test inventory DB file to the location the plugin download-bundle command expects
	fakeImageOperations.DownloadImageAndSaveFilesToDirCalls(func(image, path string) error {
		downloadImageCalled = true
		// Verify that the --image flag is respected by checking that the proper
		// inventory image is being "downloaded"
		assert.Equal(t, dpbo.pluginDiscoveryOCIImage, image)

		testDBFile := filepath.Join(
			common.DefaultCacheDir,
			common.PluginInventoryDirName,
			config.DefaultStandaloneDiscoveryName,
			plugininventory.SQliteDBFileName,
		)
		err := utils.CopyFile(testDBFile, filepath.Join(path, plugininventory.SQliteDBFileName))
		assert.Nil(t, err)

		return nil
	})
	imageProcessorForDownloadBundleComp = fakeImageOperations

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			downloadImageCalled = false

			rootCmd, err := NewRootCmd()
			assert.Nil(err)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Nil(err)

			assert.Equal(spec.expected, out.String())

			assert.Equal(spec.imageMustBeDownloaded, downloadImageCalled)

			resetPluginCommandFlags()
		})
	}
}
