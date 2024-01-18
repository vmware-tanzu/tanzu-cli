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
	"time"

	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var defaultTimeout = 5 * time.Second

// Discovery is the interface to fetch the list of available plugins
type Discovery interface {
	// Name of the repository.
	Name() string

	// List available plugins.
	List() ([]Discovered, error)

	// Type returns type of discovery.
	Type() string
}

type GroupDiscovery interface {
	// Name of the discovery
	Name() string

	// GetGroups returns the plugin groups defined in the discovery
	GetGroups() ([]*plugininventory.PluginGroup, error)
}

// DiscoveryOpts used to customize the plugin discovery process or mechanism
type DiscoveryOpts struct {
	UseLocalCacheOnly       bool // UseLocalCacheOnly used to pull the plugin data from the cache
	ForceRefresh            bool // ForceRefresh used to force a refresh of the plugin data
	PluginDiscoveryCriteria *PluginDiscoveryCriteria
	GroupDiscoveryCriteria  *GroupDiscoveryCriteria
}

type DiscoveryOptions func(options *DiscoveryOpts)

// WithUseLocalCacheOnly used to get the plugin inventory data without first refreshing the cache
// even if the cache's TTL has expired
func WithUseLocalCacheOnly() DiscoveryOptions {
	return func(o *DiscoveryOpts) {
		o.UseLocalCacheOnly = true
	}
}

// WithForceRefresh used to force a refresh of the plugin inventory data
// even when the cache's TTL has not expired
func WithForceRefresh() DiscoveryOptions {
	return func(o *DiscoveryOpts) {
		o.ForceRefresh = true
	}
}

// WithPluginDiscoveryCriteria used to specify the plugin discovery criteria
func WithPluginDiscoveryCriteria(criteria *PluginDiscoveryCriteria) DiscoveryOptions {
	return func(o *DiscoveryOpts) {
		o.PluginDiscoveryCriteria = criteria
	}
}

// WithGroupDiscoveryCriteria used to specify the group discovery criteria
func WithGroupDiscoveryCriteria(criteria *GroupDiscoveryCriteria) DiscoveryOptions {
	return func(o *DiscoveryOpts) {
		o.GroupDiscoveryCriteria = criteria
	}
}

func NewDiscoveryOpts() *DiscoveryOpts {
	return &DiscoveryOpts{}
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

// GroupDiscoveryCriteria provides criteria to look for
// plugin groups in a discovery.
type GroupDiscoveryCriteria struct {
	// Vendor of the group
	Vendor string
	// Publisher of the group
	Publisher string
	// Name of the group
	Name string
	// Version is the version for the group
	Version string
}

// CreateDiscoveryFromV1alpha1 creates discovery interface from v1alpha1 API
func CreateDiscoveryFromV1alpha1(pd configtypes.PluginDiscovery, options ...DiscoveryOptions) (Discovery, error) {
	switch {
	case pd.OCI != nil:
		// Only the OCI Discovery currently supports a criteria
		return NewOCIDiscovery(pd.OCI.Name, pd.OCI.Image, options...), nil
	case pd.Local != nil:
		return NewLocalDiscovery(pd.Local.Name, pd.Local.Path), nil
	case pd.Kubernetes != nil:
		return NewKubernetesDiscovery(pd.Kubernetes.Name, pd.Kubernetes.Path, pd.Kubernetes.Context, pd.Kubernetes.KubeConfigBytes), nil
	case pd.REST != nil:
		return NewRESTDiscovery(pd.REST.Name, pd.REST.Endpoint, pd.REST.BasePath), nil
	}
	return nil, errors.New("unknown plugin discovery source")
}

func CreateGroupDiscovery(pd configtypes.PluginDiscovery, options ...DiscoveryOptions) (GroupDiscovery, error) {
	if pd.OCI != nil {
		return NewOCIGroupDiscovery(pd.OCI.Name, pd.OCI.Image, options...), nil
	}
	return nil, errors.New("unknown group discovery source")
}
