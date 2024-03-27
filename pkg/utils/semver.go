// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package utils contains utility functions
package utils

import (
	"sort"

	"github.com/Masterminds/semver"
)

// SortVersions sorts the supported version strings in ascending semver 2.0 order.
func SortVersions(vStrArr []string) error {
	vArr := make([]*semver.Version, len(vStrArr))
	for i, vStr := range vStrArr {
		v, err := semver.NewVersion(vStr)
		if err != nil {
			return err
		}
		vArr[i] = v
	}
	sort.Sort(semver.Collection(vArr))
	for i, v := range vArr {
		vStrArr[i] = v.Original()
	}
	return nil
}

// IsNewVersion compares the plugin version and the installed version.
func IsNewVersion(incomingVersionStr, existingVersionStr string) bool {
	// Parse versions using semver package
	incomingVersion, err := semver.NewVersion(incomingVersionStr)
	if err != nil {
		return false // Invalid plugin version, return false
	}

	existingVersion, err := semver.NewVersion(existingVersionStr)
	if err != nil {
		return false // Invalid installed version, return false
	}

	// Compare versions
	return incomingVersion.Compare(existingVersion) > 0 // Return true if new version is available
}

// IsPreRelease checks if the version is a pre-release version.
func IsPreRelease(versionStr string) bool {
	version, err := semver.NewVersion(versionStr)
	if err == nil && version.Prerelease() != "" {
		return true
	}
	return false
}

// IsSameMajor compares two versions and returns true if they are the same major version.
func IsSameMajor(v1str, v2str string) bool {
	v1, err := semver.NewVersion(v1str)
	if err != nil {
		return false
	}
	v2, err := semver.NewVersion(v2str)
	if err != nil {
		return false
	}
	return v1.Major() == v2.Major()
}

// IsSameMinor compares two versions and returns true if they are the same minor version.
func IsSameMinor(v1str, v2str string) bool {
	v1, err := semver.NewVersion(v1str)
	if err != nil {
		return false
	}
	v2, err := semver.NewVersion(v2str)
	if err != nil {
		return false
	}
	return v1.Major() == v2.Major() && v1.Minor() == v2.Minor()
}
