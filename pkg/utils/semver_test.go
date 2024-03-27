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

func TestIsPreRelease(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "Major version",
			version: "v3.0.0",
			want:    false,
		},
		{
			name:    "Minor version",
			version: "v2.2.0",
			want:    false,
		},
		{
			name:    "Patch version",
			version: "v1.1.1",
			want:    false,
		},
		{
			name:    "Alpha 0 version",
			version: "v1.3.0-alpha.0",
			want:    true,
		},
		{
			name:    "Alpha 1 version",
			version: "v1.3.5-alpha.1",
			want:    true,
		},
		{
			name:    "Beta version",
			version: "v1.3.0-beta.0",
			want:    true,
		},
		{
			name:    "RC version",
			version: "v1.3.0-rc.0",
			want:    true,
		},
		{
			name:    "RC caps version",
			version: "v1.3.0-RC.0",
			want:    true,
		},
		{
			name:    "Invalid version",
			version: "invalid-version",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPreRelease(tt.version); got != tt.want {
				t.Errorf("IsPreRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSameMajor(t *testing.T) {
	tests := []struct {
		name     string
		version1 string
		version2 string
		want     bool
	}{
		{
			name:     "Same version",
			version1: "v1.0.0",
			version2: "v1.0.0",
			want:     true,
		},
		{
			name:     "Same minor",
			version1: "v1.1.1",
			version2: "v1.1.0",
			want:     true,
		},
		{
			name:     "Same major",
			version1: "v1.1.1",
			version2: "v1.0.0",
			want:     true,
		},
		{
			name:     "Different major",
			version1: "v2.0.0",
			version2: "v1.0.0",
			want:     false,
		},
		{
			name:     "Different major with minor and patch",
			version1: "v2.2.2",
			version2: "v1.0.0",
			want:     false,
		},
		{
			name:     "Different major with 0",
			version1: "v0.90.0",
			version2: "v1.0.0",
			want:     false,
		},
		{
			name:     "Same major with 0",
			version1: "v0.90.1",
			version2: "v0.90.0",
			want:     true,
		},
		{
			name:     "Invalid version",
			version1: "invalid",
			version2: "v1.0.0",
			want:     false,
		},
		{
			name:     "Invalid version reversed",
			version1: "v1.0.0",
			version2: "invalid",
			want:     false,
		},
		{
			name:     "Same major with pre-release",
			version1: "v1.3.0-alpha.1",
			version2: "v1.3.0-beta.1",
			want:     true,
		},
		{
			name:     "Same major with one as pre-release",
			version1: "v1.3.0-alpha.1",
			version2: "v1.4.1",
			want:     true,
		},
		{
			name:     "Different major with same pre-release",
			version1: "v2.3.0-alpha.1",
			version2: "v1.3.0-alpha.1",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSameMajor(tt.version1, tt.version2); got != tt.want {
				t.Errorf("IsSameMajor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSameMinor(t *testing.T) {
	tests := []struct {
		name     string
		version1 string
		version2 string
		want     bool
	}{
		{
			name:     "Same version",
			version1: "v1.0.0",
			version2: "v1.0.0",
			want:     true,
		},
		{
			name:     "Same minor",
			version1: "v1.1.1",
			version2: "v1.1.0",
			want:     true,
		},
		{
			name:     "Same major",
			version1: "v1.1.1",
			version2: "v1.0.0",
			want:     false,
		},
		{
			name:     "Different minor",
			version1: "v1.1.0",
			version2: "v1.0.0",
			want:     false,
		},
		{
			name:     "Same minor with different major",
			version1: "v2.2.2",
			version2: "v1.2.2",
			want:     false,
		},
		{
			name:     "Different minor with 0",
			version1: "v0.90.0",
			version2: "v0.91.0",
			want:     false,
		},
		{
			name:     "Same major with 0",
			version1: "v1.90.0",
			version2: "v0.90.0",
			want:     false,
		},
		{
			name:     "Invalid version",
			version1: "invalid",
			version2: "v1.0.0",
			want:     false,
		},
		{
			name:     "Invalid version reversed",
			version1: "v1.0.0",
			version2: "invalid",
			want:     false,
		},
		{
			name:     "Same minor with pre-release",
			version1: "v1.3.0-alpha.1",
			version2: "v1.3.0-beta.1",
			want:     true,
		},
		{
			name:     "Same minor with one as pre-release",
			version1: "v1.3.0-alpha.1",
			version2: "v1.3.1",
			want:     true,
		},
		{
			name:     "Different minor with same pre-release",
			version1: "v1.4.0-alpha.1",
			version2: "v1.3.0-alpha.1",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSameMinor(tt.version1, tt.version2); got != tt.want {
				t.Errorf("IsSameMinor() = %v, want %v", got, tt.want)
			}
		})
	}
}
