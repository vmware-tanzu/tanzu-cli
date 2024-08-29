// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package centralconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
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
func TestGetCentralConfigEntry(t *testing.T) {
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
		cfgContent  string
		nofile      bool
		nopointer   bool
		key         string
		expected    interface{}
		expectError bool
	}{
		{
			name:        "Missing key",
			cfgContent:  "testKey: testValue",
			key:         "invalidKey",
			expected:    "", // To specify the type
			expectError: true,
		},
		{
			name:        "Empty key",
			cfgContent:  "testKey: testValue",
			key:         "",
			expected:    "",
			expectError: true,
		},
		{
			name:       "Empty value",
			cfgContent: "testKey: ",
			key:        "testKey",
			expected:   "",
		},
		{
			name: "Invalid yaml",
			cfgContent: `testKey: testValue
		- invalid`,
			key:         "testKey",
			expected:    "",
			expectError: true,
		},
		{
			name:        "No file for central config",
			nofile:      true,
			key:         "testKey",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Not passing a pointer",
			nopointer:   true,
			cfgContent:  "testKey: testValue",
			key:         "testKey",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty central config",
			cfgContent:  "",
			key:         "testKey",
			expected:    "",
			expectError: true,
		},
		{
			name:       "String value",
			cfgContent: "testKey: testValue",
			key:        "testKey",
			expected:   "testValue",
		},
		{
			name:       "String empty value",
			cfgContent: `testKey: ""`,
			key:        "testKey",
			expected:   "",
		},
		{
			name:       "Boolean true value",
			cfgContent: "testKey: true",
			key:        "testKey",
			expected:   true,
		},
		{
			name:       "Boolean TRUE value",
			cfgContent: "testKey: TRUE",
			key:        "testKey",
			expected:   true,
		},
		{
			name:       "Boolean FALSE value",
			cfgContent: "testKey: FALSE",
			key:        "testKey",
			expected:   false,
		},
		{
			name:       "Int value",
			cfgContent: "testKey: 1",
			key:        "testKey",
			expected:   1,
		},
		{
			name:       "Negative int value",
			cfgContent: "testKey: -1",
			key:        "testKey",
			expected:   -1,
		},
		{
			name:       "Float value",
			cfgContent: "testKey: 1.0",
			key:        "testKey",
			expected:   1.0,
		},
		{
			name:       "Negative float value",
			cfgContent: "testKey: -1.0",
			key:        "testKey",
			expected:   -1.0,
		},
		{
			name:       "Negative int value",
			cfgContent: "testKey: -1",
			key:        "testKey",
			expected:   -1,
		},
		{
			name:       "Float value",
			cfgContent: "testKey: 1.0",
			key:        "testKey",
			expected:   1.0,
		},
		{
			name:       "Negative float value",
			cfgContent: "testKey: -1.0",
			key:        "testKey",
			expected:   -1.0,
		},
		{
			name:       "Timestamp value",
			cfgContent: "testKey: " + timestampStr,
			key:        "testKey",
			expected:   timestamp,
		},
		{
			name: "String array value",
			cfgContent: `testKey:
- testValue1
- testValue2`,
			key:      "testKey",
			expected: []string{"testValue1", "testValue2"},
		},
		{
			name: "String map value",
			cfgContent: `testKey:
  testSubKey1: testValue1
  testSubKey2: testValue2`,
			key:      "testKey",
			expected: map[string]string{"testSubKey1": "testValue1", "testSubKey2": "testValue2"},
		},
		{
			name: "Any array value",
			cfgContent: `testKey:
- true
- false`,
			key:      "testKey",
			expected: []interface{}{true, false},
		},
		{
			name: "Any map value",
			cfgContent: `testKey:
  testSubKey1: true
  testSubKey2: testValue`,
			key:      "testKey",
			expected: map[string]interface{}{"testSubKey1": true, "testSubKey2": "testValue"},
		},
		{
			name: "A complex string",
			cfgContent: `testKey: |-
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
			cfgContent: `testKey:
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

	dir, err := os.MkdirTemp("", "test-central-config")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	common.DefaultCacheDir = dir

	reader := newCentralConfigReader("my_discovery")

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			path := reader.(*centralConfigYamlReader).configFile
			if tc.nofile {
				// Allows to test without a central config file
				err = os.Remove(path)
				assert.Nil(t, err)
			} else {
				// Write the central config test content to the file
				err = os.MkdirAll(filepath.Dir(path), 0755)
				assert.Nil(t, err)

				err = os.WriteFile(path, []byte(tc.cfgContent), 0644)
				assert.Nil(t, err)
			}

			var genericVar interface{}
			if tc.nopointer {
				var result string
				// Don't pass a pointer.  This should trigger an error
				err = reader.GetCentralConfigEntry(tc.key, result)
				genericVar = result
			} else {
				switch expectedType := reflect.TypeOf(tc.expected); expectedType {
				case stringType:
					var result string
					err = reader.GetCentralConfigEntry(tc.key, &result)
					genericVar = result
				case boolType:
					var result bool
					err = reader.GetCentralConfigEntry(tc.key, &result)
					genericVar = result
				case intType:
					var result int
					err = reader.GetCentralConfigEntry(tc.key, &result)
					genericVar = result
				case floatType:
					var result float64
					err = reader.GetCentralConfigEntry(tc.key, &result)
					genericVar = result
				case stringArrayType:
					var result []string
					err = reader.GetCentralConfigEntry(tc.key, &result)
					genericVar = result
				case stringMapType:
					var result map[string]string
					err = reader.GetCentralConfigEntry(tc.key, &result)
					genericVar = result
				case timeType:
					var result time.Time
					err = reader.GetCentralConfigEntry(tc.key, &result)
					genericVar = result
				case arrayType:
					var result []interface{}
					err = reader.GetCentralConfigEntry(tc.key, &result)
					genericVar = result
				case mapType:
					var result map[string]interface{}
					err = reader.GetCentralConfigEntry(tc.key, &result)
					genericVar = result
				case reflect.TypeOf(TestArtifact{}):
					result := TestArtifact{}
					err = reader.GetCentralConfigEntry(tc.key, &result)
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
}
