// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package framework defines the integration and end-to-end test case for cli core
package framework

import (
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo"
)

const (
	CliCore = "[CLI-Core]"

	TanzuInit    = "tanzu init"
	TanzuVersion = "tanzu version"

	// Config commands
	ConfigSet          = "tanzu config set "
	ConfigGet          = "tanzu config get "
	ConfigUnset        = "tanzu config unset "
	ConfigInit         = "tanzu config init"
	ConfigServerList   = "tanzu config server list"
	ConfigServerDelete = "tanzu config server delete %s -y"

	// Plugin commands
	AddPluginSource                     = "tanzu plugin source add --name %s --type %s --uri %s"
	UpdatePluginSource                  = "tanzu plugin source update %s --type %s --uri %s"
	ListPluginSourcesWithJSONOutputFlag = "tanzu plugin source list -o json"
	DeletePluginSource                  = "tanzu plugin source delete %s"
	ListPluginsCmdWithJSONOutputFlag    = "tanzu plugin list -o json"
	SearchPluginsCmd                    = "tanzu plugin search"
	SearchPluginGroupsCmd               = "tanzu plugin group search"
	InstallPluginCmd                    = "tanzu plugin install %s"
	InstallPluginFromGroupCmd           = "tanzu plugin install %s --group %s"
	InstallAllPluginsFromGroupCmd       = "tanzu plugin install --group %s"
	DescribePluginCmd                   = "tanzu plugin describe %s"
	UninstallPLuginCmd                  = "tanzu plugin delete %s --yes"
	CleanPluginsCmd                     = "tanzu plugin clean"
	JSONOutput                          = " -o json"
	TestPluginsPrefix                   = "test-plugin-"
	PluginSubCommand                    = "tanzu %s"
	PluginKey                           = "%s_%s_%s" // Plugins - Name_Target_Versions

	// Central repository
	CentralRepositoryPreReleaseRepoImage     = "TANZU_CLI_PRE_RELEASE_REPO_IMAGE"
	TanzuCliE2ETestCentralRepositoryURL      = "TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL"
	TanzuCliE2ETestLocalCentralRepositoryURL = "TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL"

	// General constants
	True      = "true"
	Installed = "installed"

	// Context commands
	CreateContextWithEndPoint              = "tanzu context create --endpoint %s --name %s"
	CreateContextWithEndPointStaging       = "tanzu context create --endpoint %s --name %s --staging"
	CreateContextWithKubeconfigFile        = "tanzu context create --kubeconfig %s --kubecontext %s --name %s"
	CreateContextWithDefaultKubeconfigFile = "tanzu context create --kubecontext %s --name %s"
	UseContext                             = "tanzu context use %s"
	GetContext                             = "tanzu context get %s"
	ListContextOutputInJSON                = "tanzu context list -o json"
	DeleteContext                          = "tanzu context delete %s --yes"
	TanzuAPIToken                          = "TANZU_API_TOKEN" //nolint:gosec
	TanzuCliTmcUnstableURL                 = "TANZU_CLI_TMC_UNSTABLE_URL"

	KindClusterCreate = "kind create cluster --name %s"
	KindClusterStatus = "kubectl cluster-info --context %s"
	KindClusterDelete = "kind delete cluster --name %s"
	KindClusterGet    = "kind get clusters "
	KindClusterInfo   = "kubectl cluster-info --context %s"

	KindCreateCluster = "kind create cluster --name "
	DockerInfo        = "docker info"
	StartDockerUbuntu = "sudo systemctl start docker"
	StopDockerUbuntu  = "sudo systemctl stop docker"

	TMC            = "tmc"
	TKG            = "tkg"
	TestDir        = ".tanzu-cli-e2e"
	TestPluginsDir = ".e2e-test-plugins"
	SourceType     = "oci"
	GlobalTarget   = "global"

	// log info
	ExecutingCommand = "Executing command: %s"

	// Error messages
	UnableToFindPluginForTarget                   = "unable to find plugin '%s' for target '%s'"
	UnableToFindPlugin                            = "unable to find plugin '%s'"
	InvalidTargetSpecified                        = "invalid target specified. Please specify correct value of `--target` or `-t` flag from 'global/kubernetes/k8s/mission-control/tmc'"
	InvalidTargetGlobal                           = "invalid target for plugin: global"
	UnknownDiscoverySourceType                    = "unknown discovery source type"
	DiscoverySourceNotFound                       = "cli discovery source not found"
	ErrorLogForCommandWithErrAndStdErr            = "error while executing command:'%s', error:'%s' stdErr:'%s'"
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

var (
	TestDirPath               string
	TestPluginsDirPath        string
	TestStandalonePluginsPath string
)

// PluginsForLifeCycleTests is list of plugins (which are published in local central repo) used in plugin life cycle test cases
var PluginsForLifeCycleTests []*PluginInfo

// PluginGroupsForLifeCycleTests is list of plugin groups (which are published in local central repo) used in plugin group life cycle test cases
var PluginGroupsForLifeCycleTests []*PluginGroup

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
	os.Setenv("HOME", TestDirPath)
	TestPluginsDirPath = filepath.Join(TestDirPath, TestPluginsDir)
	TestStandalonePluginsPath = filepath.Join(filepath.Join(filepath.Join(filepath.Join(TestDirPath, ".config"), "tanzu-plugins"), "discovery"), "standalone")
	_ = CreateDir(TestStandalonePluginsPath)
	// TODO:cpamuluri: need to move plugins info to configuration file with positive and negative use cases - github issue: https://github.com/vmware-tanzu/tanzu-cli/issues/122
	PluginsForLifeCycleTests = make([]*PluginInfo, 3)
	PluginsForLifeCycleTests = []*PluginInfo{{Name: "cluster", Target: "kubernetes", Version: "v9.9.9", Description: "cluster functionality"}, {Name: "cluster", Target: "mission-control", Version: "v9.9.9", Description: "cluster functionality"}, {Name: "pinniped-auth", Target: "global", Version: "v9.9.9", Description: "pinniped-auth functionality"}}

	// TODO:cpamuluri: need to move Plugin Groups to configuration file with positive and negative use cases - github issue: https://github.com/vmware-tanzu/tanzu-cli/issues/122
	PluginGroupsForLifeCycleTests = make([]*PluginGroup, 2)
	PluginGroupsForLifeCycleTests = []*PluginGroup{{Group: "vmware-tmc/v9.9.9"}, {Group: "vmware-tkg/v9.9.9"}}
}
