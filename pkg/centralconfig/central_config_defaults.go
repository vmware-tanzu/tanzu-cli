// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package centralconfig

var (
	// NOTE: This value will be overwritten from the value specified in the central configuration.
	// It serves as a fallback default only if reading the central configuration fails.
	DefaultTanzuPlatformEndpoint = "https://api.tanzu.cloud.vmware.com"

	// DefaultPluginDBCacheRefreshThresholdSeconds is the default value for db cache refresh
	// For testing, it can be overridden using the environment variable TANZU_CLI_PLUGIN_DB_CACHE_REFRESH_THRESHOLD_SECONDS.
	// It serves as a fallback default only if reading the central configuration fails.
	DefaultPluginDBCacheRefreshThresholdSeconds = 24 * 60 * 60 // 24 hours

	// DefaultInventoryRefreshTTLSeconds is the interval in seconds between two checks of the inventory digest.
	// For testing, it can be overridden using the environment variable TANZU_CLI_PLUGIN_DB_CACHE_TTL_SECONDS.
	DefaultInventoryRefreshTTLSeconds = 30 * 60 // 30 minutes

	defaultSaaSEndpoints = []string{
		"https://(www.)?platform(.)*.tanzu.broadcom.com",
		"https://api.tanzu(.)*.cloud.vmware.com",
	}

	// DefaultCentralConfigReader is a pre-initialized default central configuration reader instance.
	// This global object can be used directly to read configuration from the default central configuration.
	DefaultCentralConfigReader = newDefaultCentralConfigReader()
)

func init() {
	// initialize the value of `DefaultTanzuPlatformEndpoint` from default central configuration
	endpoint, _ := DefaultCentralConfigReader.GetDefaultTanzuEndpoint()
	if endpoint != "" {
		DefaultTanzuPlatformEndpoint = endpoint
	}
	// initialize the value of `DefaultPluginDBCacheRefreshThresholdSeconds` from default central configuration if specified there
	secondsThreshold, err := DefaultCentralConfigReader.GetPluginDBCacheRefreshThresholdSeconds()
	if err == nil && secondsThreshold > 0 {
		DefaultPluginDBCacheRefreshThresholdSeconds = secondsThreshold
	}
	// initialize the value of `DefaultInventoryRefreshTTLSeconds` from default central configuration if specified there
	secondsTTL, err := DefaultCentralConfigReader.GetInventoryRefreshTTLSeconds()
	if err == nil && secondsTTL > 0 {
		DefaultInventoryRefreshTTLSeconds = secondsTTL
	}
}
