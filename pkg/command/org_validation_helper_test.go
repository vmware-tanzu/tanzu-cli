// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"errors"
	"testing"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
)

func TestValidateTokenForTanzuScopes(t *testing.T) {
	tests := []struct {
		name         string
		claims       *common.Claims
		scopesGetter tapScopesGetter
		expectedBool bool
		expectedErr  error
	}{
		{
			name:         "When the central config does not have any Tanzu Platform for Kubernetes scopes listed, it should return success.",
			claims:       &common.Claims{Permissions: []string{}},
			scopesGetter: func() ([]string, error) { return []string{}, nil },
			expectedBool: true,
			expectedErr:  nil,
		},
		{
			name:   "When the token has at least one of the specified Tanzu Platform for Kubernetes scopes listed in central config, it should return success",
			claims: &common.Claims{Permissions: []string{"external/UID-123-567/matching-scope", "csp:org_admin", "csp:developer"}},
			scopesGetter: func() ([]string, error) {
				return []string{"matching-scope", "matching-another-scope"}, nil
			},
			expectedBool: true,
			expectedErr:  nil,
		},
		{
			name:   "When the token lacks at least one of the specified Tanzu Platform for Kubernetes scopes listed in the central config, it should return an error",
			claims: &common.Claims{Permissions: []string{"external/UID-123-567/non-matching-scope", "csp:org_member", "csp:software_installer"}},
			scopesGetter: func() ([]string, error) {
				return []string{"matching-scope", "matching-another-scope"}, nil
			},
			expectedBool: false,
			expectedErr:  nil,
		},
		{
			name:   "It should return an error if fetching the Tanzu Platform for Kubernetes scopes from the central config fails",
			claims: &common.Claims{},
			scopesGetter: func() ([]string, error) {
				return nil, errors.New("error retrieving Tanzu Platform for Kubernetes scopes")
			},
			expectedBool: false,
			expectedErr:  errors.New("error retrieving Tanzu Platform for Kubernetes scopes from the central config: error retrieving Tanzu Platform for Kubernetes scopes"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualBool, actualErr := validateTokenForTAPScopes(tc.claims, tc.scopesGetter)
			if actualBool != tc.expectedBool {
				t.Errorf("Test case %s failed: Expected bool value %t, but got %t", tc.name, tc.expectedBool, actualBool)
			}
			if actualErr != nil && tc.expectedErr != nil && actualErr.Error() != tc.expectedErr.Error() {
				t.Errorf("Test case %s failed: Expected error %v, but got %v", tc.name, tc.expectedErr, actualErr)
			}
		})
	}
}
