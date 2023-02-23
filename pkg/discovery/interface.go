// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package discovery is implements discovery interface for plugin discovery
// Discovery is the interface to fetch the list of available plugins, their
// supported versions and how to download them either stand-alone or scoped to a server.
// A separate interface for discovery helps to decouple discovery (which is usually
// tied to a server or user identity) from distribution (which can be shared).
package discovery

import (
	"errors"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// Discovery is the interface to fetch the list of available plugins
type Discovery interface {
	// Name of the repository.
	Name() string

	// List available plugins.
	List() ([]Discovered, error)

	// Type returns type of discovery.
	Type() string
}

// PluginDiscoveryCriteria provides criteria to look for plugins
// in a discovery.
type PluginDiscoveryCriteria struct {
	// Name is the name of the plugin
	Name string
	// Target is the target of the plugin
	Target configtypes.Target
	// Version is the version for the plugin
	Version string
	// OS of the plugin binary in `GOOS` format.
	OS string
	// Arch of the plugin binary in `GOARCH` format.
	Arch string
}

// CreateDiscoveryFromV1alpha1 creates discovery interface from v1alpha1 API
func CreateDiscoveryFromV1alpha1(pd configtypes.PluginDiscovery, criteria *PluginDiscoveryCriteria) (Discovery, error) {
	switch {
	case pd.OCI != nil:
		// Only the OCI Discovery currently supports a criteria
		return NewOCIDiscovery(pd.OCI.Name, pd.OCI.Image, criteria), nil
	case pd.Local != nil:
		return NewLocalDiscovery(pd.Local.Name, pd.Local.Path), nil
	case pd.Kubernetes != nil:
		return NewKubernetesDiscovery(pd.Kubernetes.Name, pd.Kubernetes.Path, pd.Kubernetes.Context), nil
	case pd.REST != nil:
		return NewRESTDiscovery(pd.REST.Name, pd.REST.Endpoint, pd.REST.BasePath), nil
	}
	return nil, errors.New("unknown plugin discovery source")
}
