// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package utils contains utility functions
package utils

import (
	"strings"
)

// ContainsString checks the string contains in string array
func ContainsString(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// GenerateKey is a utility function that takes an arbitrary number of string arguments,
// concatenates them with a colon (":") separator, and returns the resulting string.
// This function is typically used to create a unique key or identifier from multiple string parts.
//
// Parameters:
// parts: A variadic parameter that accepts an arbitrary number of strings.
//
// Returns:
// A single string that is the result of concatenating all input strings with a colon (":") separator.
func GenerateKey(parts ...string) string {
	return strings.Join(parts, ":")
}
