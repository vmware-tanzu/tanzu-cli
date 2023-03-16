// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package common defines generic constants and structs
package common

// IsValidScope is provided scope value is valid or not
func IsValidScope(scope string, caseSensitive bool) bool {
	return scope == PluginScopeStandalone || scope == PluginScopeContext
}
