// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"testing"

	"github.com/tj/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

func TestMergePluginManifest(t *testing.T) {
	assert := assert.New(t)

	baseManifest := cli.Manifest{
		Plugins: []cli.Plugin{
			cli.Plugin{
				Name:        "plugin1",
				Target:      "target-foo",
				Description: "desc-1-foo",
				Versions:    []string{"v1.1.1"},
			},
			cli.Plugin{
				Name:        "plugin2",
				Target:      "target-foo",
				Description: "desc-2-foo",
				Versions:    []string{"v4.0.0"},
			},
			cli.Plugin{
				Name:        "plugin3",
				Target:      "target-baz",
				Description: "desc-3",
				Versions:    []string{"v3.0.0"},
			},
		},
	}

	incomingManifest := cli.Manifest{
		Plugins: []cli.Plugin{
			cli.Plugin{
				Name:        "plugin1",
				Target:      "target-foo",
				Description: "desc-1-foo",
				Versions:    []string{"v1.0.0"},
			},
			cli.Plugin{
				Name:        "plugin2",
				Target:      "target-bar",
				Description: "desc-2",
				Versions:    []string{"v2.0.0"},
			},
		},
	}

	expectedMergedManifest := cli.Manifest{
		Plugins: []cli.Plugin{
			cli.Plugin{
				Name:        "plugin1",
				Target:      "target-foo",
				Description: "desc-1-foo",
				Versions:    []string{"v1.1.1"},
			},
			cli.Plugin{
				Name:        "plugin3",
				Target:      "target-baz",
				Description: "desc-3",
				Versions:    []string{"v3.0.0"},
			},
			cli.Plugin{
				Name:        "plugin2",
				Target:      "target-foo",
				Description: "desc-2-foo",
				Versions:    []string{"v4.0.0"},
			},
			cli.Plugin{
				Name:        "plugin2",
				Target:      "target-bar",
				Description: "desc-2",
				Versions:    []string{"v2.0.0"},
			},
		},
	}

	mergedManifest := mergePluginManifest(baseManifest, incomingManifest)

	for _, plugin := range expectedMergedManifest.Plugins {
		foundPlugin := findpluginInManifest(mergedManifest, plugin)
		assert.NotNil(foundPlugin, "expected plugin:%v, target:%v not found", plugin.Name, plugin.Target)
		assert.Equal(foundPlugin.Name, plugin.Name)
		assert.Equal(foundPlugin.Target, plugin.Target)
		assert.Equal(foundPlugin.Description, plugin.Description)
		assert.Equal(foundPlugin.Versions, plugin.Versions)
	}
}

func TestMergePluginGroupManifest(t *testing.T) {
	assert := assert.New(t)

	basePGManifest := cli.PluginGroupManifest{
		Plugins: []cli.PluginNameTargetScopeVersion{
			cli.PluginNameTargetScopeVersion{
				PluginNameTargetScope: cli.PluginNameTargetScope{
					Name:            "plugin1",
					Target:          "target-foo",
					IsContextScoped: true,
				},
				Version: "v1.1.1",
			},
			cli.PluginNameTargetScopeVersion{
				PluginNameTargetScope: cli.PluginNameTargetScope{
					Name:            "plugin2",
					Target:          "target-foo",
					IsContextScoped: true,
				},
				Version: "v4.0.0",
			},
			cli.PluginNameTargetScopeVersion{
				PluginNameTargetScope: cli.PluginNameTargetScope{
					Name:            "plugin3",
					Target:          "target-baz",
					IsContextScoped: false,
				},
				Version: "v3.0.0",
			},
		},
	}

	incomingPGManifest := cli.PluginGroupManifest{
		Plugins: []cli.PluginNameTargetScopeVersion{
			cli.PluginNameTargetScopeVersion{
				PluginNameTargetScope: cli.PluginNameTargetScope{
					Name:            "plugin1",
					Target:          "target-foo",
					IsContextScoped: false,
				},
				Version: "v1.0.0",
			},
			cli.PluginNameTargetScopeVersion{
				PluginNameTargetScope: cli.PluginNameTargetScope{
					Name:            "plugin2",
					Target:          "target-bar",
					IsContextScoped: false,
				},
				Version: "v2.0.0",
			},
		},
	}

	expectedMergedPGManifest := cli.PluginGroupManifest{
		Plugins: []cli.PluginNameTargetScopeVersion{
			cli.PluginNameTargetScopeVersion{
				PluginNameTargetScope: cli.PluginNameTargetScope{
					Name:            "plugin1",
					Target:          "target-foo",
					IsContextScoped: true,
				},
				Version: "v1.1.1",
			},
			cli.PluginNameTargetScopeVersion{
				PluginNameTargetScope: cli.PluginNameTargetScope{
					Name:            "plugin3",
					Target:          "target-baz",
					IsContextScoped: false,
				},
				Version: "v3.0.0",
			},
			cli.PluginNameTargetScopeVersion{
				PluginNameTargetScope: cli.PluginNameTargetScope{
					Name:            "plugin2",
					Target:          "target-foo",
					IsContextScoped: true,
				},
				Version: "v4.0.0",
			},
			cli.PluginNameTargetScopeVersion{
				PluginNameTargetScope: cli.PluginNameTargetScope{
					Name:            "plugin2",
					Target:          "target-bar",
					IsContextScoped: false,
				},
				Version: "v2.0.0",
			},
		},
	}

	mergedManifest := mergePluginGroupManifest(basePGManifest, incomingPGManifest)

	for _, plugin := range expectedMergedPGManifest.Plugins {
		foundPlugin := findpluginInPluginGroupManifest(mergedManifest, plugin)
		assert.NotNil(foundPlugin, "expected plugin:%v, target:%v not found", plugin.Name, plugin.Target)
		assert.Equal(foundPlugin.Name, plugin.Name)
		assert.Equal(foundPlugin.Target, plugin.Target)
		assert.Equal(foundPlugin.IsContextScoped, plugin.IsContextScoped)
		assert.Equal(foundPlugin.Version, plugin.Version)
	}
}
