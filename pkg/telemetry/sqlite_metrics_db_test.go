// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/pkg/errors"
)

var _ = Describe("Inserting CLI metrics to database and verifying it by fetching the metrics from database", func() {
	var (
		err    error
		db     *sqliteMetricsDB
		dbFile *os.File
		tmpDir string
	)
	BeforeEach(func() {
		tmpDir, err = os.MkdirTemp(os.TempDir(), "")
		Expect(err).To(BeNil(), "unable to create temporary directory")

		// Create DB file
		dbFile, err = os.Create(filepath.Join(tmpDir, SQliteDBFileName))
		Expect(err).To(BeNil())

		db = &sqliteMetricsDB{metricsDBFile: dbFile.Name()}
		err = db.CreateSchema()
		Expect(err).To(BeNil(), "failed to create DB schema for testing")
	})
	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})
	Context("When inserting the cli metrics data", func() {
		It("operation should be successful inserted into the database", func() {

			metricsPayload := &OperationMetricsPayload{
				CliID:         "fake-cli-cliID",
				StartTime:     time.Now(),
				EndTime:       time.Now().Add(10 * time.Millisecond),
				NameArg:       "fake-name-arg",
				CommandName:   "fake-cmd-name",
				ExitStatus:    0,
				PluginName:    "fake-plugin",
				Flags:         `{"v":"6","longflag":"lvalue"}`,
				CliVersion:    "v1.0.0",
				PluginVersion: "v0.0.1",
				Target:        "kubernetes",
				Endpoint:      "fake-endpoint-hash",
				IsInternal:    false,
				Error:         "",
			}

			err = db.SaveOperationMetric(metricsPayload)
			Expect(err).ToNot(HaveOccurred(), "failed to save the metrics")
			metricsRows, err := getOperationMetrics(db)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(metricsRows)).To(Equal(1))
			Expect(metricsRows[0].cliID).To(Equal("fake-cli-cliID"))
			Expect(metricsRows[0].cliVersion).To(Equal("v1.0.0"))
			Expect(metricsRows[0].osName).To(Equal(runtime.GOOS))
			Expect(metricsRows[0].osArch).To(Equal(runtime.GOARCH))
			Expect(metricsRows[0].nameArg).To(Equal("fake-name-arg"))
			Expect(metricsRows[0].command).To(Equal("fake-cmd-name"))
			Expect(metricsRows[0].commandStartTSMsec).ToNot(BeEmpty())
			Expect(metricsRows[0].commandEndTSMsec).ToNot(BeEmpty())
			Expect(metricsRows[0].pluginName).To(Equal("fake-plugin"))
			Expect(metricsRows[0].pluginVersion).To(Equal("v0.0.1"))
			Expect(metricsRows[0].flags).To(Equal(`{"v":"6","longflag":"lvalue"}`))
			Expect(metricsRows[0].target).To(Equal("kubernetes"))
			Expect(metricsRows[0].endpoint).To(Equal("fake-endpoint-hash"))
			Expect(metricsRows[0].isInternal).To(Equal(false))
			Expect(metricsRows[0].exitStatus).To(Equal(0))
			Expect(metricsRows[0].error).To(BeEmpty())

			//validate the GetRowCount()
			count, err := db.GetRowCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
		})
	})

})

const selectAllFromCLIOperationMetrics = "SELECT cli_version,os_name,os_arch,plugin_name,plugin_version,command,cli_id,command_start_ts,command_end_ts," +
	"target,name_arg,endpoint,flags,exit_status,is_internal,error FROM tanzu_cli_operations"

func getOperationMetrics(metricsDB *sqliteMetricsDB) ([]*cliOperationsRow, error) {
	db, err := sql.Open("sqlite", metricsDB.metricsDBFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open the DB from '%s' file", metricsDB.metricsDBFile)
	}
	defer db.Close()

	dbQuery := selectAllFromCLIOperationMetrics
	rows, err := db.Query(dbQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute the DB query : %v", dbQuery)
	}
	defer rows.Close()

	var metricsRows []*cliOperationsRow
	for rows.Next() {
		var row cliOperationsRow
		err = rows.Scan(&row.cliVersion, &row.osName, &row.osArch, &row.pluginName, &row.pluginVersion, &row.command, &row.cliID, &row.commandStartTSMsec, &row.commandEndTSMsec,
			&row.target, &row.nameArg, &row.endpoint, &row.flags, &row.exitStatus, &row.isInternal, &row.error)
		if err != nil {
			return nil, errors.New("failed to scan the metrics row")
		}
		metricsRows = append(metricsRows, &row)
	}
	return metricsRows, nil
}
