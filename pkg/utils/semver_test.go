// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"testing"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/assert"
)

func TestSortVersion(t *testing.T) {
	tcs := []struct {
		name string
		act  []string
		exp  []string
		err  error
	}{
		{
			name: "Success",
			act:  []string{"v1.0.0", "v0.0.1", "0.0.1-dev"},
			exp:  []string{"0.0.1-dev", "v0.0.1", "v1.0.0"},
		},
		{
			name: "Success",
			act:  []string{"1.0.0", "0.0.a"},
			exp:  []string{"1.0.0", "0.0.a"},
			err:  semver.ErrInvalidSemVer,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := SortVersions(tc.act)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.exp, tc.act)
		})
	}
}

// TestIsNewVersion tests the IsNewVersion function.
func TestIsNewVersion(t *testing.T) {
	tests := []struct {
		name                string
		pluginVersionStr    string
		installedVersionStr string
		want                bool
	}{
		{
			name:                "New version available",
			pluginVersionStr:    "2.0.0",
			installedVersionStr: "1.0.0",
			want:                true,
		},
		{
			name:                "Same version",
			pluginVersionStr:    "1.0.0",
			installedVersionStr: "1.0.0",
			want:                false,
		},
		{
			name:                "Old version",
			pluginVersionStr:    "1.0.0",
			installedVersionStr: "2.0.0",
			want:                false,
		},
		{
			name:                "Invalid plugin version",
			pluginVersionStr:    "invalid",
			installedVersionStr: "1.0.0",
			want:                false,
		},
		{
			name:                "Invalid installed version",
			pluginVersionStr:    "1.0.0",
			installedVersionStr: "invalid",
			want:                false,
		},

		{
			name:                "New version available",
			pluginVersionStr:    "v2.0.0",
			installedVersionStr: "v1.0.0",
			want:                true,
		},
		{
			name:                "Same version",
			pluginVersionStr:    "v1.0.0",
			installedVersionStr: "v1.0.0",
			want:                false,
		},
		{
			name:                "Old version",
			pluginVersionStr:    "v1.0.0",
			installedVersionStr: "v2.0.0",
			want:                false,
		},
		{
			name:                "Invalid plugin version",
			pluginVersionStr:    "invalid",
			installedVersionStr: "v1.0.0",
			want:                false,
		},
		{
			name:                "Invalid installed version",
			pluginVersionStr:    "v1.0.0",
			installedVersionStr: "invalid",
			want:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNewVersion(tt.pluginVersionStr, tt.installedVersionStr); got != tt.want {
				t.Errorf("IsNewVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
