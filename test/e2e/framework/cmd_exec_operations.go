// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// CmdOps performs the Command line exec operations
type CmdOps interface {
	Exec(command string, opts ...E2EOption) (stdOut, stdErr *bytes.Buffer, err error)
	ExecContainsString(command, contains string, opts ...E2EOption) error
	ExecContainsAnyString(command string, contains []string, opts ...E2EOption) error
	ExecContainsErrorString(command, contains string, opts ...E2EOption) error
	ExecNotContainsStdErrorString(command, contains string, opts ...E2EOption) error
	ExecNotContainsString(command, contains string, opts ...E2EOption) error
	TanzuCmdExec(command string, opts ...E2EOption) (stdOut, stdErr *bytes.Buffer, err error)
}

// cmdOps is the implementation of CmdOps
type cmdOps struct {
	CmdOps
}

func NewCmdOps() CmdOps {
	return &cmdOps{}
}

// TanzuCmdExec executes the tanzu command by default uses `tanzu` prefix
func (co *cmdOps) TanzuCmdExec(command string, opts ...E2EOption) (stdOut, stdErr *bytes.Buffer, err error) {
	// Get Default options, and initialize Tanzu Command Prefix value as 'tanzu'
	var options *E2EOptions
	options = NewE2EOptions(
		WithTanzuBinary(TanzuBinary),
	)

	// Apply the options provided, which allow the user to override the Tanzu command prefix value, such as 'tz'
	for _, opt := range opts {
		opt(options)
	}

	// Verify whether the Tanzu prefix or binary is set and if not, set the tanzu binary if tanzu binary path is specified or default to tanzu prefix available at PATH variable
	if strings.Index(command, "%s") == 0 {
		command = fmt.Sprintf(command, options.TanzuBinary)
	}

	// If any additional flags are added, then append them to the command
	if options.AdditionalFlags != "" {
		command += options.AdditionalFlags
	}
	return co.Exec(command)
}

// Exec the command, exit on error
func (co *cmdOps) Exec(command string, opts ...E2EOption) (stdOut, stdErr *bytes.Buffer, err error) {
	// Default options
	options := NewE2EOptions()

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	log.Infof(ExecutingCommand, command)

	cmdInput := strings.Split(command, " ")
	cmdName := cmdInput[0]
	cmdArgs := cmdInput[1:]
	cmd := exec.Command(cmdName, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return &stdout, &stderr, fmt.Errorf("x error while running '%s' (detailed arguments: %q), stdOut: %s, stdErr: %s, err: %s", command, cmdArgs, stdout.String(), stderr.String(), err.Error())
	}
	fmt.Printf("running '%s' (detailed arguments: %q), stdOut: %s, stdErr: %s", command, cmdArgs, stdout.String(), stderr.String())
	return &stdout, &stderr, err
}

// ExecContainsString checks that the given command output contains the string.
func (co *cmdOps) ExecContainsString(command, contains string, opts ...E2EOption) error {
	stdOut, _, err := co.Exec(command, opts...)
	if err != nil {
		return err
	}
	return ContainsString(stdOut, contains)
}

// ExecContainsAnyString checks that the given command output contains any of the given set of strings.
func (co *cmdOps) ExecContainsAnyString(command string, contains []string, opts ...E2EOption) error {
	stdOut, _, err := co.Exec(command, opts...)
	if err != nil {
		return err
	}
	return ContainsAnyString(stdOut, contains)
}

// ExecContainsErrorString checks that the given command stdErr output contains the string
func (co *cmdOps) ExecContainsErrorString(command, contains string, opts ...E2EOption) error {
	_, stdErr, err := co.Exec(command, opts...)
	if err != nil {
		return err
	}
	return ContainsString(stdErr, contains)
}

// ExecNotContainsStdErrorString checks that the given command stdErr output contains the string
func (co *cmdOps) ExecNotContainsStdErrorString(command, contains string, opts ...E2EOption) error {
	_, stdErr, err := co.Exec(command, opts...)
	if err != nil && stdErr == nil {
		return err
	}
	return NotContainsString(stdErr, contains)
}

// NotContainsString checks that the given buffer not contains the string if contains then throws error.
func NotContainsString(stdOut *bytes.Buffer, contains string) error {
	so := stdOut.String()
	if strings.Contains(so, contains) {
		return fmt.Errorf("stdOut %q contains %q", so, contains)
	}
	return nil
}

// ContainsString checks that the given buffer contains the string.
func ContainsString(stdOut *bytes.Buffer, contains string) error {
	so := stdOut.String()
	if !strings.Contains(so, contains) {
		return fmt.Errorf("stdOut %q did not contain %q", so, contains)
	}
	return nil
}

// ContainsAnyString checks that the given buffer contains any of the given set of strings.
func ContainsAnyString(stdOut *bytes.Buffer, contains []string) error {
	var containsAny bool
	so := stdOut.String()

	for _, str := range contains {
		containsAny = containsAny || strings.Contains(so, str)
	}

	if !containsAny {
		return fmt.Errorf("stdOut %q did not contain of the following %q", so, contains)
	}
	return nil
}

// ExecNotContainsString checks that the given command output not contains the string.
func (co *cmdOps) ExecNotContainsString(command, contains string, opts ...E2EOption) error {
	stdOut, _, err := co.Exec(command, opts...)
	if err != nil {
		return err
	}
	return co.NotContainsString(stdOut, contains)
}

// NotContainsString checks that the given buffer not contains the string.
func (co *cmdOps) NotContainsString(stdOut *bytes.Buffer, contains string) error {
	so := stdOut.String()
	if strings.Contains(so, contains) {
		return fmt.Errorf("stdOut %q does contain %q", so, contains)
	}
	return nil
}
