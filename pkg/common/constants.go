// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package common defines generic constants and structs
package common

// Plugin status and scope constants
const (
	PluginStatusInstalled       = "installed"
	PluginStatusNotInstalled    = "not installed"
	PluginStatusUpdateAvailable = "update available"
	PluginScopeStandalone       = "Standalone"
	PluginScopeContext          = "Context"
)

// DiscoveryType constants
const (
	DiscoveryTypeOCI        = "oci"
	DiscoveryTypeLocal      = "local"
	DiscoveryTypeGCP        = "gcp"
	DiscoveryTypeKubernetes = "kubernetes"
	DiscoveryTypeREST       = "rest"
)

// DistributionType constants
const (
	DistributionTypeOCI   = "oci"
	DistributionTypeLocal = "local"
)

const (
	// TargetK8s is a kubernetes target of the CLI
	// This target applies if the plugin is interacting with a Kubernetes cluster
	TargetK8s string = "kubernetes"
	targetK8s string = "k8s"

	// TargetTMC is a Tanzu Mission Control target of the CLI
	// This target applies if the plugin is interacting with a Tanzu Mission Control endpoint
	TargetTMC string = "mission-control"
	targetTMC string = "tmc"

	// TargetGlobal is used for plugins that are not associated with any target
	TargetGlobal string = "global"

	// TargetUnknown specifies that the target is not currently known
	TargetUnknown string = ""
)

// Shared strings
const (
	TargetList = "kubernetes[k8s]/mission-control[tmc]/global"
)

// CoreName is the name of the core binary.
const CoreName = "core"

// CommandTypePlugin represents the command type is plugin
const CommandTypePlugin = "plugin"
