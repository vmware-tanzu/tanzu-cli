// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugininventory

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
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
	// SQliteDBFileName is the name of the DB file that is stored in
	// the OCI image describing the inventory of plugins.
	SQliteDBFileName = "plugin_inventory.db"

	// pluginSelectClause is the SELECT section of the SQL query to be used when querying the inventory DB.
	pluginSelectClause = "SELECT PluginName,Target,RecommendedVersion,Version,Hidden,Description,Publisher,Vendor,OS,Architecture,Digest,URI FROM PluginBinaries"

	// pluginOrderClause is the ORDER section of the SQL query to be used when querying the inventory DB.
	// It MUST be used as the order of the results is required by the functions processing the results.
	// The column order must also match the order used in getNextRow().
	pluginOrderClause = "ORDER BY PluginName,Target,Version"

	// groupSelectQuery is the query used to extract plugin groups from the PluginGroups table
	groupSelectQuery = "SELECT Vendor,Publisher,GroupName,PluginName,Target,Version FROM PluginGroups ORDER by Vendor,Publisher,GroupName,PluginName"
)

// Structure of each row of the PluginBinaries table within the SQLite database
type pluginDBRow struct {
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

// Structure of each row of the PluginGroups table within the SQLite database
type groupDBRow struct {
	vendor     string
	publisher  string
	groupName  string
	pluginName string
	target     string
	version    string
}

// NewSQLiteInventory returns a new PluginInventory connected to the data found at 'inventoryFile'.
func NewSQLiteInventory(inventoryFile, prefix string) PluginInventory {
	return &SQLiteInventory{
		inventoryFile: inventoryFile,
		uriPrefix:     prefix,
	}
}

// GetAllPlugins returns all plugins found in the inventory.
func (b *SQLiteInventory) GetAllPlugins() ([]*PluginInventoryEntry, error) {
	return b.getPluginsFromDB(nil)
}

// GetPlugins returns the plugin found in the inventory that matches the provided parameters.
func (b *SQLiteInventory) GetPlugins(filter *PluginInventoryFilter) ([]*PluginInventoryEntry, error) {
	// Since the Central Repo does not have its RecommendedVersion field set yet,
	// we first search for it by looking for the latest version amongst all versions.
	if filter.Version == cli.VersionLatest {
		if filter.Name == "" {
			return nil, fmt.Errorf("cannot get the recommended version of a plugin without a plugin name")
		}
		// Ask for all versions
		filter.Version = ""
		plugins, err := b.getPluginsFromDB(filter)
		if err != nil {
			return nil, err
		}
		// We could end up with two plugins if we didn't filter on target.
		// We know this will cause an error as it trickles back up so we just return what
		// we found without further processing.  This is NOT generic, but a temporary workaround.
		// Also, if we have no plugins found, we can return immediately.
		if len(plugins) != 1 {
			return plugins, nil
		}

		// We can now use the RecommendedVersion field which was filled when parsing the DB.
		filter.Version = plugins[0].RecommendedVersion
	}

	return b.getPluginsFromDB(filter)
}

func (b *SQLiteInventory) GetAllGroups() ([]*PluginGroup, error) {
	return b.getGroupsFromDB()
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
	dbQuery := fmt.Sprintf("%s %s %s", pluginSelectClause, whereClause, pluginOrderClause)
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
			whereClause = fmt.Sprintf("%s Target='%s' AND", whereClause, string(filter.Target))
		}
		if filter.Version != "" {
			if filter.Version == cli.VersionLatest {
				// We want the recommended version of the plugin.
				// Note that currently the plugin repositories do not fill the RecommendedVersion column
				// of the DB; therefore this query would fail to return any matches.
				// To deal with this situation, the calling function finds the correct version
				// and never sends a filter using filter.Version == cli.VersionLatest.
				// This implies that the query below will never be triggered.
				// We leave it in to prepare for the time when the repositories will have a
				// RecommendedVersion column with correct values.
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
		row, err := getPluginNextRow(rows)
		if err != nil {
			return allPlugins, err
		}

		target := configtypes.StringToTarget(strings.ToLower(row.target))
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

			hidden, _ := strconv.ParseBool(row.hidden)
			currentPlugin = &PluginInventoryEntry{
				Name:               row.name,
				Target:             target,
				Description:        row.description,
				Publisher:          row.publisher,
				Vendor:             row.vendor,
				RecommendedVersion: row.recommendedVersion,
				Hidden:             hidden,
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

// getGroupsFromDB returns all the plugin groups found in the DB 'inventoryFile'
func (b *SQLiteInventory) getGroupsFromDB() ([]*PluginGroup, error) {
	db, err := sql.Open("sqlite3", b.inventoryFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open the DB at '%s' for groups", b.inventoryFile)
	}
	defer db.Close()

	rows, err := db.Query(groupSelectQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to setup DB query for DB at '%s' for groups", b.inventoryFile)
	}
	defer rows.Close()

	return b.extractGroupsFromRows(rows)
}

// extractGroupsFromRows loops through all DB rows and builds an array
// of PluginGroups based on the data extracted.
func (b *SQLiteInventory) extractGroupsFromRows(rows *sql.Rows) ([]*PluginGroup, error) {
	currentGroupID := ""
	var currentGroup *PluginGroup
	var allGroups []*PluginGroup
	var pluginsOfGroup []*PluginIdentifier

	for rows.Next() {
		row, err := getGroupNextRow(rows)
		if err != nil {
			return allGroups, err
		}

		groupIDFromRow := fmt.Sprintf("%s-%s/%s", row.vendor, row.publisher, row.groupName)
		if currentGroupID != groupIDFromRow {
			// Found a new group.
			// Store the current one in the array and prepare the new one.
			if currentGroup != nil {
				currentGroup.Plugins = pluginsOfGroup
				pluginsOfGroup = nil
				allGroups = append(allGroups, currentGroup)
			}
			currentGroupID = groupIDFromRow

			currentGroup = &PluginGroup{
				Vendor:    row.vendor,
				Publisher: row.publisher,
				Name:      row.groupName,
				Plugins:   nil,
			}
		}

		pluginsOfGroup = append(pluginsOfGroup, &PluginIdentifier{
			Name:    row.pluginName,
			Target:  convertTargetFromDB(row.target),
			Version: row.version,
		})
	}
	// Don't forget to store the very last group we were building
	if currentGroup != nil {
		currentGroup.Plugins = pluginsOfGroup
		allGroups = append(allGroups, currentGroup)
	}
	return allGroups, rows.Err()
}

// getPluginNextRow simply extracts the next row of data from the DB.
func getPluginNextRow(rows *sql.Rows) (*pluginDBRow, error) {
	var row pluginDBRow
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

// getGroupNextRow simply extracts the next row of data from the DB.
func getGroupNextRow(rows *sql.Rows) (*groupDBRow, error) {
	var row groupDBRow
	// The order of the fields MUST match the order specified in the
	// SELECT query that generated the rows.
	err := rows.Scan(
		&row.vendor,
		&row.publisher,
		&row.groupName,
		&row.pluginName,
		&row.target,
		&row.version,
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

// CreateSchema creates table schemas to the provided database.
// returns error if table creation fails for any reason
func (b *SQLiteInventory) CreateSchema() error {
	db, err := sql.Open("sqlite3", b.inventoryFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB at '%s'", b.inventoryFile)
	}
	defer db.Close()

	_, err = db.Exec(CreateTablesSchema)
	if err != nil {
		return errors.Wrap(err, "error while creating tables to the database")
	}

	return nil
}

// InsertPlugin inserts plugin to the inventory
func (b *SQLiteInventory) InsertPlugin(pluginInventoryEntry *PluginInventoryEntry) error {
	db, err := sql.Open("sqlite3", b.inventoryFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB from '%s' file", b.inventoryFile)
	}
	defer db.Close()

	for version, artifacts := range pluginInventoryEntry.Artifacts {
		for _, a := range artifacts {
			row := pluginDBRow{
				name:               pluginInventoryEntry.Name,
				target:             string(pluginInventoryEntry.Target),
				recommendedVersion: "",
				version:            version,
				hidden:             strconv.FormatBool(pluginInventoryEntry.Hidden),
				description:        pluginInventoryEntry.Description,
				publisher:          pluginInventoryEntry.Publisher,
				vendor:             pluginInventoryEntry.Vendor,
				os:                 a.OS,
				arch:               a.Arch,
				digest:             a.Digest,
				uri:                a.Image,
			}

			_, err = db.Exec("INSERT INTO PluginBinaries VALUES(?,?,?,?,?,?,?,?,?,?,?,?);", row.name, row.target, row.recommendedVersion, row.version, row.hidden, row.description, row.publisher, row.vendor, row.os, row.arch, row.digest, row.uri)
			if err != nil {
				return errors.Wrapf(err, "unable to insert plugin row %v", row)
			}
		}
	}
	return nil
}

// UpdatePluginActivationState updates plugin metadata to activate or deactivate plugin
func (b *SQLiteInventory) UpdatePluginActivationState(pluginInventoryEntry *PluginInventoryEntry) error {
	db, err := sql.Open("sqlite3", b.inventoryFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB from '%s' file", b.inventoryFile)
	}
	defer db.Close()

	for version := range pluginInventoryEntry.Artifacts {
		result, err := db.Exec("UPDATE PluginBinaries SET hidden = ? WHERE PluginName = ? AND Target = ? AND Version = ? AND Publisher = ? AND Vendor = ? ;", strconv.FormatBool(pluginInventoryEntry.Hidden), pluginInventoryEntry.Name, string(pluginInventoryEntry.Target), version, pluginInventoryEntry.Publisher, pluginInventoryEntry.Vendor)
		if err != nil {
			return errors.Wrapf(err, "unable to update plugin %v_%v", pluginInventoryEntry.Name, string(pluginInventoryEntry.Target))
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return errors.Errorf("unable to update plugin %v_%v", pluginInventoryEntry.Name, string(pluginInventoryEntry.Target))
		}
	}
	return nil
}
