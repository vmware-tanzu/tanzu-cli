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
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

func TestPluginList(t *testing.T) {
	tests := []struct {
		test            string
		plugins         []string
		versions        []string
		targets         []configtypes.Target
		args            []string
		expected        string
		expectedFailure bool
	}{
		{
			test:            "no '--local' option",
			args:            []string{"plugin", "list", "--local", "someDirectory"},
			expectedFailure: true,
			expected:        "unknown flag: --local",
		},
		{
			test:            "With empty config file(no discovery sources added) and no plugins installed",
			plugins:         []string{},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION TARGET INSTALLED STATUS",
		},
		{
			test:            "With empty config file(no discovery sources added) and when one additional plugin installed",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			targets:         []configtypes.Target{configtypes.TargetK8s},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION TARGET INSTALLED STATUS foo some foo description kubernetes v0.1.0 installed",
		},
		{
			test:            "With empty config file(no discovery sources added) and when more than one plugin is installed",
			plugins:         []string{"foo", "bar"},
			versions:        []string{"v0.1.0", "v0.2.0"},
			targets:         []configtypes.Target{configtypes.TargetTMC, configtypes.TargetK8s},
			args:            []string{"plugin", "list"},
			expectedFailure: false,
			expected:        "NAME DESCRIPTION TARGET INSTALLED STATUS bar some bar description kubernetes v0.2.0 installed foo some foo description mission-control v0.1.0 installed",
		},
		{
			test:            "when json output is requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			targets:         []configtypes.Target{configtypes.TargetK8s},
			args:            []string{"plugin", "list", "-o", "json"},
			expectedFailure: false,
			expected:        `[ { "active": true, "context": "", "description": "some foo description", "installed": "v0.1.0", "name": "foo", "recommended": "", "status": "installed", "target": "kubernetes", "version": "v0.1.0" } ]`,
		},
		{
			test:            "when yaml output is requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			targets:         []configtypes.Target{configtypes.TargetK8s},
			args:            []string{"plugin", "list", "-o", "yaml"},
			expectedFailure: false,
			expected:        `- active: true context: "" description: some foo description installed: v0.1.0 name: foo recommended: "" status: installed target: kubernetes version: v0.1.0`,
		},
		{
			test:            "plugin describe json output requested",
			plugins:         []string{"foo"},
			versions:        []string{"v0.1.0"},
			targets:         []configtypes.Target{configtypes.TargetK8s},
			args:            []string{"plugin", "describe", "foo", "-o", "json"},
			expectedFailure: false,
			expected:        `[ { "description": "some foo description", "installationpath": "%v", "name": "foo", "status": "installed", "target": "kubernetes", "version": "v0.1.0" } ]`,
		},
	}

	for _, spec := range tests {
		tkgConfigFile, _ := os.CreateTemp("", "config")
		os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())
		defer os.RemoveAll(tkgConfigFile.Name())

		tkgConfigFileNG, _ := os.CreateTemp("", "config_ng")
		os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
		defer os.RemoveAll(tkgConfigFileNG.Name())

		dir, err := os.MkdirTemp("", "tanzu-cli-root-cmd")
		assert.Nil(t, err)
		defer os.RemoveAll(dir)
		os.Setenv("TEST_CUSTOM_CATALOG_CACHE_DIR", dir)
		os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
		os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")

		// Always turn on the context feature
		featureArray := strings.Split(constants.FeatureContextCommand, ".")
		err = config.SetFeature(featureArray[1], featureArray[2], "true")
		assert.Nil(t, err)

		var completionType uint8
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)
			cc, err := catalog.NewContextCatalogUpdater("")
			assert.Nil(err)
			pluginInstallationPath := ""
			for i, pluginName := range spec.plugins {
				err = setupFakePlugin(dir, pluginName, spec.versions[i], plugin.SystemCmdGroup, completionType, spec.targets[i], 1, false, []string{pluginName[:2]})
				assert.Nil(err)
				pluginInstallationPath = filepath.Join(dir, fmt.Sprintf("%s_%s", pluginName, string(spec.targets[i])))
				pi := &cli.PluginInfo{
					Name:             pluginName,
					Description:      fmt.Sprintf("some %s description", pluginName),
					Group:            plugin.SystemCmdGroup,
					Aliases:          []string{pluginName[:2]},
					Version:          spec.versions[i],
					InstallationPath: pluginInstallationPath,
					Status:           common.PluginStatusInstalled,
					Target:           spec.targets[i],
				}
				assert.Nil(err)
				err = cc.Upsert(pi)
				assert.Nil(err)
			}
			cc.Unlock()

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

					if strings.Contains(spec.expected, "installationpath") {
						spec.expected = fmt.Sprintf(spec.expected, pluginInstallationPath)
					}

					// whitespace-agnostic match
					assert.Contains(strings.Join(strings.Fields(string(got)), " "), spec.expected)
				}
			}
		})
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
	}
}

func TestDeletePlugin(t *testing.T) {
	tests := []struct {
		test             string
		plugins          []string
		remainingPlugins []bool
		versions         []string
		targets          []configtypes.Target
		args             []string
		expectedErrorMsg string
		expectedFailure  bool
	}{
		{
			test:             "delete an uninstalled plugin",
			plugins:          []string{},
			versions:         []string{},
			targets:          []configtypes.Target{},
			args:             []string{"plugin", "delete", "foo", "-y"},
			expectedFailure:  true,
			expectedErrorMsg: "unable to find plugin 'foo'",
		},
		{
			test:             "delete an installed plugin",
			plugins:          []string{"foo"},
			remainingPlugins: []bool{false},
			versions:         []string{"v0.1.0"},
			targets:          []configtypes.Target{configtypes.TargetK8s},
			args:             []string{"plugin", "delete", "foo", "-y"},
			expectedFailure:  false,
		},
		{
			test:             "delete an installed plugin present for multiple targets",
			plugins:          []string{"foo", "foo"},
			versions:         []string{"v0.1.0", "v0.2.0"},
			targets:          []configtypes.Target{configtypes.TargetTMC, configtypes.TargetK8s},
			args:             []string{"plugin", "delete", "foo", "-y"},
			expectedFailure:  true,
			expectedErrorMsg: "unable to uniquely identify plugin 'foo'. Please specify the target (kubernetes[k8s]/mission-control[tmc]/operations[ops]/global) of the plugin using the `--target` flag",
		},
		{
			test:             "delete an installed plugin present for multiple targets using --target",
			plugins:          []string{"foo", "foo"},
			remainingPlugins: []bool{true, false},
			versions:         []string{"v0.1.0", "v0.2.0"},
			targets:          []configtypes.Target{configtypes.TargetTMC, configtypes.TargetK8s},
			args:             []string{"plugin", "delete", "foo", "--target", string(configtypes.TargetK8s), "-y"},
			expectedFailure:  false,
		},
		{
			test:             "delete all installed plugins without using --target",
			plugins:          []string{"foo", "bar"},
			versions:         []string{"v0.1.0", "v0.2.0"},
			targets:          []configtypes.Target{configtypes.TargetTMC, configtypes.TargetK8s},
			args:             []string{"plugin", "delete", "all", "-y"},
			expectedFailure:  true,
			expectedErrorMsg: "the 'all' argument can only be used with the '--target' flag",
		},
		{
			test:             "delete all installed plugins using --target",
			plugins:          []string{"foo", "bar", "spaz"},
			remainingPlugins: []bool{true, false, false},
			versions:         []string{"v0.1.0", "v0.2.0", "v0.3.0"},
			targets:          []configtypes.Target{configtypes.TargetTMC, configtypes.TargetK8s, configtypes.TargetK8s},
			args:             []string{"plugin", "delete", "all", "--target", string(configtypes.TargetK8s), "-y"},
			expectedFailure:  false,
		},
	}

	for _, spec := range tests {
		dir, err := os.MkdirTemp("", "tanzu-cli-root-cmd")
		assert.Nil(t, err)
		defer os.RemoveAll(dir)

		// Setup a temporary configuration.  This means no
		// plugin source will be available which will prevent
		// trying to install the essential plugins.
		// This speeds up the test.
		configFile, _ := os.CreateTemp("", "config")
		os.Setenv("TANZU_CONFIG", configFile.Name())
		defer os.RemoveAll(configFile.Name())

		configFileNG, _ := os.CreateTemp("", "config_ng")
		os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		defer os.RemoveAll(configFileNG.Name())

		os.Setenv("TEST_CUSTOM_CATALOG_CACHE_DIR", dir)
		os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
		os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "Yes")
		var completionType uint8
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)
			cupdater, err := catalog.NewContextCatalogUpdater("")
			assert.Nil(err)

			var pluginInstallationPath string
			for i, pluginName := range spec.plugins {
				err = setupFakePlugin(dir, pluginName, spec.versions[i], plugin.SystemCmdGroup, completionType, spec.targets[i], 1, false, []string{pluginName[:2]})
				assert.Nil(err)

				pluginInstallationPath = filepath.Join(dir, fmt.Sprintf("%s_%s", pluginName, string(spec.targets[i])))
				pi := &cli.PluginInfo{
					Name:             pluginName,
					Description:      fmt.Sprintf("some %s description", pluginName),
					Group:            plugin.SystemCmdGroup,
					Aliases:          []string{pluginName[:2]},
					Version:          spec.versions[i],
					Target:           spec.targets[i],
					InstallationPath: pluginInstallationPath,
				}
				assert.Nil(err)
				err = cupdater.Upsert(pi)
				assert.Nil(err)
			}
			cupdater.Unlock()

			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)

			// Execute the test command
			err = rootCmd.Execute()
			assert.Equal(spec.expectedFailure, err != nil)
			if spec.expectedErrorMsg != "" {
				assert.Contains(err.Error(), spec.expectedErrorMsg)
			}

			// Verify the catalog after a successful delete
			// Need to get a new catalog because the rootCmd.Execute()
			// updated the catalog file
			creader, err := catalog.NewContextCatalog("")
			assert.Nil(err)

			if !spec.expectedFailure {
				assert.Equal(len(spec.plugins), len(spec.remainingPlugins), "test definition error")

				for i := range spec.plugins {
					if spec.remainingPlugins[i] {
						// The plugin must still be present
						_, exists := creader.Get(catalog.PluginNameTarget(spec.plugins[i], spec.targets[i]))
						assert.Equal(true, exists)
					} else {
						// The plugin should have been removed
						_, exists := creader.Get(catalog.PluginNameTarget(spec.plugins[i], spec.targets[i]))
						assert.Equal(false, exists)
					}

					// tanzu plugin uninstall does not remove the binary
					_, err := os.Stat(filepath.Join(dir, fmt.Sprintf("%s_%s", spec.plugins[i], string(spec.targets[i]))))
					assert.Nil(err)
				}
			}
		})
		os.Unsetenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
	}
}

func TestInstallPlugin(t *testing.T) {
	tests := []struct {
		test             string
		args             []string
		expectedErrorMsg string
		expectedFailure  bool
	}{
		{
			test:             "need plugin name if no group",
			args:             []string{"plugin", "install"},
			expectedFailure:  true,
			expectedErrorMsg: "missing plugin name as an argument",
		},
		{
			test:             "no 'all' option",
			args:             []string{"plugin", "install", "all"},
			expectedFailure:  true,
			expectedErrorMsg: "the 'all' argument can only be used with the '--group' flag",
		},
		{
			test:             "invalid target",
			args:             []string{"plugin", "install", "--target", "invalid", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: invalidTargetMsg,
		},
		{
			test:             "no --group and --local-source together",
			args:             []string{"plugin", "install", "--group", "testgroup", "--local-source", "./", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: "if any flags in the group [group local-source] are set none of the others can be",
		},
		{
			test:             "no --group and --target together",
			args:             []string{"plugin", "install", "--group", "testgroup", "--target", "k8s", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: "if any flags in the group [group target] are set none of the others can be",
		},
		{
			test:             "no --group and --version together",
			args:             []string{"plugin", "install", "--group", "testgroup", "--version", "v1.1.1", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: "if any flags in the group [group version] are set none of the others can be",
		},
	}

	assert := assert.New(t)

	tkgConfigFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

	tkgConfigFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
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
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
	}()

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)
			if spec.expectedErrorMsg != "" {
				assert.Contains(err.Error(), spec.expectedErrorMsg)
			}
		})
	}
}

func TestUpgradePlugin(t *testing.T) {
	tests := []struct {
		test             string
		args             []string
		expectedErrorMsg string
		expectedFailure  bool
	}{
		{
			test:             "invalid target",
			args:             []string{"plugin", "upgrade", "--target", "invalid", "myplugin"},
			expectedFailure:  true,
			expectedErrorMsg: invalidTargetMsg,
		},
	}

	assert := assert.New(t)

	tkgConfigFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

	tkgConfigFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
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
		os.RemoveAll(tkgConfigFile.Name())
		os.RemoveAll(tkgConfigFileNG.Name())
	}()

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			rootCmd, err := NewRootCmd()
			assert.Nil(err)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Equal(err != nil, spec.expectedFailure)
			if spec.expectedErrorMsg != "" {
				assert.Contains(err.Error(), spec.expectedErrorMsg)
			}
		})
	}
}

func TestCompletionPlugin(t *testing.T) {
	// This is global logic and needs not be tested for each
	// command.  Let's deactivate it.
	os.Setenv("TANZU_ACTIVE_HELP", "no_short_help")
	// Test local discovery

	localSourcePath := filepath.Join("..", "fakes", "plugins", cli.GOOS, cli.GOARCH)

	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		// =====================
		// tanzu plugin list
		// =====================
		{
			test: "no completion after the plugin list command",
			args: []string{"__complete", "plugin", "list", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "completion for the --output flag value of the plugin list command",
			args: []string{"__complete", "plugin", "list", "--output", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForOutputFlag + ":4\n",
		},
		// =====================
		// tanzu plugin clean
		// =====================
		{
			test: "no completions for the plugin clean command",
			args: []string{"__complete", "plugin", "clean", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// =====================
		// tanzu plugin sync
		// =====================
		{
			test: "no completions for the plugin sync command",
			args: []string{"__complete", "plugin", "sync", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// =====================
		// tanzu plugin install
		// =====================
		{
			test: "completion for the plugin install command",
			args: []string{"__complete", "plugin", "install", ""},
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
			test: "completion for the plugin install command using --group",
			args: []string{"__complete", "plugin", "install", "--group", "vmware-tkg/default:v1.1.1", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			// There are no descriptions in this case because the plugin group only contains plugin names
			expected: "all\n" +
				"isolated-cluster\n" +
				"login\n" +
				"management-cluster\n" +
				"package\n" +
				"secret\n" +
				":4\n",
		},
		{
			test: "completion for the plugin install command using --group with no version",
			args: []string{"__complete", "plugin", "install", "--group", "vmware-tkg/default", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			// There are no descriptions in this case because the plugin group only contains plugin names
			expected: "all\n" +
				"isolated-cluster\n" +
				":4\n",
		},
		{
			test: "completion for the plugin install command using --target",
			args: []string{"__complete", "plugin", "install", "--target", "kubernetes", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "cluster\tPlugin cluster/kubernetes description\n" +
				"feature\tPlugin feature/kubernetes description\n" +
				"management-cluster\tPlugin management-cluster/kubernetes description\n" +
				"package\tPlugin package/kubernetes description\n" +
				"secret\tPlugin secret/kubernetes description\n" +
				":4\n",
		},
		{
			test: "completion for the plugin install command using --local-source",
			args: []string{"__complete", "plugin", "install", "--local-source", localSourcePath, ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "all\tAll plugins of the local source\n" +
				"builder\tBuild Tanzu components\n" +
				"secret\tTanzu secret management\n" +
				":4\n",
		},
		{
			test: "completion for the plugin install command using --local-source and --target",
			args: []string{"__complete", "plugin", "install", "--local-source", localSourcePath, "--target", "global", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "all\tAll plugins of the local source\n" +
				"builder\tBuild Tanzu components\n" +
				":4\n",
		},
		{
			test: "completion for the plugin install command using --local-source and --target with no plugin match",
			args: []string{"__complete", "plugin", "install", "--local-source", localSourcePath, "--target", "tmc", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		{
			test: "completion for the --group flag value for the group name part of the plugin install command",
			args: []string{"__complete", "plugin", "install", "--group", ""},
			// ":6" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveNoSpace
			expected: "vmware-tap/default\tPlugins for TAP\n" +
				"vmware-tkg/default\tPlugins for TKG\n" +
				":6\n",
		},
		{
			test: "completion for the --group flag value for the version part of the plugin install command",
			args: []string{"__complete", "plugin", "install", "--group", "vmware-tkg/default:"},
			// ":36" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveKeepOrder
			expected: "vmware-tkg/default:v2.2.2\n" +
				"vmware-tkg/default:v2.2.2-beta\n" +
				"vmware-tkg/default:v1.1.1\n" +
				":36\n",
		},
		{
			test: "completion for the --group flag value for the version part of an invalid group for the plugin install command",
			args: []string{"__complete", "plugin", "install", "--group", "invalid:"},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ Invalid group format: 'invalid'\n" +
				":4\n",
		},
		{
			test: "completion for the --group flag value for the version part of a missing group for the plugin install command",
			args: []string{"__complete", "plugin", "install", "--group", "vmware-tkg/invalid:"},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ There is no group named: 'vmware-tkg/invalid'\n" +
				":4\n",
		},

		{
			test: "completion for the --local-source flag value",
			args: []string{"__complete", "plugin", "install", "--local-source", ""},
			// ":0" is the value of the ShellCompDirectiveDefault which indicates
			// that file completion will be performed
			expected: ":0\n",
		},
		{
			test: "completion for the --target flag value for the plugin install command with no plugin name",
			args: []string{"__complete", "plugin", "install", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				compOpsTarget + "\n" +
				":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin install command with a plugin name",
			args: []string{"__complete", "plugin", "install", "isolated-cluster", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin install command with an invalid plugin",
			args: []string{"__complete", "plugin", "install", "invalid", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				compOpsTarget + "\n" +
				":4\n",
		},

		{
			test: "no completion for the --version flag value for the plugin install command with no plugin name",
			args: []string{"__complete", "plugin", "install", "--version", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ You must first specify a plugin name to be able to complete its version\n" +
				":4\n",
		},
		{
			test: "completion for the --version flag value for the plugin install command with a plugin name",
			args: []string{"__complete", "plugin", "install", "management-cluster", "--version", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ Unable to uniquely identify this plugin. Please specify a target using the `--target` flag\n" +
				":4\n",
		},
		{
			test: "completion for the --version flag value for the plugin install command with an invalid plugin name",
			args: []string{"__complete", "plugin", "install", "invalid", "--version", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ Unable to find plugin 'invalid'\n" +
				":4\n",
		},
		{
			test: "completion for the --version flag value for the plugin install command with an invalid plugin name for a specified target",
			args: []string{"__complete", "plugin", "install", "feature", "--target", "tmc", "--version", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ Unable to find plugin 'feature' for target 'tmc'\n" +
				":4\n",
		},
		{
			test: "completion for the --version flag value for the plugin install command with a plugin name and --target",
			args: []string{"__complete", "plugin", "install", "management-cluster", "--target", "tmc", "--version", ""},
			// ":36" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveKeepOrder
			expected: "v0.2.0\n" +
				"v0.0.3\n" +
				"v0.0.2\n" +
				"v0.0.1\n" +
				":36\n",
		},
		{
			test: "completion for the plugin install command when the specified plugin name is not unique",
			args: []string{"__complete", "plugin", "install", "cluster", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "--target\n" +
				":4\n",
		},
		{
			test: "no more completions for the plugin install command when the specified plugin name is not unique and --target is specified",
			args: []string{"__complete", "plugin", "install", "cluster", "--target", "k8s", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "no completion after the first arg for the plugin install command",
			args: []string{"__complete", "plugin", "install", "builder", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// =====================
		// tanzu plugin upgrade
		// =====================
		{
			test: "completion for the plugin upgrade command",
			args: []string{"__complete", "plugin", "upgrade", ""},
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
			test: "completion for the plugin upgrade command using --target",
			args: []string{"__complete", "plugin", "upgrade", "--target", "kubernetes", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "cluster\tPlugin cluster/kubernetes description\n" +
				"feature\tPlugin feature/kubernetes description\n" +
				"management-cluster\tPlugin management-cluster/kubernetes description\n" +
				"package\tPlugin package/kubernetes description\n" +
				"secret\tPlugin secret/kubernetes description\n" +
				":4\n",
		},
		{
			test: "completion for the plugin upgrade command when the specified plugin name is not unique",
			args: []string{"__complete", "plugin", "upgrade", "cluster", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "--target\n" +
				":4\n",
		},
		{
			test: "no more completions for the plugin upgrade command when the specified plugin name is not unique and --target is specified",
			args: []string{"__complete", "plugin", "upgrade", "cluster", "--target", "k8s", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "completion for the --target flag value for the plugin upgrade command with no plugin name",
			args: []string{"__complete", "plugin", "upgrade", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				compOpsTarget + "\n" +
				":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin upgrade command with a plugin name",
			args: []string{"__complete", "plugin", "upgrade", "management-cluster", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin upgrade command with an invalid plugin",
			args: []string{"__complete", "plugin", "upgrade", "invalid", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				compOpsTarget + "\n" +
				":4\n",
		},

		{
			test: "no completion after the first arg for the plugin upgrade command",
			args: []string{"__complete", "plugin", "upgrade", "builder", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// =====================
		// tanzu plugin uninstall
		// =====================
		{
			test: "completion for the plugin uninstall command",
			args: []string{"__complete", "plugin", "uninstall", ""},
			// ":36" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveKeepOrder
			expected: "all\tAll plugins for a target. You will need to use the --target flag.\n" +
				"cluster\tMultiple entries for plugin cluster. You will need to use the --target flag.\n" +
				"feature\tTarget: kubernetes for feature\n" +
				"management-cluster\tMultiple entries for plugin management-cluster. You will need to use the --target flag.\n" +
				"secret\tTarget: kubernetes for secret\n" +
				":36\n",
		},
		{
			test: "completion for the plugin uninstall command using --target",
			args: []string{"__complete", "plugin", "uninstall", "--target", "k8s", ""},
			// ":36" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveKeepOrder
			expected: "all\tAll plugins of target kubernetes\n" +
				"cluster\tTarget: kubernetes for cluster\n" +
				"feature\tTarget: kubernetes for feature\n" +
				"management-cluster\tTarget: kubernetes for management-cluster\n" +
				"secret\tTarget: kubernetes for secret\n" +
				":36\n",
		},
		{
			test: "completion for the plugin uninstall command when the specified plugin name is not unique",
			args: []string{"__complete", "plugin", "uninstall", "cluster", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "--target\n" +
				":4\n",
		},
		{
			test: "no more completions for the plugin uninstall command when the specified plugin name is not unique and --target is specified",
			args: []string{"__complete", "plugin", "uninstall", "cluster", "--target", "k8s", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "no more completions after the first arg for the plugin uninstall command",
			args: []string{"__complete", "plugin", "uninstall", "feature", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "completion of the --target flag if 'all' is specified",
			args: []string{"__complete", "plugin", "uninstall", "all", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "--target\n" +
				":4\n",
		},
		{
			test: "no more completions after 'all' if --target is specified",
			args: []string{"__complete", "plugin", "uninstall", "--target", "k8s", "all", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "completion for the --target flag value for the plugin uninstall command with no plugin name",
			args: []string{"__complete", "plugin", "uninstall", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				compOpsTarget + "\n" +
				":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin uninstall command with a plugin name",
			args: []string{"__complete", "plugin", "uninstall", "cluster", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin uninstall command with 'all'",
			args: []string{"__complete", "plugin", "uninstall", "all", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				compOpsTarget + "\n" +
				":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin uninstall command with an invalid plugin",
			args: []string{"__complete", "plugin", "uninstall", "invalid", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				compOpsTarget + "\n" +
				":4\n",
		},
		// =====================
		// tanzu plugin describe
		// =====================
		{
			test: "completion for the plugin describe command",
			args: []string{"__complete", "plugin", "describe", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "cluster\tMultiple entries for plugin cluster. You will need to use the --target flag.\n" +
				"feature\tTarget: kubernetes for feature\n" +
				"management-cluster\tMultiple entries for plugin management-cluster. You will need to use the --target flag.\n" +
				"secret\tTarget: kubernetes for secret\n" +
				":4\n",
		},
		{
			test: "completion for the plugin describe command when the specified plugin name is not unique",
			args: []string{"__complete", "plugin", "describe", "cluster", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "--target\n" +
				":4\n",
		},
		{
			test: "no more completions for the plugin describe command when the specified plugin name is not unique and --target is specified",
			args: []string{"__complete", "plugin", "describe", "cluster", "--target", "k8s", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "completion for the plugin describe command using --target",
			args: []string{"__complete", "plugin", "describe", "--target", "k8s", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "cluster\tTarget: kubernetes for cluster\n" +
				"feature\tTarget: kubernetes for feature\n" +
				"management-cluster\tTarget: kubernetes for management-cluster\n" +
				"secret\tTarget: kubernetes for secret\n" +
				":4\n",
		},
		{
			test: "completion for the --output flag value of the plugin describe command",
			args: []string{"__complete", "plugin", "describe", "--output", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: expectedOutForOutputFlag + ":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin describe command with no plugin name",
			args: []string{"__complete", "plugin", "describe", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				compOpsTarget + "\n" +
				":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin describe command with a plugin name",
			args: []string{"__complete", "plugin", "describe", "secret", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compK8sTarget + "\n" +
				":4\n",
		},
		{
			test: "completion for the --target flag value for the plugin describe command with an invalid plugin",
			args: []string{"__complete", "plugin", "describe", "invalid", "--target", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: compGlobalTarget + "\n" +
				compK8sTarget + "\n" +
				compTMCTarget + "\n" +
				compOpsTarget + "\n" +
				":4\n",
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

	os.Unsetenv("TANZU_ACTIVE_HELP")
}

func resetPluginCommandFlags() {
	targetStr = ""
	local = ""
	version = ""
	forceDelete = false
	outputFormat = ""
	targetStr = ""
	group = ""
	showNonMandatory = false
	groupID = ""
	showDetails = false
	pluginName = ""
}
