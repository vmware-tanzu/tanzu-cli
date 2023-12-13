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

	for _, c := range cfg.KnownContexts {
		err := config.SetContext(c, false)
		if err != nil {
			return errors.Wrap(err, "failed to set context")
		}
	}

	return nil
}
