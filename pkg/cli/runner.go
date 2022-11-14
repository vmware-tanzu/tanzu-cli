// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"

	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aunum/log"
)

// Runner is a plugin runner.
type Runner struct {
	name          string
	args          []string
	pluginAbsPath string
}

// NewRunner creates an instance of Runner.
func NewRunner(name, pluginAbsPath string, args []string) *Runner {
	r := &Runner{
		name:          name,
		args:          args,
		pluginAbsPath: pluginAbsPath,
	}
	return r
}

// Run runs a plugin.
func (r *Runner) Run(ctx context.Context) error {
	return r.runStdOutput(ctx, r.pluginPath())
}

// RunTest runs a plugin test.
func (r *Runner) RunTest(ctx context.Context) error {
	return r.runStdOutput(ctx, r.testPluginPath())
}

// runStdOutput runs a plugin and writes any output to the standard os.Stdout and os.Stderr.
func (r *Runner) runStdOutput(ctx context.Context, pluginPath string) error {
	err := r.run(ctx, pluginPath, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// RunOutput runs a plugin and returns the output.
func (r *Runner) RunOutput(ctx context.Context) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := r.run(ctx, r.pluginPath(), &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}

// run executes a command at pluginPath. If stdout and stderr are nil, any output from command
// execution is emitted to os.Stdout and os.Stderr respectively. Otherwise any command output
// is captured in the bytes.Buffer.
func (r *Runner) run(ctx context.Context, pluginPath string, stdout, stderr *bytes.Buffer) error {
	if BuildArch().IsWindows() && !strings.HasSuffix(pluginPath, ".exe") {
		pluginPath += ".exe"
	}

	info, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("plugin %q does not exist, try using `tanzu plugin install %s` to install or `tanzu plugin list` to find plugins", r.name, r.name)
	}

	if info.IsDir() {
		return fmt.Errorf("%q is a directory", pluginPath)
	}

	log.Debugf("running command path %s args: %+v", pluginPath, r.args)
	cmd := exec.CommandContext(ctx, pluginPath, r.args...) //nolint:gosec

	cmd.Stdin = os.Stdin
	// Check if the execution output should be captured
	if stderr != nil {
		cmd.Stderr = stderr
	} else {
		cmd.Stderr = os.Stderr
	}
	if stdout != nil {
		cmd.Stdout = stdout
	} else {
		cmd.Stdout = os.Stdout
	}

	err = cmd.Run()
	return err
}

func (r *Runner) pluginPath() string {
	return r.pluginAbsPath
}

func (r *Runner) testPluginPath() string {
	return TestPluginPathFromPluginPath(r.pluginAbsPath)
}

func TestPluginPathFromPluginPath(pluginPath string) string {
	testPluginFilename := "test-" + filepath.Base(pluginPath)
	return filepath.Join(filepath.Dir(pluginPath), testPluginFilename)
}
