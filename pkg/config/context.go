// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// SyncContextsAndServers populate or sync contexts and servers
func SyncContextsAndServers() error {
	cfg, err := config.GetClientConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get client config")
	}

	hasChanged := config.PopulateContexts(cfg)
	needsSync := doServersNeedUpdate(cfg)
	if !hasChanged && !needsSync {
		return nil
	}

	// Now write the context to the configuration file.  This will also create any missing server for its corresponding context
	for _, c := range cfg.KnownContexts {
		err := config.SetContext(c, false)
		if err != nil {
			return errors.Wrap(err, "failed to set context")
		}
	}

	// Now write the active contexts to the configuration file. This will also create any missing active server for its corresponding context
	activeContexts, _ := cfg.GetAllActiveContextsList()
	for _, c := range activeContexts {
		err := config.SetActiveContext(c)
		if err != nil {
			return errors.Wrap(err, "failed to set active context")
		}
	}
	return nil
}

func doServersNeedUpdate(cfg *types.ClientConfig) bool {
	if cfg == nil {
		return false
	}

	for _, c := range cfg.KnownContexts {
		if c.ContextType == types.ContextTypeTanzu || cfg.HasServer(c.Name) { //nolint:staticcheck // Deprecated
			// context of type "tanzu" don't get synched
			// or context already present in servers; skip
			continue
		}
		// Found a context that is not in the servers.  We need to update
		// the servers section
		return true
	}

	return false
}
