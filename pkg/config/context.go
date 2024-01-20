// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

// SyncContextsAndServers populate or sync contexts and servers
func SyncContextsAndServers() error {
	cfg, err := config.GetClientConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get client config")
	}

	config.PopulateContexts(cfg)

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
