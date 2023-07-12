// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"testing"
)

// TestContainsString tests the ContainsString function.
func TestContainsString(t *testing.T) {
	tests := []struct {
		name string
		arr  []string
		str  string
		want bool
	}{
		{
			name: "String present",
			arr:  []string{"apple", "banana", "cherry"},
			str:  "banana",
			want: true,
		},
		{
			name: "String absent",
			arr:  []string{"apple", "banana", "cherry"},
			str:  "grape",
			want: false,
		},
		{
			name: "Empty array",
			arr:  []string{},
			str:  "apple",
			want: false,
		},
		{
			name: "Empty string",
			arr:  []string{"apple", "banana", "cherry"},
			str:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsString(tt.arr, tt.str); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateKey tests the GenerateKey function.
func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "No parts",
			parts: []string{},
			want:  "",
		},
		{
			name:  "One part",
			parts: []string{"part1"},
			want:  "part1",
		},
		{
			name:  "Two parts",
			parts: []string{"part1", "part2"},
			want:  "part1:part2",
		},
		{
			name:  "Three parts",
			parts: []string{"part1", "part2", "part3"},
			want:  "part1:part2:part3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateKey(tt.parts...); got != tt.want {
				t.Errorf("GenerateKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
