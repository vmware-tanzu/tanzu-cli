// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// ContextCmdOps helps to run Context lifecycle operations
type ContextCmdOps interface {
	// ContextCreateOps helps context create operations
	ContextCreateOps
	// UseContext helps to run 'context use' command
	UseContext(contextName string, opts ...E2EOption) error
	// GetContext helps to run `context get` command
	GetContext(contextName string, opts ...E2EOption) (ContextInfo, error)
	// ListContext helps to run `context list` command
	ListContext(opts ...E2EOption) ([]*ContextListInfo, error)
	// DeleteContext helps to run `context delete` command
	DeleteContext(contextName string, opts ...E2EOption) (stdOutStr, stdErrStr string, err error)
	// GetActiveContext returns current active context
	GetActiveContext(targetType string, opts ...E2EOption) (string, error)
	// GetActiveContexts returns all active contexts
	GetActiveContexts(opts ...E2EOption) ([]*ContextListInfo, error)
	// UnsetContext unsets the given context with 'tanzu context unset' and returns stdOut, stdErr and error
	UnsetContext(contextName string, opts ...E2EOption) (stdOutStr, stdErrStr string, err error)
}

// contextCmdOps implements the interface ContextCmdOps
type contextCmdOps struct {
	ContextCreateOps
	cmdExe CmdOps
}

func NewContextCmdOps() ContextCmdOps {
	return &contextCmdOps{
		cmdExe:           NewCmdOps(),
		ContextCreateOps: NewContextCreateOps(),
	}
}

func (cc *contextCmdOps) UseContext(contextName string, opts ...E2EOption) error {
	useContextCmd := fmt.Sprintf(UseContext, "%s", contextName)
	_, _, err := cc.cmdExe.TanzuCmdExec(useContextCmd, opts...)
	return err
}

func (cc *contextCmdOps) UnsetContext(contextName string, opts ...E2EOption) (string, string, error) {
	unsetCmd := UnsetContext
	if contextName != "" {
		unsetCmd += " " + contextName
	}
	stdOut, stdErr, err := cc.cmdExe.TanzuCmdExec(unsetCmd, opts...)
	return stdOut.String(), stdErr.String(), err
}

func (cc *contextCmdOps) GetContext(contextName string, opts ...E2EOption) (ContextInfo, error) {
	getContextCmd := fmt.Sprintf(GetContext, "%s", contextName)
	out, _, err := cc.cmdExe.TanzuCmdExec(getContextCmd, opts...)
	if err != nil {
		return ContextInfo{}, err
	}
	jsonStr := out.String()
	var contextInfo ContextInfo
	err = json.Unmarshal([]byte(jsonStr), &contextInfo)
	if err != nil {
		return ContextInfo{}, errors.Wrap(err, "failed to construct json node from context get output")
	}
	return contextInfo, nil
}

func (cc *contextCmdOps) ListContext(opts ...E2EOption) ([]*ContextListInfo, error) {
	list, _, _, err := ExecuteCmdAndBuildJSONOutput[ContextListInfo](cc.cmdExe, ListContextOutputInJSON, opts...)
	return list, err
}

func (cc *contextCmdOps) GetActiveContext(targetType string, opts ...E2EOption) (string, error) {
	list, err := cc.ListContext(opts...)
	if err != nil {
		return "", err
	}
	activeCtx := ""
	for _, context := range list {
		if context.Iscurrent == True && context.Type == targetType {
			if activeCtx != "" {
				return "", errors.New("more than one context is active")
			}
			activeCtx = context.Name
		}
	}
	return activeCtx, nil
}

func (cc *contextCmdOps) GetActiveContexts(opts ...E2EOption) ([]*ContextListInfo, error) {
	list, err := cc.ListContext(opts...)
	contexts := make([]*ContextListInfo, 0)
	if err != nil {
		return contexts, err
	}
	for i, _ := range list {
		if list[i].Iscurrent == True {
			contexts = append(contexts, list[i])
		}
	}
	return contexts, nil
}

func (cc *contextCmdOps) DeleteContext(contextName string, opts ...E2EOption) (string, string, error) {
	deleteContextCmd := fmt.Sprintf(DeleteContext, "%s", contextName)
	stdOut, stdErr, err := cc.cmdExe.TanzuCmdExec(deleteContextCmd, opts...)
	if err != nil {
		log.Infof("failed to delete context:%s", contextName)
		return stdOut.String(), stdErr.String(), errors.Wrapf(err, FailedToDeleteContext+", stderr:%s stdout:%s , ", stdErr.String(), stdOut.String())
	}
	log.Infof(ContextDeleted, contextName)
	return stdOut.String(), stdErr.String(), err
}
