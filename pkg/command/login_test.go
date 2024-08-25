// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"testing"

	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
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
			endpoint:  centralconfig.DefaultTanzuPlatformEndpoint,
			isStaging: false,
			expected:  "MyOrg",
		},
		// Test case for normal input with staging environment and default endpoint.
		{
			orgName:   "MyOrg",
			endpoint:  centralconfig.DefaultTanzuPlatformEndpoint,
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

func TestIsSubDomain(t *testing.T) {
	tests := []struct {
		name    string
		parent  string
		child   string
		want    bool
		wantErr bool
	}{
		{"same host", "https://example.vmware.com", "https://example.vmware.com", true, false},
		{"same host different protocol", "http://example.vmware.com", "https://example.vmware.com", false, false},
		{"subdomain", "https://example.vmware.com", "https://child.example.vmware.com", true, false},
		{"not subdomain", "https://example.vmware.com", "https://child.random.vmware.com", false, false},
		{"not subdomain different protocol", "http://example.vmware.com", "https://child.random.vmware.com", false, false},
		{"not subdomain different host", "https://example.vmware.com", "https://child.random.random.com", false, false},
		{"invalid parent", "invalid://example.vmware.com", "https://example.vmware.com", false, true},
		{"invalid child", "https://example.vmware.com", "invalid://example.vmware.com", false, true},
		{"parent with path", "https://example.vmware.com/path", "https://example.vmware.com", false, false},
		{"child with path", "https://example.vmware.com", "https://example.vmware.com/path", false, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := isSubdomain(tt.parent, tt.child)
			if got != tt.want {
				t.Errorf("isSubdomain(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}
