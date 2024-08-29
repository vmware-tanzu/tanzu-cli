// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package centralconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
)

func TestGetDefaultTanzuEndpoint(t *testing.T) {
	tcs := []struct {
		name             string
		cfgContent       string
		expectedEndpoint interface{}
		expectError      bool
	}{
		{
			name:             "when default endpoint does not exist",
			cfgContent:       "testKey: testValue",
			expectedEndpoint: "",
			expectError:      true,
		},
		{
			name: "when default endpoint exists",
			cfgContent: `
testKey: testValue
cli.core.tanzu_default_endpoint: https://fake.endpoint.example.com
`,
			expectedEndpoint: "https://fake.endpoint.example.com",
		},
		{
			name: "when default endpoint key exists as empty value",
			cfgContent: `
testKey: testValue
cli.core.tanzu_default_endpoint: ""
`,
			expectedEndpoint: "",
		},
	}

	dir, err := os.MkdirTemp("", "test-central-config")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)
	common.DefaultCacheDir = dir
	reader := newCentralConfigReader("my_discovery")

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			path := reader.(*centralConfigYamlReader).configFile
			// Write the central config test content to the file
			err = os.MkdirAll(filepath.Dir(path), 0755)
			assert.Nil(t, err)

			err = os.WriteFile(path, []byte(tc.cfgContent), 0644)
			assert.Nil(t, err)

			endpoint, err := reader.GetDefaultTanzuEndpoint()

			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.expectedEndpoint, endpoint)
		})
	}
}

func TestGetTanzuPlatformEndpointToServiceEndpointMap(t *testing.T) {
	tcs := []struct {
		name                           string
		cfgContent                     string
		tpEndpointToServiceEndpointMap TanzuPlatformEndpointToServiceEndpointMap
		expectError                    bool
	}{
		{
			name:                           "when endpoint map does not exist",
			cfgContent:                     "",
			tpEndpointToServiceEndpointMap: TanzuPlatformEndpointToServiceEndpointMap{},
			expectError:                    true,
		},
		{
			name: "when endpoint map exists",
			cfgContent: `
cli.core.tanzu_endpoint_map:
  https://fake.endpoint1.example.com:
    ucp: https://custom.ucp.fake.endpoint1.example.com
    tmc: https://custom.tmc.fake.endpoint1.example.com
    hub: https://custom.hub.fake.endpoint1.example.com
  https://fake.endpoint2.example.com:
    ucp: https://ucp.fake.endpoint2.example.com
    tmc: https://tmc.fake.endpoint2.example.com
    hub: https://hub.fake.endpoint2.example.com
`,
			tpEndpointToServiceEndpointMap: map[string]ServiceEndpointMap{
				"https://fake.endpoint1.example.com": {
					UCPEndpoint: "https://custom.ucp.fake.endpoint1.example.com",
					TMCEndpoint: "https://custom.tmc.fake.endpoint1.example.com",
					HubEndpoint: "https://custom.hub.fake.endpoint1.example.com",
				},
				"https://fake.endpoint2.example.com": {
					UCPEndpoint: "https://ucp.fake.endpoint2.example.com",
					TMCEndpoint: "https://tmc.fake.endpoint2.example.com",
					HubEndpoint: "https://hub.fake.endpoint2.example.com",
				},
			},
			expectError: false,
		},
		{
			name: "when endpoint map exists but the format is different and all keys cannot be parsed",
			cfgContent: `
testKey: testValue
cli.core.tanzu_endpoint_map:
  https://fake.endpoint1.example.com:
    extra: https://custom.tmc.fake.endpoint1.example.com
  https://fake.endpoint2.example.com:
    fake: https://ucp.fake.endpoint2.example.com
`,
			tpEndpointToServiceEndpointMap: map[string]ServiceEndpointMap{
				"https://fake.endpoint1.example.com": {
					UCPEndpoint: "",
					TMCEndpoint: "",
					HubEndpoint: "",
				},
				"https://fake.endpoint2.example.com": {
					UCPEndpoint: "",
					TMCEndpoint: "",
					HubEndpoint: "",
				},
			},
			expectError: false,
		},
	}

	dir, err := os.MkdirTemp("", "test-central-config")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)
	common.DefaultCacheDir = dir
	reader := newCentralConfigReader("my_discovery")

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			path := reader.(*centralConfigYamlReader).configFile
			// Write the central config test content to the file
			err = os.MkdirAll(filepath.Dir(path), 0755)
			assert.Nil(t, err)

			err = os.WriteFile(path, []byte(tc.cfgContent), 0644)
			assert.Nil(t, err)

			tpEndpointToServiceEndpointMap, err := reader.GetTanzuPlatformEndpointToServiceEndpointMap()
			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.tpEndpointToServiceEndpointMap, tpEndpointToServiceEndpointMap)
		})
	}
}

func TestGetTanzuPlatformSaaSEndpointList(t *testing.T) {
	tcs := []struct {
		name            string
		cfgContent      string
		tpSaaSEndpoints []string
	}{
		{
			name:            "when saas endpoint list does not exist, it should return default endpoints from local variable",
			cfgContent:      "",
			tpSaaSEndpoints: defaultSaaSEndpoints,
		},
		{
			name: "when saas endpoint list exists",
			cfgContent: `
cli.core.tanzu_cli_platform_saas_endpoints_as_regular_expression:
  - https://platform*.fake.example.com
  - https://api.fake.example.com
  - https://api.tanzu*.example.com
`,
			tpSaaSEndpoints: []string{"https://platform*.fake.example.com", "https://api.fake.example.com", "https://api.tanzu*.example.com"},
		},
		{
			name: "when saas endpoint list exists but the format is different and cannot be parsed, it should return default endpoints from local variable",
			cfgContent: `
cli.core.tanzu_cli_platform_saas_endpoints_as_regular_expression:
  https://platform*.fake.example.com
`,
			tpSaaSEndpoints: defaultSaaSEndpoints,
		},
	}

	dir, err := os.MkdirTemp("", "test-central-config")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)
	common.DefaultCacheDir = dir
	reader := newCentralConfigReader("my_discovery")

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			path := reader.(*centralConfigYamlReader).configFile
			// Write the central config test content to the file
			err = os.MkdirAll(filepath.Dir(path), 0755)
			assert.Nil(t, err)

			err = os.WriteFile(path, []byte(tc.cfgContent), 0644)
			assert.Nil(t, err)

			tpSaaSEndpoints := reader.GetTanzuPlatformSaaSEndpointList()
			assert.Equal(t, tc.tpSaaSEndpoints, tpSaaSEndpoints)
		})
	}
}

func TestGetTanzuConfigEndpointUpdateVersion(t *testing.T) {
	tcs := []struct {
		name                  string
		cfgContent            string
		expectedUpdateVersion interface{}
		expectError           bool
	}{
		{
			name:                  "when endpoint update version key does not exist",
			cfgContent:            "",
			expectedUpdateVersion: "",
			expectError:           true,
		},
		{
			name: "when endpoint update version key exists",
			cfgContent: `
testKey: testValue
cli.core.tanzu_cli_config_endpoint_update_version: v1
`,
			expectedUpdateVersion: "v1",
		},
		{
			name: "when default endpoint key exists as empty value",
			cfgContent: `
testKey: testValue
cli.core.tanzu_cli_config_endpoint_update_version: ""
`,
			expectedUpdateVersion: "",
		},
		{
			name: "when default endpoint key exists as numeric value",
			cfgContent: `
testKey: testValue
cli.core.tanzu_cli_config_endpoint_update_version: 10
`,
			expectedUpdateVersion: "10",
		},
	}

	dir, err := os.MkdirTemp("", "test-central-config")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)
	common.DefaultCacheDir = dir
	reader := newCentralConfigReader("my_discovery")

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			path := reader.(*centralConfigYamlReader).configFile
			// Write the central config test content to the file
			err = os.MkdirAll(filepath.Dir(path), 0755)
			assert.Nil(t, err)

			err = os.WriteFile(path, []byte(tc.cfgContent), 0644)
			assert.Nil(t, err)

			updateVersion, err := reader.GetTanzuConfigEndpointUpdateVersion()

			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tc.expectedUpdateVersion, updateVersion)
		})
	}
}

func TestGetTanzuConfigEndpointUpdateMapping(t *testing.T) {
	tcs := []struct {
		name              string
		cfgContent        string
		endpointUpdateMap map[string]string
		expectError       bool
	}{
		{
			name:              "when endpoint update map does not exist",
			cfgContent:        "",
			endpointUpdateMap: map[string]string{},
			expectError:       true,
		},
		{
			name: "when endpoint update map exists",
			cfgContent: `
cli.core.tanzu_cli_config_endpoint_update_mapping:
  https://fake.endpoint1.example.com: https://fake.endpoint2.example.com
  https://fake.endpoint3.example.com: https://fake.endpoint4.example.com
`,
			endpointUpdateMap: map[string]string{
				"https://fake.endpoint1.example.com": "https://fake.endpoint2.example.com",
				"https://fake.endpoint3.example.com": "https://fake.endpoint4.example.com",
			},
			expectError: false,
		},
		{
			name: "when endpoint update map exists but the format is different and cannot be parsed",
			cfgContent: `
cli.core.tanzu_cli_config_endpoint_update_mapping: https://fake.endpoint4.example.com
`,
			endpointUpdateMap: map[string]string{},
			expectError:       true,
		},
	}

	dir, err := os.MkdirTemp("", "test-central-config")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)
	common.DefaultCacheDir = dir
	reader := newCentralConfigReader("my_discovery")

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			path := reader.(*centralConfigYamlReader).configFile
			// Write the central config test content to the file
			err = os.MkdirAll(filepath.Dir(path), 0755)
			assert.Nil(t, err)

			err = os.WriteFile(path, []byte(tc.cfgContent), 0644)
			assert.Nil(t, err)

			endpointUpdateMap, err := reader.GetTanzuConfigEndpointUpdateMapping()
			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.endpointUpdateMap, endpointUpdateMap)
		})
	}
}
