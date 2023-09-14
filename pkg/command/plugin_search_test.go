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
		test                string
		centralRepoDisabled string
		args                []string
		expected            string
		expectedFailure     bool
	}{
		{
			test:                "no 'plugin search' without central repo",
			centralRepoDisabled: "true",
			args:                []string{"plugin", "search"},
			expected:            "Provides all lifecycle operations for plugins",
		},
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
			// Disable the Central Repository feature if needed
			enabled := "true"
			if !strings.EqualFold(spec.centralRepoDisabled, "true") {
				enabled = "false"
			}
			featureArray := strings.Split(constants.FeatureDisableCentralRepositoryForTesting, ".")
			err := config.SetFeature(featureArray[1], featureArray[2], enabled)
			assert.Nil(err)

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
