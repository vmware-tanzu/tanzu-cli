// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig/fakes"
	"github.com/vmware-tanzu/tanzu-cli/pkg/datastore"
)

func TestUpdateConfigWithTanzuPlatformEndpointChanges(t *testing.T) {
	tests := []struct {
		name                           string
		existingContext                *configtypes.Context
		existingEndpointUpdateVersion  string
		requestedEndpointUpdateVersion string
		endpointUpdateMap              map[string]string
		expectedContext                *configtypes.Context
	}{
		{
			name: "When endpoint update version mismatch and endpoint needs to be updated - part1",
			existingContext: &configtypes.Context{
				Name:        "tanzu-context-1",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://api.fake.endpoint.example.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.fake.endpoint.example.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey:            "https://hub.fake.endpoint.example.com/hub",
					config.TanzuMissionControlEndpointKey: "https://tmc.fake.endpoint.example.com",
				},
			},
			existingEndpointUpdateVersion:  "",
			requestedEndpointUpdateVersion: "v1",
			endpointUpdateMap: map[string]string{
				"https://api.fake.endpoint.example.com":        "https://update-ucp.test.com",
				"https://hub.fake.endpoint.example.com":        "https://update-hub.test.com",
				"https://tmc.fake.endpoint.example.com":        "https://update-tmc.test.com",
				"https://additional.fake.endpoint.example.com": "https://update-additional.test.com",
			},
			expectedContext: &configtypes.Context{
				Name:        "tanzu-context-1",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://update-ucp.test.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://update-ucp.test.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey:            "https://update-hub.test.com/hub",
					config.TanzuMissionControlEndpointKey: "https://update-tmc.test.com",
				},
			},
		},
		{
			name: "When endpoint update version mismatch and endpoint needs to be updated - part2",
			existingContext: &configtypes.Context{
				Name:        "tanzu-context-2",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey:            "https://hub.fake-dev.endpoint2.example.com/hub",
					config.TanzuMissionControlEndpointKey: "https://tmc.fake-dev.endpoint.example.com",
				},
			},
			existingEndpointUpdateVersion:  "v1",
			requestedEndpointUpdateVersion: "v2",
			endpointUpdateMap: map[string]string{
				"https://api.fake-dev.endpoint.example.com": "https://update-ucp.test.com",
				"https://hub.fake-dev.endpoint.example.com": "https://update-hub.test.com",
				"https://tmc.fake-dev.endpoint.example.com": "https://update-tmc.test.com",
			},
			expectedContext: &configtypes.Context{
				Name:        "tanzu-context-2",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://update-ucp.test.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://update-ucp.test.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey:            "https://hub.fake-dev.endpoint2.example.com/hub", // does not get updated because no matching mapping
					config.TanzuMissionControlEndpointKey: "https://update-tmc.test.com",
				},
			},
		},
		{
			name: "When endpoint update version mismatch but no endpoint mapping is available, no updates should be made",
			existingContext: &configtypes.Context{
				Name:        "tanzu-context-3",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey: "https://hub.fake-dev.endpoint2.example.com/hub",
				},
			},
			existingEndpointUpdateVersion:  "",
			requestedEndpointUpdateVersion: "v1",
			endpointUpdateMap:              map[string]string{},
			expectedContext: &configtypes.Context{
				Name:        "tanzu-context-3",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey: "https://hub.fake-dev.endpoint2.example.com/hub",
				},
			},
		},
		{
			name: "When endpoint update version matches, no updates should be made even if matching endpoint mapping is found",
			existingContext: &configtypes.Context{
				Name:        "tanzu-context-3",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey: "https://hub.fake-dev.endpoint2.example.com/hub",
				},
			},
			existingEndpointUpdateVersion:  "v1",
			requestedEndpointUpdateVersion: "v1",
			endpointUpdateMap: map[string]string{
				"https://api.fake-dev.endpoint.example.com": "https://update-ucp.test.com",
				"https://hub.fake-dev.endpoint.example.com": "https://update-hub.test.com",
				"https://tmc.fake-dev.endpoint.example.com": "https://update-tmc.test.com",
			},
			expectedContext: &configtypes.Context{
				Name:        "tanzu-context-3",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey: "https://hub.fake-dev.endpoint2.example.com/hub",
				},
			},
		},
		{
			name: "When requestedEndpointVersion is empty, no updates should be made even if matching endpoint mapping is found",
			existingContext: &configtypes.Context{
				Name:        "tanzu-context-3",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey: "https://hub.fake-dev.endpoint2.example.com/hub",
				},
			},
			existingEndpointUpdateVersion:  "v1",
			requestedEndpointUpdateVersion: "",
			endpointUpdateMap: map[string]string{
				"https://api.fake-dev.endpoint.example.com": "https://update-ucp.test.com",
				"https://hub.fake-dev.endpoint.example.com": "https://update-hub.test.com",
				"https://tmc.fake-dev.endpoint.example.com": "https://update-tmc.test.com",
			},
			expectedContext: &configtypes.Context{
				Name:        "tanzu-context-3",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com",
				},
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://api.fake-dev.endpoint.example.com/org/random-org-id",
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuHubEndpointKey: "https://hub.fake-dev.endpoint2.example.com/hub",
				},
			},
		},
		{
			name: "When TMC context is present, updates should be made as well",
			existingContext: &configtypes.Context{
				Name:        "tmc-context-1",
				ContextType: configtypes.ContextTypeTMC,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://tmc.endpoint.example.com",
				},
			},
			existingEndpointUpdateVersion:  "",
			requestedEndpointUpdateVersion: "v1",
			endpointUpdateMap: map[string]string{
				"https://tmc.endpoint.example.com": "https://new.tmc.endpoint.test.com",
			},
			expectedContext: &configtypes.Context{
				Name:        "tmc-context-1",
				ContextType: configtypes.ContextTypeTMC,
				GlobalOpts: &configtypes.GlobalServer{
					Endpoint: "https://new.tmc.endpoint.test.com",
				},
			},
		},
		{
			name: "When k8s context is present, no updates should be made",
			existingContext: &configtypes.Context{
				Name:        "k8s-context-1",
				ContextType: configtypes.ContextTypeK8s,
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://management-cluster.endpoint.example.com",
				},
			},
			existingEndpointUpdateVersion:  "",
			requestedEndpointUpdateVersion: "v1",
			endpointUpdateMap: map[string]string{
				"https://management-cluster.endpoint.example.com": "https://new.management-cluster.endpoint.test.com",
			},
			expectedContext: &configtypes.Context{
				Name:        "k8s-context-1",
				ContextType: configtypes.ContextTypeK8s,
				ClusterOpts: &configtypes.ClusterServer{
					Endpoint: "https://management-cluster.endpoint.example.com",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup the base test environment for testing
			env := setupTestCLIEnvironment(t)
			defer tearDownTestCLIEnvironment(env)

			// Configure fake central configuration reader
			fakeDefaultCentralConfigReader := fakes.CentralConfig{}
			fakeDefaultCentralConfigReader.GetTanzuConfigEndpointUpdateVersionReturns(test.requestedEndpointUpdateVersion, nil)
			fakeDefaultCentralConfigReader.GetTanzuConfigEndpointUpdateMappingReturns(test.endpointUpdateMap, nil)
			centralconfig.DefaultCentralConfigReader = &fakeDefaultCentralConfigReader

			// Configure fake datastore file
			fakeDatastoreFile := filepath.Join(os.TempDir(), ".datastore")
			os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", fakeDatastoreFile)
			err := datastore.SetDataStoreValue(existingEndpointUpdateVersionKey, test.existingEndpointUpdateVersion)
			assert.NoError(t, err)
			defer func() {
				os.Unsetenv("TEST_CUSTOM_DATA_STORE_FILE")
				os.Remove(fakeDatastoreFile)
			}()

			// Configure existing contexts in the configuration file to run tests against
			err = config.SetContext(test.existingContext, false)
			assert.NoError(t, err)

			// Invoke the test function
			updateConfigWithTanzuPlatformEndpointChanges()

			// Verify that endpoints were updated as expected
			updatedContext, err := config.GetContext(test.expectedContext.Name)
			assert.NoError(t, err)
			if test.expectedContext.GlobalOpts != nil {
				assert.Equal(t, test.expectedContext.GlobalOpts.Endpoint, updatedContext.GlobalOpts.Endpoint)
			}
			if test.expectedContext.ClusterOpts != nil {
				assert.Equal(t, test.expectedContext.ClusterOpts.Endpoint, updatedContext.ClusterOpts.Endpoint)
			}
			assert.Equal(t, test.expectedContext.AdditionalMetadata, updatedContext.AdditionalMetadata)
		})
	}
}
