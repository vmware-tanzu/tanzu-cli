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
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// SQLiteInventory is an inventory stored using SQLite
type SQLiteInventory struct {
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

	// querySelectClause is the SELECT section of the SQL query to be used when querying the inventory DB.
	querySelectClause = "SELECT PluginName,Target,RecommendedVersion,Version,Hidden,Description,Publisher,Vendor,OS,Architecture,Digest,URI FROM PluginBinaries"

	// queryOrderClause is the ORDER section of the SQL query to be used when querying the inventory DB.
	// It MUST be used as the order of the results is required by the functions processing the results.
	// The column order must also match the order used in getNextRow().
	queryOrderClause = "ORDER BY PluginName,Target,Version"
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
func NewSQLiteInventory(inventoryDir, prefix string) PluginInventory {
	return &SQLiteInventory{
		inventoryFile: filepath.Join(inventoryDir, sqliteDBFileName),
		uriPrefix:     prefix,
	}
}

// GetAllPlugins returns all plugins found in the inventory.
func (b *SQLiteInventory) GetAllPlugins() ([]*PluginInventoryEntry, error) {
	return b.getPluginsFromDB(nil)
}

// GetPlugins returns the plugin found in the inventory that matches the provided parameters.
func (b *SQLiteInventory) GetPlugins(filter *PluginInventoryFilter) ([]*PluginInventoryEntry, error) {
	return b.getPluginsFromDB(filter)
}

// getPluginsFromDB returns the plugins found in the DB 'inventoryFile' that match the filter
func (b *SQLiteInventory) getPluginsFromDB(filter *PluginInventoryFilter) ([]*PluginInventoryEntry, error) {
	db, err := sql.Open("sqlite3", b.inventoryFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open the DB at '%s'", b.inventoryFile)
	}
	defer db.Close()

	whereClause, err := createWhereClause(filter)
	if err != nil {
		return nil, err
	}

	// Build the final query with the SELECT, WHERE and ORDER clauses.
	// The ORDER clause is essential because the parsing algorithm of extractPluginsFromRows()
	// assumes that ordering.
	dbQuery := fmt.Sprintf("%s %s %s", querySelectClause, whereClause, queryOrderClause)
	rows, err := db.Query(dbQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to setup DB query for DB at '%s'", b.inventoryFile)
	}
	defer rows.Close()

	return b.extractPluginsFromRows(rows)
}

// createWhereClause parses the filter and creates the WHERE clause for the DB query.
func createWhereClause(filter *PluginInventoryFilter) (string, error) {
	var whereClause string

	// If there is a filter, create a WHERE clause for the query.
	if filter != nil {
		if filter.Name != "" {
			whereClause = fmt.Sprintf("%s PluginName='%s' AND", whereClause, filter.Name)
		}
		if filter.Target != "" {
			var target string
			switch filter.Target {
			case configtypes.TargetK8s:
				target = "k8s"
			case configtypes.TargetTMC:
				target = "tmc"
			default:
				return whereClause, fmt.Errorf("invalid target for plugin: %s", string(filter.Target))
			}

			whereClause = fmt.Sprintf("%s Target='%s' AND", whereClause, target)
		}
		if filter.Version != "" {
			if filter.Version == cli.VersionLatest {
				// We want the recommended version of the plugin
				whereClause = fmt.Sprintf("%s Version=RecommendedVersion AND", whereClause)
			} else {
				// We want a specific version of the plugin
				whereClause = fmt.Sprintf("%s Version='%s' AND", whereClause, filter.Version)
			}
		}
		if filter.OS != "" {
			whereClause = fmt.Sprintf("%s OS='%s' AND", whereClause, filter.OS)
		}
		if filter.Arch != "" {
			whereClause = fmt.Sprintf("%s Architecture='%s' AND", whereClause, filter.Arch)
		}
		if filter.Publisher != "" {
			whereClause = fmt.Sprintf("%s Publisher='%s' AND", whereClause, filter.Publisher)
		}
		if filter.Vendor != "" {
			whereClause = fmt.Sprintf("%s Vendor='%s' AND", whereClause, filter.Vendor)
		}

		if whereClause != "" {
			// Remove the last added "AND"
			whereClause = strings.TrimSuffix(whereClause, "AND")
			// Add the "WHERE" keyword
			whereClause = fmt.Sprintf("WHERE %s", whereClause)
		}
	}
	return whereClause, nil
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
			}
			currentVersion = ""
			artifacts = distribution.Artifacts{}
		}

		// Check if we have a new version
		if currentVersion != row.version {
			// We know this is a new version of our current plugin since we have
			// requested the list of plugins from the database ordered by version.

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

// appendPlugin appends a PluginInventoryEntry to the specified array.
// This function needs to be used to do post-processing on the new plugin before storing it.
func appendPlugin(allPlugins []*PluginInventoryEntry, plugin *PluginInventoryEntry) []*PluginInventoryEntry {
	// Now that we are done gathering the information for the plugin
	// we need to compute the recommendedVersion if it wasn't provided
	// by the database
	if plugin.RecommendedVersion == "" && len(plugin.Artifacts) > 0 {
		var versions []string
		for v := range plugin.Artifacts {
			versions = append(versions, v)
		}
		if err := utils.SortVersions(versions); err != nil {
			fmt.Fprintf(os.Stderr, "error parsing versions for plugin %s: %v\n", plugin.Name, err)
		}
		plugin.RecommendedVersion = versions[len(versions)-1]
	}
	allPlugins = append(allPlugins, plugin)
	return allPlugins
}
