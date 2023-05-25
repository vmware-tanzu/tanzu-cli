// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// CliOps performs basic cli operations
type CliOps interface {
	CLIInit(opts ...E2EOption) error
	CLIVersion(opts ...E2EOption) (string, error)
	UninstallTanzuCLI(opts ...E2EOption) error
	InstallLegacyTanzuCLI(opts ...E2EOption) error
	RollbackToLegacyTanzuCLI(tf *Framework, opts ...E2EOption) error
	InstallNewTanzuCLI(opts ...E2EOption) error
	ReinstallNewTanzuCLI(opts ...E2EOption) error
	// CompletionCmd executes `tanzu completion` command for given shell as input value, and returns stdout, stderr and error
	CompletionCmd(shell string, opts ...E2EOption) (string, string, error)
}

type cliOps struct {
	cmdExe CmdOps
}

func NewCliOps() CliOps {
	return &cliOps{
		cmdExe: NewCmdOps(),
	}
}

func (co *cliOps) CompletionCmd(shell string, opts ...E2EOption) (string, string, error) {
	completionCmdWithShell := CompletionCmd
	if shell != "" {
		completionCmdWithShell += " " + shell
	}
	out, stdErr, err := co.cmdExe.TanzuCmdExec(completionCmdWithShell, opts...)
	if err != nil {
		log.Info(fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
		return out.String(), stdErr.String(), errors.Wrap(err, fmt.Sprintf(FailedToCreateContextWithStdout, out.String()))
	}
	return out.String(), stdErr.String(), err
}

// CLIInit initializes the CLI
func (co *cliOps) CLIInit(opts ...E2EOption) error {
	_, _, err := co.cmdExe.TanzuCmdExec(InitCmd, opts...)
	return err
}

// CLIVersion returns the CLI version info
// opts E2EOptions to customize tanzu prefix; default value is tanzu; tz prefix is used in coexistence tests to differentiate with legacy and new Tanzu CLI
func (co *cliOps) CLIVersion(opts ...E2EOption) (string, error) {
	stdOut, _, err := co.cmdExe.TanzuCmdExec(VersionCmd, opts...)
	return stdOut.String(), err
}

// UninstallTanzuCLI removes the Tanzu CLI and its relevant files
// CLIOptions provides a way to pass the file path to user tanzu installation location to uninstall
func (co *cliOps) UninstallTanzuCLI(opts ...E2EOption) error {
	log.Info("Uninstalling Tanzu CLI and its relevant files")
	// Default options
	options := &E2EOptions{}

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	// If filePath is specified remove all files from the path else remove files from default installation directories
	if options.FilePath != "" {
		log.Infof("Uninstalling file %s", options.FilePath)
		err := os.RemoveAll(options.FilePath)
		if err != nil {
			return err
		}
	} else {
		// Remove previously downloaded cli files
		err := os.RemoveAll("$HOME/tanzu/cli")
		if err != nil {
			return fmt.Errorf("failed to remove $HOME/tanzu/cli with error %v", err)
		}

		err = os.RemoveAll("$HOME/go/bin/tanzu")
		if err != nil {
			return fmt.Errorf("failed to remove $HOME/go/bin/tanzu with error %v", err)
		}

		err = os.RemoveAll("$HOME/bin/tanzu")
		if err != nil {
			return fmt.Errorf("failed to remove $HOME/bin/tanzu with error %v", err)
		}

		// Remove CLI binary (executable)
		err = exec.Command("rm", "-rf", "/usr/local/bin/tanzu").Run()
		if err != nil {
			return fmt.Errorf("failed to remove CLI binary tanzu (executable) at /usr/local/bin/tanzu with error %v", err)
		}

		// Remove new CLI binary (executable)
		err = exec.Command("rm", "-rf", "/usr/local/bin/tz").Run()
		if err != nil {
			return fmt.Errorf("failed to remove CLI binary tz (executable) at /usr/local/bin/tanzu with error %v", err)
		}

		// Remove CLI binary (executable)
		err = exec.Command("rm", "-rf", "/usr/bin/tanzu").Run()
		if err != nil {
			return fmt.Errorf("failed to remove CLI binary tanzu (executable) at /usr/bin/tanzu with error %v", err)
		}

		// Remove new CLI binary (executable)
		err = exec.Command("rm", "-rf", "/usr/bin/tz").Run()
		if err != nil {
			return fmt.Errorf("failed to remove CLI binary tz (executable) at /usr/bin/tanzu with error %v", err)
		}
	}

	// current location # Remove config directory
	err := os.RemoveAll("~/.config/tanzu/")
	if err != nil {
		return fmt.Errorf("failed to remove config directory with error %v", err)
	}

	// old location # Remove config directory
	err = os.RemoveAll("~/.tanzu/")
	if err != nil {
		return fmt.Errorf("failed to remove legacy config directory with error %v", err)
	}

	// remove cached catalog.yaml
	err = os.RemoveAll("~/.cache/tanzu")
	if err != nil {
		return fmt.Errorf("failed to remove cached catalog.yaml with error %v", err)
	}

	// Remove plug-ins
	err = exec.Command("rm", "-rf", "~/Library/Application Support/tanzu-cli/*").Run()
	if err != nil {
		return fmt.Errorf("failed to remove plug-ins with error %v", err)
	}

	log.Info("Uninstalled Tanzu CLI")
	return nil
}

// RollbackToLegacyTanzuCLI rolls back to the legacy version of Tanzu CLI
// CLIOptions provides a way to pass the file path to user tanzu installation location
// This method performs rollback instructions steps of the Tanzu CLI like remove config dir, install legacy Tanzu CLI, Run tanzu init and sync commands
func (co *cliOps) RollbackToLegacyTanzuCLI(tf *Framework, opts ...E2EOption) error {
	log.Info("Executing Rollback instructions to legacy Tanzu CLI")
	// Default options
	options := NewE2EOptions(
		WithTanzuCommandPrefix(TanzuPrefix),
	)

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	// current location # Remove config directory
	err := os.RemoveAll("~/.config/tanzu/")
	if err != nil {
		return err
	}

	err = co.InstallLegacyTanzuCLI(opts...)

	if err != nil {
		return err
	}

	err = co.CLIInit(opts...)

	if err != nil {
		return err
	}

	log.Info("Clean and Sync/Install the plugins using legacy Tanzu CLI")

	err = tf.PluginCmd.CleanPlugins(opts...)
	if err != nil {
		return err
	}

	_, err = tf.PluginCmd.Sync(opts...)

	if err != nil {
		return err
	}

	return nil
}

// InstallLegacyTanzuCLI installs the legacy tanzu cli
// CLIOptions provides a way to pass the file path to user tanzu installation location
func (co *cliOps) InstallLegacyTanzuCLI(opts ...E2EOption) error {
	log.Info("Installing Legacy Tanzu CLI")

	legacyCLIFilePath := os.Getenv(CLICoexistenceLegacyTanzuCLIInstallationPath)

	// Default options
	options := NewE2EOptions(
		WithFilePath(legacyCLIFilePath),
	)

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	stdOut, stdErr, err := co.cmdExe.Exec(fmt.Sprintf("cp -r %s/tanzu /usr/bin", options.FilePath))
	log.Info(stdOut.String())

	if stdErr != nil {
		log.Errorf(stdErr.String())
	}

	if err != nil {
		return err
	}

	version, err := co.CLIVersion(opts...)
	log.Info(version)

	if err != nil {
		return err
	}
	return nil
}

// ReinstallNewTanzuCLI re-installs the new tanzu cli
// CLIOptions provides a way to pass the file path to user tanzu installation location
func (co *cliOps) ReinstallNewTanzuCLI(opts ...E2EOption) error {
	log.Info("Reinstalling new Tanzu CLI")

	// current location # Remove config directory
	err := os.RemoveAll("~/.config/tanzu/")
	if err != nil {
		return err
	}

	return co.InstallNewTanzuCLI(opts...)
}

// InstallNewTanzuCLI installs the new tanzu cli to the installation path specified in the env CLICoexistenceNewTanzuCLIInstallationPath
// CLIOptions provides a way to pass the file path to user tanzu installation location and override option to determine whether the new Tanzu CLI should override the installation of legacy Tanzu CLI
// Ex: tanzu prefix is used when new CLI overrides the installation of legacy CLI in cli-coexistence tests and tz prefix is used when new CLI coexists along with the legacy CLI
func (co *cliOps) InstallNewTanzuCLI(opts ...E2EOption) error {
	log.Info("Installing new Tanzu CLI")

	newTanzuCLIFilePath := os.Getenv(CLICoexistenceNewTanzuCLIInstallationPath)

	// Default options
	options := NewE2EOptions(
		WithTanzuCommandPrefix(TzPrefix),
		WithOverride(false),
		WithFilePath(newTanzuCLIFilePath),
	)

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	if options.Override {
		log.Info("New Tanzu CLI to override existing legacy Tanzu CLI")

		// Remove existing Tanzu CLI
		stdOut, stdErr, err := co.cmdExe.Exec("rm -rf /usr/bin/tanzu", opts...)
		log.Info(stdOut.String())
		if stdErr != nil {
			log.Errorf(stdErr.String())
		}

		if err != nil {
			return err
		}

		// Move Tanzu binary to /usr/bin
		copyTanzuCommand := fmt.Sprintf("cp -r %s/tanzu /usr/bin", options.FilePath)
		stdOut, stdErr, err = co.cmdExe.Exec(copyTanzuCommand, opts...)
		log.Info(stdOut.String())
		if stdErr != nil {
			log.Errorf(stdErr.String())
		}

		if err != nil {
			return err
		}

		version, err := co.CLIVersion(opts...)
		log.Info(version)

		if err != nil {
			return err
		}
	} else {
		log.Info("New Tanzu CLI to coexist along with Old Tanzu CLI")

		// Create a copy of new tanzu with name tz
		stdOut, stdErr, err := co.cmdExe.Exec(fmt.Sprintf("cp -r %s/tanzu %s/%s", options.FilePath, options.FilePath, options.TanzuCommandPrefix))
		log.Info(stdOut.String())

		if stdErr != nil {
			log.Errorf(stdErr.String())
		}

		if err != nil {
			return err
		}

		// Move tz binary to /usr/bin
		stdOut, stdErr, err = co.cmdExe.Exec(fmt.Sprintf("cp -r %s/%s /usr/bin", options.FilePath, options.TanzuCommandPrefix))
		log.Info(stdOut.String())

		if stdErr != nil {
			log.Errorf(stdErr.String())
		}

		if err != nil {
			return err
		}

		version, err := co.CLIVersion(opts...)
		log.Info(version)

		if err != nil {
			return err
		}
	}
	return nil
}
