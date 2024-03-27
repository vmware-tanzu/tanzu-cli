// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package centralconfig implements an interface to deal with the central configuration.
package centralconfig

import (
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

type centralConfigYamlReader struct {
	// configFile is the path to the central config file.
	configFile string
}

// Make sure centralConfigYamlReader implements CentralConfig
var _ CentralConfig = &centralConfigYamlReader{}

// parseConfigFile reads the central config file and returns the parsed yaml content.
// If the file does not exist, it does not return an error because some central repositories
// may choose not to have a central config file.
func (c *centralConfigYamlReader) parseConfigFile() (map[string]interface{}, error) {
	// Check if the central config file exists.
	if _, err := os.Stat(c.configFile); os.IsNotExist(err) {
		// The central config file is optional, don't return an error if it does not exist.
		return nil, nil
	}

	bytes, err := os.ReadFile(c.configFile)
	if err != nil {
		return nil, err
	}

	var content map[string]interface{}
	err = yaml.Unmarshal(bytes, &content)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (c *centralConfigYamlReader) GetCentralConfigEntry(key string, out interface{}) error {
	values, err := c.parseConfigFile()
	if err != nil {
		return err
	}

	ok, err := extractValue(out, values, key)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("key %s not found in central config", key)
	}

	return nil
}

func extractValue(out interface{}, values map[string]interface{}, key string) (bool, error) {
	res, ok := values[key]
	if !ok {
		return false, nil
	}

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return false, fmt.Errorf("out must be a pointer to a value")
	}
	yamlBytes, err := yaml.Marshal(res)
	if err == nil {
		err = yaml.Unmarshal(yamlBytes, out)
	}

	return ok, err
}
