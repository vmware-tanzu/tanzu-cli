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
