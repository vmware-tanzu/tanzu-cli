// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

// DiscoveryBackend is the interface to extract plugin information from
// a backend used for plugin discovery.
type DiscoveryBackend interface {
	GetAllPlugins() ([]*Discovered, error)
}
