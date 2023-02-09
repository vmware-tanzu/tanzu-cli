// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugininventory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	// Import the sqlite3 driver
	_ "github.com/mattn/go-sqlite3"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// SQLiteInventory is an inventory stored using SQLite
type SQLiteInventory struct {
	// discoveryName is the name of the discovery powered by this backend
	discoveryName string
	// inventoryFile represents the full path to the SQLite DB file
	inventoryFile string
	// uriPrefix is the prefix that must be added to the extracted URIs.
	// To be future-proof the DB stores image URIs that are relative to
	// the inventory location.
	uriPrefix string
}

const (
	// sqliteDBFileName is the name of the DB file that is stored in
	// the OCI image describing the inventory of plugins.
	sqliteDBFileName = "plugin_inventory.db"
)

// Structure of each row of the PluginBinaries table within the SQLite database
type inventoryDBRow struct {
	name               string
	target             string
	recommendedVersion string
	version            string
	hidden             string
	description        string
	publisher          string
	vendor             string
	os                 string
	arch               string
	digest             string
	uri                string
}

// NewSQLiteInventory returns a new PluginInventory connected to the data found at 'inventoryDir'.
func NewSQLiteInventory(discoveryName, inventoryDir, prefix string) PluginInventory {
	return &SQLiteInventory{
		discoveryName: discoveryName,
		inventoryFile: filepath.Join(inventoryDir, sqliteDBFileName),
		uriPrefix:     prefix,
	}
}

// GetAllPlugins returns all plugins discovered in this backend.
func (b *SQLiteInventory) GetAllPlugins() ([]*PluginInventoryEntry, error) {
	return b.getPluginsFromDB()
}

// getPluginsFromDB returns all plugins found in the DB 'inventoryFile'
func (b *SQLiteInventory) getPluginsFromDB() ([]*PluginInventoryEntry, error) {
	db, err := sql.Open("sqlite3", b.inventoryFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open the DB for discovery '%s'", b.discoveryName)
	}
	defer db.Close()

	// We need to order the results properly because the logic of extractPluginsFromRows()
	// expects an ordering of PluginName, then Target, then Version.
	// The column order must also match the order used in getNextRow().
	dbQuery := "SELECT PluginName,Target,RecommendedVersion,Version,Hidden,Description,Publisher,Vendor,OS,Architecture,Digest,URI FROM PluginBinaries ORDER BY PluginName,Target,Version;"
	rows, err := db.Query(dbQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to setup DB query for discovery '%s'", b.discoveryName)
	}
	defer rows.Close()

	return b.extractPluginsFromRows(rows)
}

// extractPluginsFromRows loops through all DB rows and builds an array
// of Discovered plugins based on the data extracted.
func (b *SQLiteInventory) extractPluginsFromRows(rows *sql.Rows) ([]*PluginInventoryEntry, error) {
	currentPluginID := ""
	currentVersion := ""
	var currentPlugin *PluginInventoryEntry
	allPlugins := make([]*PluginInventoryEntry, 0)
	var artifactList distribution.ArtifactList
	var artifacts distribution.Artifacts

	for rows.Next() {
		row, err := getNextRow(rows)
		if err != nil {
			return allPlugins, err
		}

		target := convertTargetFromDB(row.target)
		pluginIDFromRow := catalog.PluginNameTarget(row.name, target)
		if currentPluginID != pluginIDFromRow {
			// Found a new plugin.
			// Store the current one in the array and prepare the new one.
			if currentPlugin != nil {
				artifacts[currentVersion] = artifactList
				artifactList = distribution.ArtifactList{}
				currentPlugin.Artifacts = artifacts
				allPlugins = appendPlugin(allPlugins, currentPlugin)
			}
			currentPluginID = pluginIDFromRow

			currentPlugin = &PluginInventoryEntry{
				Name:               row.name,
				Target:             target,
				Description:        row.description,
				Publisher:          row.publisher,
				Vendor:             row.vendor,
				RecommendedVersion: row.recommendedVersion,
				AvailableVersions:  []string{}, // Will be filled gradually below.
			}
			currentVersion = ""
			artifacts = distribution.Artifacts{}
		}

		// Check if we have a new version
		if currentVersion != row.version {
			// This is a new version of our current plugin.  Add it to the array of versions.
			// We can do this without verifying if the version is already there because
			// we have requested the list of plugins from the database ordered by version.
			currentPlugin.AvailableVersions = append(currentPlugin.AvailableVersions, row.version)

			// Store the list of artifacts for the previous version then start building
			// the artifact list for the new version.
			if currentVersion != "" {
				artifacts[currentVersion] = artifactList
				artifactList = distribution.ArtifactList{}
			}
			currentVersion = row.version
		}

		// The DB uses relative URIs to be future-proof.
		// Build the full URI before creating the artifact.
		fullImagePath := fmt.Sprintf("%s/%s", b.uriPrefix, row.uri)
		// Create the artifact for this row.
		artifact := distribution.Artifact{
			Image:  fullImagePath,
			URI:    "",
			Digest: row.digest,
			OS:     row.os,
			Arch:   row.arch,
		}
		artifactList = append(artifactList, artifact)
	}
	// Don't forget to store the very last plugin we were building
	if currentPlugin != nil {
		artifacts[currentVersion] = artifactList
		currentPlugin.Artifacts = artifacts
		allPlugins = appendPlugin(allPlugins, currentPlugin)
	}
	return allPlugins, rows.Err()
}

// getNextRow simply extracts the next row of data from the DB.
func getNextRow(rows *sql.Rows) (*inventoryDBRow, error) {
	var row inventoryDBRow
	// The order of the fields MUST match the order specified in the
	// SELECT query that generated the rows.
	err := rows.Scan(
		&row.name,
		&row.target,
		&row.recommendedVersion,
		&row.version,
		&row.hidden,
		&row.description,
		&row.publisher,
		&row.vendor,
		&row.os,
		&row.arch,
		&row.digest,
		&row.uri,
	)
	return &row, err
}

func convertTargetFromDB(target string) configtypes.Target {
	target = strings.ToLower(target)
	if target == "global" {
		target = ""
	}
	return configtypes.StringToTarget(target)
}

// appendPlugin appends a Discovered plugins to the specified array.
// This function needs to be used to do post-processing on the new plugin before storing it.
func appendPlugin(allPlugins []*PluginInventoryEntry, plugin *PluginInventoryEntry) []*PluginInventoryEntry {
	// Now that we are done gathering the information for the plugin
	// we need to compute the recommendedVersion if it wasn't provided
	// by the database
	if err := utils.SortVersions(plugin.AvailableVersions); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing supported versions for plugin %s: %v", plugin.Name, err)
	}
	if plugin.RecommendedVersion == "" {
		plugin.RecommendedVersion = plugin.AvailableVersions[len(plugin.AvailableVersions)-1]
	}
	allPlugins = append(allPlugins, plugin)
	return allPlugins
}
