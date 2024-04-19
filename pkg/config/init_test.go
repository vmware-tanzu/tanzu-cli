// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains useful functionality for config updates
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

func TestFeatureContextCommandDoesNotGetSet(t *testing.T) {
	assert := assert.New(t)

	// Setup a temporary configuration
	configFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	err = os.Setenv("TANZU_CONFIG", configFile.Name())
	assert.Nil(err)
	defer os.RemoveAll(configFile.Name())
	defer os.Unsetenv("TANZU_CONFIG")

	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	err = os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
	assert.Nil(err)
	defer os.RemoveAll(configFileNG.Name())
	defer os.Unsetenv("TANZU_CONFIG_NEXT_GEN")

	runInit()

	// Starting with CLI 1.3.0 we no longer set the feature flag features.global.context-target-v2.
	// This is important because it allows us to determine if the last version executed was < 1.3.0.
	assert.False(config.IsFeatureActivated(constants.FeatureContextCommand))
}

func TestCentralRepoGetsSet(t *testing.T) {
	assert := assert.New(t)

	// Setup a temporary configuration
	configFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	err = os.Setenv("TANZU_CONFIG", configFile.Name())
	assert.Nil(err)
	defer os.RemoveAll(configFile.Name())
	defer os.Unsetenv("TANZU_CONFIG")

	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	err = os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
	assert.Nil(err)
	defer os.RemoveAll(configFileNG.Name())
	defer os.Unsetenv("TANZU_CONFIG_NEXT_GEN")

	// Check that the central repo gets set
	runInit()

	sources, err := config.GetCLIDiscoverySources()
	assert.Nil(err)
	assert.Equal(1, len(sources))
	assert.Equal(DefaultStandaloneDiscoveryName, sources[0].OCI.Name)
	assert.Equal(constants.TanzuCLIDefaultCentralPluginDiscoveryImage, sources[0].OCI.Image)

	// Check that the central repo does not get set if the user has deleted it
	err = config.DeleteCLIDiscoverySource(DefaultStandaloneDiscoveryName)
	assert.Nil(err)

	sources, err = config.GetCLIDiscoverySources()
	assert.Nil(err)
	assert.Equal(0, len(sources))

	runInit()

	sources, err = config.GetCLIDiscoverySources()
	assert.Nil(err)
	assert.Equal(0, len(sources))
}
