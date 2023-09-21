// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	// Import the sqlite3 driver
	_ "modernc.org/sqlite"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
)

// sqliteMetricsDB implements the MetricDB interface using SQLite database
type sqliteMetricsDB struct {
	// metricsDBFile represents the full path to the SQLite DB file
	metricsDBFile string
}

const (
	// SQliteDBFileName is the name of the DB file that has CLI metrics
	SQliteDBFileName = "cli_metrics.db"

	// TanzuCLITelemetryMaxRowCount Max metric instances to be accumulated before pausing the collection
	TanzuCLITelemetryMaxRowCount = 10000

	// cliOperationMetricRowClause is the SELECT section of the SQL query to be used when querying the Metric DB row count.
	cliOperationMetricRowClause = "SELECT count(*) FROM tanzu_cli_operations"
)

// Structure of each row of the PluginBinaries table within the SQLite database
type cliOperationsRow struct {
	cliVersion         string
	osName             string
	osArch             string
	pluginName         string
	pluginVersion      string
	command            string
	cliID              string
	commandStartTSMsec string
	commandEndTSMsec   string
	target             string
	nameArg            string
	endpoint           string
	flags              string
	exitStatus         int
	isInternal         bool
	error              string
}

// newSQLiteMetricsDB returns a new PluginInventory connected to the data found at 'metricsDBFile'.
func newSQLiteMetricsDB() MetricsDB {
	dbFile := filepath.Join(common.DefaultCLITelemetryDir, SQliteDBFileName)
	return &sqliteMetricsDB{
		metricsDBFile: dbFile,
	}
}

// CreateSchema creates table schemas to the provided database.
// returns error if table creation fails for any reason
func (b *sqliteMetricsDB) CreateSchema() error {
	dirName := filepath.Dir(b.metricsDBFile)
	if _, serr := os.Stat(dirName); serr != nil {
		merr := os.MkdirAll(dirName, os.ModePerm)
		if merr != nil {
			return merr
		}
	}
	db, err := sql.Open("sqlite", b.metricsDBFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB at '%s'", b.metricsDBFile)
	}
	defer db.Close()

	_, err = db.Exec(CreateTablesSchema)
	if err != nil {
		return errors.Wrap(err, "error while creating tables to the database")
	}

	return nil
}

func (b *sqliteMetricsDB) SaveOperationMetric(entry *OperationMetricsPayload) error {
	err := AcquireTanzuMetricDBLock()
	if err != nil {
		return err
	}
	defer ReleaseTanzuMetricDBLock()

	db, err := sql.Open("sqlite", b.metricsDBFile)
	if err != nil {
		return errors.Wrapf(err, "failed to open the DB from '%s' file", b.metricsDBFile)
	}
	defer db.Close()

	atThreshold, err := isDBRowCountThresholdReached(db)
	if err != nil {
		return errors.Wrap(err, "failed to validate the DB size threshold")
	}
	if atThreshold {
		return errors.New("metrics DB size threshold reached")
	}

	row := cliOperationsRow{
		cliVersion:         entry.CliVersion,
		osName:             runtime.GOOS,
		osArch:             runtime.GOARCH,
		pluginName:         entry.PluginName,
		pluginVersion:      entry.PluginVersion,
		command:            entry.CommandName,
		cliID:              entry.CliID,
		commandStartTSMsec: strconv.FormatInt(entry.StartTime.UnixMilli(), 10),
		commandEndTSMsec:   strconv.FormatInt(entry.EndTime.UnixMilli(), 10),
		target:             entry.Target,
		nameArg:            entry.NameArg,
		endpoint:           entry.Endpoint,
		flags:              entry.Flags,
		exitStatus:         entry.ExitStatus,
		isInternal:         entry.IsInternal,
		error:              entry.Error,
	}

	_, err = db.Exec("INSERT INTO tanzu_cli_operations VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);", row.cliVersion, row.osName, row.osArch, row.pluginName, row.pluginVersion, row.command, row.cliID, row.commandStartTSMsec, row.commandEndTSMsec, row.target, row.nameArg, row.endpoint, row.flags, row.exitStatus, row.isInternal, row.error)
	if err != nil {
		return errors.Wrapf(err, "unable to insert clioperations row %v", row)
	}

	return nil
}

func (b *sqliteMetricsDB) GetRowCount() (int, error) {
	err := AcquireTanzuMetricDBLock()
	if err != nil {
		return 0, err
	}
	defer ReleaseTanzuMetricDBLock()
	db, err := sql.Open("sqlite", b.metricsDBFile)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to open the DB from '%s' file", b.metricsDBFile)
	}
	defer db.Close()

	dbQuery := cliOperationMetricRowClause
	rows, err := db.Query(dbQuery) //nolint:rowserrcheck
	if err != nil {
		return 0, errors.Wrapf(err, "failed to execute the DB query : %v", dbQuery)
	}
	defer rows.Close()
	count := 0
	if rows.Next() {
		err = rows.Scan(&count)
	}
	return count, err
}
func isDBRowCountThresholdReached(db *sql.DB) (bool, error) {
	dbQuery := cliOperationMetricRowClause
	rows, err := db.Query(dbQuery) //nolint:rowserrcheck
	if err != nil {
		return false, errors.Wrapf(err, "failed to execute the DB query : %v", dbQuery)
	}
	defer rows.Close()
	count := 0
	if rows.Next() {
		err = rows.Scan(&count)
	}

	return count >= TanzuCLITelemetryMaxRowCount, err
}
