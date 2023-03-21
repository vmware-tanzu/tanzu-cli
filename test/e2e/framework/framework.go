// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package framework defines the integration and end-to-end test case for cli core
package framework

import (
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
)

const (
	CliCore = "[CLI-Core]"

	TanzuInit    = "%s init"
	TanzuVersion = "%s version"
	TanzuPrefix  = "tanzu"
	TzPrefix     = "tz"

	// Config commands
	ConfigSet          = "%s config set "
	ConfigGet          = "%s config get "
	ConfigUnset        = "%s config unset "
	ConfigInit         = "%s config init"
	ConfigServerList   = "%s config server list"
	ConfigServerDelete = "%s config server delete %s -y"

	// Plugin commands
	UpdatePluginSource                  = "%s plugin source update %s --uri %s"
	ListPluginSourcesWithJSONOutputFlag = "%s plugin source list -o json"
	DeletePluginSource                  = "%s plugin source delete %s"
	InitPluginDiscoverySource           = "%s plugin source init"
	ListPluginsCmdWithJSONOutputFlag    = "%s plugin list -o json"
	SearchPluginsCmd                    = "%s plugin search"
	SearchPluginGroupsCmd               = "%s plugin group search"
	InstallPluginCmd                    = "%s plugin install %s"
	InstallPluginFromGroupCmd           = "%s plugin install %s --group %s"
	InstallAllPluginsFromGroupCmd       = "%s plugin install --group %s"
	DescribePluginCmd                   = "%s plugin describe %s"
	UninstallPLuginCmd                  = "%s plugin delete %s --yes"
	CleanPluginsCmd                     = "%s plugin clean"
	pluginSyncCmd                       = "%s plugin sync"
	JSONOutput                          = " -o json"
	TestPluginsPrefix                   = "test-plugin-"
	PluginSubCommand                    = "%s %s"
	PluginKey                           = "%s_%s_%s" // Plugins - Name_Target_Versions
	LegacyPluginKey                     = "%s_%s"    // Plugins - Name_Target_Versions or Name_Version_Status for legacy cli plugins

	// Central repository
	TanzuCliE2ETestCentralRepositoryURL      = "TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL"
	TanzuCliE2ETestLocalCentralRepositoryURL = "TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL"

	// CLI Coexistence
	CLICoexistenceLegacyTanzuCLIInstallationPath = "TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR"
	CLICoexistenceNewTanzuCLIInstallationPath    = "TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR"
	CLICoexistenceLegacyTanzuCLIVersion          = "TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_VERSION"
	CLICoexistenceNewTanzuCLIVersion             = "TANZU_CLI_BUILD_VERSION"
	CLICoexistenceTanzuCEIPParticipation         = "TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER"

	// General constants
	True         = "true"
	Installed    = "installed"
	NotInstalled = "not installed"

	// Context commands
	CreateContextWithEndPoint              = "%s context create --endpoint %s --name %s"
	CreateContextWithEndPointStaging       = "%s context create --endpoint %s --name %s --staging"
	CreateContextWithKubeconfigFile        = "%s context create --kubeconfig %s --kubecontext %s --name %s"
	CreateContextWithDefaultKubeconfigFile = "%s context create --kubecontext %s --name %s"
	UseContext                             = "%s context use %s"
	GetContext                             = "%s context get %s"
	ListContextOutputInJSON                = "%s context list -o json"
	DeleteContext                          = "%s context delete %s --yes"
	TanzuAPIToken                          = "TANZU_API_TOKEN" //nolint:gosec
	TanzuCliTmcUnstableURL                 = "TANZU_CLI_TMC_UNSTABLE_URL"

	// context logs
	ContextShouldNotExists       = "the context %s should not exists"
	ContextShouldExistsAsCreated = "the context %s should exists as its been created"

	KindClusterCreate = "kind create cluster --name %s"
	KindClusterStatus = "kubectl cluster-info --context %s"
	KindClusterDelete = "kind delete cluster --name %s"
	KindClusterGet    = "kind get clusters "
	KindClusterInfo   = "kubectl cluster-info --context %s"
	KubectlApply      = "kubectl --context %s apply -f %s"

	// specific plugin custom resource file name cr_<pluginName>_<target>_<versions>.yaml to apply on kind cluster
	PluginCRFileName = "cr_%s_%s_%s.yaml"

	KindCreateCluster = "kind create cluster --name "
	DockerInfo        = "docker info"
	StartDockerUbuntu = "sudo systemctl start docker"
	StopDockerUbuntu  = "sudo systemctl stop docker"

	TMC                  = "tmc"
	TKG                  = "tkg"
	SourceType           = "oci"
	GlobalTarget         = "global"
	KubernetesTarget     = "kubernetes"
	MissionControlTarget = "mission-control"

	// log info
	ExecutingCommand = "Executing command: %s"
	FileContent      = "file: %s , content: %s"

	// Error messages
	UnableToFindPluginForTarget                   = "unable to find plugin '%s' for target '%s'"
	UnableToFindPlugin                            = "unable to find plugin '%s'"
	InvalidTargetSpecified                        = "invalid target specified. Please specify correct value of `--target` or `-t` flag from 'global/kubernetes/k8s/mission-control/tmc'"
	InvalidTargetGlobal                           = "invalid target for plugin: global"
	DiscoverySourceNotFound                       = "discovery %q does not exist"
	ErrorLogForCommandWithErrStdErrAndStdOut      = "error while executing command:'%s', error:'%s' stdErr:'%s' stdOut: '%s'"
	FailedToConstructJSONNodeFromOutputAndErrInfo = "failed to construct json node from output:'%s' error:'%s' "
	FailedToConstructJSONNodeFromOutput           = "failed to construct json node from output:'%s'"

	// config related constants
	FailedToCreateContext           = "failed to create context"
	FailedToCreateContextWithStdout = FailedToCreateContext + ", stdout:%s"
	ContextCreated                  = "context %s created successfully"
	ContextDeleted                  = "context %s deleted successfully"
	ConfigServerDeleted             = "config server %s deleted successfully"
	FailedToDeleteContext           = "failed to delete context"
)

// TestDir is the directory under $HOME, created during framework initialization, and the $HOME updated as $HOME/$TestDir, to create all Tanzu CLI specific files
// and not to disturb any existing Tanzu CLI files
const TestDir = ".tanzu-cli-e2e"

// TestPluginsDir is the directory under $HOME/$TestDir, to store test plugins for E2E tests
const TestPluginsDir = ".e2e-test-plugins"

// TempDirInTestDirPath is the directory under $HOME/$TestDir, to create temporary files (if any) for E2E test execution
const TempDirInTestDirPath = "temp"

const ConfigFolder = ".config"
const TanzuFolder = "tanzu"
const TanzuPluginsFolder = "tanzu-plugins"
const ConfigFile = "config.yaml"
const ConfigNGFile = "config-ng.yaml"

var (
	// TestDirPath is the absolute directory path for the E2E test execution uses to create all Tanzu CLI specific files (config, local plugins etc)
	TestDirPath               string
	TestPluginsDirPath        string
	TestStandalonePluginsPath string
	// FullPathForTempDir is the absolute path for the temp directory under $TestDir
	FullPathForTempDir string

	// ConfigFilePath represents config.yaml file path which under $HOME/.tanzu-cli-e2e/.config/tanzu
	ConfigFilePath string
	// ConfigFilePath represents config-ng.yaml file path which under $HOME/.tanzu-cli-e2e/.config/tanzu
	ConfigNGFilePath string
	TanzuFolderPath  string
)

// PluginsForLifeCycleTests is list of plugins (which are published in local central repo) used in plugin life cycle test cases
var PluginsForLifeCycleTests []*PluginInfo

// PluginGroupsForLifeCycleTests is list of plugin groups (which are published in local central repo) used in plugin group life cycle test cases
var PluginGroupsForLifeCycleTests []*PluginGroup

// PluginGroupsLatestToOldVersions is plugin group names mapping latest version to old version, of same target,
// we are mapping because the 'tanzu plugin search' is not displaying plugins for old versions, showing only latest version of plugins
// we need this mapping for plugin sync test cases, we want to install same plugins but for different versions
var PluginGroupsLatestToOldVersions map[string]string

// CLICoreDescribe annotates the test with the CLICore label.
func CLICoreDescribe(text string, body func()) bool {
	return ginkgo.Describe(CliCore+text, body)
}

// Framework has all helper functions to write CLI e2e test cases
type Framework struct {
	CliOps
	Config       ConfigLifecycleOps
	KindCluster  ClusterOps
	PluginCmd    PluginCmdOps    // performs plugin command operations
	PluginHelper PluginHelperOps // helper (pre-setup) for plugin cmd operations
	ContextCmd   ContextCmdOps
}

// E2EOptions used to configure certain options to customize the E2E framework
// TanzuCommandPrefix should be set to customize the tanzu command prefix; default is tanzu
type E2EOptions struct {
	TanzuCommandPrefix string
	CLIOptions
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

// WithTanzuCommandPrefix is to set the tanzu command prefix; default is tanzu
func WithTanzuCommandPrefix(prefix string) E2EOption {
	return func(opts *E2EOptions) {
		opts.TanzuCommandPrefix = prefix
	}
}

// WithFilePath is the installation file path location
func WithFilePath(filePath string) E2EOption {
	return func(opts *E2EOptions) {
		opts.FilePath = filePath
	}
}

// WithOverride is to provide whether new Tanzu CLI overrides the installation of legacy Tanzu CLI
func WithOverride(override bool) E2EOption {
	return func(opts *E2EOptions) {
		opts.Override = override
	}
}

func NewFramework() *Framework {
	return &Framework{
		CliOps:       NewCliOps(),
		Config:       NewConfOps(),
		KindCluster:  NewKindCluster(NewDocker()),
		PluginCmd:    NewPluginLifecycleOps(),
		PluginHelper: NewPluginOps(NewScriptBasedPlugins(), NewLocalOCIPluginOps(NewLocalOCIRegistry(DefaultRegistryName, DefaultRegistryPort))),
		ContextCmd:   NewContextCmdOps(),
	}
}

func init() {
	homeDir := GetHomeDir()
	TestDirPath = filepath.Join(homeDir, TestDir)
	FullPathForTempDir = filepath.Join(TestDirPath, TempDirInTestDirPath)
	// Update $HOME as $HOME/.tanzu-cli-e2e
	os.Setenv("HOME", TestDirPath)
	TestPluginsDirPath = filepath.Join(TestDirPath, TestPluginsDir)
	TanzuFolderPath = filepath.Join(filepath.Join(TestDirPath, ConfigFolder), TanzuFolder)
	ConfigFilePath = filepath.Join(TanzuFolderPath, ConfigFile)
	ConfigNGFilePath = filepath.Join(TanzuFolderPath, ConfigNGFile)
	// Create a directory (if not exists) $HOME/.tanzu-cli-e2e/.config/tanzu-plugins/discovery/standalone
	TestStandalonePluginsPath = filepath.Join(filepath.Join(filepath.Join(filepath.Join(TestDirPath, ConfigFolder), TanzuPluginsFolder), "discovery"), "standalone")
	_ = CreateDir(TestStandalonePluginsPath)
	// Create a directory (if not exists) $HOME/.tanzu-cli-e2e/test
	_ = CreateDir(FullPathForTempDir)

	// TODO:cpamuluri: need to move plugins info to configuration file with positive and negative use cases - github issue: https://github.com/vmware-tanzu/tanzu-cli/issues/122
	PluginsForLifeCycleTests = []*PluginInfo{{Name: "cluster", Target: "kubernetes", Version: "v9.9.9", Description: "cluster functionality"}, {Name: "cluster", Target: "mission-control", Version: "v9.9.9", Description: "cluster functionality"}, {Name: "pinniped-auth", Target: "global", Version: "v9.9.9", Description: "pinniped-auth functionality"}}

	// TODO:cpamuluri: need to move Plugin Groups to configuration file with positive and negative use cases - github issue: https://github.com/vmware-tanzu/tanzu-cli/issues/122
	PluginGroupsForLifeCycleTests = []*PluginGroup{{Group: "vmware-tkg/v9.9.9"}, {Group: "vmware-tmc/v9.9.9"}}

	PluginGroupsLatestToOldVersions = make(map[string]string)
	PluginGroupsLatestToOldVersions["vmware-tmc/v9.9.9"] = "vmware-tmc/v0.0.1"
	PluginGroupsLatestToOldVersions["vmware-tkg/v9.9.9"] = "vmware-tkg/v0.0.1"
}
