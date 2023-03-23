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
	// CreateConextWithEndPoint creates a context with a given endpoint URL
	CreateConextWithEndPoint(contextName, endpoint string) error
	// CreateConextWithEndPointStaging creates a context with a given endpoint URL for staging
	CreateConextWithEndPointStaging(contextName, endpoint string) error
	// CreateConextWithKubeconfig creates a context with the given kubeconfig file path and a context from the kubeconfig file
	CreateConextWithKubeconfig(contextName, kubeconfigPath, kubeContext string) error
	// CreateContextWithDefaultKubeconfig creates a context with the default kubeconfig file and a given input context name if it exists in the default kubeconfig file
	CreateContextWithDefaultKubeconfig(contextName, kubeContext string) error
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

func (cc *contextCreateOps) CreateConextWithEndPoint(contextName, endpoint string) error {
	createContextCmd := fmt.Sprintf(CreateContextWithEndPoint, endpoint, contextName)
	out, _, err := cc.cmdExe.Exec(createContextCmd)
	if err != nil {
		log.Info(fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
		return errors.Wrap(err, fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
	}
	log.Infof(ContextCreated, contextName)
	return err
}

func (cc *contextCreateOps) CreateConextWithEndPointStaging(contextName, endpoint string) error {
	createContextCmd := fmt.Sprintf(CreateContextWithEndPointStaging, endpoint, contextName)
	out, _, err := cc.cmdExe.Exec(createContextCmd)
	if err != nil {
		log.Info(fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
		return errors.Wrap(err, fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
	}
	log.Infof(ContextCreated, contextName)
	return err
}

func (cc *contextCreateOps) CreateConextWithKubeconfig(contextName, kubeconfigPath, kubeContext string) error {
	createContextCmd := fmt.Sprintf(CreateContextWithKubeconfigFile, kubeconfigPath, kubeContext, contextName)
	out, _, err := cc.cmdExe.Exec(createContextCmd)
	if err != nil {
		log.Info(fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
		return errors.Wrap(err, fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
	}
	log.Infof(ContextCreated, contextName)
	return err
}

func (cc *contextCreateOps) CreateContextWithDefaultKubeconfig(contextName, kubeContext string) error {
	createContextCmd := fmt.Sprintf(CreateContextWithDefaultKubeconfigFile, kubeContext, contextName)
	out, _, err := cc.cmdExe.Exec(createContextCmd)
	if err != nil {
		log.Info(fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
		return errors.Wrap(err, fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
	}
	log.Infof(ContextCreated, contextName)
	return err
}
