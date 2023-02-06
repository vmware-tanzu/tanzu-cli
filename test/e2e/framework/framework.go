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
	ConfigServerDelete = "tanzu config server delete "

	// Plugin commands
	AddPluginSource      = "tanzu plugin source add --name %s --type %s --uri %s"
	DeletePluginSource   = "tanzu plugin source delete %s"
	ListPluginsCmdInJSON = "tanzu plugin list -o json"

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

	TestDir        = ".tanzu-cli-e2e"
	TestPluginsDir = ".e2e-test-plugins"
	TargetTypeTMC  = "mission-control"
	TargetTypeK8s  = "kubernetes"
)

var TestDirPath string
var TestPluginsDirPath string
var TestStandalonePluginsPath string

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
}
