// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugininventory

import (
	"reflect"
	"testing"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

func TestRemoveDuplicatePluginInventoryEntries(t *testing.T) {
	entry1 := &PluginInventoryEntry{Name: "Plugin1", Target: configtypes.TargetK8s, RecommendedVersion: "1.0"}
	entry2 := &PluginInventoryEntry{Name: "Plugin2", Target: configtypes.TargetK8s, RecommendedVersion: "2.0"}
	entry3 := &PluginInventoryEntry{Name: "Plugin1", Target: configtypes.TargetK8s, RecommendedVersion: "1.0"} // Duplicate
	entry4 := &PluginInventoryEntry{Name: "Plugin3", Target: configtypes.TargetK8s, RecommendedVersion: "3.0"}

	tests := []struct {
		name   string
		input  []*PluginInventoryEntry
		output []*PluginInventoryEntry
	}{
		{
			name:   "NoDuplicates",
			input:  []*PluginInventoryEntry{entry1, entry2, entry4},
			output: []*PluginInventoryEntry{entry1, entry2, entry4},
		},
		{
			name:   "WithDuplicates",
			input:  []*PluginInventoryEntry{entry1, entry2, entry3, entry4},
			output: []*PluginInventoryEntry{entry1, entry2, entry4},
		},
		{
			name:   "EmptyInput",
			input:  []*PluginInventoryEntry{},
			output: []*PluginInventoryEntry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveDuplicatePluginInventoryEntries(tt.input)
			if !reflect.DeepEqual(got, tt.output) {
				t.Errorf("TestCase: %v, RemoveDuplicatePluginInventoryEntries() = %v, want %v", tt.name, got, tt.output)
			}
		})
	}
}

func TestRemoveDuplicatePluginGroups(t *testing.T) {
	entry1 := &PluginGroup{Vendor: "vmware", Publisher: "tkg", Name: "Plugin1", RecommendedVersion: "v1.0"}
	entry2 := &PluginGroup{Vendor: "vmware", Publisher: "tkg", Name: "Plugin2", RecommendedVersion: "v2.0"}
	entry3 := &PluginGroup{Vendor: "vmware", Publisher: "tkg", Name: "Plugin1", RecommendedVersion: "v1.0"} // Duplicate
	entry4 := &PluginGroup{Vendor: "vmware", Publisher: "tkg", Name: "Plugin1", RecommendedVersion: "v1.1"}

	tests := []struct {
		name   string
		input  []*PluginGroup
		output []*PluginGroup
	}{
		{
			name:   "NoDuplicates",
			input:  []*PluginGroup{entry1, entry2, entry4},
			output: []*PluginGroup{entry1, entry2, entry4},
		},
		{
			name:   "WithDuplicates",
			input:  []*PluginGroup{entry1, entry2, entry3, entry4},
			output: []*PluginGroup{entry1, entry2, entry4},
		},
		{
			name:   "EmptyInput",
			input:  []*PluginGroup{},
			output: []*PluginGroup{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveDuplicatePluginGroups(tt.input)
			if !reflect.DeepEqual(got, tt.output) {
				t.Errorf("TestCase: %v, RemoveDuplicatePluginGroups() = %v, want %v", tt.name, got, tt.output)
			}
		})
	}
}
