// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"github.com/spf13/cobra"

	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/config"
)

func newEULACmd() *cobra.Command {
	var eulaCmd = &cobra.Command{
		Use:   "eula",
		Short: "Manage EULA acceptance",
		Long:  "Manage EULA acceptance for Tanzu CLI use",
	}
	eulaCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	showEULACmd := newShowEULACmd()
	acceptEULACmd := newAcceptEULACmd()

	eulaCmd.AddCommand(
		showEULACmd,
		acceptEULACmd,
	)

	return eulaCmd
}

func newShowEULACmd() *cobra.Command {
	var showEULACmd = &cobra.Command{
		Use:               "show",
		Short:             "Present EULA",
		Long:              "Present EULA for review",
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.ConfigureEULA(true)
		},
	}
	return showEULACmd
}

func newAcceptEULACmd() *cobra.Command {
	var acceptEULACmd = &cobra.Command{
		Use:               "accept",
		Short:             "Accept the EULA",
		Long:              "Accept the EULA for Tanzu CLI non-interactively.",
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := configlib.SetEULAStatus(configlib.EULAStatusAccepted)
			if err != nil {
				return err
			}
			log.Successf("Marking agreement as accepted.")
			return nil
		},
	}
	return acceptEULACmd
}
