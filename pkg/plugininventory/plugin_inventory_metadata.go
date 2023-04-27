// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugininventory

// PluginInventoryMetadata is the interface to interact with a plugin inventory
// metadata database and plugin inventory database.
// It can be used to create database schema for metadata db, insert
// plugin and plugin group identifier and merging metadata database
// This interface also provides function to update plugin inventory database
// based on the plugin inventory metadata database
type PluginInventoryMetadata interface {
	// CreateInventoryMetadataDBSchema creates table schemas for
	// plugin inventory metadata database
	// returns error if table creation fails for any reason
	CreateInventoryMetadataDBSchema() error

	// InsertPluginIdentifier inserts the PluginIdentifier entry to the
	// AvailablePluginBinaries table
	InsertPluginIdentifier(*PluginIdentifier) error

	// InsertPluginGroupIdentifier inserts the PluginGroupIdentifier entry to the
	// AvailablePluginGroups table
	InsertPluginGroupIdentifier(*PluginGroupIdentifier) error

	// MergeInventoryMetadataDatabase merges two inventory metadata database by
	// merging the content of AvailablePluginBinaries and AvailablePluginGroups tables
	MergeInventoryMetadataDatabase(additionalMetadataDBFilePath string) error

	// UpdatePluginInventoryDatabase updates the plugin inventory database based
	// on the plugin inventory metadata database by deleting entries that don't
	// exists in plugin inventory metadata database
	UpdatePluginInventoryDatabase(pluginInventoryDBFilePath string) error
}
