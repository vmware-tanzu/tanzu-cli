// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// Test_config_MalformedPathArg validates functionality when an invalid argument is provided.
func TestConfigMalformedPathArg(t *testing.T) {
	err := setConfiguration(nil, "invalid-arg", "")
	if err == nil {
		t.Error("Malformed path argument should have resulted in an error")
	}

	if !strings.Contains(err.Error(), "unable to parse config path parameter") {
		t.Errorf("Unexpected error message returned for malformed path argument: %s", err.Error())
	}
}

// Test_config_InvalidPathArg validates functionality when an invalid argument is provided.
func TestConfigInvalidPathArg(t *testing.T) {
	err := setConfiguration(nil, "shouldbefeatures.plugin.feature", "")
	if err == nil {
		t.Error("Invalid path argument should have resulted in an error")
	}

	if !strings.Contains(err.Error(), "unsupported config path parameter") {
		t.Errorf("Unexpected error message returned for invalid path argument: %s", err.Error())
	}
}

// TestConfigUnstableVersions validates functionality when path argument unstable-versions is provided.
func TestConfigUnstableVersions(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	const path = "unstable-versions"
	const value = "experimental"
	err := setConfiguration(cfg, path, value)
	if err != nil {
		t.Errorf("Unexpected error returned for %s path argument: %s", path, err.Error())
	}

	if cfg.ClientOptions.CLI.UnstableVersionSelector != value {
		t.Error("Unstable version was not set correctly for valid value")
	}
}

// TestConfigUnstableVersions validates functionality when path argument cli.unstable-versions is provided.
func TestConfigCliUnstableVersions(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	const path = "cli.unstable-versions"
	const value = "all"
	err := setConfiguration(cfg, path, value)
	if err != nil {
		t.Errorf("Unexpected error returned for %s path argument: %s", path, err.Error())
	}

	if cfg.ClientOptions.CLI.UnstableVersionSelector != value {
		t.Error("Unstable version was not set correctly for valid value")
	}
}

// TestConfigInvalidUnstableVersions validates functionality when invalid unstable-versions is provided.
func TestConfigInvalidUnstableVersions(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	err := setConfiguration(cfg, "unstable-versions", "invalid-unstable-versions-value")
	if err == nil {
		t.Error("Invalid unstable-versions should have resulted in error")
	}

	if !strings.Contains(err.Error(), "unknown unstable-versions setting") {
		t.Errorf("Unexpected error message returned for invalid unstable-versions value: %s", err.Error())
	}
}

// TestConfigGlobalFeature validates functionality when global feature path argument is provided.
func TestConfigGlobalFeature(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	value := "bar"
	err := setConfiguration(cfg, "features.global.foo", value)
	if err != nil {
		t.Errorf("Unexpected error returned for global features path argument: %s", err.Error())
	}

	if cfg.ClientOptions.Features["global"]["foo"] != value {
		t.Error("cfg.ClientOptions.Features[\"global\"][\"foo\"] was not assigned the value \"" + value + "\"")
	}
}

// TestConfigFeature validates functionality when normal feature path argument is provided.
func TestConfigFeature(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	value := "barr"
	err := setConfiguration(cfg, "features.any-plugin.foo", value)
	if err != nil {
		t.Errorf("Unexpected error returned for any-plugin features path argument: %s", err.Error())
	}

	if cfg.ClientOptions.Features["any-plugin"]["foo"] != value {
		t.Error("cfg.ClientOptions.Features[\"any-plugin\"][\"foo\"] was not assigned the value \"" + value + "\"")
	}
}

// TestConfigSetUnsetEnv validates set and unset functionality when env config path argument is provided.
func TestConfigSetUnsetEnv(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	value := "baar"
	err := setConfiguration(cfg, "env.foo", value)
	assert.Nil(t, err)
	assert.Equal(t, value, cfg.ClientOptions.Env["foo"])

	err = unsetConfiguration("env.foo")
	assert.Nil(t, err)

	cfg, err = config.GetClientConfigNoLock()
	assert.NoError(t, err)
	assert.Equal(t, cfg.ClientOptions.Env["foo"], "")
}

// TestConfigIncorrectConfigLiteral validates incorrect config literal
func TestConfigIncorrectConfigLiteral(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	value := "b"
	err := setConfiguration(cfg, "fake.any-plugin.foo", value)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unsupported config path parameter [fake] (was expecting 'features.<plugin>.<feature>' or 'env.<env_variable>')")
}

// TestConfigEnv validates functionality when normal env path argument is provided.
func TestConfigEnv(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	value := "baarr"
	err := setConfiguration(cfg, "env.any-plugin", value)
	if err != nil {
		t.Errorf("Unexpected error returned for any-plugin env path argument: %s", err.Error())
	}
	if cfg.ClientOptions.Env["any-plugin"] != value {
		t.Error("cfg.ClientOptions.Features[\"any-plugin\"][\"foo\"] was not assigned the value \"" + value + "\"")
	}
}

func TestConfigEditionCommunity(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	value := configtypes.EditionCommunity
	err := setConfiguration(cfg, "cli.edition", value)
	if err != nil {
		t.Errorf("Unexpected error returned for cli.edition argument: %s", err.Error())
	}

	if cfg.ClientOptions.CLI.Edition != configtypes.EditionCommunity {
		t.Error("cfg.ClientOptions.CLI.Edition was not assigned the value \"" + value + "\"")
	}
}

func TestConfigEditionStandard(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	value := configtypes.EditionStandard
	err := setConfiguration(cfg, "cli.edition", value)
	if err != nil {
		t.Errorf("Unexpected error returned for cli.edition argument: %s", err.Error())
	}

	if cfg.ClientOptions.CLI.Edition != configtypes.EditionStandard {
		t.Error("cfg.ClientOptions.CLI.Edition was not assigned the value \"" + value + "\"")
	}
}

func TestConfigEditionInvalid(t *testing.T) {
	cfg := &configtypes.ClientConfig{}
	value := "invalidEdition"
	err := setConfiguration(cfg, "cli.edition", value)
	if err == nil {
		t.Errorf("Expected error returned for cli.edition argument: %s", value)
	}
}

func TestCompletionConfig(t *testing.T) {
	// Setup a temporary configuration
	configFile, err := os.CreateTemp("", "config")
	assert.Nil(t, err)
	os.Setenv("TANZU_CONFIG", configFile.Name())
	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(t, err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

	// Set some env vars
	_ = config.SetEnv("VAR1", "value1")
	_ = config.SetEnv("VAR2", "value2")

	// Set some features
	_ = config.SetFeature("global", "feat1", "val1")
	_ = config.SetFeature("plugin2", "feat2", "val2")

	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		// ======================
		// tanzu config get
		// ======================
		{
			test: "no completion for the config get command",
			args: []string{"__complete", "config", "get", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// ======================
		// tanzu config set
		// ======================
		{
			test: "completions for the config set command",
			args: []string{"__complete", "config", "set", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ You can modify the below entries, or provide a new one\n" +
				"env.VAR1\tValue: \"value1\"\n" +
				"env.VAR2\tValue: \"value2\"\n" +
				"features.global.feat1\tValue: \"val1\"\n" +
				"features.plugin2.feat2\tValue: \"val2\"\n" +
				":4\n",
		},
		{
			test: "active help after the first arg for the config set command",
			args: []string{"__complete", "config", "set", "env.VAR", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ You must provide a value as a second argument\n:4\n",
		},
		{
			test: "no completion after the second arg for the config set command",
			args: []string{"__complete", "config", "set", "env.VAR", "val", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// ======================
		// tanzu config unset
		// ======================
		{
			test: "completions for the config unset command",
			args: []string{"__complete", "config", "unset", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "env.VAR1\tValue: \"value1\"\n" +
				"env.VAR2\tValue: \"value2\"\n" +
				"features.global.feat1\tValue: \"val1\"\n" +
				"features.plugin2.feat2\tValue: \"val2\"\n" +
				":4\n",
		},
		{
			test: "no completion after the first arg for the config unset command",
			args: []string{"__complete", "config", "unset", "env.VAR", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// ======================
		// tanzu config init
		// ======================
		{
			test: "no completion for the config init command",
			args: []string{"__complete", "config", "init", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
	}

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

			resetCertCommandFlags()
		})
	}

	os.RemoveAll(configFile.Name())
	os.RemoveAll(configFileNG.Name())
	os.Unsetenv("TANZU_CONFIG")
	os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
}
