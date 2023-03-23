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
	UseContext(contextName string) error
	// GetContext helps to run `context get` command
	GetContext(contextName string) (ContextInfo, error)
	// ListContext helps to run `context list` command
	ListContext() ([]*ContextListInfo, error)
	// DeleteContext helps to run `context delete` command
	DeleteContext(contextName string) error
	// GetActiveContext returns current active context
	GetActiveContext(targetType string) (string, error)
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

func (cc *contextCmdOps) UseContext(contextName string) error {
	useContextCmd := fmt.Sprintf(UseContext, contextName)
	_, _, err := cc.cmdExe.Exec(useContextCmd)
	return err
}

func (cc *contextCmdOps) GetContext(contextName string) (ContextInfo, error) {
	getContextCmd := fmt.Sprintf(GetContext, contextName)
	out, _, err := cc.cmdExe.Exec(getContextCmd)
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

func (cc *contextCmdOps) ListContext() ([]*ContextListInfo, error) {
	return ExecuteCmdAndBuildJSONOutput[ContextListInfo](cc.cmdExe, ListContextOutputInJSON)
}

func (cc *contextCmdOps) GetActiveContext(targetType string) (string, error) {
	list, err := cc.ListContext()
	if err != nil {
		return "", err
	}
	activeCtx := ""
	for _, context := range list {
		if context.Iscurrent == "true" && context.Type == targetType {
			if activeCtx != "" {
				return "", errors.New("more than one context is active")
			}
			activeCtx = context.Name
		}
	}
	return activeCtx, nil
}

func (cc *contextCmdOps) DeleteContext(contextName string) error {
	deleteContextCmd := fmt.Sprintf(DeleteContext, contextName)
	stdOut, stdErr, err := cc.cmdExe.Exec(deleteContextCmd)
	if err != nil {
		log.Infof("failed to delete context:%s", contextName)
		return errors.Wrapf(err, FailedToDeleteContext+", stderr:%s stdout:%s , ", stdErr.String(), stdOut.String())
	}
	log.Infof(ContextCreated, contextName)
	return err
}
