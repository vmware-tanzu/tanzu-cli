// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package centralconfig implements an interface to deal with the central configuration.
package centralconfig

import (
	"path/filepath"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

//go:generate counterfeiter -o ./fakes/central_config_fake.go --fake-name CentralConfig . CentralConfig

// CentralConfig is used to interact with the central configuration.
type CentralConfig interface {
	// GetCentralConfigEntry reads the central configuration and
	// returns the value for the given key. The value is unmarshalled
	// into the out parameter. The out parameter must be a non-nil
	// pointer to a value.  If the key does not exist, the out parameter
	// is not modified and an error is returned.
	GetCentralConfigEntry(key string, out interface{}) error

	// GetDefaultTanzuEndpoint returns default endpoint for the tanzu platform from the default
	// central configuration file
	GetDefaultTanzuEndpoint() (string, error)
	// GetTanzuPlatformEndpointToServiceEndpointMap returns Map of tanzu platform endpoint to service endpoints
	// from the default central configuration file
	GetTanzuPlatformEndpointToServiceEndpointMap() (TanzuPlatformEndpointToServiceEndpointMap, error)
	// GetTanzuPlatformSaaSEndpointList returns list of tanzu platform saas endpoints which can be a regular
	// expression. When comparing the result please make sure to use regex match instead of string comparison
	GetTanzuPlatformSaaSEndpointList() []string
	// GetTanzuConfigEndpointUpdateVersion returns current version for the local configuration file update
	// If the version specified here does not match with the local version stored in the datastore that means
	// the local configuration file endpoint updates are required
	GetTanzuConfigEndpointUpdateVersion() (string, error)
	// GetTanzuConfigEndpointUpdateMapping returns mapping of old endpoints to new endpoints that needs to be updated
	// in the user's local configuration file
	GetTanzuConfigEndpointUpdateMapping() (map[string]string, error)
}

// newCentralConfigReader returns a CentralConfig reader that can be used to read central configuration values.
// The reader is initialized with the specified plugin discovery name and reads the central configuration data from the cache.
//
// Note: This function is currently private because CLI does not require custom discovery mechanisms beyond the default discovery.
// For default discovery please use the pre-initialized `DefaultCentralConfigReader` object instead.
func newCentralConfigReader(pluginDiscoveryName string) CentralConfig {
	// The central config is stored in the cache
	centralConfigFile := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, pluginDiscoveryName, constants.CentralConfigFileName)
	return &centralConfigYamlReader{configFile: centralConfigFile}
}

// newDefaultCentralConfigReader returns a CentralConfig reader that can be used to read default central configuration values.
//
// Note: This function is currently private because the pre-initialized `DefaultCentralConfigReader` object should be used instead.
func newDefaultCentralConfigReader() CentralConfig {
	return newCentralConfigReader("default")
}
