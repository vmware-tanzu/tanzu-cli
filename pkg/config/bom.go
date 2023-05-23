// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

// Deprecated: This configuration variables are defined to support older plugins that are relying on
// this configuration to be set in the tanzu configuration file.
// This is pointing to the production registry to make sure existing plugins continue to work with
// newer version of the Tanzu CLI
const (
	tkgDefaultImageRepo              = "projects.registry.vmware.com/tkg"
	tkgDefaultCompatibilityImagePath = "tkg-compatibility"
)

// Deprecated: This method is deprecated
func addCompatibilityFile(c *configtypes.ClientConfig, compatibilityFilePath string) {
	if c.ClientOptions == nil {
		c.ClientOptions = &configtypes.ClientOptions{}
	}
	if c.ClientOptions.CLI == nil {
		c.ClientOptions.CLI = &configtypes.CLIOptions{}
	}
	// CompatibilityFilePath has been deprecated and will be removed from future version
	c.ClientOptions.CLI.CompatibilityFilePath = compatibilityFilePath
}

// Deprecated: This method is deprecated
func addBomRepo(c *configtypes.ClientConfig, repo string) {
	if c.ClientOptions == nil {
		c.ClientOptions = &configtypes.ClientOptions{}
	}
	if c.ClientOptions.CLI == nil {
		c.ClientOptions.CLI = &configtypes.CLIOptions{}
	}
	// BOMRepo has been deprecated and will be removed from future version
	c.ClientOptions.CLI.BOMRepo = repo
}

// AddCompatibilityFileIfMissing adds the compatibility file to the client configuration to ensure it can be downloaded
//
// Deprecated: This method is deprecated
func AddCompatibilityFileIfMissing(config *configtypes.ClientConfig) bool {
	// CompatibilityFilePath has been deprecated and will be removed from future version
	if config.ClientOptions == nil || config.ClientOptions.CLI == nil || config.ClientOptions.CLI.CompatibilityFilePath == "" {
		addCompatibilityFile(config, tkgDefaultCompatibilityImagePath)
		return true
	}
	return false
}

// AddBomRepoIfMissing adds the bomRepository to the client configuration if it is not already present
//
// Deprecated: This method is deprecated
func AddBomRepoIfMissing(config *configtypes.ClientConfig) bool {
	// BOMRepo has been deprecated and will be removed from future version
	if config.ClientOptions == nil || config.ClientOptions.CLI == nil || config.ClientOptions.CLI.BOMRepo == "" {
		addBomRepo(config, tkgDefaultImageRepo)
		return true
	}
	return false
}
