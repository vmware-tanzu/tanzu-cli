// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package lastversion contains functionality to store and retrieve the last
// executed CLI version in the datastore.
package lastversion

import (
	"strings"

	"github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/datastore"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

// lastExecutedCLIVersionKey is the key used to store the last executed CLI version in the datastore.
const (
	lastExecutedCLIVersionKey = "lastExecutedCLIVersion"
	OlderThan1_3_0            = "olderThan1_3_0"
)

// lastExecutedCLIVersion is a struct used to store the last executed CLI version in the datastore.
// We use a struct to be able to add more fields in the future if needed.
type lastExecutedCLIVersion struct {
	Version string `json:"version" yaml:"version"`
}

// GetLastExecutedCLIVersion gets the last executed CLI version from the datastore.
// If the last executed version is < 1.3.0, it returns an empty string, otherwise
// it returns the last executed version.
func GetLastExecutedCLIVersion() string {
	if config.IsFeatureActivated(constants.FeatureContextCommand) {
		// If this feature flag is present, then we know that the
		// last version executed was < 1.3.0.  We cannot know which version
		// specifically was last run because version 1.3.0 is the first version
		// to set the last executed version.  In this case, we return an empty string.
		// Don't return an empty string as it would be the same as the value returned
		// if the datastore is removed. Instead, return a constant value.
		return OlderThan1_3_0
	}

	var lastVersion lastExecutedCLIVersion
	_ = datastore.GetDataStoreValue(lastExecutedCLIVersionKey, &lastVersion)
	return lastVersion.Version
}

// SetLastExecutedCLIVersion sets the last executed CLI version in the datastore.
func SetLastExecutedCLIVersion() {
	var prevLastVersion lastExecutedCLIVersion
	_ = datastore.GetDataStoreValue(lastExecutedCLIVersionKey, &prevLastVersion)
	if prevLastVersion.Version != buildinfo.Version {
		// Only update the last executed version if it is different from the one already stored.
		_ = datastore.SetDataStoreValue(lastExecutedCLIVersionKey, lastExecutedCLIVersion{Version: buildinfo.Version})
	}

	// Just in case the 'features.global.context-target-v2' feature flag is still set
	// because the last version executed was < 1.3.0, we must remove it.
	parts := strings.Split(constants.FeatureContextCommand, ".")
	if enabled, err := config.IsFeatureEnabled(parts[1], parts[2]); err == nil && enabled {
		_ = config.DeleteFeature(parts[1], parts[2])
	}
}
