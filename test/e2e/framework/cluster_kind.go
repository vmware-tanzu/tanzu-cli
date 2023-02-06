// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"gopkg.in/yaml.v3"
	configapi "k8s.io/client-go/tools/clientcmd/api/v1"
)

// KindCluster performs k8s KIND cluster operations
type KindCluster interface {
	ClusterOps
}

// kindCluster implements ClusterOps interface
type kindCluster struct {
	CmdOps
	Docker
}

func NewKindCluster(docker Docker) KindCluster {
	return &kindCluster{
		CmdOps: NewCmdOps(),
		Docker: docker,
	}
}

// CreateCluster creates kind cluster with given name and returns stdout info
// if container runtime not running or any error then returns stdout and error info
func (kc *kindCluster) CreateCluster(kindClusterName string) (output string, err error) {
	stdOut, err := kc.ContainerRuntimeStatus()
	if err != nil {
		return stdOut, err
	}
	createCmd := fmt.Sprintf(KindClusterCreate, kindClusterName)
	stdOutBuffer, stdErrBuffer, err := kc.Exec(createCmd)
	if err != nil {
		return stdOutBuffer.String(), fmt.Errorf(stdErrBuffer.String(), err)
	}
	return stdOutBuffer.String(), err
}

// DeleteCluster creates kind cluster with given name and returns stdout info
// if container runtime not running or any error then returns stdout and error info
func (kc *kindCluster) DeleteCluster(kindClusterName string) (output string, err error) {
	stdOut, err := kc.ContainerRuntimeStatus()
	if err != nil {
		return stdOut, err
	}
	delCmd := fmt.Sprintf(KindClusterDelete, kindClusterName)
	stdOutBuffer, stdErrBuffer, err := kc.Exec(delCmd)
	if err != nil {
		return stdOutBuffer.String(), fmt.Errorf(stdErrBuffer.String(), err)
	}
	return stdOutBuffer.String(), err
}

// ClusterStatus checks given kind cluster status and returns stdout info
// if container runtime not running or any error then returns stdout and error info
func (kc *kindCluster) ClusterStatus(kindClusterName string) (output string, err error) {
	stdOut, err := kc.ContainerRuntimeStatus()
	if err != nil {
		return stdOut, err
	}
	statusCmd := fmt.Sprintf(KindClusterStatus, kc.GetClusterContext(kindClusterName))
	stdOutBuffer, stdErrBuffer, err := kc.Exec(statusCmd)
	if err != nil {
		return stdOutBuffer.String(), fmt.Errorf(stdErrBuffer.String(), err)
	}
	return stdOutBuffer.String(), err
}

// GetClusterEndpoint returns given kind cluster control plane endpoint
func (kc *kindCluster) GetClusterEndpoint(kindClusterName string) (endpoint string, err error) {
	stdOut, err := kc.ContainerRuntimeStatus()
	if err != nil {
		return stdOut, err
	}
	path := kc.GetKubeconfigPath()
	file, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var conf *configapi.Config
	err = yaml.Unmarshal(file, &conf)
	if err != nil {
		return "", errors.Wrap(err, "failed to construct yaml node from kubeconfig file")
	}
	ctx := kc.GetClusterContext(kindClusterName)
	for i := range conf.Clusters {
		if conf.Clusters[i].Name == ctx {
			return conf.Clusters[i].Cluster.Server, nil
		}
	}
	return "", errors.New("the '" + path + "' kubeconfig file does not have context '" + ctx + "' details")
}

func (kc *kindCluster) GetClusterContext(kindClusterName string) string {
	return "kind-" + kindClusterName
}

func (kc *kindCluster) GetKubeconfigPath() string {
	return os.Getenv("HOME") + "/.kube/config"
}
