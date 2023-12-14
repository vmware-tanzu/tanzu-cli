// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// CreateCluster creates kind cluster with given cluster name
func (kc *kindCluster) CreateCluster(kindClusterName string) (stdOut, stdErr string, err error) {
	stdOut, stdErr, err = kc.ContainerRuntimeStatus()
	if err != nil {
		return stdOut, stdErr, err
	}
	createCmd := fmt.Sprintf(KindClusterCreate, kindClusterName)
	stdOutBuffer, stdErrBuffer, err := kc.Exec(createCmd)
	return stdOutBuffer.String(), stdErrBuffer.String(), err
}

// DeleteCluster deletes given kind cluster
func (kc *kindCluster) DeleteCluster(kindClusterName string) (stdOut, stdErr string, err error) {
	stdOut, stdErr, err = kc.ContainerRuntimeStatus()
	if err != nil {
		return stdOut, stdErr, err
	}
	delCmd := fmt.Sprintf(KindClusterDelete, kindClusterName)
	stdOutBuffer, stdErrBuffer, err := kc.Exec(delCmd)
	return stdOutBuffer.String(), stdErrBuffer.String(), err
}

// ClusterStatus checks given kind cluster status
func (kc *kindCluster) ClusterStatus(kindClusterName string) (stdOut, stdErr string, err error) {
	stdOut, stdErr, err = kc.ContainerRuntimeStatus()
	if err != nil {
		return stdOut, stdErr, err
	}
	statusCmd := fmt.Sprintf(KindClusterStatus, kc.GetClusterContext(kindClusterName))
	stdOutBuffer, stdErrBuffer, err := kc.Exec(statusCmd)
	return stdOutBuffer.String(), stdErrBuffer.String(), err
}

// ApplyConfig applies given config file on to the given kind cluster context
func (kc *kindCluster) ApplyConfig(contextName, configFilePath string) (stdOut, stdErr string, err error) {
	applyCmd := fmt.Sprintf(KubectlApply, contextName, configFilePath)
	stdOutBuff, stdErrBuff, err := kc.CmdOps.Exec(applyCmd)
	return stdOutBuff.String(), stdErrBuff.String(), err
}

// WaitForCondition waits for certain condition on cluster to be true,
// or returns error otherwise (including after a timeout period of 30s has elapsed)
func (kc *kindCluster) WaitForCondition(contextName string, waitArgs []string) error {
	waitString := strings.Join(waitArgs, " ")
	waitCmd := fmt.Sprintf(KubectlWait, contextName, waitString)
	_, _, err := kc.CmdOps.Exec(waitCmd)
	return err
}

// GetClusterEndpoint returns given kind cluster control plane endpoint
func (kc *kindCluster) GetClusterEndpoint(kindClusterName string) (endpoint, stdOut, stdErr string, err error) {
	stdOut, stdErr, err = kc.ContainerRuntimeStatus()
	if err != nil {
		return "", stdOut, stdErr, err
	}
	path := kc.GetKubeconfigPath()
	file, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", err
	}
	var conf *configapi.Config
	err = yaml.Unmarshal(file, &conf)
	if err != nil {
		return "", "", "", errors.Wrap(err, "failed to construct yaml node from kubeconfig file")
	}
	ctx := kc.GetClusterContext(kindClusterName)
	for i := range conf.Clusters {
		if conf.Clusters[i].Name == ctx {
			return conf.Clusters[i].Cluster.Server, "", "", nil
		}
	}
	return "", "", "", errors.Errorf("the '%s' kubeconfig file does not have context '%s' details", path, ctx)
}

func (kc *kindCluster) GetClusterContext(kindClusterName string) string {
	return "kind-" + kindClusterName
}

func (kc *kindCluster) GetKubeconfigPath() string {
	return filepath.Join(GetE2EHomeDir(), ".kube", "config")
}
