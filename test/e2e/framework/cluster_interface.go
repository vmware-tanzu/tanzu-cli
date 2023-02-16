// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

// ClusterOps has helper operations to perform on cluster
type ClusterOps interface {
	// CreateCluster creates the cluster with given name
	CreateCluster(clusterName string) (output string, err error)
	// DeleteCluster deletes the cluster with given name
	DeleteCluster(clusterName string) (output string, err error)
	// ClusterStatus checks the status of the cluster for given cluster name
	ClusterStatus(clusterName string) (output string, err error)
	// GetClusterEndpoint returns the cluster endpoint for the given cluster name
	GetClusterEndpoint(clusterName string) (endpoint string, err error)
	// GetClusterContext returns the given cluster kubeconfig context
	GetClusterContext(clusterName string) string
	// GetKubeconfigPath returns the default kubeconfig path
	GetKubeconfigPath() string
}

// ClusterInfo holds the general cluster details
type ClusterInfo struct {
	Name           string
	ClusterContext string
	EndPoint       string
	KubeConfigPath string
	APIKey         string
}
