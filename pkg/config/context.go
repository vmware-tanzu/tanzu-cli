// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/vmware-tanzu/tanzu-cli/pkg/datastore"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

type contextAndServerSha struct {
	ContextsSha        string
	CurrentContextsSha string
	ServersSha         string
	CurrentServerSha   string
}

const lastContextServerShasKey = "lastContextServerShas"

// SyncContextsAndServers populate or sync contexts and servers
func SyncContextsAndServers() error {
	cfg, err := config.GetClientConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get client config")
	}

	if !shouldSyncContextsAndServers(cfg) {
		return nil
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

	// After the sync, save the shas using the updated config
	// so that next time, we avoid synching if the shas are the same
	cfg, err = config.GetClientConfig()
	if err == nil {
		_ = saveContextAndServerShas(cfg)
	}
	return nil
}

func shouldSyncContextsAndServers(cfg *types.ClientConfig) bool {
	var prevShas contextAndServerSha
	err := datastore.GetDataStoreValue(lastContextServerShasKey, &prevShas)
	if err != nil {
		return true
	}

	newShas := computeShasForSync(cfg)
	if newShas == nil || prevShas != *newShas {
		return true
	}
	return false
}

//nolint:staticcheck // Deprecated
func computeShasForSync(cfg *types.ClientConfig) *contextAndServerSha {
	// tanzu contexts are not synced so we don't include them in the sha
	// because even if they change, we don't need to sync them
	var nonTanzuContexts []*types.Context
	for i := range cfg.KnownContexts {
		if cfg.KnownContexts[i].ContextType != types.ContextTypeTanzu {
			nonTanzuContexts = append(nonTanzuContexts, cfg.KnownContexts[i])
		}
	}
	ctxBytes, err := yaml.Marshal(nonTanzuContexts)
	if err != nil {
		return nil
	}

	// tanzu servers are not synced so we don't include them in the sha
	// because even if they change, we don't need to sync them.
	// Note that tanzu servers should normally not be in the known servers list
	// but could happen for some older plugins or CLI versions which used to sync them.
	var nonTanzuServers []*types.Server
	for i := range cfg.KnownServers {
		if cfg.KnownServers[i].Type != types.ServerType(types.ContextTypeTanzu) {
			nonTanzuServers = append(nonTanzuServers, cfg.KnownServers[i])
		}
	}
	serverBytes, err := yaml.Marshal(nonTanzuServers)
	if err != nil {
		return nil
	}

	return &contextAndServerSha{
		ContextsSha: hashString(string(ctxBytes)),
		// Only the k8s current context is synched
		CurrentContextsSha: hashString(cfg.CurrentContext[types.ContextTypeK8s]),
		ServersSha:         hashString(string(serverBytes)),
		CurrentServerSha:   hashString(cfg.CurrentServer),
	}
}

func saveContextAndServerShas(cfg *types.ClientConfig) error {
	newShas := computeShasForSync(cfg)
	return datastore.SetDataStoreValue(lastContextServerShasKey, newShas)
}

func hashString(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
