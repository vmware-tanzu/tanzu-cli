// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexflint/go-filemutex"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
)

const (
	LocalTanzuCLIMetricsDBFileLock = ".cli_metrics_db.lock"
	// DefaultMetricsDBLockTimeout is the default time waiting on the filelock
	DefaultMetricsDBLockTimeout = 3 * time.Second
)

var cliMetricDBLockFile string

// cliMetricDBLock used as a static lock variable that stores fslock
// This is used for interprocess locking of the tanzu cli metrics DB file
var cliMetricDBLock *filemutex.FileMutex

// cliMetricDBMutex is used to handle the locking behavior between concurrent calls
// within the existing process trying to acquire the lock
var cliMetricDBMutex sync.Mutex

// AcquireTanzuMetricDBLock tries to acquire lock to update tanzu cli metrics DB file with timeout
func AcquireTanzuMetricDBLock() error {
	var err error

	if cliMetricDBLockFile == "" {
		cliMetricDBLockFile = filepath.Join(common.DefaultCLITelemetryDir, LocalTanzuCLIMetricsDBFileLock)
	}

	// using fslock to handle interprocess locking
	lock, err := getFileLockWithTimeout(cliMetricDBLockFile, DefaultMetricsDBLockTimeout)
	if err != nil {
		return fmt.Errorf("cannot acquire lock for Tanzu CLI metrics DB, reason: %v", err)
	}

	// Lock the mutex to prevent concurrent calls to acquire and configure the cliMetricDBLock
	cliMetricDBMutex.Lock()
	cliMetricDBLock = lock
	return nil
}

// ReleaseTanzuMetricDBLock releases the lock if the cliMetricDBLock was acquired
func ReleaseTanzuMetricDBLock() {
	if cliMetricDBLock == nil {
		return
	}
	if errUnlock := cliMetricDBLock.Close(); errUnlock != nil {
		panic(fmt.Sprintf("cannot release lock for Tanzu CLI metrics DB, reason: %v", errUnlock))
	}

	cliMetricDBLock = nil
	// Unlock the mutex to allow other concurrent calls to acquire and configure the cliMetricDBLock
	cliMetricDBMutex.Unlock()
}

// getFileLockWithTimeout returns a file lock with timeout
func getFileLockWithTimeout(lockPath string, lockDuration time.Duration) (*filemutex.FileMutex, error) {
	dir := filepath.Dir(lockPath)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
	}

	flock, err := filemutex.New(lockPath)
	if err != nil {
		return nil, err
	}

	result := make(chan error)
	cancel := make(chan struct{})
	go func() {
		err := flock.Lock()
		select {
		case <-cancel:
			// Timed out, cleanup if necessary.
			_ = flock.Close()
		case result <- err:
		}
	}()

	select {
	case err := <-result:
		return flock, err
	case <-time.After(lockDuration):
		close(cancel)
		return flock, fmt.Errorf("timeout waiting for lock")
	}
}
