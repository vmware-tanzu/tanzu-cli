// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines helper structure definitions
package types

// Metadata specifies plugin metadata
type Metadata struct {
	Name   string `json:"name" yaml:"name"`
	Target string `json:"target" yaml:"target"`
}
