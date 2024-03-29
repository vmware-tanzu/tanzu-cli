// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"testing"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
)

// TestIsAllPluginsFromGroupInstalled tests the isAllPluginsFromGroupInstalled function.
func TestIsAllPluginsFromGroupInstalled(t *testing.T) {
	tests := []struct {
		name             string
		plugins          []*plugininventory.PluginGroupPluginEntry
		installedPlugins []cli.PluginInfo
		want             bool
	}{
		{
			name: "All plugins installed",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "1.0.0",
					},
					Mandatory: true,
				},
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin2",
						Target:  "target2",
						Version: "1.0.0",
					},
					Mandatory: true,
				},
			},
			installedPlugins: []cli.PluginInfo{
				{
					Name:    "plugin1",
					Target:  "target1",
					Version: "1.0.0",
				},
				{
					Name:    "plugin2",
					Target:  "target2",
					Version: "1.0.0",
				},
			},
			want: true,
		},
		{
			name: "Some plugins not installed",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "1.0.0",
					},
					Mandatory: true,
				},
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin2",
						Target:  "target2",
						Version: "1.0.0",
					},
					Mandatory: true,
				},
			},
			installedPlugins: []cli.PluginInfo{
				{
					Name:    "plugin1",
					Target:  "target1",
					Version: "1.0.0",
				},
			},
			want: false,
		},

		{
			name: "No plugins installed",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "1.0.0",
					},
					Mandatory: true,
				},
			},
			installedPlugins: []cli.PluginInfo{},
			want:             false,
		},
		{
			name: "Installed plugins list is empty",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "1.0.0",
					},
					Mandatory: true,
				},
			},
			installedPlugins: nil,
			want:             false,
		},
		{
			name: "Mandatory plugin not installed",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "1.0.0",
					},
					Mandatory: true,
				},
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin2",
						Target:  "target2",
						Version: "1.0.0",
					},
					Mandatory: false,
				},
			},
			installedPlugins: []cli.PluginInfo{
				{
					Name:    "plugin2",
					Target:  "target2",
					Version: "1.0.0",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAllPluginsFromGroupInstalled(tt.plugins, tt.installedPlugins); got != tt.want {
				t.Errorf("isAllPluginsFromGroupInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsNewPluginVersionAvailable tests the isNewPluginVersionAvailable function.
func TestIsNewPluginVersionAvailable(t *testing.T) {
	tests := []struct {
		name             string
		plugins          []*plugininventory.PluginGroupPluginEntry
		installedPlugins []cli.PluginInfo
		want             bool
	}{
		{
			name: "New version available",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "2.0.0",
					},
				},
			},
			installedPlugins: []cli.PluginInfo{
				{
					Name:    "plugin1",
					Target:  "target1",
					Version: "1.0.0",
				},
			},
			want: true,
		},
		{
			name: "Same version",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "1.0.0",
					},
				},
			},
			installedPlugins: []cli.PluginInfo{
				{
					Name:    "plugin1",
					Target:  "target1",
					Version: "1.0.0",
				},
			},
			want: false,
		},
		{
			name: "Old version",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "1.0.0",
					},
				},
			},
			installedPlugins: []cli.PluginInfo{
				{
					Name:    "plugin1",
					Target:  "target1",
					Version: "2.0.0",
				},
			},
			want: false,
		},
		{
			name: "Plugin not installed",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "1.0.0",
					},
				},
			},
			installedPlugins: []cli.PluginInfo{},
			want:             false,
		},

		{
			name: "New version available",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "v2.0.0",
					},
				},
			},
			installedPlugins: []cli.PluginInfo{
				{
					Name:    "plugin1",
					Target:  "target1",
					Version: "v1.0.0",
				},
			},
			want: true,
		},
		{
			name: "Same version",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "v1.0.0",
					},
				},
			},
			installedPlugins: []cli.PluginInfo{
				{
					Name:    "plugin1",
					Target:  "target1",
					Version: "v1.0.0",
				},
			},
			want: false,
		},
		{
			name: "Old version",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "v1.0.0",
					},
				},
			},
			installedPlugins: []cli.PluginInfo{
				{
					Name:    "plugin1",
					Target:  "target1",
					Version: "v2.0.0",
				},
			},
			want: false,
		},
		{
			name: "Plugin not installed",
			plugins: []*plugininventory.PluginGroupPluginEntry{
				{
					PluginIdentifier: plugininventory.PluginIdentifier{
						Name:    "plugin1",
						Target:  "target1",
						Version: "v1.0.0",
					},
				},
			},
			installedPlugins: []cli.PluginInfo{},
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewPluginVersionAvailable(tt.plugins, tt.installedPlugins); got != tt.want {
				t.Errorf("isNewPluginVersionAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
