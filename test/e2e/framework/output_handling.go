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
	Installed   string `json:"installed"`
	Recommended string `json:"recommended"`
	Context     string `json:"context"`
}

type PluginSearch struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Target      string `json:"target"`
	Latest      string `json:"latest"`
}

type PluginGroup struct {
	Name        string   `json:"name"`
	Group       string   `json:"group"`
	Description string   `json:"description"`
	Latest      string   `json:"latest"`
	Versions    []string `json:"versions"`
}

type PluginGroupGet struct {
	Group         string `json:"group"`
	PluginName    string `json:"pluginname"`
	PluginTarget  string `json:"plugintarget"`
	PluginVersion string `json:"pluginversion"`
}

type PluginSourceInfo struct {
	Name  string `json:"name"`
	Image string `json:"image"`
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

// TMCPluginsResponse is to mock the tmc endpoint response for plugins info
type TMCPluginsResponse struct {
	PluginsInfo TMCPluginsInfo `yaml:"pluginsInfo"`
}

type TMCPlugin struct {
	Name               string `yaml:"name"`
	Description        string `yaml:"description"`
	RecommendedVersion string `yaml:"recommendedVersion"`
}

type TMCPluginsInfo struct {
	Plugins []TMCPlugin `yaml:"plugins"`
}

type TMCPluginsMockRequestResponseMapping struct {
	Request struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"request"`
	Response struct {
		Status  int    `json:"status"`
		Body    string `json:"body"`
		Headers struct {
			ContentType string `json:"Content-Type"`
			Accept      string `json:"Accept"`
		} `json:"headers"`
	} `json:"response"`
}

type CertDetails struct {
	CaCertificate        string `json:"ca-certificate"`
	Host                 string `json:"host"`
	Insecure             string `json:"insecure"`
	SkipCertVerification string `json:"skip-cert-verification"`
}

type PluginDescribe struct {
	Buildsha                     string `yaml:"buildsha"`
	Completiontype               string `yaml:"completiontype"`
	Defaultfeatureflags          string `yaml:"defaultfeatureflags"`
	Description                  string `yaml:"description"`
	Digest                       string `yaml:"digest"`
	Discoveredrecommendedversion string `yaml:"discoveredrecommendedversion"`
	Discovery                    string `yaml:"discovery"`
	Docurl                       string `yaml:"docurl"`
	Group                        string `yaml:"group"`
	Installationpath             string `yaml:"installationpath"`
	Name                         string `yaml:"name"`
	Scope                        string `yaml:"scope"`
	Status                       string `yaml:"status"`
	Target                       string `yaml:"target"`
	Version                      string `yaml:"version"`
}
