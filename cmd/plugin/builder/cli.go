// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/command"
)

var (
	dryRun      bool
	description string
)

// NewCLICmd creates the CLI builder commands.
func NewCLICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cli",
		Short: "Build CLIs",
	}

	cmd.AddCommand(newAddPluginCmd())
	return cmd
}

// newAddPluginCmd adds a cli plugin to the repository.
func newAddPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-plugin NAME",
		Short: "Add a plugin to a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			name := args[0]
			if description == "" {
				description, err = askDescription()
				if err != nil {
					return err
				}
			}

			return command.AddPlugin(name, description, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print generated files to stdout")
	cmd.Flags().StringVar(&description, "description", "", "Required plugin description")

	return cmd
}

func askDescription() (answer string, err error) {
	questioncfg := &component.QuestionConfig{
		Message: "provide a description",
	}
	err = component.Ask(questioncfg, &answer)
	if err != nil {
		return
	}
	return
}
