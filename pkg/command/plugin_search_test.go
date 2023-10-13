// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

func TestPluginSearch(t *testing.T) {
	tests := []struct {
		test            string
		args            []string
		expected        string
		expectedFailure bool
	}{
		{
			test:            "invalid target",
			args:            []string{"plugin", "search", "--target", "invalid"},
			expectedFailure: true,
			expected:        invalidTargetMsg,
		},
		{
			test:            "no --local and --name together",
			args:            []string{"plugin", "search", "--local", "./", "--name", "myplugin"},
			expectedFailure: true,
			expected:        "if any flags in the group [local name] are set none of the others can be",
		},
		{
			test:            "no --local and --target together",
			args:            []string{"plugin", "search", "--local", "./", "--target", "tmc"},
			expectedFailure: true,
			expected:        "if any flags in the group [local target] are set none of the others can be",
		},
		{
			test:            "no --local and --show-details together",
			args:            []string{"plugin", "search", "--local", "./", "--show-details"},
			expectedFailure: true,
			expected:        "if any flags in the group [local show-details] are set none of the others can be",
		},
	}

	assert := assert.New(t)

	configFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG", configFile.Name())

	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
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
		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())
	}()

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
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
	}
}

func TestCompletionPluginSearch(t *testing.T) {
	expectedOutforTargetFlag := compGlobalTarget + "\n" + compK8sTarget + "\n" + compTMCTarget + "\n"

	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		{
			test: "no completion after the plugin search command",
			args: []string{"__complete", "plugin", "search", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "completion for the --name flag value",
			args: []string{"__complete", "plugin", "search", "--name", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "cluster\tMultiple entries for plugin cluster. You will need to use the --target flag.\n" +
				"feature\tPlugin feature/kubernetes description\n" +
				"isolated-cluster\tPlugin isolated-cluster/global description\n" +
				"login\tPlugin login/global description\n" +
				"management-cluster\tMultiple entries for plugin management-cluster. You will need to use the --target flag.\n" +
				"package\tPlugin package/kubernetes description\n" +
				"secret\tPlugin secret/kubernetes description\n" +
				":4\n",
		},
		{
			test: "completion for the --name flag value when --target is specified",
			args: []string{"__complete", "plugin", "search", "--target", "global", "--name", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "isolated-cluster\tPlugin isolated-cluster/global description\n" +
				"login\tPlugin login/global description\n" +
				":4\n",
		},
		{
			test: "completion for the --output flag value",
			args: []string{"__complete", "plugin", "search", "--output", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForOutputFlag + ":4\n",
		},
		{
			test: "completion for the --target flag value",
			args: []string{"__complete", "plugin", "search", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutforTargetFlag + ":4\n",
		},
		{
			test: "completion for the --local-source flag value",
			args: []string{"__complete", "plugin", "search", "--local-source", ""},
			// ":0" is the value of the ShellCompDirectiveDefault which indicates
			// that file completion will be performed
			expected: ":0\n",
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
