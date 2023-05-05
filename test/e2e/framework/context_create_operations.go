// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// ContextCreateOps helps to run context create command
type ContextCreateOps interface {
	// CreateContextWithEndPoint creates a context with a given endpoint URL
	CreateContextWithEndPoint(contextName, endpoint string, opts ...E2EOption) error
	// CreateContextWithEndPointStaging creates a context with a given endpoint URL for staging, returns stdout and error
	CreateContextWithEndPointStaging(contextName, endpoint string, opts ...E2EOption) (string, error)
	// CreateContextWithKubeconfig creates a context with the given kubeconfig file path and a context from the kubeconfig file
	CreateContextWithKubeconfig(contextName, kubeconfigPath, kubeContext string, opts ...E2EOption) error
	// CreateContextWithDefaultKubeconfig creates a context with the default kubeconfig file and a given input context name if it exists in the default kubeconfig file
	CreateContextWithDefaultKubeconfig(contextName, kubeContext string, opts ...E2EOption) error
}

type contextCreateOps struct {
	ContextCreateOps
	cmdExe CmdOps
}

func NewContextCreateOps() ContextCreateOps {
	return &contextCreateOps{
		cmdExe: NewCmdOps(),
	}
}

func (cc *contextCreateOps) CreateContextWithEndPoint(contextName, endpoint string, opts ...E2EOption) error {
	createContextCmd := fmt.Sprintf(CreateContextWithEndPoint, "%s", endpoint, contextName)
	out, _, err := cc.cmdExe.TanzuCmdExec(createContextCmd, opts...)
	if err != nil {
		log.Info(fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
		return errors.Wrap(err, fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
	}
	log.Infof(ContextCreated, contextName)
	return err
}

func (cc *contextCreateOps) CreateContextWithEndPointStaging(contextName, endpoint string, opts ...E2EOption) (string, error) {
	createContextCmd := fmt.Sprintf(CreateContextWithEndPointStaging, "%s", endpoint, contextName)
	out, stdErr, err := cc.cmdExe.TanzuCmdExec(createContextCmd, opts...)
	log.Infof("out:%s stdErr:%s", out.String(), stdErr.String())
	if err != nil {
		log.Info(fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
		return out.String(), errors.Wrap(err, fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
	}
	log.Infof(ContextCreated, contextName)
	return out.String(), err
}

func (cc *contextCreateOps) CreateContextWithKubeconfig(contextName, kubeconfigPath, kubeContext string, opts ...E2EOption) error {
	createContextCmd := fmt.Sprintf(CreateContextWithKubeconfigFile, "%s", kubeconfigPath, kubeContext, contextName)
	out, _, err := cc.cmdExe.TanzuCmdExec(createContextCmd, opts...)
	if err != nil {
		log.Info(fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
		return errors.Wrap(err, fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
	}
	log.Infof(ContextCreated, contextName)
	return err
}

func (cc *contextCreateOps) CreateContextWithDefaultKubeconfig(contextName, kubeContext string, opts ...E2EOption) error {
	createContextCmd := fmt.Sprintf(CreateContextWithDefaultKubeconfigFile, "%s", kubeContext, contextName)
	out, _, err := cc.cmdExe.TanzuCmdExec(createContextCmd, opts...)
	if err != nil {
		log.Info(fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
		return errors.Wrap(err, fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
	}
	log.Infof(ContextCreated, contextName)
	return err
}
