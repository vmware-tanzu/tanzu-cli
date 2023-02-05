// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
)

func Test_CheckDiscoveryName(t *testing.T) {
	assert := assert.New(t)

	gcpDiscovery := configtypes.PluginDiscovery{GCP: &configtypes.GCPDiscovery{Name: "gcp-test"}}
	result := CheckDiscoveryName(gcpDiscovery, "gcp-test")
	assert.True(result)
	result = CheckDiscoveryName(gcpDiscovery, "test")
	assert.False(result)

	ociDiscovery := configtypes.PluginDiscovery{OCI: &configtypes.OCIDiscovery{Name: "oci-test"}}
	result = CheckDiscoveryName(ociDiscovery, "oci-test")
	assert.True(result)
	result = CheckDiscoveryName(ociDiscovery, "test")
	assert.False(result)

	k8sDiscovery := configtypes.PluginDiscovery{Kubernetes: &configtypes.KubernetesDiscovery{Name: "k8s-test"}}
	result = CheckDiscoveryName(k8sDiscovery, "k8s-test")
	assert.True(result)
	result = CheckDiscoveryName(k8sDiscovery, "test")
	assert.False(result)

	localDiscovery := configtypes.PluginDiscovery{Local: &configtypes.LocalDiscovery{Name: "local-test"}}
	result = CheckDiscoveryName(localDiscovery, "local-test")
	assert.True(result)
	result = CheckDiscoveryName(localDiscovery, "test")
	assert.False(result)

	restDiscovery := configtypes.PluginDiscovery{REST: &configtypes.GenericRESTDiscovery{Name: "rest-test"}}
	result = CheckDiscoveryName(restDiscovery, "rest-test")
	assert.True(result)
	result = CheckDiscoveryName(restDiscovery, "test")
	assert.False(result)
}

func Test_CompareDiscoverySource(t *testing.T) {
	assert := assert.New(t)

	ds1 := configtypes.PluginDiscovery{GCP: &configtypes.GCPDiscovery{Name: "gcp-test", Bucket: "bucket1", ManifestPath: "manifest1"}}
	ds2 := configtypes.PluginDiscovery{GCP: &configtypes.GCPDiscovery{Name: "gcp-test", Bucket: "bucket1", ManifestPath: "manifest1"}}
	result := CompareDiscoverySource(ds1, ds2, common.DiscoveryTypeGCP)
	assert.True(result)
	ds2 = configtypes.PluginDiscovery{GCP: &configtypes.GCPDiscovery{Name: "gcp-test", Bucket: "bucket2", ManifestPath: "manifest1"}}
	result = CompareDiscoverySource(ds1, ds2, common.DiscoveryTypeGCP)
	assert.False(result)

	ds1 = configtypes.PluginDiscovery{Local: &configtypes.LocalDiscovery{Name: "gcp-test", Path: "path1"}}
	ds2 = configtypes.PluginDiscovery{Local: &configtypes.LocalDiscovery{Name: "gcp-test", Path: "path1"}}
	result = CompareDiscoverySource(ds1, ds2, common.DiscoveryTypeLocal)
	assert.True(result)
	ds2 = configtypes.PluginDiscovery{Local: &configtypes.LocalDiscovery{Name: "gcp-test", Path: "path2"}}
	result = CompareDiscoverySource(ds1, ds2, common.DiscoveryTypeLocal)
	assert.False(result)

	ds1 = configtypes.PluginDiscovery{OCI: &configtypes.OCIDiscovery{Name: "gcp-test", Image: "image1"}}
	ds2 = configtypes.PluginDiscovery{OCI: &configtypes.OCIDiscovery{Name: "gcp-test", Image: "image1"}}
	result = CompareDiscoverySource(ds1, ds2, common.DiscoveryTypeOCI)
	assert.True(result)
	ds2 = configtypes.PluginDiscovery{OCI: &configtypes.OCIDiscovery{Name: "gcp-test", Image: "image2"}}
	result = CompareDiscoverySource(ds1, ds2, common.DiscoveryTypeOCI)
	assert.False(result)

	ds1 = configtypes.PluginDiscovery{OCI: &configtypes.OCIDiscovery{Name: "gcp-test", Image: "image1"}}
	ds2 = configtypes.PluginDiscovery{Local: &configtypes.LocalDiscovery{Name: "gcp-test", Path: "path1"}}
	result = CompareDiscoverySource(ds1, ds2, common.DiscoveryTypeOCI)
	assert.False(result)
	result = CompareDiscoverySource(ds1, ds2, common.DiscoveryTypeLocal)
	assert.False(result)
	result = CompareDiscoverySource(ds1, ds2, common.DiscoveryTypeREST)
	assert.False(result)
}
