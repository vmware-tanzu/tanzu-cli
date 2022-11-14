// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides commands
package command

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates a root command.
func NewRootCmd() (*cobra.Command, error) {
	var rootCmd = &cobra.Command{
		Use: "tanzu",
		// Don't have Cobra print the error message, the CLI will
		// print it itself in a nicer format.
		SilenceErrors: true,
	}

	return rootCmd, nil
}

// Execute executes the CLI.
func Execute() error {
	root, err := NewRootCmd()
	if err != nil {
		return err
	}
	return root.Execute()
}
