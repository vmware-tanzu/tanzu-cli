// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	commonauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/uaa"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

func newAPITokenCmd() *cobra.Command {
	apiTokenCmd := &cobra.Command{
		Use:     "api-token",
		Short:   "Manage API Tokens for Tanzu Platform Self-managed",
		Aliases: []string{"apitoken"},
		Annotations: map[string]string{
			"group": string(plugin.SystemCmdGroup),
		},
	}

	apiTokenCmd.SetUsageFunc(cli.SubCmdUsageFunc)
	apiTokenCmd.AddCommand(
		newAPITokenCreateCmd(),
	)

	return apiTokenCmd
}

func newAPITokenCreateCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new API Token for Tanzu Platform Self-managed",
		Aliases: []string{},
		Example: `
    # Create an API Token for the Tanzu Platform Self-managed
    tanzu api-token create

    # Note: The retrieved token can be used as the value of TANZU_API_TOKEN
    # environment variable when running 'tanzu login' for non-interactive workflow.`,
		RunE:              createAPIToken,
		ValidArgsFunction: noMoreCompletions,
	}

	return createCmd
}

func createAPIToken(cmd *cobra.Command, _ []string) (err error) {
	c, err := config.GetActiveContext(types.ContextTypeTanzu)
	if err != nil {
		return errors.New("no active context found for Tanzu Platform. Please login to Tanzu Platform first to generate an API token")
	}
	if c == nil || c.GlobalOpts == nil || c.GlobalOpts.Auth.Issuer == "" {
		return errors.New("invalid active context found for Tanzu Platform. Please login to Tanzu Platform first to generate an API token")
	}
	// Make sure it is of type tanzu with tanzuIdpType as `uaa` else return error
	if idpType, exist := c.AdditionalMetadata[config.TanzuIdpTypeKey]; !exist || idpType != string(config.UAAIdpType) {
		return errors.New("command no supported. Please refer to documentation on how to generate an API token for a public Tanzu Platform endpoint via https://console.tanzu.broadcom.com")
	}

	var token *commonauth.Token
	// If user chooses to use a specific local listener port, use it
	// Also specify the client ID to use for token generation
	loginOptions := []commonauth.LoginOption{
		commonauth.WithListenerPortFromEnv(constants.TanzuCLIOAuthLocalListenerPort),
		commonauth.WithClientIDAndSecret(uaa.GetAlternateClientID(), uaa.GetClientSecret()),
	}

	token, err = uaa.TanzuLogin(c.GlobalOpts.Auth.Issuer, loginOptions...)
	if err != nil {
		return errors.Wrap(err, "unable to login")
	}

	// Get tanzu platform endpoint as best effort from the existing context
	tpEndpoint := "<tanzu-platform-endpoint>"
	if hubEndpoint, exist := c.AdditionalMetadata[config.TanzuHubEndpointKey]; exist && hubEndpoint != nil {
		u, err := url.Parse(hubEndpoint.(string))
		if err == nil {
			tpEndpoint = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		}
	}

	cyanBold := color.New(color.FgCyan).Add(color.Bold)
	bold := color.New(color.Bold)

	fmt.Fprint(cmd.OutOrStdout(), bold.Sprint("==\n\n"))
	fmt.Fprintf(cmd.OutOrStdout(), "%s Your generated API token is: %s\n\n", bold.Sprint("API Token Generation Successful!"), cyanBold.Sprint(token.RefreshToken))
	fmt.Fprintf(cmd.OutOrStdout(), "For Tanzu CLI use in non-interactive settings, set the environment variable %s before authenticating with the command %s\n\n", cyanBold.Sprintf("TANZU_API_TOKEN=%s", token.RefreshToken), cyanBold.Sprintf("tanzu login --endpoint %s", tpEndpoint))
	fmt.Fprint(cmd.OutOrStdout(), "Please copy and save your token securely. Note that you will need to regenerate a new token before expiration time and login again to continue using the CLI.\n")

	return nil
}
