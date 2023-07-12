// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

var testData1 = `---
apiVersion: cli.tanzu.vmware.com/v1alpha1
kind: CLIPlugin
metadata:
 name: foo
spec:
 artifacts:
   v0.0.1:
   - arch: amd64
     image: tanzu-cli-plugins/foo-darwin-amd64:latest
     os: darwin
     type: oci
   - arch: amd64
     image: tanzu-cli-plugins/foo-linux-amd64:latest
     os: linux
     type: oci
   - arch: amd64
     image: tanzu-cli-plugins/foo-windows-amd64:latest
     os: windows
     type: foo
 description: Foo description
 optional: false
 recommendedVersion: v0.0.1
`

var testData2 = `---
apiVersion: cli.tanzu.vmware.com/v1alpha1
kind: CLIPlugin
metadata:
 name: foo
spec:
 artifacts:
   v0.0.1:
   - arch: amd64
     image: tanzu-cli-plugins/foo-darwin-amd64:latest
     os: darwin
     type: oci
   - arch: amd64
     image: tanzu-cli-plugins/foo-linux-amd64:latest
     os: linux
     type: oci
   - arch: amd64
     image: tanzu-cli-plugins/foo-windows-amd64:latest
     os: windows
     type: oci
 description: Foo description
 optional: false
 recommendedVersion: v0.0.1
---
apiVersion: cli.tanzu.vmware.com/v1alpha1
kind: CLIPlugin
metadata:
 name: bar
spec:
 artifacts:
   v0.0.1:
   - arch: amd64
     image: tanzu-cli-plugins/foo-darwin-amd64:latest
     os: darwin
     type: oci
   - arch: amd64
     image: tanzu-cli-plugins/foo-linux-amd64:latest
     os: linux
     type: oci
   - arch: amd64
     image: tanzu-cli-plugins/foo-windows-amd64:latest
     os: windows
     type: oci
   v0.0.2:
   - arch: amd64
     image: tanzu-cli-plugins/foo-darwin-amd64:latest
     os: darwin
     type: oci
   - arch: amd64
     image: tanzu-cli-plugins/foo-linux-amd64:latest
     os: linux
     type: oci
   - arch: amd64
     image: tanzu-cli-plugins/foo-windows-amd64:latest
     os: windows
     type: oci
 description: Bar description
 optional: false
 recommendedVersion: v0.0.2
`

func Test_ProcessOCIPluginManifest(t *testing.T) {
	assert := assert.New(t)

	plugins, err := processDiscoveryManifestData([]byte(testData1), "test-discovery")
	assert.Nil(err)
	assert.NotNil(plugins)
	assert.Equal(1, len(plugins))
	assert.Equal("foo", plugins[0].Name)
	assert.Equal("v0.0.1", plugins[0].RecommendedVersion)
	assert.Equal("Foo description", plugins[0].Description)
	assert.Equal("test-discovery", plugins[0].Source)
	assert.EqualValues([]string{"v0.0.1"}, plugins[0].SupportedVersions)

	plugins, err = processDiscoveryManifestData([]byte(testData2), "test-discovery")
	assert.Nil(err)
	assert.NotNil(plugins)
	assert.Equal(2, len(plugins))

	assert.Equal("foo", plugins[0].Name)
	assert.Equal("v0.0.1", plugins[0].RecommendedVersion)
	assert.Equal("Foo description", plugins[0].Description)
	assert.Equal("test-discovery", plugins[0].Source)
	assert.Equal(1, len(plugins[0].SupportedVersions))
	assert.EqualValues([]string{"v0.0.1"}, plugins[0].SupportedVersions)

	assert.Equal("bar", plugins[1].Name)
	assert.Equal("v0.0.2", plugins[1].RecommendedVersion)
	assert.Equal("Bar description", plugins[1].Description)
	assert.Equal("test-discovery", plugins[1].Source)
	assert.Equal(2, len(plugins[1].SupportedVersions))
	assert.Contains(plugins[1].SupportedVersions, "v0.0.1")
	assert.Contains(plugins[1].SupportedVersions, "v0.0.2")
}

func Test_NewOCIDiscovery(t *testing.T) {
	assert := assert.New(t)

	configFile, err := os.CreateTemp("", "config")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG", configFile.Name())

	configFileNG, err := os.CreateTemp("", "config_ng")
	assert.Nil(err)
	os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

	featureArray := strings.Split(constants.FeatureContextCommand, ".")
	err = config.SetFeature(featureArray[1], featureArray[2], "true")
	assert.Nil(err)

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

	// Turn off central repo feature
	featureArray = strings.Split(constants.FeatureDisableCentralRepositoryForTesting, ".")
	err = config.SetFeature(featureArray[1], featureArray[2], "true")
	assert.Nil(err)

	// Check that the correct discovery type is returned
	pd = NewOCIDiscovery(discoveryName, discoveryImage, WithPluginDiscoveryCriteria(discoveryCriteria))
	assert.NotNil(pd)
	assert.Equal(discoveryName, pd.Name())
	assert.Equal(common.DiscoveryTypeOCI, pd.Type())

	ociDiscovery, ok := pd.(*OCIDiscovery)
	assert.True(ok)
	assert.Equal(discoveryName, ociDiscovery.name)
	assert.Equal(discoveryImage, ociDiscovery.image)
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
