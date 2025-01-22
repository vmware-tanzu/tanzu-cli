// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lastversion

import (
	"os"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/datastore"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

func TestGetLastExecutedCLIVersion(t *testing.T) {
	tests := []struct {
		test            string
		lastVersion     string
		expectedVersion string
	}{
		{
			test:            "last version > 1.3.0",
			lastVersion:     "1.4.0",
			expectedVersion: "1.4.0",
		},
		{
			test:            "last version == 1.3.0",
			lastVersion:     "1.3.0",
			expectedVersion: "1.3.0",
		},
		{
			test:            "last version > 1.3.0 pre-release",
			lastVersion:     "2.0.1-dev",
			expectedVersion: "2.0.1-dev",
		},
		{
			test:            "last version older than 1.3.0 ",
			lastVersion:     "1.2.0",
			expectedVersion: olderThan1_3_0,
		},
	}

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

	tmpDataStoreFile, _ := os.CreateTemp("", "data-store.yaml")
	defer os.RemoveAll(tmpDataStoreFile.Name())
	os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDataStoreFile.Name())
	defer os.Unsetenv("TEST_CUSTOM_DATA_STORE_FILE")

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			if utils.IsNewVersion("v1.3.0", spec.lastVersion) {
				// We need to set the 'features.global.context-target-v2' feature flag
				// to indicate that the last version executed was < 1.3.0.
				parts := strings.Split(constants.FeatureContextCommand, ".")
				_ = config.SetFeature(parts[1], parts[2], "true")
			}
			// Set the last executed version in the datastore
			_ = datastore.SetDataStoreValue(lastExecutedCLIVersionKey, lastExecutedCLIVersion{Version: spec.lastVersion})

			// Get the last executed version and verify
			lastVersion := getLastExecutedCLIVersion()
			assert.Equal(spec.expectedVersion, lastVersion)

			// Clean up
			parts := strings.Split(constants.FeatureContextCommand, ".")
			_ = config.DeleteFeature(parts[1], parts[2])
		})
	}
}

func TestSetLastExecutedCLIVersion(t *testing.T) {
	tests := []struct {
		test            string
		lastVersion     string
		expectedVersion string
	}{
		{
			test:            "last version is 1.3.0",
			lastVersion:     "1.3.0",
			expectedVersion: "1.3.0",
		},
		{
			test:            "last version is > 1.3.0",
			lastVersion:     "1.4.0",
			expectedVersion: "1.4.0",
		},
		{
			test:            "last version is a pre-release",
			lastVersion:     "2.0.1-dev",
			expectedVersion: "2.0.1-dev",
		},
	}

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

	tmpDataStoreFile, _ := os.CreateTemp("", "data-store.yaml")
	defer os.RemoveAll(tmpDataStoreFile.Name())
	os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDataStoreFile.Name())
	defer os.Unsetenv("TEST_CUSTOM_DATA_STORE_FILE")

	originalVersion := buildinfo.Version
	defer func() {
		buildinfo.Version = originalVersion
	}()

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			// Run the test twice.  Once with the 'features.global.context-target-v2' feature flag set
			// and once without it set.
			for i := 0; i < 2; i++ {
				if i == 1 {
					parts := strings.Split(constants.FeatureContextCommand, ".")
					_ = config.SetFeature(parts[1], parts[2], "true")
				}

				buildinfo.Version = spec.lastVersion
				SetLastExecutedCLIVersion()

				// Get the last executed version and verify
				var lastVersion lastExecutedCLIVersion
				err := datastore.GetDataStoreValue(lastExecutedCLIVersionKey, &lastVersion)
				assert.Nil(err)
				assert.Equal(spec.expectedVersion, lastVersion.Version)

				// Check that the 'features.global.context-target-v2' feature flag is removed
				assert.False(config.IsFeatureActivated(constants.FeatureContextCommand))
			}
		})
	}
}

func TestIsLessThan(t *testing.T) {
	tests := []struct {
		test            string
		lastVersion     string
		lessThanVersion string
		expectedResult  bool
	}{
		{
			test:            "last version older than 1.3.0 compared to 1.5.0",
			lastVersion:     olderThan1_3_0,
			lessThanVersion: "1.5.0",
			expectedResult:  true,
		},
		{
			test:            "last version older than 1.3.0 compared to 2.0.0",
			lastVersion:     olderThan1_3_0,
			lessThanVersion: "2.0.0",
			expectedResult:  true,
		},
		{
			// Not supported so returns false
			test:            "last version older than 1.3.0 compared to 1.2.0",
			lastVersion:     olderThan1_3_0,
			lessThanVersion: "1.2.0",
			expectedResult:  false,
		},
		{
			test:            "last version 1.5.3 compared to 1.3.0",
			lastVersion:     "1.5.3",
			lessThanVersion: "1.3.0",
			expectedResult:  false,
		},
		{
			test:            "last version 1.5.3 compared to 1.5.0",
			lastVersion:     "1.5.3",
			lessThanVersion: "1.5.0",
			expectedResult:  false,
		},
		{
			test:            "last version 1.5.3 compared to 1.5.4",
			lastVersion:     "1.5.3",
			lessThanVersion: "1.5.4",
			expectedResult:  true,
		},
		{
			test:            "last version 1.5.3 compared to 1.6.0",
			lastVersion:     "1.5.3",
			lessThanVersion: "1.6.0",
			expectedResult:  true,
		},
		{
			test:            "last version 1.5.3 compared to 2.0.0",
			lastVersion:     "1.5.3",
			lessThanVersion: "2.0.0",
			expectedResult:  true,
		},
	}

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

	tmpDataStoreFile, _ := os.CreateTemp("", "data-store.yaml")
	defer os.RemoveAll(tmpDataStoreFile.Name())
	os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDataStoreFile.Name())
	defer os.Unsetenv("TEST_CUSTOM_DATA_STORE_FILE")

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			if spec.lastVersion == olderThan1_3_0 {
				// We need to set the 'features.global.context-target-v2' feature flag
				// to indicate that the last version executed was < 1.3.0.
				parts := strings.Split(constants.FeatureContextCommand, ".")
				_ = config.SetFeature(parts[1], parts[2], "true")
			} else {
				// Set the last executed version in the datastore
				_ = datastore.SetDataStoreValue(lastExecutedCLIVersionKey, lastExecutedCLIVersion{Version: spec.lastVersion})
			}

			// Compare the versions
			assert.Equal(spec.expectedResult, IsLessThan(semver.MustParse(spec.lessThanVersion)))

			// Clean up
			parts := strings.Split(constants.FeatureContextCommand, ".")
			_ = config.DeleteFeature(parts[1], parts[2])
		})
	}
}
