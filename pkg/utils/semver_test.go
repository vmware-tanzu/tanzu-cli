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
