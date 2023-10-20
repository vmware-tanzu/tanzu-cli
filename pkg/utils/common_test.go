// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
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

func setupMultiConcurrentContexts(t *testing.T) func() {
	configFile, err := os.CreateTemp("", "config")
	assert.NoError(t, err)
	err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), configFile.Name())
	assert.NoError(t, err, "Error while copying tanzu config file for testing")
	os.Setenv("TANZU_CONFIG", configFile.Name())

	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.NoError(t, err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
	err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng_2.yaml"), configFileNG.Name())
	assert.NoError(t, err, "Error while copying tanzu-ng config file for testing")

	cleanup := func() {
		err = os.Remove(configFile.Name())
		assert.NoError(t, err)
		err = os.Unsetenv("TANZU_CONFIG")
		assert.NoError(t, err)

		err = os.Remove(configFileNG.Name())
		assert.NoError(t, err)
		err = os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		assert.NoError(t, err)
	}
	return cleanup
}

// TestEnsureMutualExclusiveCurrentContexts tests the EnsureMutualExclusiveCurrentContexts function.
func TestEnsureMutualExclusiveCurrentContexts(t *testing.T) {
	cleanup := setupMultiConcurrentContexts(t)

	defer cleanup()

	// it should remove the tanzu current context and keep k8s and tmc current context
	err := EnsureMutualExclusiveCurrentContexts()
	assert.NoError(t, err)

	ccmap, err := config.GetAllActiveContextsMap()
	assert.NoError(t, err)
	assert.Equal(t, ccmap[configtypes.ContextTypeK8s].Name, "test-mc-context")
	assert.Equal(t, ccmap[configtypes.ContextTypeTMC].Name, "test-tmc-context")
	assert.Nil(t, ccmap[configtypes.ContextTypeTanzu])

	// if there is only k8s current context, calling again should not affect the current contexts
	err = EnsureMutualExclusiveCurrentContexts()
	assert.NoError(t, err)

	ccmap, err = config.GetAllActiveContextsMap()
	assert.NoError(t, err)
	assert.Equal(t, ccmap[configtypes.ContextTypeK8s].Name, "test-mc-context")
	assert.Equal(t, ccmap[configtypes.ContextTypeTMC].Name, "test-tmc-context")
	assert.Nil(t, ccmap[configtypes.ContextTypeTanzu])

	// if there is only tanzu current context, calling again should not affect the current contexts
	err = config.SetActiveContext("test-tanzu-context")
	assert.NoError(t, err)

	err = EnsureMutualExclusiveCurrentContexts()
	assert.NoError(t, err)

	ccmap, err = config.GetAllActiveContextsMap()
	assert.NoError(t, err)
	assert.Nil(t, ccmap[configtypes.ContextTypeK8s])
	assert.Equal(t, ccmap[configtypes.ContextTypeTMC].Name, "test-tmc-context")
	assert.Equal(t, ccmap[configtypes.ContextTypeTanzu].Name, "test-tanzu-context")

	// if there are no current context, calling again should not affect the current contexts
	err = config.RemoveActiveContext(configtypes.ContextTypeTanzu)
	assert.NoError(t, err)
	err = config.RemoveActiveContext(configtypes.ContextTypeTMC)
	assert.NoError(t, err)

	err = EnsureMutualExclusiveCurrentContexts()
	assert.NoError(t, err)

	ccmap, err = config.GetAllActiveContextsMap()
	assert.NoError(t, err)
	assert.Equal(t, len(ccmap), 0)
}
