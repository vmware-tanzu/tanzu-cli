// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package helpers

import "runtime"

var minConcurrent = 2

// ErrInfo is used to return error information for
type ErrInfo struct {
	Err  error
	Path string
	ID   string
}

// GetMaxParallelism return the maximum concurrent threads to use.
// Limit the number of concurrent operations we perform so we don't overwhelm the system.
func GetMaxParallelism() int {
	maxConcurrent := runtime.NumCPU() - 2
	if maxConcurrent < minConcurrent {
		maxConcurrent = minConcurrent
	}
	return maxConcurrent
}

// Identifiers are the emoji symbols to specify progress happening on different threads
var Identifiers = []string{
	string('\U0001F435'),
	string('\U0001F43C'),
	string('\U0001F436'),
	string('\U0001F430'),
	string('\U0001F98A'),
	string('\U0001F431'),
	string('\U0001F981'),
	string('\U0001F42F'),
	string('\U0001F42E'),
	string('\U0001F437'),
	string('\U0001F42D'),
	string('\U0001F428'),
}

// GetID return a unique ID based on the identifiers
func GetID(i int) string {
	index := i
	if i >= len(Identifiers) {
		// Well aren't you lucky
		index = i % len(Identifiers)
	}
	return Identifiers[index]
}
