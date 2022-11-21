// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"testing"

	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestGenerateDescriptor(t *testing.T) {
	m := MainUsage{}

	f := m.UsageFunc()

	c := &cobra.Command{
		Use:   "tanzu",
		Short: aurora.Bold(`Tanzu CLI`).String(),
	}

	var subCmd = &cobra.Command{
		Use:   "sub",
		Short: "subcommand",
		Annotations: map[string]string{
			"group": "system",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	c.AddCommand(subCmd)

	err := f(c)
	require.NoError(t, err)
}

func TestSubcommandUsage(t *testing.T) {
	c := &cobra.Command{
		Use:   "myplugin",
		Short: aurora.Bold(`My Plugin`).String(),
	}
	err := SubCmdUsageFunc(c)
	require.NoError(t, err)
}
