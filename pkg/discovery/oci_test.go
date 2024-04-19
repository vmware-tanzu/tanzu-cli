// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
)

func Test_NewOCIDiscovery(t *testing.T) {
	assert := assert.New(t)

	configFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG", configFile.Name())

	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

	defer func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())
	}()

	discoveryName := "test-discovery"
	discoveryImage := "test-image:latest"
	criteriaName := "test-criteria"
	discoveryCriteria := &PluginDiscoveryCriteria{Name: criteriaName}

	// Check that the correct discovery type is returned
	pd := NewOCIDiscovery(discoveryName, discoveryImage, WithPluginDiscoveryCriteria(discoveryCriteria))
	assert.NotNil(pd)
	assert.Equal(discoveryName, pd.Name())
	assert.Equal(common.DiscoveryTypeOCI, pd.Type())

	dbDiscovery, ok := pd.(*DBBackedOCIDiscovery)
	assert.True(ok)
	assert.Equal(discoveryName, dbDiscovery.name)
	assert.Equal(discoveryImage, dbDiscovery.image)
	assert.Equal(discoveryCriteria, dbDiscovery.pluginCriteria)
	assert.Nil(dbDiscovery.groupCriteria)
}

func Test_NewOCIGroupDiscovery(t *testing.T) {
	assert := assert.New(t)

	discoveryName := "test-discovery2"
	discoveryImage := "test-image2:latest"
	criteriaName := "test-criteria2"
	groupCriteria := &GroupDiscoveryCriteria{Name: criteriaName}

	// Check that the correct discovery is returned
	pd := NewOCIGroupDiscovery(discoveryName, discoveryImage, WithGroupDiscoveryCriteria(groupCriteria))
	assert.NotNil(pd)
	assert.Equal(discoveryName, pd.Name())

	dbDiscovery, ok := pd.(*DBBackedOCIDiscovery)
	assert.True(ok)
	assert.Equal(discoveryName, dbDiscovery.name)
	assert.Equal(discoveryImage, dbDiscovery.image)
	assert.Equal(groupCriteria, dbDiscovery.groupCriteria)
	assert.Nil(dbDiscovery.pluginCriteria)
}
