// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package centralconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

func TestTriggerForInventoryCacheInvalidation(t *testing.T) {
	tcs := []struct {
		name                 string
		numDiscoveries       int
		missingCentralConfig []bool
		expectedToTrigger    bool
	}{
		{
			name:              "No discovery sources",
			numDiscoveries:    0,
			expectedToTrigger: false,
		},
		{
			name:                 "One discovery source with central config",
			numDiscoveries:       1,
			missingCentralConfig: []bool{false},
			expectedToTrigger:    false,
		},
		{
			name:                 "One discovery source with missing central config",
			numDiscoveries:       1,
			missingCentralConfig: []bool{true},
			expectedToTrigger:    true,
		},
		{
			name:                 "Two discovery sources both with central config",
			numDiscoveries:       2,
			missingCentralConfig: []bool{false, false},
			expectedToTrigger:    false,
		},
		{
			name:                 "Two discovery sources with only one missing central config",
			numDiscoveries:       2,
			missingCentralConfig: []bool{false, true},
			expectedToTrigger:    true,
		},
	}

	configFile, err := os.CreateTemp("", "config")
	assert.Nil(t, err)
	os.Setenv("TANZU_CONFIG", configFile.Name())

	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(t, err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

	defer func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())
	}()

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Create a cache directory for the plugin inventory and the central config file
			cacheDir, err := os.MkdirTemp("", "test-cache-dir")
			assert.Nil(t, err)

			common.DefaultCacheDir = cacheDir

			// Create the discovery sources
			var discoveries []types.PluginDiscovery
			for i := 0; i < tc.numDiscoveries; i++ {
				discName := fmt.Sprintf("discovery%d", i)
				discoveries = append(discoveries, types.PluginDiscovery{
					OCI: &types.OCIDiscovery{
						Name: discName,
					},
				})

				// Create the directory for this discovery source
				centralCfgDir := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, discName)
				err = os.MkdirAll(centralCfgDir, 0755)
				assert.Nil(t, err)

				// Create the central config file if needed by the test
				if !tc.missingCentralConfig[i] {
					centralCfgFile := filepath.Join(centralCfgDir, constants.CentralConfigFileName)
					file, err := os.Create(centralCfgFile)
					assert.Nil(t, err)
					assert.NotNil(t, file)
					file.Close()
				}
			}

			err = config.SetCLIDiscoverySources(discoveries)
			assert.Nil(t, err)

			assert.Equal(t, triggerForInventoryCacheInvalidation(), tc.expectedToTrigger)

			os.RemoveAll(cacheDir)
		})
	}
}
