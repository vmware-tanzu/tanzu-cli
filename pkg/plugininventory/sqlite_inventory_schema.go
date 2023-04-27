// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugininventory

import (
	_ "embed"
	"strings"
)

var (
	// CreateTablesSchema defines the database schema to create sqlite database
	CreateTablesSchema = strings.TrimSpace(createTablesSchema)
	//go:embed data/sqlite/create_tables.sql
	createTablesSchema string

	// PluginInventoryMetadataCreateTablesSchema defines the database schema to create sqlite database for available plugins
	PluginInventoryMetadataCreateTablesSchema = strings.TrimSpace(pluginInventoryMetadataCreateTablesSchema)
	//go:embed data/sqlite/plugin_inventory_metadata_tables.sql
	pluginInventoryMetadataCreateTablesSchema string
)
