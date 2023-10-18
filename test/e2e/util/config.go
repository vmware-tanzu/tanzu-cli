// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package util contains utility functions for the CLI e2e tests.
package util

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const CliE2EConfigInputFilePath = "TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH"

type InputConfigData struct {
	PluginsForLifeCycleTests      []*framework.PluginInfo  `json:"PluginsForLifeCycleTests"`
	PluginGroupsForLifeCycleTests []*framework.PluginGroup `json:"PluginGroupsForLifeCycleTests"`
	EssentialPluginGroups         []*framework.PluginGroup `json:"EssentialPluginGroups"`
}

// PluginsForLifeCycleTests is list of plugins (which are published in local central repo) used in plugin life cycle test cases
var PluginsForLifeCycleTests []*framework.PluginInfo

// PluginGroupsForLifeCycleTests is list of plugin groups (which are published in local central repo) used in plugin group life cycle test cases
var PluginGroupsForLifeCycleTests []*framework.PluginGroup

// EssentialPluginGroups is list of essential plugin groups
var EssentialPluginGroups []*framework.PluginGroup

// init reads the input config file data and initializes the required variables for the CLI e2e tests.
func init() {
	GetInputConfigData()
}

// GetInputConfigData returns the input config data for the CLI e2e tests.
func GetInputConfigData() {
	// Read the file path from the environment variable
	configFile := os.Getenv(CliE2EConfigInputFilePath)
	log.Infof("reading input config data from file %s", configFile)
	if configFile == "" {
		log.Fatal(nil, fmt.Sprintf("the environment variable %s is not set to read the input config data for CLI E2E tests", CliE2EConfigInputFilePath))
		return
	}

	// Configure Viper with the file path and type
	viper.SetConfigType("json")
	viper.SetConfigFile(configFile)

	// Read and unmarshal the configuration
	var confData InputConfigData

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err, fmt.Sprintf("error while reading input config file %s", configFile))
		return
	}

	if err := viper.Unmarshal(&confData); err != nil {
		log.Fatal(err, fmt.Sprintf("error while unmarshaling config file %s", configFile))
		return
	}

	PluginsForLifeCycleTests = confData.PluginsForLifeCycleTests
	PluginGroupsForLifeCycleTests = confData.PluginGroupsForLifeCycleTests
	EssentialPluginGroups = confData.EssentialPluginGroups
}
