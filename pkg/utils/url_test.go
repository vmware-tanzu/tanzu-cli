// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinUrl(t *testing.T) {
	testCases := []struct {
		baseURL     string
		relativeURL string
		expected    string
	}{
		{"https://www.example.com/", "/test/path/", "https://www.example.com/test/path"},
		{"https://www.example.com", "/test/path/", "https://www.example.com/test/path"},
		{"https://www.example.com", "test/path/", "https://www.example.com/test/path"},
		{"https://www.example.com/", "test/path/", "https://www.example.com/test/path"},
		{"https://www.example.com", "test/path", "https://www.example.com/test/path"},
		{"https://www.example.com/", "/test/path", "https://www.example.com/test/path"},
		{"https://www.example.com/", "test/path", "https://www.example.com/test/path"},
		{"https://www.example.com", "/test/path", "https://www.example.com/test/path"},
		{"https://www.example.com", "", "https://www.example.com"},
		{"https://www.example.com/", "", "https://www.example.com/"},
		{"https://www.example.com/", "/", "https://www.example.com/"},

		{"https://www.example.com:5001/", "/test/path/", "https://www.example.com:5001/test/path"},
		{"https://www.example.com:5001", "/test/path/", "https://www.example.com:5001/test/path"},
		{"https://www.example.com:5001", "test/path/", "https://www.example.com:5001/test/path"},
		{"https://www.example.com:5001/", "test/path/", "https://www.example.com:5001/test/path"},
		{"https://www.example.com:5001", "test/path", "https://www.example.com:5001/test/path"},
		{"https://www.example.com:5001/", "/test/path", "https://www.example.com:5001/test/path"},
		{"https://www.example.com:5001/", "test/path", "https://www.example.com:5001/test/path"},
		{"https://www.example.com:5001", "/test/path", "https://www.example.com:5001/test/path"},
		{"https://www.example.com:5001", "", "https://www.example.com:5001"},
		{"https://www.example.com:5001/", "", "https://www.example.com:5001/"},
		{"https://www.example.com:5001/", "/", "https://www.example.com:5001/"},

		{"www.example.com/", "/test/path/", "www.example.com/test/path"},
		{"www.example.com", "/test/path/", "www.example.com/test/path"},
		{"www.example.com", "test/path/", "www.example.com/test/path"},
		{"www.example.com/", "test/path/", "www.example.com/test/path"},
		{"www.example.com", "test/path", "www.example.com/test/path"},
		{"www.example.com/", "/test/path", "www.example.com/test/path"},
		{"www.example.com/", "test/path", "www.example.com/test/path"},
		{"www.example.com", "/test/path", "www.example.com/test/path"},
		{"www.example.com", "", "www.example.com"},
		{"www.example.com/", "", "www.example.com"},
		{"www.example.com/", "/", "www.example.com"},

		{"www.example.com:5001/", "/test/path/", "www.example.com:5001/test/path"},
		{"www.example.com:5001", "/test/path/", "www.example.com:5001/test/path"},
		{"www.example.com:5001", "test/path/", "www.example.com:5001/test/path"},
		{"www.example.com:5001/", "test/path/", "www.example.com:5001/test/path"},
		{"www.example.com:5001", "test/path", "www.example.com:5001/test/path"},
		{"www.example.com:5001/", "/test/path", "www.example.com:5001/test/path"},
		{"www.example.com:5001/", "test/path", "www.example.com:5001/test/path"},
		{"www.example.com:5001", "/test/path", "www.example.com:5001/test/path"},
		{"www.example.com:5001", "", "www.example.com:5001"},
		{"www.example.com:5001/", "", "www.example.com:5001"},
		{"www.example.com:5001/", "/", "www.example.com:5001"},

		{"example.com/", "/test/path/", "example.com/test/path"},
		{"example.com", "/test/path/", "example.com/test/path"},
		{"example.com", "test/path/", "example.com/test/path"},
		{"example.com/", "test/path/", "example.com/test/path"},
		{"example.com", "test/path", "example.com/test/path"},
		{"example.com/", "/test/path", "example.com/test/path"},
		{"example.com/", "test/path", "example.com/test/path"},
		{"example.com", "/test/path", "example.com/test/path"},
		{"example.com", "", "example.com"},
		{"example.com/", "", "example.com"},
		{"example.com/", "/", "example.com"},

		{"example.com:5001/", "/test/path/", "example.com:5001/test/path"},
		{"example.com:5001", "/test/path/", "example.com:5001/test/path"},
		{"example.com:5001", "test/path/", "example.com:5001/test/path"},
		{"example.com:5001/", "test/path/", "example.com:5001/test/path"},
		{"example.com:5001", "test/path", "example.com:5001/test/path"},
		{"example.com:5001/", "/test/path", "example.com:5001/test/path"},
		{"example.com:5001/", "test/path", "example.com:5001/test/path"},
		{"example.com:5001", "/test/path", "example.com:5001/test/path"},
		{"example.com:5001", "", "example.com:5001"},
		{"example.com:5001/", "", "example.com:5001"},
		{"example.com:5001/", "/", "example.com:5001"},

		{"localhost:5001/", "/test/path/", "localhost:5001/test/path"},
		{"localhost:5001", "/test/path/", "localhost:5001/test/path"},
		{"localhost:5001", "test/path/", "localhost:5001/test/path"},
		{"localhost:5001/", "test/path/", "localhost:5001/test/path"},
		{"localhost:5001", "test/path", "localhost:5001/test/path"},
		{"localhost:5001/", "/test/path", "localhost:5001/test/path"},
		{"localhost:5001/", "test/path", "localhost:5001/test/path"},
		{"localhost:5001", "/test/path", "localhost:5001/test/path"},
		{"localhost:5001", "", "localhost:5001"},
		{"localhost:5001/", "", "localhost:5001"},
		{"localhost:5001/", "/", "localhost:5001"},

		{"localhost:5001/tanzu-plugins/", "/test/path/", "localhost:5001/tanzu-plugins/test/path"},
		{"localhost:5001/tanzu-plugins", "/test/path/", "localhost:5001/tanzu-plugins/test/path"},
		{"localhost:5001/tanzu-plugins", "test/path/", "localhost:5001/tanzu-plugins/test/path"},
		{"localhost:5001/tanzu-plugins/", "test/path/", "localhost:5001/tanzu-plugins/test/path"},
		{"localhost:5001/tanzu-plugins", "test/path", "localhost:5001/tanzu-plugins/test/path"},
		{"localhost:5001/tanzu-plugins/", "/test/path", "localhost:5001/tanzu-plugins/test/path"},
		{"localhost:5001/tanzu-plugins/", "test/path", "localhost:5001/tanzu-plugins/test/path"},
		{"localhost:5001/tanzu-plugins", "/test/path", "localhost:5001/tanzu-plugins/test/path"},
		{"localhost:5001/tanzu-plugins", "", "localhost:5001/tanzu-plugins"},
		{"localhost:5001/tanzu-plugins/", "", "localhost:5001/tanzu-plugins"},
		{"localhost:5001/tanzu-plugins/", "/", "localhost:5001/tanzu-plugins"},
	}

	for _, tt := range testCases {
		testName := fmt.Sprintf("%v,%v", tt.baseURL, tt.relativeURL)
		t.Run(testName, func(t *testing.T) {
			ans := JoinURL(tt.baseURL, tt.relativeURL)
			assert.Equal(t, tt.expected, ans)
		})
	}
}
