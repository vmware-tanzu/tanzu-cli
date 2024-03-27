// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package framework defines the integration and end-to-end test case for cli core
package framework

import (
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
)

var (
	// TestDirPath is the absolute directory path for the E2E test execution uses to create all Tanzu CLI specific files (config, local plugins etc)
	TestDirPath               string
	TestPluginsDirPath        string
	TestStandalonePluginsPath string
	// FullPathForTempDir is the absolute path for the temp directory under $TestDir
	FullPathForTempDir string

	// OriginalHomeDir is the actual HOME directory of the machine before E2E test overwrites it
	OriginalHomeDir string

	// TestHomeDir is the HOME directory during E2E test execution
	TestHomeDir string

	// ConfigFilePath represents config.yaml file path which under $HOME/.tanzu-cli-e2e/.config/tanzu
	ConfigFilePath string
	// ConfigFilePath represents config-ng.yaml file path which under $HOME/.tanzu-cli-e2e/.config/tanzu
	ConfigNGFilePath string
	TanzuFolderPath  string
	TanzuBinary      string // Tanzu binary name if available in PATH variable or full path to binary with tanzu name
)

// CLICoreDescribe annotates the test with the CLICore label.
func CLICoreDescribe(text string, body func()) bool {
	return ginkgo.Describe(CliCore+text, body)
}

// Framework has all helper functions to write CLI e2e test cases
type Framework struct {
	Exec CmdOps
	CliOps
	Config       ConfigCmdOps
	KindCluster  ClusterOps
	PluginCmd    PluginCmdOps    // performs plugin command operations
	PluginHelper PluginHelperOps // helper (pre-setup) for plugin cmd operations
	ContextCmd   ContextCmdOps
}

// E2EOptions used to configure certain options to customize the E2E framework

type E2EOptions struct {
	TanzuBinary string // TanzuBinary should be set to customize the tanzu binary either the binary name if available with PATH variable or full binary path with name ; default is tanzu from PATH variable
	CLIOptions
	AdditionalFlags string
}

// CLIOptions used to configure certain options that are used in CLI lifecycle operations
// FilePath should be set to customize the installation path for legacy Tanzu CLI or new Tanzu CLI
// Override is set to determine whether the new Tanzu CLI should override or coexist the installation of legacy Tanzu CLI
type CLIOptions struct {
	FilePath string // file path to tanzu installation; Default values for legacy Tanzu CLI is set using TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR and new Tanzu CLI is set using TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR
	Override bool   // new tanzu cli overrides the installation of legacy tanzu cli; Default value is false
}

type E2EOption func(*E2EOptions)
type CLIOption func(*CLIOptions)

func NewE2EOptions(options ...E2EOption) *E2EOptions {
	e := &E2EOptions{}
	for _, option := range options {
		option(e)
	}
	return e
}

func NewFramework() *Framework {
	return &Framework{
		Exec:         NewCmdOps(),
		CliOps:       NewCliOps(),
		Config:       NewConfOps(),
		KindCluster:  NewKindCluster(NewDocker()),
		PluginCmd:    NewPluginLifecycleOps(),
		PluginHelper: NewPluginOps(NewScriptBasedPlugins(), NewLocalOCIPluginOps(NewLocalOCIRegistry(DefaultRegistryName, DefaultRegistryPort))),
		ContextCmd:   NewContextCmdOps(),
	}
}

func init() {
	OriginalHomeDir = GetHomeDir()
	TestDirPath = filepath.Join(OriginalHomeDir, TestDir)
	FullPathForTempDir = filepath.Join(TestDirPath, TempDirInTestDirPath)
	// Update $HOME as $HOME/.tanzu-cli-e2e
	os.Setenv("HOME", TestDirPath)
	TestHomeDir = TestDirPath
	TestPluginsDirPath = filepath.Join(TestDirPath, TestPluginsDir)
	TanzuFolderPath = filepath.Join(TestDirPath, ConfigFolder, TanzuFolder)
	ConfigFilePath = filepath.Join(TanzuFolderPath, ConfigFile)
	ConfigNGFilePath = filepath.Join(TanzuFolderPath, ConfigNGFile)
	TanzuBinary = os.Getenv(TanzuCLIE2ETestBinaryPath)
	// Set `tanzu` as default binary if not specified tanzu cli binary path
	if TanzuBinary == "" {
		TanzuBinary = TanzuPrefix
	}
	// Create a directory (if not exists) $HOME/.tanzu-cli-e2e/.config/tanzu-plugins/discovery/standalone
	TestStandalonePluginsPath = filepath.Join(TestDirPath, ConfigFolder, TanzuPluginsFolder, "discovery", "standalone")
	_ = CreateDir(TestStandalonePluginsPath)
	// Create a directory (if not exists) $HOME/.tanzu-cli-e2e/temp
	_ = CreateDir(FullPathForTempDir)
}
