// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

// CeipOptOutStatus and CeipOptInStatus are constants for the CEIP opt-in/out verbiage
const (
	CeipOptInStatus  = "Opt-in"
	CeipOptOutStatus = "Opt-out"
)

// Note(TODO:prkalle): The below ceip-participation command(experimental) added may be removed in the next release,
//       If we decide to fold this functionality into existing 'tanzu telemetry' plugin

func newCEIPParticipationCmd() *cobra.Command {
	var ceipParticipationCmd = &cobra.Command{
		Use:   "ceip-participation",
		Short: "Manage VMware's Customer Experience Improvement Program (CEIP) Participation (subject to change)",
		Long: "Manage VMware's Customer Experience Improvement Program (CEIP) participation which provides VMware with " +
			"information that enables VMware to improve its products and services and fix problems (subject to change)",
		Aliases: []string{"ceip"},
		Annotations: map[string]string{
			"group": string(plugin.SystemCmdGroup),
		},
		Hidden: true,
	}
	ceipParticipationCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	ceipParticipationCmd.AddCommand(
		newCEIPParticipationSetCmd(),
		newCEIPParticipationGetCmd(),
	)

	return ceipParticipationCmd
}

func newCEIPParticipationSetCmd() *cobra.Command {
	var setCmd = &cobra.Command{
		Use:               "set OPT_IN_BOOL",
		Short:             "Set the opt-in preference for CEIP (subject to change)",
		Long:              "Set the opt-in preference for CEIP (subject to change)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeCeipSet,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !strings.EqualFold(args[0], "true") && !strings.EqualFold(args[0], "false") {
				return errors.Errorf("incorrect boolean argument: %q", args[0])
			}
			err := configlib.SetCEIPOptIn(strconv.FormatBool(strings.EqualFold(args[0], "true")))
			if err != nil {
				return errors.Wrapf(err, "failed to update the configuration")
			}
			return nil
		},
	}

	return setCmd
}

func newCEIPParticipationGetCmd() *cobra.Command {
	var getCmd = &cobra.Command{
		Use:               "get",
		Short:             "Get the current CEIP opt-in status (subject to change)",
		Long:              "Get the current CEIP opt-in status (subject to change)",
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			optInVal, err := configlib.GetCEIPOptIn()
			if err != nil {
				return errors.Wrapf(err, "failed to get the CEIP opt-in status")
			}
			ceipStatus := ""
			if strings.EqualFold(optInVal, "true") {
				ceipStatus = CeipOptInStatus
			} else {
				ceipStatus = CeipOptOutStatus
			}
			t := component.NewOutputWriter(cmd.OutOrStdout(), outputFormat, "CEIP-Status")
			t.AddRow(ceipStatus)
			t.Render()
			return nil
		},
	}

	return getCmd
}

func completeCeipSet(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Keep the "true" choice first by using ShellCompDirectiveKeepOrder (may not work for all shells)
	// This is just to make it "easier" to opt-in :-)
	return []string{
			"true\tAccept to participate",
			"false\tRefuse to participate"},
		cobra.ShellCompDirectiveKeepOrder | cobra.ShellCompDirectiveNoFileComp
}
