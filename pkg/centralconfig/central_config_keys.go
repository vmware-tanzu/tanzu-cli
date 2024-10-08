// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package centralconfig implements an interface to deal with the central configuration.
package centralconfig

const (
	KeyDefaultTanzuEndpoint                          = "cli.core.tanzu_default_endpoint"
	KeyDefaultPluginDBCacheRefreshThresholdSeconds   = "cli.core.tanzu_cli_default_plugin_db_cache_refresh_threshold_seconds"
	KeyDefaultInventoryRefreshTTLSeconds             = "cli.core.tanzu_cli_default_inventory_refresh_ttl_seconds"
	KeyTanzuEndpointMap                              = "cli.core.tanzu_endpoint_map"
	KeyTanzuPlatformSaaSEndpointsAsRegularExpression = "cli.core.tanzu_cli_platform_saas_endpoints_as_regular_expression"
	KeyTanzuConfigEndpointUpdateVersion              = "cli.core.tanzu_cli_config_endpoint_update_version"
	KeyTanzuConfigEndpointUpdateMapping              = "cli.core.tanzu_cli_config_endpoint_update_mapping"
)
