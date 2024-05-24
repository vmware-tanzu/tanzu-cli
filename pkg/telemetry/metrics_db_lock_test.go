// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAcquireAndReleaseTanzuMetricDBLock(t *testing.T) {
	// Test acquiring the lock
	err := AcquireTanzuMetricDBLock()
	assert.NoError(t, err, "Expected no error while acquiring the lock")

	// Verify the lock is held
	assert.NotNil(t, cliMetricDBLock, "Expected the lock to be acquired")

	// Test releasing the lock
	assert.NotPanics(t, func() {
		ReleaseTanzuMetricDBLock()
	}, "Expected no panic while releasing the lock")

	// Verify the lock is released
	assert.Nil(t, cliMetricDBLock, "Expected the lock to be released")
}

func TestLockTimeout(t *testing.T) {
	// Acquire the lock for the first time
	err := AcquireTanzuMetricDBLock()
	assert.NoError(t, err, "Expected no error while acquiring the lock the first time")

	// Try acquiring the lock again, should time out
	err = AcquireTanzuMetricDBLock()
	assert.Error(t, err, "Expected a timeout error while trying to acquire the lock again")
	assert.ErrorContains(t, err, "timeout waiting for lock")

	// Release the initial lock
	ReleaseTanzuMetricDBLock()
}

func TestMultipleAcquireAndRelease(t *testing.T) {
	// Acquire and release the lock multiple times
	for i := 0; i < 3; i++ {
		err := AcquireTanzuMetricDBLock()
		assert.NoError(t, err, "Expected no error while acquiring the lock")

		assert.NotNil(t, cliMetricDBLock, "Expected the lock to be acquired")

		assert.NotPanics(t, func() {
			ReleaseTanzuMetricDBLock()
		}, "Expected no panic while releasing the lock")

		assert.Nil(t, cliMetricDBLock, "Expected the lock to be released")
	}
}

func TestParallelLocking(t *testing.T) {
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	successCount := int32(0)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			err := AcquireTanzuMetricDBLock()
			if err == nil {
				// sleep and hold lock for more than timeout period
				// so that all other go routines fail to acquire lock
				time.Sleep(2 * DefaultMetricsDBLockTimeout)
				defer ReleaseTanzuMetricDBLock()
				successCount++
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(1), successCount, "Expected only one goroutine to successfully acquire the lock")
}

func TestParallelLockingAndUnlocking(t *testing.T) {
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	successCount := int32(0)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			err := AcquireTanzuMetricDBLock()
			if err == nil {
				// sleep and hold lock for less than timeout period
				// so that all other go routines could acquire and release lock successfully
				time.Sleep(100 * time.Millisecond)
				defer ReleaseTanzuMetricDBLock()
				successCount++
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(10), successCount, "Expected all the goroutine to successfully acquire the lock")
}
