// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

func addEdition(c *configtypes.ClientConfig, edition configtypes.EditionSelector) {
	if c.ClientOptions == nil {
		c.ClientOptions = &configtypes.ClientOptions{}
	}
	if c.ClientOptions.CLI == nil {
		c.ClientOptions.CLI = &configtypes.CLIOptions{}
	}
	c.ClientOptions.CLI.Edition = edition //nolint:staticcheck
}

// addDefaultEditionIfMissing returns true if the default edition was added to the configuration (because there was no edition)
func addDefaultEditionIfMissing(config *configtypes.ClientConfig) bool {
	if config.ClientOptions == nil || config.ClientOptions.CLI == nil || config.ClientOptions.CLI.Edition == "" { //nolint:staticcheck
		addEdition(config, DefaultEdition)
		return true
	}
	return false
}
