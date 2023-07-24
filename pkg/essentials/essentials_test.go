// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package essentials

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

// TestGetEssentialsPluginGroupDetails tests the GetEssentialsPluginGroupDetails function.
func TestGetEssentialsPluginGroupDetails(t *testing.T) {
	tests := []struct {
		name     string
		envName  string
		envVer   string
		wantName string
		wantVer  string
	}{
		{
			name:     "Default values",
			envName:  "",
			envVer:   "",
			wantName: constants.DefaultCLIEssentialsPluginGroupName,
			wantVer:  "",
		},
		{
			name:     "Environment variable set for name",
			envName:  "customName",
			envVer:   "",
			wantName: "customName",
			wantVer:  "",
		},
		{
			name:     "Environment variable set for version",
			envName:  "",
			envVer:   "1.0.0",
			wantName: constants.DefaultCLIEssentialsPluginGroupName,
			wantVer:  "1.0.0",
		},
		{
			name:     "Environment variables set for both name and version",
			envName:  "customName",
			envVer:   "1.0.0",
			wantName: "customName",
			wantVer:  "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables.
			err := os.Setenv(constants.TanzuCLIEssentialsPluginGroupName, tt.envName)
			assert.Nil(t, err)
			err = os.Setenv(constants.TanzuCLIEssentialsPluginGroupVersion, tt.envVer)
			assert.Nil(t, err)

			// Call the function and check the results.
			gotName, gotVer := GetEssentialsPluginGroupDetails()
			if gotName != tt.wantName || gotVer != tt.wantVer {
				t.Errorf("GetEssentialsPluginGroupDetails() = (%v, %v), want (%v, %v)", gotName, gotVer, tt.wantName, tt.wantVer)
			}

			// Clean up environment variables.
			err = os.Unsetenv(constants.TanzuCLIEssentialsPluginGroupName)
			assert.Nil(t, err)

			err = os.Unsetenv(constants.TanzuCLIEssentialsPluginGroupVersion)
			assert.Nil(t, err)
		})
	}
}
