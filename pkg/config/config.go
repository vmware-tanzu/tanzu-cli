// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config provides functions for the tanzu cli configuration
package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/interfaces"
)

var (
	configClient interfaces.ConfigClientWrapper
	// TODO(prkalle): Update the below CEIP message if necessary
	ceipPromptMsg = `VMware's Customer Experience Improvement Program ("CEIP") provides VMware with information that enables VMware to improve its products and services and fix problems. By choosing to participate in CEIP, you agree that VMware may collect technical information about your use of VMware products and services on a regular basis. This information does not personally identify you.
For more details about the Program, please see http://www.vmware.com/trustvmware/ceip.html

Do you agree to Participate in the Customer Experience Improvement Program? 

`

	// TODO(vuil): Placeholder content, update to actual verbiage
	eulaPromptMsg = `You must agree to the VMware General Terms in order to install software via
Tanzu CLI.  The term of acceptance of the VMware General Terms will cover all
software installed via the Tanzu CLI until/unless there's a change to VMware
General Terms, a major release of the Tanzu CLI has been installed, or the
underlying software distribution registry has been changed.

To view the VMware General Terms, please see https://network.tanzu.vmware.com/legal_document_agreements/1982364

Do you agree to the VMware General Terms?
`
)

func init() {
	configClient = interfaces.NewConfigClient()
}

// ConfigureEnvVariables reads and configures provided environment variables
// as part of tanzu configuration file
func ConfigureEnvVariables() {
	envMap := configClient.GetEnvConfigurations()
	if envMap == nil {
		return
	}
	for variable, value := range envMap {
		// If environment variable is not already set
		// set the environment variable
		if os.Getenv(variable) == "" {
			os.Setenv(variable, value)
		}
	}
}

// ConfigureCEIPOptIn checks and configures the User CEIP Opt-in choice in the tanzu configuration file
func ConfigureCEIPOptIn() error {
	ceipOptInConfigVal, _ := configlib.GetCEIPOptIn()
	// If CEIP Opt-In config parameter is already set, do nothing
	if ceipOptInConfigVal != "" {
		return nil
	}

	ceipOptInUserVal, err := getCEIPUserOptIn()
	if err != nil {
		return errors.Wrapf(err, "failed to get CEIP Opt-In status")
	}

	err = configlib.SetCEIPOptIn(strconv.FormatBool(ceipOptInUserVal))
	if err != nil {
		return errors.Wrapf(err, "failed to update the CEIP Opt-In status")
	}

	return nil
}

func getCEIPUserOptIn() (bool, error) {
	var ceipOptIn string
	optInPromptChoiceEnvVal := os.Getenv(constants.CEIPOptInUserPromptAnswer)
	if optInPromptChoiceEnvVal != "" {
		return strings.EqualFold(optInPromptChoiceEnvVal, "Yes"), nil
	}

	// prompt user and record their choice
	err := component.Prompt(
		&component.PromptConfig{
			Message: ceipPromptMsg,
			Options: []string{"Yes", "No"},
			Default: "Yes",
		},
		&ceipOptIn,
	)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(ceipOptIn, "Yes"), nil
}

// ConfigureEULA checks and configures the user's EULA acceptance status
func ConfigureEULA(alwaysPrompt bool) error {
	configVal, _ := configlib.GetEULAStatus()

	// Unless forced to always prompt, it is a no-op if EULA is already accepted.
	if !alwaysPrompt && configVal == configlib.EULAStatusAccepted {
		return nil
	}

	accepted, err := promptForEULA()
	if err != nil {
		return errors.Wrapf(err, "failed to get EULA status")
	}

	status := configlib.EULAStatusShown
	if accepted {
		status = configlib.EULAStatusAccepted
	}

	err = configlib.SetEULAStatus(status)
	if err != nil {
		return errors.Wrapf(err, "failed to update the EULA status to %s", status)
	}

	return nil
}

func promptForEULA() (bool, error) {
	var eulaAccepted string

	eulaPromptChoiceEnvVal := os.Getenv(constants.EULAPromptAnswer)
	if eulaPromptChoiceEnvVal != "" {
		return strings.EqualFold(eulaPromptChoiceEnvVal, "Yes"), nil
	}

	// prompt user and record their choice
	err := component.Prompt(
		&component.PromptConfig{
			Message: eulaPromptMsg,
			Options: []string{"Yes", "No"},
			Default: "Yes",
		},
		&eulaAccepted,
	)
	if err != nil {
		return false, errors.Wrapf(err, "prompt failed")
	}

	return strings.EqualFold(eulaAccepted, "Yes"), nil
}
