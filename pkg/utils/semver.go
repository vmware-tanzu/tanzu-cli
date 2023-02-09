// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package utils contains utility functions
package utils

import (
	"sort"

	"github.com/Masterminds/semver"
)

// SortVersions sorts the supported version strings in semver 2.0 order.
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
