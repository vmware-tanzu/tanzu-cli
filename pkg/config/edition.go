// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

//nolint:staticcheck // Deprecated
func addEdition(c *configtypes.ClientConfig, edition configtypes.EditionSelector) {
	if c.ClientOptions == nil {
		c.ClientOptions = &configtypes.ClientOptions{}
	}
	if c.ClientOptions.CLI == nil { //nolint:staticcheck // Deprecated
		c.ClientOptions.CLI = &configtypes.CLIOptions{} //nolint:staticcheck // Deprecated
	}
	c.ClientOptions.CLI.Edition = edition //nolint:staticcheck
}

// addDefaultEditionIfMissing returns true if the default edition was added to the configuration (because there was no edition)
//
// Deprecated: This method is deprecated
func addDefaultEditionIfMissing(config *configtypes.ClientConfig) bool {
	if config.ClientOptions == nil || config.ClientOptions.CLI == nil || config.ClientOptions.CLI.Edition == "" {
		addEdition(config, DefaultEdition)
		return true
	}
	return false
}
