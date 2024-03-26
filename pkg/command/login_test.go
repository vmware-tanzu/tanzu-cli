// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"testing"
)

func TestPrepareTanzuContextName(t *testing.T) {
	testCases := []struct {
		orgName   string
		endpoint  string
		isStaging bool
		expected  string
	}{
		// Test case for normal input with no staging environment and default endpoint.
		{
			orgName:   "MyOrg",
			endpoint:  defaultTanzuEndpoint,
			isStaging: false,
			expected:  "MyOrg",
		},
		// Test case for normal input with staging environment and default endpoint.
		{
			orgName:   "MyOrg",
			endpoint:  defaultTanzuEndpoint,
			isStaging: true,
			expected:  "MyOrg-staging",
		},
		// Test case for normal input with no staging environment and custom endpoint.
		{
			orgName:   "MyOrg",
			endpoint:  "https://custom-endpoint.com",
			isStaging: false,
			expected:  "MyOrg-86fd8133",
		},
		// Test case for normal input with staging environment and custom endpoint.
		{
			orgName:   "MyOrg",
			endpoint:  "https://custom-endpoint.com",
			isStaging: true,
			expected:  "MyOrg-staging-86fd8133",
		},
	}

	for _, tc := range testCases {
		actual := prepareTanzuContextName(tc.orgName, tc.endpoint, tc.isStaging)
		if actual != tc.expected {
			t.Errorf("orgName: %s, endpoint: %s, isStaging: %t - expected: %s, got: %s", tc.orgName, tc.endpoint, tc.isStaging, tc.expected, actual)
		}
	}
}
