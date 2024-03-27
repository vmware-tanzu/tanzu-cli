// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package datastore

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	stringType      = reflect.TypeOf("")
	boolType        = reflect.TypeOf(true)
	intType         = reflect.TypeOf(1)
	floatType       = reflect.TypeOf(1.0)
	stringArrayType = reflect.TypeOf([]string{})
	stringMapType   = reflect.TypeOf(map[string]string{})
	arrayType       = reflect.TypeOf([]interface{}{})
	mapType         = reflect.TypeOf(map[string]interface{}{})
	timeType        = reflect.TypeOf(time.Time{})
)

//nolint:gocyclo
func TestGetDataStoreValue(t *testing.T) {
	// Create a timestamp in the RFC3339 format
	timestampStr := time.Now().Format(time.RFC3339)
	// Now convert it back to a time.Time object so the two can be compared
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	assert.Nil(t, err)

	type TestArtifact struct {
		Image string
		OS    string
		Arch  string
	}

	tcs := []struct {
		name        string
		dsContent   string
		nodir       bool
		nofile      bool
		nopointer   bool
		key         string
		expected    interface{}
		expectError bool
	}{
		{
			name:     "No directory for data store",
			nodir:    true,
			key:      "testKey",
			expected: "", // To specify the type
		},
		{
			name:     "No file for data store",
			nofile:   true,
			key:      "testKey",
			expected: "",
		},
		{
			name:        "Not passing a pointer",
			nopointer:   true,
			dsContent:   "testKey: testValue",
			key:         "testKey",
			expected:    "",
			expectError: true,
		},
		{
			name:      "Empty data store",
			dsContent: "",
			key:       "testKey",
			expected:  "",
		},
		{
			name:        "Missing key",
			dsContent:   "testKey: testValue",
			key:         "invalidKey",
			expected:    "", // To specify the type
			expectError: true,
		},
		{
			name:        "Empty key",
			dsContent:   "testKey: testValue",
			key:         "",
			expected:    "",
			expectError: true,
		},
		{
			name:      "Empty value",
			dsContent: "testKey: ",
			key:       "testKey",
			expected:  "",
		},
		{
			name: "Invalid yaml",
			dsContent: `testKey: testValue
		- invalid`,
			key:         "testKey",
			expected:    "",
			expectError: true,
		},
		{
			name:      "String value",
			dsContent: "testKey: testValue",
			key:       "testKey",
			expected:  "testValue",
		},
		{
			name:      "String empty value",
			dsContent: `testKey: ""`,
			key:       "testKey",
			expected:  "",
		},
		{
			name:      "Boolean true value",
			dsContent: "testKey: true",
			key:       "testKey",
			expected:  true,
		},
		{
			name:      "Boolean TRUE value",
			dsContent: "testKey: TRUE",
			key:       "testKey",
			expected:  true,
		},
		{
			name:      "Boolean FALSE value",
			dsContent: "testKey: FALSE",
			key:       "testKey",
			expected:  false,
		},
		{
			name:      "Int value",
			dsContent: "testKey: 1",
			key:       "testKey",
			expected:  1,
		},
		{
			name:      "Negative int value",
			dsContent: "testKey: -1",
			key:       "testKey",
			expected:  -1,
		},
		{
			name:      "Float value",
			dsContent: "testKey: 1.0",
			key:       "testKey",
			expected:  1.0,
		},
		{
			name:      "Negative float value",
			dsContent: "testKey: -1.0",
			key:       "testKey",
			expected:  -1.0,
		},
		{
			name:      "Negative int value",
			dsContent: "testKey: -1",
			key:       "testKey",
			expected:  -1,
		},
		{
			name:      "Float value",
			dsContent: "testKey: 1.0",
			key:       "testKey",
			expected:  1.0,
		},
		{
			name:      "Negative float value",
			dsContent: "testKey: -1.0",
			key:       "testKey",
			expected:  -1.0,
		},
		{
			name:      "Timestamp value",
			dsContent: "testKey: " + timestampStr,
			key:       "testKey",
			expected:  timestamp,
		},
		{
			name: "String array value",
			dsContent: `testKey:
- testValue1
- testValue2`,
			key:      "testKey",
			expected: []string{"testValue1", "testValue2"},
		},
		{
			name: "String map value",
			dsContent: `testKey:
  testSubKey1: testValue1
  testSubKey2: testValue2`,
			key:      "testKey",
			expected: map[string]string{"testSubKey1": "testValue1", "testSubKey2": "testValue2"},
		},
		{
			name: "Any array value",
			dsContent: `testKey:
- true
- false`,
			key:      "testKey",
			expected: []interface{}{true, false},
		},
		{
			name: "Any map value",
			dsContent: `testKey:
  testSubKey1: true
  testSubKey2: testValue`,
			key:      "testKey",
			expected: map[string]interface{}{"testSubKey1": true, "testSubKey2": "testValue"},
		},
		{
			name: "A complex string",
			dsContent: `testKey: |-
  {
    "testSubKey": [
      "testValue1",
      "testValue1"
    ]
  }`,
			key: "testKey",
			expected: `{
  "testSubKey": [
    "testValue1",
    "testValue1"
  ]
}`,
		},
		{
			name: "Any custom type",
			dsContent: `testKey:
  image: "fake.test.image.io/fake-image"
  os: darwin
  arch: amd64`,
			key: "testKey",
			expected: TestArtifact{
				Image: "fake.test.image.io/fake-image",
				OS:    "darwin",
				Arch:  "amd64",
			},
		},
	}

	tmpDir, err := os.MkdirTemp("", "data_store_test")
	assert.Nil(t, err)
	assert.NotNil(t, tmpDir)
	defer os.RemoveAll(tmpDir)

	tmpDSFile, err := os.CreateTemp(tmpDir, ".data-store.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, tmpDSFile)

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.nodir {
				// Set the environment variable to a nonexistent directory and file
				nonExistentDir := tmpDir + "_nonexistentdir"
				os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", filepath.Join(nonExistentDir, ".data-store.yaml"))
				defer os.RemoveAll(nonExistentDir)
			} else {
				if tc.nofile {
					// Set the environment variable to a nonexistent file
					os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDSFile.Name()+"_nonexistent")
				} else {
					// Write the data store test content to the file
					err = os.WriteFile(tmpDSFile.Name(), []byte(tc.dsContent), 0644)
					assert.Nil(t, err)
					os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDSFile.Name())
				}
			}

			var genericVar interface{}
			if tc.nopointer {
				var result string
				// Don't pass a pointer.  This should trigger an error
				err = GetDataStoreValue(tc.key, result)
				genericVar = result
			} else {
				switch expectedType := reflect.TypeOf(tc.expected); expectedType {
				case stringType:
					var result string
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				case boolType:
					var result bool
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				case intType:
					var result int
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				case floatType:
					var result float64
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				case stringArrayType:
					var result []string
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				case stringMapType:
					var result map[string]string
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				case timeType:
					var result time.Time
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				case arrayType:
					var result []interface{}
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				case mapType:
					var result map[string]interface{}
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				case reflect.TypeOf(TestArtifact{}):
					result := TestArtifact{}
					err = GetDataStoreValue(tc.key, &result)
					genericVar = result
				default:
					t.Fatalf("unsupported type: %v", expectedType)
				}
			}

			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			if tc.expected == nil {
				assert.Nil(t, genericVar)
			} else {
				assert.Equal(t, tc.expected, genericVar)
			}
		})
	}
	os.Unsetenv("TEST_CUSTOM_DATA_STORE_FILE")
}

func TestSetDataStoreValue(t *testing.T) {
	// Create a timestamp in the RFC3339 format
	timestampStr := time.Now().Format(time.RFC3339)
	// Now convert it back to a time.Time object so the two can be compared
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	assert.Nil(t, err)

	tcs := []struct {
		name   string
		nofile bool
		nodir  bool
		key    string
		value  interface{}
		// expected is the expected value to be returned by GetDataStoreValue.
		// Normally, it should be the same as value, but in some cases, it can be different.
		expected interface{}
	}{
		{
			name:  "No directory for data store",
			nodir: true,
			key:   "testKey",
			value: "testValue",
		},
		{
			name:   "No file for data store",
			nofile: true,
			key:    "testKey",
			value:  "testValue",
		},
		{
			name:  "String value",
			key:   "testKey",
			value: "testValue",
		},
		{
			name:  "Boolean true value",
			key:   "testKey",
			value: true,
		},
		{
			name:  "Boolean TRUE value",
			key:   "testKey",
			value: true,
		},
		{
			name:  "Boolean FALSE value",
			key:   "testKey",
			value: false,
		},
		{
			name:  "Boolean int value",
			key:   "testKey",
			value: 1,
		},
		{
			name:  "Timestamp value",
			key:   "testKey",
			value: timestamp,
		},
		{
			name:     "Complex map value",
			key:      "testKey",
			value:    map[string]string{"testSubKey": "testValue"},
			expected: map[string]interface{}{"testSubKey": "testValue"},
		},
		{
			name:     "More map of array value",
			key:      "testKey",
			value:    map[string][]string{"testSubKey": {"testValue1", "testValue2"}},
			expected: map[string]interface{}{"testSubKey": []interface{}{"testValue1", "testValue2"}},
		},
		{
			name:  "Empty key",
			key:   "",
			value: "testValue",
		},
		{
			name:  "Empty value",
			key:   "testKey",
			value: "",
		},
		{
			name:  "Both empty",
			key:   "",
			value: "",
		},
	}

	tmpDir, err := os.MkdirTemp("", "data_store_test")
	assert.Nil(t, err)
	assert.NotNil(t, tmpDir)
	defer os.RemoveAll(tmpDir)

	tmpDSFile, err := os.CreateTemp(tmpDir, ".data-store.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, tmpDSFile)

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.nodir {
				// Set the environment variable to a nonexistent directory and file
				nonExistentDir := tmpDir + "_nonexistentdir2"
				os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", filepath.Join(nonExistentDir, ".data-store.yaml"))
				defer os.RemoveAll(nonExistentDir)
			} else {
				if tc.nofile {
					// Set the environment variable to a nonexistent file
					os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDSFile.Name()+"_nonexistent2")
				} else {
					// Set the environment variable to the file we already created
					os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDSFile.Name())
				}
			}

			if tc.expected == nil {
				tc.expected = tc.value
			}

			err = SetDataStoreValue(tc.key, tc.value)
			assert.Nil(t, err)

			var value interface{}
			err := GetDataStoreValue(tc.key, &value)
			assert.Nil(t, err)
			assert.Equal(t, tc.expected, value)
		})
	}
}

func TestDeleteDataStoreValue(t *testing.T) {
	tcs := []struct {
		name        string
		dsContent   string
		nofile      bool
		nodir       bool
		key         string
		expectError bool
	}{
		{
			name:        "No directory for data store",
			nodir:       true,
			key:         "testKey",
			expectError: true,
		},
		{
			name:        "No file for data store",
			nofile:      true,
			key:         "testKey",
			expectError: true,
		},
		{
			name:        "Empty data store",
			dsContent:   "",
			key:         "testKey",
			expectError: true,
		},
		{
			name:      "String value",
			dsContent: "testKey: testValue",
			key:       "testKey",
		},
		{
			name:      "Boolean true value",
			dsContent: "testKey: true",
			key:       "testKey",
		},
		{
			name:      "Boolean TRUE value",
			dsContent: "testKey: TRUE",
			key:       "testKey",
		},
		{
			name:      "Boolean FALSE value",
			dsContent: "testKey: FALSE",
			key:       "testKey",
		},
		{
			name:      "Boolean int value",
			dsContent: "testKey: 1",
			key:       "testKey",
		},
		{
			name:      "Timestamp value",
			dsContent: "testKey: 1",
			key:       "testKey",
		},
		{
			name: "Complex map value",
			dsContent: `testKey:
  testSubKey: testValue`,
			key: "testKey",
		},
		{
			name: "More map of array value",
			dsContent: `testKey:
  testSubKey:
  - testValue1
  - testValue2`,
			key: "testKey",
		},
		{
			name: "Missing key",
			dsContent: `testKey:
  testSubKey:
  - testValue1
  - testValue2`,
			key:         "invalidKey",
			expectError: true,
		},
		{
			name: "Empty key",
			dsContent: `testKey:
  testSubKey:
  - testValue1
  - testValue2`,
			key:         "",
			expectError: true,
		},
		{
			name:      "Empty value",
			dsContent: "testKey: ",
			key:       "testKey",
		},
	}

	tmpDir, err := os.MkdirTemp("", "data_store_test")
	assert.Nil(t, err)
	assert.NotNil(t, tmpDir)
	defer os.RemoveAll(tmpDir)

	tmpDSFile, err := os.CreateTemp(tmpDir, ".data-store.yaml")
	assert.Nil(t, err)
	assert.NotNil(t, tmpDSFile)

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.nodir {
				// Set the environment variable to a nonexistent directory and file
				nonExistentDir := tmpDir + "_nonexistentdir3"
				os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", filepath.Join(nonExistentDir, ".data-store.yaml"))
				defer os.RemoveAll(nonExistentDir)
			} else {
				if tc.nofile {
					// Set the environment variable to a nonexistent file
					os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDSFile.Name()+"_nonexistent3")
				} else {
					// Write the data store test content to the file
					err = os.WriteFile(tmpDSFile.Name(), []byte(tc.dsContent), 0644)
					assert.Nil(t, err)
					os.Setenv("TEST_CUSTOM_DATA_STORE_FILE", tmpDSFile.Name())
				}
			}

			err := DeleteDataStoreValue(tc.key)
			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			// Make sure the key is deleted
			var value interface{}
			err = GetDataStoreValue(tc.key, &value)
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "not found in the data store")
		})
	}
	os.Unsetenv("TEST_CUSTOM_DATA_STORE_FILE")
}

func TestGetDataStorePath(t *testing.T) {
	// Verify that the data store path is in the .config directory (not the .cache directory)
	path := getDataStorePath()
	assert.Contains(t, path, ".config")
}
