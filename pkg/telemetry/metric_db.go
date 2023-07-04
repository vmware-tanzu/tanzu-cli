// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

type MetricsDB interface {
	// CreateSchema creates table schemas to the provided database.
	// returns error if table creation fails for any reason
	CreateSchema() error

	// SaveOperationMetric inserts CLI operation metrics collected into database
	SaveOperationMetric(*OperationMetricsPayload) error
}
