// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

type PluginInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Target      string `json:"target"`
	Discovery   string `json:"discovery"`
	Scope       string `json:"scope"`
	Status      string `json:"status"`
	Version     string `json:"version"`
}

type PluginSearch struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Target      string `json:"target"`
	Latest      string `json:"latest"`
}

type PluginGroup struct {
	Group string `json:"group"`
}

type PluginSourceInfo struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
	Type  string `json:"type"`
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

type Server struct {
	Context  string `json:"context"`
	Endpoint string `json:"endpoint"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
}
