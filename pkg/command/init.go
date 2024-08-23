// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

func newInitCmd() *cobra.Command {
	var initCmd = &cobra.Command{
		Use:    "init",
		Hidden: true,
		Short:  "Initialize the CLI",
		Annotations: map[string]string{
			"group": string(plugin.SystemCmdGroup),
		},
		SilenceErrors:     true,
		ValidArgsFunction: noMoreCompletions,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Currently nothing to initialize.
			// We are keeping this command as it may become useful
			// again in the future.
			log.Success("successfully initialized CLI")
			return nil
		},
	}
	initCmd.SetUsageFunc(cli.SubCmdUsageFunc)
	return initCmd
}
