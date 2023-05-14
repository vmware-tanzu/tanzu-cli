// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugininventory

import (
	"database/sql"

	// Import the sqlite3 driver
	_ "modernc.org/sqlite"

	"github.com/pkg/errors"
)

// SQLiteInventoryMetadata is an inventory metadata stored using SQLite
type SQLiteInventoryMetadata struct {
	// inventoryMetadataDBFile represents the full path to the SQLite DB file
	inventoryMetadataDBFile string
}

const (
	// SQliteInventoryMetadataDBFileName is the name of the DB file that is stored in
	// the OCI image describing the plugin inventory metadata.
	SQliteInventoryMetadataDBFileName = "plugin_inventory_metadata.db"
)

// NewSQLiteInventoryMetadata returns a new PluginInventoryMetadata connected to the data found at 'inventoryMetadataDBFile'.
func NewSQLiteInventoryMetadata(inventoryMetadataDBFile string) PluginInventoryMetadata {
	return &SQLiteInventoryMetadata{
		inventoryMetadataDBFile: inventoryMetadataDBFile,
	}
}

// CreateInventoryMetadataDBSchema creates table schemas for
// plugin inventory metadata database
// returns error if table creation fails for any reason
func (b *SQLiteInventoryMetadata) CreateInventoryMetadataDBSchema() error {
	db, err := sql.Open("sqlite", b.inventoryMetadataDBFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB at '%s'", b.inventoryMetadataDBFile)
	}
	defer db.Close()

	_, err = db.Exec(PluginInventoryMetadataCreateTablesSchema)
	if err != nil {
		return errors.Wrap(err, "error while creating tables to the database")
	}

	return nil
}

// InsertPluginIdentifier inserts the PluginIdentifier entry to the
// AvailablePluginBinaries table
func (b *SQLiteInventoryMetadata) InsertPluginIdentifier(pi *PluginIdentifier) error {
	db, err := sql.Open("sqlite", b.inventoryMetadataDBFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB from '%s' file", b.inventoryMetadataDBFile)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO AvailablePluginBinaries VALUES(?,?,?);", pi.Name, pi.Target, pi.Version)
	if err != nil {
		return errors.Wrapf(err, "unable to insert plugin identifier %v", pi)
	}
	return nil
}

// InsertPluginGroupIdentifier inserts the PluginGroupIdentifier entry to the
// AvailablePluginGroups table
func (b *SQLiteInventoryMetadata) InsertPluginGroupIdentifier(pgi *PluginGroupIdentifier) error {
	db, err := sql.Open("sqlite", b.inventoryMetadataDBFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB from '%s' file", b.inventoryMetadataDBFile)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO AvailablePluginGroups VALUES(?,?,?,?);", pgi.Vendor, pgi.Publisher, pgi.Name, pgi.Version)
	if err != nil {
		return errors.Wrapf(err, "unable to insert plugin group identifier %v", pgi)
	}
	return nil
}

// MergeInventoryMetadataDatabase merges two inventory metadata database by
// merging the content of AvailablePluginBinaries and AvailablePluginGroups tables
func (b *SQLiteInventoryMetadata) MergeInventoryMetadataDatabase(additionalMetadataDBFilePath string) error {
	db, err := sql.Open("sqlite", b.inventoryMetadataDBFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB from '%s' file", b.inventoryMetadataDBFile)
	}
	defer db.Close()

	mergeQuery := `ATTACH ? as additionalMetadataDB;
	INSERT OR REPLACE INTO AvailablePluginGroups SELECT Vendor,Publisher,GroupName,GroupVersion FROM additionalMetadataDB.AvailablePluginGroups;
	INSERT OR REPLACE INTO AvailablePluginBinaries SELECT PluginName,Target,Version FROM additionalMetadataDB.AvailablePluginBinaries;`

	_, err = db.Exec(mergeQuery, additionalMetadataDBFilePath)
	if err != nil {
		return errors.Wrapf(err, "unable to execute the query %v", mergeQuery)
	}
	return nil
}

// UpdatePluginInventoryDatabase updates the plugin inventory database based
// on the plugin inventory metadata database by deleting entries that don't
// exists in plugin inventory metadata database
func (b *SQLiteInventoryMetadata) UpdatePluginInventoryDatabase(pluginInventoryDBFilePath string) error {
	db, err := sql.Open("sqlite", b.inventoryMetadataDBFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB from '%s' file", b.inventoryMetadataDBFile)
	}
	defer db.Close()

	updateQuery := `ATTACH ? as piDB;
	DELETE FROM piDB.PluginGroups WHERE ROWID IN (SELECT a.ROWID FROM piDB.PluginGroups a LEFT JOIN AvailablePluginGroups b ON b.Vendor = a.Vendor AND b.Publisher = a.Publisher AND b.GroupName = a.GroupName AND b.GroupVersion = a.GroupVersion WHERE b.GroupVersion IS null);
	DELETE FROM piDB.PluginBinaries WHERE ROWID IN (SELECT a.ROWID FROM piDB.PluginBinaries a LEFT JOIN AvailablePluginBinaries b ON b.PluginName = a.PluginName AND b.Target = a.Target AND b.Version = a.Version WHERE b.PluginName IS null);`

	_, err = db.Exec(updateQuery, pluginInventoryDBFilePath)
	if err != nil {
		return errors.Wrap(err, "error while updating plugin inventory database")
	}
	return nil
}
