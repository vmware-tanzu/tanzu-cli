// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package airgapped contains e2e tests related to airgapped env scenarios
package airgapped

import "github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"

const functionality = "functionality"

// TODO(anujc): Remove the hardcoding of the plugins within the plugin-groups here once the `tanzu plugin group get`
// command is implemented. We can use the output of `tanzu plugin group get` command with the original repository
// and use the created plugin map to do the validation of plugins availability after plugin gets migrated to new repository.
var pluginsForPGTKG001 = []*framework.PluginInfo{
	{Name: "cluster", Target: "kubernetes", Version: "v0.0.1", Description: "cluster " + functionality},
	{Name: "feature", Target: "kubernetes", Version: "v0.0.1", Description: "feature " + functionality},
	{Name: "kubernetes-release", Target: "kubernetes", Version: "v0.0.1", Description: "kubernetes-release " + functionality},
	{Name: "management-cluster", Target: "kubernetes", Version: "v0.0.1", Description: "management-cluster " + functionality},
	{Name: "package", Target: "kubernetes", Version: "v0.0.1", Description: "package " + functionality},
	{Name: "secret", Target: "kubernetes", Version: "v0.0.1", Description: "secret " + functionality},
}

var pluginsForPGTKG999 = []*framework.PluginInfo{
	{Name: "cluster", Target: "kubernetes", Version: "v9.9.9", Description: "cluster " + functionality},
	{Name: "feature", Target: "kubernetes", Version: "v9.9.9", Description: "feature " + functionality},
	{Name: "kubernetes-release", Target: "kubernetes", Version: "v9.9.9", Description: "kubernetes-release " + functionality},
	{Name: "management-cluster", Target: "kubernetes", Version: "v9.9.9", Description: "management-cluster " + functionality},
	{Name: "package", Target: "kubernetes", Version: "v9.9.9", Description: "package " + functionality},
	{Name: "secret", Target: "kubernetes", Version: "v9.9.9", Description: "secret " + functionality},
}

//nolint:dupl
var pluginsForPGTMC001 = []*framework.PluginInfo{
	{Name: "account", Target: "mission-control", Version: "v0.0.1", Description: "account " + functionality},
	{Name: "apply", Target: "mission-control", Version: "v0.0.1", Description: "apply " + functionality},
	{Name: "audit", Target: "mission-control", Version: "v0.0.1", Description: "audit " + functionality},
	{Name: "cluster", Target: "mission-control", Version: "v0.0.1", Description: "cluster " + functionality},
	{Name: "clustergroup", Target: "mission-control", Version: "v0.0.1", Description: "clustergroup " + functionality},
	{Name: "data-protection", Target: "mission-control", Version: "v0.0.1", Description: "data-protection " + functionality},
	{Name: "ekscluster", Target: "mission-control", Version: "v0.0.1", Description: "ekscluster " + functionality},
	{Name: "events", Target: "mission-control", Version: "v0.0.1", Description: "events " + functionality},
	{Name: "iam", Target: "mission-control", Version: "v0.0.1", Description: "iam " + functionality},
	{Name: "inspection", Target: "mission-control", Version: "v0.0.1", Description: "inspection " + functionality},
	{Name: "integration", Target: "mission-control", Version: "v0.0.1", Description: "integration " + functionality},
	{Name: "management-cluster", Target: "mission-control", Version: "v0.0.1", Description: "management-cluster " + functionality},
	{Name: "policy", Target: "mission-control", Version: "v0.0.1", Description: "policy " + functionality},
	{Name: "workspace", Target: "mission-control", Version: "v0.0.1", Description: "workspace " + functionality},
	{Name: "helm", Target: "mission-control", Version: "v0.0.1", Description: "helm " + functionality},
	{Name: "secret", Target: "mission-control", Version: "v0.0.1", Description: "secret " + functionality},
	{Name: "continuousdelivery", Target: "mission-control", Version: "v0.0.1", Description: "continuousdelivery " + functionality},
	{Name: "tanzupackage", Target: "mission-control", Version: "v0.0.1", Description: "tanzupackage " + functionality},
}

//nolint:dupl
var pluginsForPGTMC999 = []*framework.PluginInfo{
	{Name: "account", Target: "mission-control", Version: "v9.9.9", Description: "account " + functionality},
	{Name: "apply", Target: "mission-control", Version: "v9.9.9", Description: "apply " + functionality},
	{Name: "audit", Target: "mission-control", Version: "v9.9.9", Description: "audit " + functionality},
	{Name: "cluster", Target: "mission-control", Version: "v9.9.9", Description: "cluster " + functionality},
	{Name: "clustergroup", Target: "mission-control", Version: "v9.9.9", Description: "clustergroup " + functionality},
	{Name: "data-protection", Target: "mission-control", Version: "v9.9.9", Description: "data-protection " + functionality},
	{Name: "ekscluster", Target: "mission-control", Version: "v9.9.9", Description: "ekscluster " + functionality},
	{Name: "events", Target: "mission-control", Version: "v9.9.9", Description: "events " + functionality},
	{Name: "iam", Target: "mission-control", Version: "v9.9.9", Description: "iam " + functionality},
	{Name: "inspection", Target: "mission-control", Version: "v9.9.9", Description: "inspection " + functionality},
	{Name: "integration", Target: "mission-control", Version: "v9.9.9", Description: "integration " + functionality},
	{Name: "management-cluster", Target: "mission-control", Version: "v9.9.9", Description: "management-cluster " + functionality},
	{Name: "policy", Target: "mission-control", Version: "v9.9.9", Description: "policy " + functionality},
	{Name: "workspace", Target: "mission-control", Version: "v9.9.9", Description: "workspace " + functionality},
	{Name: "helm", Target: "mission-control", Version: "v9.9.9", Description: "helm " + functionality},
	{Name: "secret", Target: "mission-control", Version: "v9.9.9", Description: "secret " + functionality},
	{Name: "continuousdelivery", Target: "mission-control", Version: "v9.9.9", Description: "continuousdelivery " + functionality},
	{Name: "tanzupackage", Target: "mission-control", Version: "v9.9.9", Description: "tanzupackage " + functionality},
}

var pluginsNotInAnyPG999 = []*framework.PluginInfo{
	{Name: "isolated-cluster", Target: "global", Version: "v9.9.9", Description: "isolated-cluster " + functionality},
	{Name: "pinniped-auth", Target: "global", Version: "v9.9.9", Description: "pinniped-auth " + functionality},
}

var essentialPlugins = []*framework.PluginInfo{
	{Name: "telemetry", Target: "global", Version: "v9.9.9", Description: "telemetry " + functionality},
}

var pluginsNotInAnyPGAndUsingSha = []*framework.PluginInfo{
	{Name: "plugin-with-sha", Target: "global", Version: "v9.9.9", Description: "plugin-with-sha " + functionality},
}
