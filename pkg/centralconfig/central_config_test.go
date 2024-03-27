// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package centralconfig

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

func TestNewCentralConfigReader(t *testing.T) {
	// Verify that the central config reader points to the correct location
	discoveryName := "my_discovery"
	reader := NewCentralConfigReader(&types.PluginDiscovery{
		OCI: &types.OCIDiscovery{
			Name:  discoveryName,
			Image: "image",
		},
	})

	path := reader.(*centralConfigYamlReader).configFile
	expectedPath := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, discoveryName, CentralConfigFileName)

	assert.Equal(t, expectedPath, path)
}
