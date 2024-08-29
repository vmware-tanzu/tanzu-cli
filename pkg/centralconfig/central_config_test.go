// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package centralconfig

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

func TestNewCentralConfigReader(t *testing.T) {
	// Verify that the central config reader points to the correct location
	discoveryName := "my_discovery"
	reader := newCentralConfigReader(discoveryName)

	path := reader.(*centralConfigYamlReader).configFile
	expectedPath := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, discoveryName, constants.CentralConfigFileName)

	assert.Equal(t, expectedPath, path)
}
