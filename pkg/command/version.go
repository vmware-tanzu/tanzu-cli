// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

func newVersionCmd() *cobra.Command {
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Version information",
		Annotations: map[string]string{
			"group": string(plugin.SystemCmdGroup),
		},
		ValidArgsFunction: noMoreCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf(
				"version: %s\nbuildDate: %s\nsha: %s\narch: %s\n",
				buildinfo.Version, buildinfo.Date, buildinfo.SHA, cli.GOARCH)
			return nil
		},
	}

	versionCmd.SetUsageFunc(cli.SubCmdUsageFunc)
	return versionCmd
}
