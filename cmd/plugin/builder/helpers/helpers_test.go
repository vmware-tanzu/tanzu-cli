// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNonEmptyValidatePluginBinary(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "temp")
	assert.Nil(t, err)

	defer func(name string) {
		err := os.Remove(name)
		assert.Nil(t, err)
	}(tmpFile.Name()) // clean up

	// Add some data to the file
	data := []byte("Some data")
	_, err = tmpFile.Write(data)
	assert.Nil(t, err)

	err = tmpFile.Close()
	assert.Nil(t, err)

	// Make the file executable
	err = os.Chmod(tmpFile.Name(), 0755)
	assert.Nil(t, err)

	// Test a non-empty plugin binary file
	valid, err := ValidatePluginBinary(tmpFile.Name())
	assert.Nil(t, err)
	assert.Equal(t, true, valid)
}

func TestEmptyValidatePluginBinary(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "temp")
	assert.Nil(t, err)

	defer func(name string) {
		err := os.Remove(name)
		assert.Nil(t, err)
	}(tmpFile.Name()) // clean up

	err = tmpFile.Close()
	assert.Nil(t, err)
	// Make the file executable
	err = os.Chmod(tmpFile.Name(), 0755)
	assert.Nil(t, err)

	// Test an empty plugin binary file
	valid, err := ValidatePluginBinary(tmpFile.Name())
	assert.Nil(t, err)
	assert.Equal(t, false, valid)
}
