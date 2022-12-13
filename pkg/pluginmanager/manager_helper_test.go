// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aunum/log"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"

	cliv1alpha1 "github.com/vmware-tanzu/tanzu-framework/apis/cli/v1alpha1"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
)

func findDiscoveredPlugin(discovered []discovery.Discovered, pluginName string, target cliv1alpha1.Target) *discovery.Discovered {
	for i := range discovered {
		if pluginName == discovered[i].Name && target == discovered[i].Target {
			return &discovered[i]
		}
	}
	return nil
}

func findPluginInfo(pd []cli.PluginInfo, pluginName string, target cliv1alpha1.Target) *cli.PluginInfo {
	for i := range pd {
		if pluginName == pd[i].Name && target == pd[i].Target {
			return &pd[i]
		}
	}
	return nil
}

func setupLocalDistoForTesting() func() {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		log.Fatal(err, "unable to create temporary directory")
	}

	tmpHomeDir, err := os.MkdirTemp(os.TempDir(), "home")
	if err != nil {
		log.Fatal(err, "unable to create temporary home directory")
	}

	config.DefaultStandaloneDiscoveryType = "local"
	config.DefaultStandaloneDiscoveryLocalPath = "default"

	common.DefaultPluginRoot = filepath.Join(tmpDir, "plugin-root")
	common.DefaultLocalPluginDistroDir = filepath.Join(tmpDir, "distro")
	common.DefaultCacheDir = filepath.Join(tmpDir, "cache")

	tkgConfigFile := filepath.Join(tmpDir, "tanzu_config.yaml")
	tkgConfigNextGenFile := filepath.Join(tmpDir, "tanzu_config_ng.yaml")
	os.Setenv("TANZU_CONFIG", tkgConfigFile)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigNextGenFile)
	os.Setenv("HOME", tmpHomeDir)

	err = copy.Copy(filepath.Join("test", "local"), common.DefaultLocalPluginDistroDir)
	if err != nil {
		log.Fatal(err, "Error while setting local distro for testing")
	}

	err = copy.Copy(filepath.Join("test", "config.yaml"), tkgConfigFile)
	if err != nil {
		log.Fatal(err, "Error while coping tanzu config file for testing")
	}

	err = copy.Copy(filepath.Join("test", "config-ng.yaml"), tkgConfigNextGenFile)
	if err != nil {
		log.Fatal(err, "Error while coping tanzu config next gen file for testing")
	}

	err = configlib.SetFeature("global", "context-target-v2", "true")
	if err != nil {
		log.Fatal(err, "Error while coping tanzu config file for testing")
	}

	return func() {
		os.RemoveAll(tmpDir)
	}
}

func mockInstallPlugin(assert *assert.Assertions, name, version string, target cliv1alpha1.Target) {
	execCommand = fakeInfoExecCommand
	defer func() { execCommand = exec.Command }()

	err := InstallPlugin(name, version, target)
	assert.Nil(err)
}

// Reference: https://jamiethompson.me/posts/Unit-Testing-Exec-Command-In-Golang/
func fakeInfoExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...) //nolint:gosec
	tc := "FILE_PATH=" + command
	home := "HOME=" + os.Getenv("HOME")
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", tc, home}
	return cmd
}
