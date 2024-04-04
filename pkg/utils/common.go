// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package utils contains utility functions
package utils

import (
	"strings"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// ContainsString checks the string contains in string array
func ContainsString(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// GenerateKey is a utility function that takes an arbitrary number of string arguments,
// concatenates them with a colon (":") separator, and returns the resulting string.
// This function is typically used to create a unique key or identifier from multiple string parts.
//
// Parameters:
// parts: A variadic parameter that accepts an arbitrary number of strings.
//
// Returns:
// A single string that is the result of concatenating all input strings with a colon (":") separator.
func GenerateKey(parts ...string) string {
	return strings.Join(parts, ":")
}

// EnsureMutualExclusiveCurrentContexts ensures mutual exclusive behavior among k8s and tanzu current contexts,
// i.e, if both k8s and tanzu current contexts types are set (a case where plugin using old plugin-runtime API
// can set k8s current context though tanzu current context is set by CLI or plugin with latest plugin-runtime
// in config file) it would remove the tanzu current context to maintain backward compatibility
func EnsureMutualExclusiveCurrentContexts() error {
	ccmap, err := config.GetAllActiveContextsMap()
	if err != nil {
		return err
	}
	if ccmap[configtypes.ContextTypeK8s] != nil && ccmap[configtypes.ContextTypeTanzu] != nil {
		return config.RemoveActiveContext(configtypes.ContextTypeTanzu)
	}
	return nil
}

// PanicOnErr calls 'panic' if 'err' is non-nil.
func PanicOnErr(err error) {
	if err == nil {
		return
	}

	panic(err)
}

// ParsePluginID parses the plugin id and returns (name, target, version) strings
func ParsePluginID(pluginID string) (string, string, string) {
	var name, target, version string
	parts := strings.Split(pluginID, ":")
	if len(parts) > 1 {
		version = parts[1]
	}
	parts = strings.Split(parts[0], "@")
	name = parts[0]
	if len(parts) > 1 {
		target = parts[1]
	}
	return name, target, version
}
