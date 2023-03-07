// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

func populateDefaultStandaloneDiscovery(c *configtypes.ClientConfig) bool {
	if c.ClientOptions == nil {
		c.ClientOptions = &configtypes.ClientOptions{}
	}
	if c.ClientOptions.CLI == nil {
		c.ClientOptions.CLI = &configtypes.CLIOptions{}
	}
	if c.ClientOptions.CLI.DiscoverySources == nil {
		c.ClientOptions.CLI.DiscoverySources = make([]configtypes.PluginDiscovery, 0)
	}

	defaultDiscovery := getDefaultStandaloneDiscoverySource(GetDefaultStandaloneDiscoveryType())
	if defaultDiscovery == nil {
		return false
	}

	matchIdx := findDiscoverySourceIndex(c.ClientOptions.CLI.DiscoverySources, func(pd configtypes.PluginDiscovery) bool {
		return discovery.CheckDiscoveryName(pd, DefaultStandaloneDiscoveryName) ||
			discovery.CheckDiscoveryName(pd, DefaultStandaloneDiscoveryNameLocal)
	})

	if matchIdx >= 0 {
		if discovery.CompareDiscoverySource(c.ClientOptions.CLI.DiscoverySources[matchIdx], *defaultDiscovery, GetDefaultStandaloneDiscoveryType()) {
			return false
		}
		c.ClientOptions.CLI.DiscoverySources[matchIdx] = *defaultDiscovery
		return true
	}

	// Prepend default discovery to available discovery sources
	c.ClientOptions.CLI.DiscoverySources = append([]configtypes.PluginDiscovery{*defaultDiscovery}, c.ClientOptions.CLI.DiscoverySources...)
	return true
}

func findDiscoverySourceIndex(discoverySources []configtypes.PluginDiscovery, matcherFunc func(pd configtypes.PluginDiscovery) bool) int {
	for i := range discoverySources {
		if matcherFunc(discoverySources[i]) {
			return i
		}
	}
	return -1 // haven't found a match
}

func getDefaultStandaloneDiscoverySource(dsType string) *configtypes.PluginDiscovery {
	switch dsType {
	case common.DiscoveryTypeLocal:
		return getDefaultStandaloneDiscoverySourceLocal()
	case common.DiscoveryTypeOCI:
		return getDefaultStandaloneDiscoverySourceOCI()
	}
	log.Warning("unsupported default standalone discovery configuration")
	return nil
}

func getDefaultStandaloneDiscoverySourceOCI() *configtypes.PluginDiscovery {
	return &configtypes.PluginDiscovery{
		OCI: &configtypes.OCIDiscovery{
			Name:  DefaultStandaloneDiscoveryName,
			Image: GetDefaultStandaloneDiscoveryImage(),
		},
	}
}

func getDefaultStandaloneDiscoverySourceLocal() *configtypes.PluginDiscovery {
	return &configtypes.PluginDiscovery{
		Local: &configtypes.LocalDiscovery{
			Name: DefaultStandaloneDiscoveryNameLocal,
			Path: GetDefaultStandaloneDiscoveryLocalPath(),
		},
	}
}
