// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig/fakes"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

func TestConfigureTanzuPlatformServiceEndpoints(t *testing.T) {
	tests := []struct {
		name                                 string
		tpEndpoint                           string
		envVariableTanzuHubEndpoint          string
		envVariableTanzuUCPEndpoint          string
		envVariableTanzuTMCEndpoint          string
		saasEndpointListInCC                 []string
		saasEndpointToServiceEndpointMapInCC centralconfig.TanzuPlatformEndpointToServiceEndpointMap
		expectedTanzuHubEndpoint             string
		expectedTanzuUCPEndpoint             string
		expectedTanzuTMCEndpoint             string
		expectedErr                          string
	}{
		{
			name:        "empty endpoint",
			tpEndpoint:  "",
			expectedErr: "invalid endpoint",
		},
		{
			name:                                 "valid SaaS endpoint, endpoint not found in service endpoint map, uses fallback saas algorithm",
			tpEndpoint:                           "https://tanzu.vmware.com",
			saasEndpointListInCC:                 []string{"https://tanzu.vmware.com"},
			saasEndpointToServiceEndpointMapInCC: centralconfig.TanzuPlatformEndpointToServiceEndpointMap{},
			expectedTanzuHubEndpoint:             "https://api.tanzu.vmware.com/hub",
			expectedTanzuUCPEndpoint:             "https://ucp.tanzu.vmware.com",
			expectedTanzuTMCEndpoint:             "https://ops.tanzu.vmware.com",
		},
		{
			name:                                 "valid SaaS endpoint with http, endpoint not found in service endpoint map, uses fallback saas algorithm",
			tpEndpoint:                           "http://tanzu.vmware.com",
			saasEndpointListInCC:                 []string{"http://tanzu.vmware.com"},
			saasEndpointToServiceEndpointMapInCC: centralconfig.TanzuPlatformEndpointToServiceEndpointMap{},
			expectedTanzuHubEndpoint:             "http://api.tanzu.vmware.com/hub",
			expectedTanzuUCPEndpoint:             "http://ucp.tanzu.vmware.com",
			expectedTanzuTMCEndpoint:             "http://ops.tanzu.vmware.com",
		},
		{
			name:                                 "valid SaaS endpoint without scheme, endpoint not found in service endpoint map, uses fallback saas algorithm",
			tpEndpoint:                           "tanzu.vmware.com",
			saasEndpointListInCC:                 []string{"https://tanzu.vmware.com"},
			saasEndpointToServiceEndpointMapInCC: centralconfig.TanzuPlatformEndpointToServiceEndpointMap{},
			expectedTanzuHubEndpoint:             "https://api.tanzu.vmware.com/hub",
			expectedTanzuUCPEndpoint:             "https://ucp.tanzu.vmware.com",
			expectedTanzuTMCEndpoint:             "https://ops.tanzu.vmware.com",
		},
		{
			name:                                 "valid SaaS endpoint with www prefix, endpoint not found in service endpoint map, uses fallback saas algorithm",
			tpEndpoint:                           "https://www.tanzu.vmware.com",
			saasEndpointListInCC:                 []string{"https://(www.)?tanzu.vmware.com"},
			saasEndpointToServiceEndpointMapInCC: centralconfig.TanzuPlatformEndpointToServiceEndpointMap{},
			expectedTanzuHubEndpoint:             "https://api.tanzu.vmware.com/hub",
			expectedTanzuUCPEndpoint:             "https://ucp.tanzu.vmware.com",
			expectedTanzuTMCEndpoint:             "https://ops.tanzu.vmware.com",
		},
		{
			name:                        "when all endpoints are configured thought environment variables",
			tpEndpoint:                  "https://www.tanzu.vmware.com",
			envVariableTanzuHubEndpoint: "https://env.variable.hub.tanzu.vmware.com",
			envVariableTanzuUCPEndpoint: "https://env.variable.ucp.tanzu.vmware.com",
			envVariableTanzuTMCEndpoint: "https://env.variable.ops.tanzu.vmware.com",
			expectedTanzuHubEndpoint:    "https://env.variable.hub.tanzu.vmware.com/hub",
			expectedTanzuUCPEndpoint:    "https://env.variable.ucp.tanzu.vmware.com",
			expectedTanzuTMCEndpoint:    "https://env.variable.ops.tanzu.vmware.com",
		},
		{
			name:                                 "when some endpoints are configured thought environment variables",
			tpEndpoint:                           "https://www.tanzu.vmware.com",
			saasEndpointListInCC:                 []string{"https://(www.)?tanzu.vmware.com"},
			saasEndpointToServiceEndpointMapInCC: centralconfig.TanzuPlatformEndpointToServiceEndpointMap{},
			envVariableTanzuHubEndpoint:          "https://env.variable.hub.tanzu.vmware.com",
			expectedTanzuHubEndpoint:             "https://env.variable.hub.tanzu.vmware.com/hub",
			expectedTanzuUCPEndpoint:             "https://ucp.tanzu.vmware.com",
			expectedTanzuTMCEndpoint:             "https://ops.tanzu.vmware.com",
		},
		{
			name:                 "valid SaaS endpoint, endpoint found in service endpoint map, uses mapping from map",
			tpEndpoint:           "https://tanzu.vmware.com",
			saasEndpointListInCC: []string{"https://tanzu.vmware.com"},
			saasEndpointToServiceEndpointMapInCC: map[string]centralconfig.ServiceEndpointMap{
				"https://tanzu.vmware.com": {
					HubEndpoint: "https://customhub.tanzu.vmware.com",
					UCPEndpoint: "https://customucp.tanzu.vmware.com",
					TMCEndpoint: "https://customops.tanzu.vmware.com",
				},
			},
			expectedTanzuHubEndpoint: "https://customhub.tanzu.vmware.com/hub",
			expectedTanzuUCPEndpoint: "https://customucp.tanzu.vmware.com",
			expectedTanzuTMCEndpoint: "https://customops.tanzu.vmware.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the variables before running tests
			tanzuHubEndpoint, tanzuUCPEndpoint, tanzuTMCEndpoint = "", "", ""

			os.Setenv(constants.TPHubEndpoint, tt.envVariableTanzuHubEndpoint)
			os.Setenv(constants.TPUCPEndpoint, tt.envVariableTanzuUCPEndpoint)
			os.Setenv(constants.TPKubernetesOpsEndpoint, tt.envVariableTanzuTMCEndpoint)
			defer func() {
				os.Unsetenv(constants.TPHubEndpoint)
				os.Unsetenv(constants.TPUCPEndpoint)
				os.Unsetenv(constants.TPKubernetesOpsEndpoint)
			}()

			fakeDefaultCentralConfigReader := fakes.CentralConfig{}
			fakeDefaultCentralConfigReader.GetTanzuPlatformSaaSEndpointListReturns(tt.saasEndpointListInCC)
			fakeDefaultCentralConfigReader.GetTanzuPlatformEndpointToServiceEndpointMapReturns(tt.saasEndpointToServiceEndpointMapInCC, nil)
			centralconfig.DefaultCentralConfigReader = &fakeDefaultCentralConfigReader

			err := configureTanzuPlatformServiceEndpoints(tt.tpEndpoint)
			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
			}

			assert.Equal(t, tt.expectedTanzuHubEndpoint, tanzuHubEndpoint)
			assert.Equal(t, tt.expectedTanzuUCPEndpoint, tanzuUCPEndpoint)
			assert.Equal(t, tt.expectedTanzuTMCEndpoint, tanzuTMCEndpoint)
		})
	}
}

func TestIsTanzuPlatformSaaSEndpoint(t *testing.T) {
	tests := []struct {
		name                            string
		tpEndpoint                      string
		saasEndpointListInCentralConfig []string
		expected                        bool
	}{
		{
			name:                            "valid SaaS endpoint",
			tpEndpoint:                      "https://tanzu.vmware.com",
			saasEndpointListInCentralConfig: []string{"https://tanzu.vmware.com"},
			expected:                        true,
		},
		{
			name:                            "invalid SaaS endpoint",
			tpEndpoint:                      "https://example.com",
			saasEndpointListInCentralConfig: []string{"https://tanzu.vmware.com"},
			expected:                        false,
		},
		{
			name:                            "regex matching endpoint 1",
			tpEndpoint:                      "https://tanzu-dev.vmware.com",
			saasEndpointListInCentralConfig: []string{"https://tanzu.vmware.com", "https://tanzu.*\\.vmware\\.com"},
			expected:                        true,
		},
		{
			name:                            "regex matching endpoint 2",
			tpEndpoint:                      "https://tanzudev.vmware.com",
			saasEndpointListInCentralConfig: []string{"https://tanzu.vmware.com", "https://tanzu.*\\.vmware\\.com"},
			expected:                        true,
		},
		{
			name:                            "regex mismatching endpoint 1",
			tpEndpoint:                      "https://tanzudev.vmware1com",
			saasEndpointListInCentralConfig: []string{"https://tanzu.vmware.com", "https://tanzu.*\\.vmware\\.com"},
			expected:                        false,
		},
		{
			name:                            "regex mismatching endpoint 2",
			tpEndpoint:                      "https://dev.vmware.com",
			saasEndpointListInCentralConfig: []string{"https://tanzu.vmware.com", "https://tanzu.*\\.vmware\\.com"},
			expected:                        false,
		},
		{
			name:                            "empty string",
			tpEndpoint:                      "",
			saasEndpointListInCentralConfig: []string{""},
			expected:                        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeDefaultCentralConfigReader := fakes.CentralConfig{}
			fakeDefaultCentralConfigReader.GetTanzuPlatformSaaSEndpointListReturns(tt.saasEndpointListInCentralConfig)
			centralconfig.DefaultCentralConfigReader = &fakeDefaultCentralConfigReader

			actual := isTanzuPlatformSaaSEndpoint(tt.tpEndpoint)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
