// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package centralconfiginit is used to add inventory updater to the globalinitializer
package centralconfiginit

import (
	"io"
	"os"
	"path/filepath"

	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/globalinit"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

// The Central Configuration feature uses a central_config.yaml file that gets downloaded
// along with the plugin inventory cache and is stored in the cache.  Older versions of the
// CLI (< 1.3.0) do not include this file in the cache.  It is therefore possible that
// the plugin inventory cache was setup by an older version of the CLI and is missing
// the central_config.yaml file.  In such a case, the digest of the cache will still indicate
// that the latest plugin inventory is present and the content of the cache will not be refreshed
// until the plugin inventory data changes in the central repo itself.  In such a case, to be able
// to benefit from the central configuration feature once the CLI is upgraded to >= 1.3.0
// we need to fix the cache.  This initializer checks if the central_config.yaml file is
// present in the cache and if not, it invalidates the cache.

func init() {
	globalinit.RegisterInitializer("Central Config Initializer", triggerForInventoryCacheInvalidation, invalidateInventoryCache)
}

// triggerForInventoryCacheInvalidation returns true if the central_config.yaml file is missing
// in the plugin inventory cache.
func triggerForInventoryCacheInvalidation() bool {
	sources, err := config.GetCLIDiscoverySources()
	if err != nil {
		// No discovery source
		return false
	}

	for _, source := range sources {
		centralConfigFile := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, source.OCI.Name, constants.CentralConfigFileName)

		if _, err := os.Stat(centralConfigFile); os.IsNotExist(err) {
			// As soon as we find a source that doesn't have a central_config.yaml file,
			// we need to perform some initialization.
			return true
		}
	}
	// If we get here, then all sources have a central_config.yaml file
	return false
}

// invalidateInventoryCache performs the required actions
func invalidateInventoryCache(_ io.Writer) error {
	sources, err := config.GetCLIDiscoverySources()
	if err != nil {
		// No discovery source
		return nil
	}

	var errorList []error
	for _, source := range sources {
		centralConfigFile := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, source.OCI.Name, constants.CentralConfigFileName)

		if _, err := os.Stat(centralConfigFile); os.IsNotExist(err) {
			// This source doesn't have a central_config.yaml file,
			// we need to invalidate its plugin inventory cache.
			err = discovery.RefreshDiscoveryDatabaseForSource(source, discovery.WithForceInvalidation())
			if err != nil {
				errorList = append(errorList, err)
			}
		}
	}
	return kerrors.NewAggregate(errorList)
}
