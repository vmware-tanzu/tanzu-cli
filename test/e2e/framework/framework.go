// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package test defines the integration and end-to-end test case for cli core
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

	ConfigSet          = "tanzu config set "
	ConfigGet          = "tanzu config get "
	ConfigUnset        = "tanzu config unset "
	ConfigInit         = "tanzu config init"
	ConfigServerList   = "tanzu config server list"
	ConfigServerDelete = "tanzu config server delete "
	AddPluginSource    = "tanzu plugin source add --name %s --type %s --uri %s"
	DeletePluginSource = "tanzu plugin source delete %s"
	ListPluginsCmd     = "tanzu plugin list -o json"

	KindCreateCluster = "kind create cluster --name "
	DockerInfo        = "docker info"
	StartDockerUbuntu = "sudo systemctl start docker"
	StopDockerUbuntu  = "sudo systemctl stop docker"

	TestDir         = ".tanzu-cli-e2e"
	TestPluginsDir  = ".e2e-test-plugins"
	HackToolsBinDir = "/../../../hack/tools/bin"
	TanzuBinDir     = "/../../../bin"
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
	Config ConfigLifecycleOps
	ClusterOps
	PluginCmd    PluginCmdOps    // performs plugin command operations
	PluginHelper PluginHelperOps // helper (pre-setup) for plugin cmd operations
}

func NewFramework() *Framework {
	return &Framework{
		CliOps:       NewCliOps(),
		Config:       NewConfOps(),
		ClusterOps:   NewKindCluster(NewDocker()),
		PluginCmd:    NewPluginLifecycleOps(),
		PluginHelper: NewPluginOps(NewScriptBasedPlugins(), NewLocalOCIPluginOps(NewLocalOCIRegistry(DefaultRegistryName, DefaultRegistryPort))),
	}
}

func init() {
	homeDir, _ := os.UserHomeDir()
	TestDirPath = filepath.Join(homeDir, TestDir)
	os.Setenv("HOME", TestDirPath)
	pwd, _ := os.Getwd()
	os.Setenv("PATH", pwd+HackToolsBinDir+":"+pwd+TanzuBinDir+":"+os.Getenv("PATH"))
	TestPluginsDirPath = filepath.Join(TestDirPath, TestPluginsDir)
	TestStandalonePluginsPath = filepath.Join(filepath.Join(filepath.Join(filepath.Join(TestDirPath, ".config"), "tanzu-plugins"), "discovery"), "standalone")
	_ = CreateDir(TestStandalonePluginsPath)
}
