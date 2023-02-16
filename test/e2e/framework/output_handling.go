// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

type PluginListInfo struct {
	Description string `json:"description"`
	Discovery   string `json:"discovery"`
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Status      string `json:"status"`
	Version     string `json:"version"`
}

type ContextListInfo struct {
	Endpoint            string `json:"endpoint"`
	Iscurrent           string `json:"iscurrent"`
	Ismanagementcluster string `json:"ismanagementcluster"`
	Kubeconfigpath      string `json:"kubeconfigpath"`
	Kubecontext         string `json:"kubecontext"`
	Name                string `json:"name"`
	Type                string `json:"type"`
}

type ContextInfo struct {
	Name        string `json:"name"`
	Target      string `json:"target"`
	ClusterOpts struct {
		Path                string `json:"path"`
		Context             string `json:"context"`
		IsManagementCluster bool   `json:"isManagementCluster"`
	} `json:"clusterOpts"`
}
