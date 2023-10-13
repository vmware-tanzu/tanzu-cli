// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

func Test_createDiscoverySource(t *testing.T) {
	assert := assert.New(t)

	// When discovery source name is empty
	_, err := createDiscoverySource("", "fake/path")
	assert.NotNil(err)
	assert.Equal(err.Error(), "discovery source name cannot be empty")

	// With an invalid image
	pd, err := createDiscoverySource("fake-oci-discovery-name", "test.registry.com/test-image:v1.0.0")
	assert.NotNil(err)
	assert.Contains(err.Error(), "unable to fetch the inventory of discovery 'fake-oci-discovery-name' for plugins")
	assert.NotNil(pd.OCI)
	assert.Equal(pd.OCI.Name, "fake-oci-discovery-name")
	assert.Equal(pd.OCI.Image, "test.registry.com/test-image:v1.0.0")
}

// Test_createAndListDiscoverySources test 'tanzu plugin source list' when TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY has set test only discovery sources
func Test_createAndListDiscoverySources(t *testing.T) {
	assert := assert.New(t)

	// Set temporary configuration
	configFile, _ := os.CreateTemp("", "config")
	os.Setenv(configlib.EnvConfigKey, configFile.Name())
	defer os.RemoveAll(configFile.Name())

	configFileNG, _ := os.CreateTemp("", "config_ng")
	os.Setenv(configlib.EnvConfigNextGenKey, configFileNG.Name())
	defer os.RemoveAll(configFileNG.Name())

	os.Setenv(constants.CEIPOptInUserPromptAnswer, "No")
	os.Setenv(constants.EULAPromptAnswer, "Yes")

	// Initialize the plugin source to the default one
	err := configlib.SetCLIDiscoverySource(configtypes.PluginDiscovery{
		OCI: &configtypes.OCIDiscovery{
			Name:  config.DefaultStandaloneDiscoveryName,
			Image: constants.TanzuCLIDefaultCentralPluginDiscoveryImage,
		}})
	assert.Nil(err)

	// List with one extra plugin source
	testSource1 := "harbor-repo.vmware.com/tanzu_cli_stage/plugins/plugin-inventory:latest"
	os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, testSource1)

	rootCmd, err := NewRootCmd()
	assert.Nil(err)
	rootCmd.SetArgs([]string{"plugin", "source", "list"})
	b := bytes.NewBufferString("")
	rootCmd.SetOut(b)
	err = rootCmd.Execute()
	assert.Nil(err)

	got, err := io.ReadAll(b)
	assert.Nil(err)

	// whitespace-agnostic match
	assert.Contains(strings.Join(strings.Fields(string(got)), " "),
		config.DefaultStandaloneDiscoveryName+" "+constants.TanzuCLIDefaultCentralPluginDiscoveryImage)
	assert.Contains(strings.Join(strings.Fields(string(got)), " "),
		"disc_0 (test only) "+testSource1)

	// List with two extra plugin sources
	testSource2 := "localhost:9876/tanzu-cli/plugins/sandbox1:small"
	os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, testSource1+","+testSource2)

	rootCmd.SetArgs([]string{"plugin", "source", "list"})
	b = bytes.NewBufferString("")
	rootCmd.SetOut(b)
	err = rootCmd.Execute()
	assert.Nil(err)

	got, err = io.ReadAll(b)
	assert.Nil(err)

	// whitespace-agnostic match
	assert.Contains(strings.Join(strings.Fields(string(got)), " "),
		config.DefaultStandaloneDiscoveryName+" "+constants.TanzuCLIDefaultCentralPluginDiscoveryImage)
	assert.Contains(strings.Join(strings.Fields(string(got)), " "),
		"disc_0 (test only) "+testSource1)
	assert.Contains(strings.Join(strings.Fields(string(got)), " "),
		"disc_1 (test only) "+testSource2)

	// Reset variables
	os.Unsetenv(configlib.EnvConfigKey)
	os.Unsetenv(configlib.EnvConfigNextGenKey)
	os.Unsetenv(constants.CEIPOptInUserPromptAnswer)
	os.Unsetenv(constants.EULAPromptAnswer)
	os.Unsetenv(constants.ConfigVariableAdditionalDiscoveryForTesting)
}

func Test_initDiscoverySources(t *testing.T) {
	tests := []struct {
		test            string
		args            []string
		expected        string
		expectedFailure bool
	}{
		{
			test:            "init with extra arg error",
			args:            []string{"plugin", "source", "init", "extra"},
			expectedFailure: true,
			expected:        "accepts at most 0 arg(s), received 1",
		},
		{
			test:            "init success",
			args:            []string{"plugin", "source", "init"},
			expectedFailure: false,
			expected:        "successfully initialized discovery source",
		},
	}

	configFile, _ := os.CreateTemp("", "config")
	os.Setenv(configlib.EnvConfigKey, configFile.Name())
	defer os.RemoveAll(configFile.Name())

	configFileNG, _ := os.CreateTemp("", "config_ng")
	os.Setenv(configlib.EnvConfigNextGenKey, configFileNG.Name())
	defer os.RemoveAll(configFileNG.Name())

	os.Setenv(constants.CEIPOptInUserPromptAnswer, "No")
	os.Setenv(constants.EULAPromptAnswer, "Yes")

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			// Start with a different plugin source than the default one
			// so we can test the "plugin source init" command
			err := configlib.SetCLIDiscoverySource(configtypes.PluginDiscovery{
				OCI: &configtypes.OCIDiscovery{
					Name:  config.DefaultStandaloneDiscoveryName,
					Image: "test/uri",
				}})
			assert.Nil(err)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)
			b := bytes.NewBufferString("")
			rootCmd.SetOut(b)
			rootCmd.SetErr(b)
			log.SetStdout(b)
			log.SetStderr(b)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)

			if spec.expected != "" {
				if spec.expectedFailure {
					// Check we got the correct error
					assert.Contains(err.Error(), spec.expected)
				} else {
					got, err := io.ReadAll(b)
					assert.Nil(err)
					assert.Contains(string(got), spec.expected)

					// Check that there is only one plugin source and that it
					// is the default one
					discoverySources, err := configlib.GetCLIDiscoverySources()
					assert.Nil(err)
					assert.Equal(1, len(discoverySources))

					for _, ds := range discoverySources {
						assert.NotNil(ds.OCI)
						assert.Equal(config.DefaultStandaloneDiscoveryName, ds.OCI.Name)
						assert.Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage, ds.OCI.Image)
					}
				}
			}
		})
	}
	os.Unsetenv(configlib.EnvConfigKey)
	os.Unsetenv(configlib.EnvConfigNextGenKey)
	os.Unsetenv(constants.CEIPOptInUserPromptAnswer)
	os.Unsetenv(constants.EULAPromptAnswer)
}

func Test_updateDiscoverySources(t *testing.T) {
	tests := []struct {
		test            string
		args            []string
		expected        string
		expectedFailure bool
	}{
		{
			test:            "update missing arg error",
			args:            []string{"plugin", "source", "update"},
			expectedFailure: true,
			expected:        "accepts 1 arg(s), received 0",
		},
		{
			test:            "update extra arg error",
			args:            []string{"plugin", "source", "update", "default", "extra"},
			expectedFailure: true,
			expected:        "accepts 1 arg(s), received 2",
		},
		{
			test:            "update invalid source",
			args:            []string{"plugin", "source", "update", "invalid", "-u", constants.TanzuCLIDefaultCentralPluginDiscoveryImage},
			expectedFailure: true,
			expected:        `discovery "invalid" does not exist`,
		},
		{
			test:            "update invalid uri error",
			args:            []string{"plugin", "source", "update", "default", "-u", "example.com"},
			expectedFailure: true,
			expected:        "unable to fetch the inventory of discovery",
		},
		{
			test:            "update success",
			args:            []string{"plugin", "source", "update", "default", "-u", constants.TanzuCLIDefaultCentralPluginDiscoveryImage},
			expectedFailure: false,
			expected:        "updated discovery source",
		},
	}

	configFile, _ := os.CreateTemp("", "config")
	os.Setenv(configlib.EnvConfigKey, configFile.Name())
	defer os.RemoveAll(configFile.Name())

	configFileNG, _ := os.CreateTemp("", "config_ng")
	os.Setenv(configlib.EnvConfigNextGenKey, configFileNG.Name())
	defer os.RemoveAll(configFileNG.Name())

	os.Setenv(constants.CEIPOptInUserPromptAnswer, "No")
	os.Setenv(constants.EULAPromptAnswer, "Yes")

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			// Set the discovery source to a fake one before each test
			// to see being updated
			err := configlib.SetCLIDiscoverySource(configtypes.PluginDiscovery{
				OCI: &configtypes.OCIDiscovery{
					Name:  config.DefaultStandaloneDiscoveryName,
					Image: "test/uri",
				}})
			assert.Nil(err)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)
			b := bytes.NewBufferString("")
			rootCmd.SetOut(b)
			rootCmd.SetErr(b)
			log.SetStdout(b)
			log.SetStderr(b)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)

			if spec.expected != "" {
				if spec.expectedFailure {
					// Check we got the correct error
					assert.Contains(err.Error(), spec.expected)
				} else {
					got, err := io.ReadAll(b)
					assert.Nil(err)
					assert.Contains(string(got), spec.expected)

					// Check that there is only one plugin source and that it
					// is the default one
					discoverySources, err := configlib.GetCLIDiscoverySources()
					assert.Nil(err)
					assert.Equal(1, len(discoverySources))

					for _, ds := range discoverySources {
						assert.NotNil(ds.OCI)
						assert.Equal(config.DefaultStandaloneDiscoveryName, ds.OCI.Name)
						assert.Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage, ds.OCI.Image)
					}
				}
			}
		})
	}
	os.Unsetenv(configlib.EnvConfigKey)
	os.Unsetenv(configlib.EnvConfigNextGenKey)
	os.Unsetenv(constants.CEIPOptInUserPromptAnswer)
	os.Unsetenv(constants.EULAPromptAnswer)
}

func Test_deleteDiscoverySource(t *testing.T) {
	tests := []struct {
		test            string
		args            []string
		expected        string
		expectedFailure bool
	}{
		{
			test:            "delete missing arg error",
			args:            []string{"plugin", "source", "delete"},
			expectedFailure: true,
			expected:        "accepts 1 arg(s), received 0",
		},
		{
			test:            "delete extra arg error",
			args:            []string{"plugin", "source", "delete", "default", "extra"},
			expectedFailure: true,
			expected:        "accepts 1 arg(s), received 2",
		},
		{
			test:            "delete invalid source",
			args:            []string{"plugin", "source", "delete", "invalid"},
			expectedFailure: true,
			expected:        `discovery "invalid" does not exist`,
		},
		{
			test:            "delete success",
			args:            []string{"plugin", "source", "delete", "default"},
			expectedFailure: false,
			expected:        "deleted discovery source",
		},
	}

	configFile, _ := os.CreateTemp("", "config")
	os.Setenv(configlib.EnvConfigKey, configFile.Name())
	defer os.RemoveAll(configFile.Name())

	configFileNG, _ := os.CreateTemp("", "config_ng")
	os.Setenv(configlib.EnvConfigNextGenKey, configFileNG.Name())
	defer os.RemoveAll(configFileNG.Name())

	os.Setenv(constants.CEIPOptInUserPromptAnswer, "No")
	os.Setenv(constants.EULAPromptAnswer, "Yes")

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			// Reset the discovery source to the default one
			// before each test
			err := configlib.SetCLIDiscoverySource(configtypes.PluginDiscovery{
				OCI: &configtypes.OCIDiscovery{
					Name:  config.DefaultStandaloneDiscoveryName,
					Image: constants.TanzuCLIDefaultCentralPluginDiscoveryImage,
				}})
			assert.Nil(err)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)
			b := bytes.NewBufferString("")
			rootCmd.SetOut(b)
			rootCmd.SetErr(b)
			log.SetStdout(b)
			log.SetStderr(b)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)

			if spec.expected != "" {
				if spec.expectedFailure {
					// Check we got the correct error
					assert.Contains(err.Error(), spec.expected)
				} else {
					got, err := io.ReadAll(b)
					assert.Nil(err)
					assert.Contains(string(got), spec.expected)

					// Check that there are no more plugin sources
					discoverySources, err := configlib.GetCLIDiscoverySources()
					assert.Nil(err)
					assert.Equal(0, len(discoverySources))
				}
			}
		})
	}
	os.Unsetenv(configlib.EnvConfigKey)
	os.Unsetenv(configlib.EnvConfigNextGenKey)
	os.Unsetenv(constants.CEIPOptInUserPromptAnswer)
	os.Unsetenv(constants.EULAPromptAnswer)
}

func TestCompletionPluginSource(t *testing.T) {
	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		{
			test: "no completion after the source init command",
			args: []string{"__complete", "plugin", "source", "init", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "no completion after the source list command",
			args: []string{"__complete", "plugin", "source", "list", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "completion for the --output flag value",
			args: []string{"__complete", "plugin", "source", "list", "--output", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForOutputFlag + ":4\n",
		},
		{
			test: "completion for the source update command",
			args: []string{"__complete", "plugin", "source", "update", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "--uri\tURI for discovery source. The URI must be of an OCI image\n" +
				"-u\tURI for discovery source. The URI must be of an OCI image\n" +
				"default\texample.com/tanzu_cli/plugins/plugin-inventory:latest\n" +
				":4\n",
		},
		{
			test: "no completion after the first arg of the source update command",
			args: []string{"__complete", "plugin", "source", "update", "default", "-u", "someURI", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "completion for the source delete command",
			args: []string{"__complete", "plugin", "source", "delete", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "default\texample.com/tanzu_cli/plugins/plugin-inventory:latest\n" +
				":4\n",
		},
		{
			test: "no completion after the first arg of the source delete command",
			args: []string{"__complete", "plugin", "source", "delete", "default", ""},
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
