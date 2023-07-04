// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"os"
	"strconv"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

var showTelemetryLog = false

func init() {
	showLog := os.Getenv(constants.ShowTelemetryConsoleLogs)
	if isTrue, _ := strconv.ParseBool(showLog); isTrue {
		showTelemetryLog = true
	}
}

func LogError(err error, msg string, kvs ...interface{}) {
	if showTelemetryLog {
		log.Error(err, msg, kvs...)
	}
}
func LogWarning(msg string, kvs ...interface{}) {
	if showTelemetryLog {
		log.Warning(msg, kvs...)
	}
}
