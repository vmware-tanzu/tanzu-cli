// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package centralconfig implements an interface to deal with the central configuration.
package centralconfig

import (
	"path/filepath"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// CentralConfigFileName is the name of the central config file
const CentralConfigFileName = "central_config.yaml"

// CentralConfig is used to interact with the central configuration.
type CentralConfig interface {
	// GetCentralConfigEntry reads the central configuration and
	// returns the value for the given key. The value is unmarshalled
	// into the out parameter. The out parameter must be a non-nil
	// pointer to a value.  If the key does not exist, the out parameter
	// is not modified and an error is returned.
	GetCentralConfigEntry(key string, out interface{}) error
}

// NewCentralConfigReader returns a CentralConfig reader that can
// be used to read central configuration values.
func NewCentralConfigReader(pd *types.PluginDiscovery) CentralConfig {
	// The central config is stored in the cache
	centralConfigFile := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, pd.OCI.Name, CentralConfigFileName)

	return &centralConfigYamlReader{configFile: centralConfigFile}
}
