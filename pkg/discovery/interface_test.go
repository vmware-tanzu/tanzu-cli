// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
)

func Test_CreateDiscoveryFromV1alpha1(t *testing.T) {
	assert := assert.New(t)

	// When no discovery type is provided, it should throw error
	pd := configtypes.PluginDiscovery{}
	_, err := CreateDiscoveryFromV1alpha1(pd)
	assert.NotNil(err)
	assert.Contains(err.Error(), "unknown plugin discovery source")

	// When OCI discovery is provided
	pd = configtypes.PluginDiscovery{
		OCI: &configtypes.OCIDiscovery{Name: "fake-oci", Image: "fake.repo.com/test:v1.0.0"},
	}
	discovery, err := CreateDiscoveryFromV1alpha1(pd)
	assert.Nil(err)
	assert.Equal(common.DiscoveryTypeOCI, discovery.Type())
	assert.Equal("fake-oci", discovery.Name())

	// When Local discovery is provided
	pd = configtypes.PluginDiscovery{
		Local: &configtypes.LocalDiscovery{Name: "fake-local", Path: "test/path"},
	}
	discovery, err = CreateDiscoveryFromV1alpha1(pd)
	assert.Nil(err)
	assert.Equal(common.DiscoveryTypeLocal, discovery.Type())
	assert.Equal("fake-local", discovery.Name())

	// When K8s discovery is provided
	pd = configtypes.PluginDiscovery{
		Kubernetes: &configtypes.KubernetesDiscovery{Name: "fake-k8s"},
	}
	discovery, err = CreateDiscoveryFromV1alpha1(pd)
	assert.Nil(err)
	assert.Equal(common.DiscoveryTypeKubernetes, discovery.Type())
	assert.Equal("fake-k8s", discovery.Name())

	// When REST discovery is provided
	pd = configtypes.PluginDiscovery{
		REST: &configtypes.GenericRESTDiscovery{Name: "fake-rest"},
	}
	discovery, err = CreateDiscoveryFromV1alpha1(pd)
	assert.Nil(err)
	assert.Equal(common.DiscoveryTypeREST, discovery.Type())
	assert.Equal("fake-rest", discovery.Name())
}

func Test_CreateGroupDiscovery(t *testing.T) {
	assert := assert.New(t)

	// When no discovery type is provided, it should throw error
	pd := configtypes.PluginDiscovery{}
	_, err := CreateGroupDiscovery(pd)
	assert.NotNil(err)
	assert.Contains(err.Error(), "unknown group discovery source")

	// When OCI discovery is provided
	pd = configtypes.PluginDiscovery{
		OCI: &configtypes.OCIDiscovery{Name: "fake-oci", Image: "fake.repo.com/test:v1.0.0"},
	}
	discovery, err := CreateGroupDiscovery(pd)
	assert.Nil(err)
	assert.Equal("fake-oci", discovery.Name())

	// When OCI discovery is provided with criteria
	pd = configtypes.PluginDiscovery{
		OCI: &configtypes.OCIDiscovery{Name: "fake-oci", Image: "fake.repo.com/test:v1.0.0"},
	}
	criteria := &GroupDiscoveryCriteria{
		Vendor:    "vmware",
		Publisher: "tkg",
		Name:      "fakegroup",
	}
	discovery, err = CreateGroupDiscovery(pd, WithGroupDiscoveryCriteria(criteria))
	assert.Nil(err)
	assert.Equal("fake-oci", discovery.Name())

	groupDisc, ok := discovery.(*DBBackedOCIDiscovery)
	assert.True(ok)
	assert.Equal(criteria, groupDisc.groupCriteria)
	assert.Nil(groupDisc.pluginCriteria)

	// When Local discovery is provided
	pd = configtypes.PluginDiscovery{
		Local: &configtypes.LocalDiscovery{Name: "fake-local", Path: "test/path"},
	}
	_, err = CreateGroupDiscovery(pd, nil)
	assert.NotNil(err)
	assert.Equal(err.Error(), "unknown group discovery source")

	// When K8s discovery is provided
	pd = configtypes.PluginDiscovery{
		Kubernetes: &configtypes.KubernetesDiscovery{Name: "fake-k8s"},
	}
	_, err = CreateGroupDiscovery(pd, nil)
	assert.NotNil(err)
	assert.Equal(err.Error(), "unknown group discovery source")

	// When REST discovery is provided
	pd = configtypes.PluginDiscovery{
		REST: &configtypes.GenericRESTDiscovery{Name: "fake-rest"},
	}
	_, err = CreateGroupDiscovery(pd, nil)
	assert.NotNil(err)
	assert.Equal(err.Error(), "unknown group discovery source")
}
