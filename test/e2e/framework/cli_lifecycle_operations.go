// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

// CliOps performs basic cli operations
type CliOps interface {
	CliInit(opts ...E2EOption) error
	CliVersion(opts ...E2EOption) (string, error)
	InstallCLI(version string) error
	UninstallCLI(version string) error
}

type cliOps struct {
	CmdOps
}

func NewCliOps() CliOps {
	return &cliOps{
		CmdOps: NewCmdOps(),
	}
}

// CliInit() initializes the CLI
func (co *cliOps) CliInit(opts ...E2EOption) error {
	_, _, err := co.TanzuCmdExec(TanzuInit, opts...)
	return err
}

// CliVersion returns the CLI version info
func (co *cliOps) CliVersion(opts ...E2EOption) (string, error) {
	stdOut, _, err := co.TanzuCmdExec(TanzuInit, opts...)
	return stdOut.String(), err
}

// InstallCLI installs specific CLI version
func (co *cliOps) InstallCLI(version string) (err error) {
	return nil
}

// UninstallCLI uninstalls specific CLI version
func (co *cliOps) UninstallCLI(version string) (err error) {
	return nil
}
