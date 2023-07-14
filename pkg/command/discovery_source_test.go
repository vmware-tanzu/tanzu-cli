// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

func Test_createDiscoverySource(t *testing.T) {
	assert := assert.New(t)

	common.DefaultLocalPluginDistroDir = "../pluginmanager/test/local/"

	// When discovery source name is empty
	_, err := createDiscoverySource("LOCAL", "", "fake/path")
	assert.NotNil(err)
	assert.Equal(err.Error(), "discovery source name cannot be empty")

	// When discovery source type is empty
	_, err = createDiscoverySource("", "fake-discovery-name", "fake/path")
	assert.NotNil(err)
	assert.Contains(err.Error(), "discovery source type cannot be empty")

	// When discovery source is `local` and data is provided correctly
	// but path is invalid
	pd, err := createDiscoverySource("local", "fake-discovery-name", "fake/path")
	assert.NotNil(err)
	assert.Contains(err.Error(), "error while reading local plugin manifest directory")
	assert.NotNil(pd.Local)
	assert.Equal(pd.Local.Name, "fake-discovery-name")
	assert.Equal(pd.Local.Path, "fake/path")

	// When discovery source is `local` with a valid path
	pd, err = createDiscoverySource("local", "fake-discovery-name", "standalone")
	assert.Nil(err)
	assert.NotNil(pd.Local)
	assert.Equal(pd.Local.Name, "fake-discovery-name")
	assert.Equal(pd.Local.Path, "standalone")

	// When discovery source is `LOCAL` with a valid path
	pd, err = createDiscoverySource("LOCAL", "fake-discovery-name", "standalone")
	assert.Nil(err)
	assert.NotNil(pd.Local)
	assert.Equal(pd.Local.Name, "fake-discovery-name")
	assert.Equal(pd.Local.Path, "standalone")

	// When discovery source is `oci` with an invalid image
	pd, err = createDiscoverySource("oci", "fake-oci-discovery-name", "test.registry.com/test-image:v1.0.0")
	assert.NotNil(err)
	assert.Contains(err.Error(), "unable to fetch the inventory of discovery 'fake-oci-discovery-name' for plugins")
	assert.NotNil(pd.OCI)
	assert.Equal(pd.OCI.Name, "fake-oci-discovery-name")
	assert.Equal(pd.OCI.Image, "test.registry.com/test-image:v1.0.0")

	// When discovery source is gcp
	_, err = createDiscoverySource("gcp", "fake-discovery-name", "fake/path")
	assert.NotNil(err)
	assert.Contains(err.Error(), "not yet supported")

	// When discovery source is kubernetes
	_, err = createDiscoverySource("kubernetes", "fake-discovery-name", "fake/path")
	assert.NotNil(err)
	assert.Contains(err.Error(), "not yet supported")

	// When discovery source is rest with invalid endpoint
	pd, err = createDiscoverySource("rest", "fake-discovery-name", "fake/path")
	assert.NotNil(err)
	assert.Contains(err.Error(), "unsupported protocol scheme")
	assert.NotNil(pd.REST)
	assert.Equal(pd.REST.Name, "fake-discovery-name")
	assert.Equal(pd.REST.Endpoint, "fake/path")

	// When discovery source is an unknown value
	_, err = createDiscoverySource("unexpectedValue", "fake-discovery-name", "fake/path")
	assert.NotNil(err)
	assert.Contains(err.Error(), "unknown discovery source type 'unexpectedValue'")
}

// Test_createAndListDiscoverySources test 'tanzu plugin source list' when TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY has set test only discovery sources
func Test_createAndListDiscoverySources(t *testing.T) {
	oldVal := os.Getenv(constants.ConfigVariableAdditionalDiscoveryForTesting)
	os.Setenv(constants.CEIPOptInUserPromptAnswer, "No")
	os.Setenv(constants.EULAPromptAnswer, "Yes")

	// Initialize the plugin source
	rootCmd, err := NewRootCmd()
	assert.Nil(t, err)
	rootCmd.SetArgs([]string{"plugin", "source", "init"})
	err = rootCmd.Execute()
	assert.Nil(t, err)

	// List plugin source - one
	os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, "harbor-repo.vmware.com/tanzu_cli_stage/plugins/plugin-inventory:latest")

	rootCmd.SetArgs([]string{"plugin", "source", "list"})
	b := bytes.NewBufferString("")
	rootCmd.SetOut(b)
	err = rootCmd.Execute()
	assert.Nil(t, err)

	got, err := io.ReadAll(b)
	assert.Nil(t, err)

	str := string(got)
	assert.Contains(t, str, "disc_0 (test only)")

	// List plugin source - two
	os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, "harbor-repo.vmware.com/tanzu_cli_stage/plugins/plugin-inventory:latest,localhost:9876/tanzu-cli/plugins/sandbox1:small")

	rootCmd.SetArgs([]string{"plugin", "source", "list"})
	b = bytes.NewBufferString("")
	rootCmd.SetOut(b)
	err = rootCmd.Execute()
	assert.Nil(t, err)

	got, err = io.ReadAll(b)
	assert.Nil(t, err)

	str = string(got)
	assert.Contains(t, str, "disc_0 (test only)")
	assert.Contains(t, str, "disc_1 (test only)")
	os.Setenv(constants.ConfigVariableAdditionalDiscoveryForTesting, oldVal)
}
func Test_addDiscoverySource(t *testing.T) {
}

func Test_updateDiscoverySources(t *testing.T) {
}

func Test_deleteDiscoverySource(t *testing.T) {
}
