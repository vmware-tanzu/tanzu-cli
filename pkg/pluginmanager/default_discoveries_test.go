// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

func Test_defaultDiscoverySourceBasedOnServer(t *testing.T) {
	tanzuConfigBytes := `apiVersion: config.tanzu.vmware.com/v1alpha1
clientOptions:
  cli:
    useContextAwareDiscovery: true
current: mgmt
kind: ClientConfig
metadata:
  creationTimestamp: null
servers:
- managementClusterOpts:
    context: mgmt-admin@mgmt
    path: config
  name: mgmt
  type: managementcluster
`
	f, err := os.CreateTemp("", "tanzu_config")
	assert.Nil(t, err)
	err = os.WriteFile(f.Name(), []byte(tanzuConfigBytes), 0644)
	assert.Nil(t, err)
	defer func(name string) {
		err = os.Remove(name)
		assert.NoError(t, err)
	}(f.Name())
	err = os.Setenv("TANZU_CONFIG", f.Name())
	assert.NoError(t, err)

	f2, err := os.CreateTemp("", "tanzu_config")
	assert.Nil(t, err)
	err = os.WriteFile(f2.Name(), []byte(""), 0644)
	assert.Nil(t, err)
	defer func(name string) {
		err = os.Remove(name)
		assert.NoError(t, err)
	}(f2.Name())
	err = os.Setenv("TANZU_CONFIG_NEXT_GEN", f2.Name())
	assert.NoError(t, err)

	server, err := configlib.GetServer("mgmt")
	assert.Nil(t, err)
	pds := append(server.DiscoverySources, defaultDiscoverySourceBasedOnServer(server)...)
	assert.Equal(t, 1, len(pds))
	assert.Equal(t, pds[0].Kubernetes.Name, "default-mgmt")
	assert.Equal(t, pds[0].Kubernetes.Path, "config")
	assert.Equal(t, pds[0].Kubernetes.Context, "mgmt-admin@mgmt")
}

func Test_defaultDiscoverySourceBasedOnContext(t *testing.T) {
	// Test tmc global server
	tanzuConfigBytes := `apiVersion: config.tanzu.vmware.com/v1alpha1
clientOptions:
  cli:
    useContextAwareDiscovery: true
current: tmc-test
kind: ClientConfig
metadata:
  creationTimestamp: null
`
	nextGenCfgBytes := `contexts:
- globalOpts:
    endpoint: test.cloud.vmware.com:443
  name: tmc-test
  target: mission-control
- clusterOpts:
    context: mgmt-admin@mgmt
    path: config
  name: mgmt
  target: kubernetes
- name: tanzu-context-1
  target: tanzu
  contextType: tanzu
  globalOpts:
    endpoint: https://localhost:8443
    auth:
      issuer: https://console-stg.cloud.vmware.com/csp/gateway/am/api
      userName: anujc
      accessToken: eyJ
      IDToken: eyJ
      refresh_token: sA4
      expiration: 2024-01-18T14:56:59.557973-08:00
      type: id-token
  clusterOpts:
    endpoint: https://localhost:8443/org/testorg
    path: test/kubeconfig.yaml
    context: tanzu-cli-tanzu-context-1
  additionalMetadata:
    tanzuOrgID: testorg`
	tf, err := os.CreateTemp("", "tanzu_tmc_config")
	assert.Nil(t, err)
	err = os.WriteFile(tf.Name(), []byte(tanzuConfigBytes), 0644)
	assert.Nil(t, err)
	defer func(name string) {
		err = os.Remove(name)
		assert.NoError(t, err)
	}(tf.Name())
	err = os.Setenv("TANZU_CONFIG", tf.Name())
	assert.Nil(t, err)

	f2, err := os.CreateTemp("", "tanzu_config")
	assert.Nil(t, err)
	err = os.WriteFile(f2.Name(), []byte(nextGenCfgBytes), 0644)
	assert.Nil(t, err)
	defer func(name string) {
		err = os.Remove(name)
		assert.NoError(t, err)
	}(f2.Name())
	err = os.Setenv("TANZU_CONFIG_NEXT_GEN", f2.Name())
	assert.NoError(t, err)

	context, err := configlib.GetContext("tmc-test")
	assert.Nil(t, err)
	pdsTMC := append(context.DiscoverySources, defaultDiscoverySourceBasedOnContext(context)...)
	assert.Equal(t, 1, len(pdsTMC))
	assert.Equal(t, pdsTMC[0].REST.Endpoint, "https://test.cloud.vmware.com:443")
	assert.Equal(t, pdsTMC[0].REST.BasePath, "v1alpha1/system/binaries/plugins")
	assert.Equal(t, pdsTMC[0].REST.Name, "default-tmc-test")

	context, err = configlib.GetContext("mgmt")
	assert.Nil(t, err)
	pdsK8s := append(context.DiscoverySources, defaultDiscoverySourceBasedOnContext(context)...)
	assert.Equal(t, 1, len(pdsK8s))
	assert.Equal(t, pdsK8s[0].Kubernetes.Name, "default-mgmt")
	assert.Equal(t, pdsK8s[0].Kubernetes.Path, "config")
	assert.Equal(t, pdsK8s[0].Kubernetes.Context, "mgmt-admin@mgmt")

	// Verify that no discovery sources are returned when feature-flag is disabled
	context, err = configlib.GetContext("tanzu-context-1")
	assert.Nil(t, err)
	pdsTanzu := append(context.DiscoverySources, defaultDiscoverySourceBasedOnContext(context)...)
	assert.Equal(t, 0, len(pdsTanzu))

	// Enable feature-flag and verify that one discovery source is returned
	err = configlib.ConfigureFeatureFlags(map[string]bool{constants.FeaturePluginDiscoveryForTanzuContext: true})
	assert.Nil(t, err)
	context, err = configlib.GetContext("tanzu-context-1")
	assert.Nil(t, err)
	pdsTanzu = append(context.DiscoverySources, defaultDiscoverySourceBasedOnContext(context)...)
	assert.Equal(t, 1, len(pdsTanzu))
	assert.Equal(t, pdsTanzu[0].Kubernetes.Name, "default-tanzu-context-1")
	assert.NotEmpty(t, pdsTanzu[0].Kubernetes.KubeConfigBytes)
}

func Test_appendURLScheme(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		output   string
	}{
		{
			name:     "url does not start with any scheme and not port",
			endpoint: "tmc.cloud.vmware.com",
			output:   "https://tmc.cloud.vmware.com",
		},
		{
			name:     "url does not start with any scheme, but ends with https port",
			endpoint: "tmc.cloud.vmware.com:443",
			output:   "https://tmc.cloud.vmware.com:443",
		},

		{
			name:     "url does not start with any scheme, but ends with non-default https port",
			endpoint: "tmc.cloud.vmware.com:8443",
			output:   "https://tmc.cloud.vmware.com:8443",
		},
		{
			name:     "url does start with http, but ends with https port",
			endpoint: "http://tmc.cloud.vmware.com:443",
			output:   "http://tmc.cloud.vmware.com:443",
		},

		{
			name:     "url does start with https, but ends with https port",
			endpoint: "https://tmc.cloud.vmware.com:443",
			output:   "https://tmc.cloud.vmware.com:443",
		},

		{
			name:     "url start with http, but ends with non-default http/https port",
			endpoint: "http://tmc.cloud.vmware.com:9443",
			output:   "http://tmc.cloud.vmware.com:9443",
		},
		{
			name:     "url start with http, but ends with default http port",
			endpoint: "http://tmc.cloud.vmware.com:80",
			output:   "http://tmc.cloud.vmware.com:80",
		},
	}
	for _, spec := range tests {
		t.Run(spec.name, func(t *testing.T) {
			output := appendURLScheme(spec.endpoint)
			assert.Equal(t, output, spec.output)
		})
	}
}
