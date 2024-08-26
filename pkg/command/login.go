// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

var loginEndpoint string

func newLoginCmd() *cobra.Command {
	loginCmd := &cobra.Command{
		Use:     "login",
		Short:   "Login to Tanzu Platform for Kubernetes",
		Aliases: []string{"lo", "logins"},
		Annotations: map[string]string{
			"group": string(plugin.SystemCmdGroup),
		},
		ValidArgsFunction: noMoreCompletions,
		RunE:              login,
	}

	// "endpoint" variable from context.go cannot be used as default value varies for login command
	loginCmd.Flags().StringVar(&loginEndpoint, "endpoint", "https://api.tanzu.cloud.vmware.com", "endpoint to login to")
	utils.PanicOnErr(loginCmd.RegisterFlagCompletionFunc("endpoint", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cobra.AppendActiveHelp(nil, "Please enter the endpoint for which to create the context"), cobra.ShellCompDirectiveNoFileComp
	}))
	loginCmd.Flags().StringVar(&apiToken, "api-token", "", "API token for the SaaS login")
	utils.PanicOnErr(loginCmd.RegisterFlagCompletionFunc("api-token", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cobra.AppendActiveHelp(nil, fmt.Sprintf("Please enter your api-token (you can instead set the variable %s)", config.EnvAPITokenKey)), cobra.ShellCompDirectiveNoFileComp
	}))
	loginCmd.Flags().BoolVar(&staging, "staging", false, "use CSP staging issuer")
	loginCmd.Flags().StringVar(&endpointCACertPath, "endpoint-ca-certificate", "", "path to the endpoint public certificate")
	loginCmd.Flags().BoolVar(&skipTLSVerify, "insecure-skip-tls-verify", false, "skip endpoint's TLS certificate verification")
	loginCmd.Flags().StringVar(&tanzuHubEndpoint, "tanzu-hub-endpoint", "", "customize the Tanzu Hub endpoint associated with the context")

	utils.PanicOnErr(loginCmd.Flags().MarkHidden("api-token"))
	utils.PanicOnErr(loginCmd.Flags().MarkHidden("staging"))
	utils.PanicOnErr(loginCmd.Flags().MarkHidden("tanzu-hub-endpoint"))
	loginCmd.SetUsageFunc(cli.SubCmdUsageFunc)
	loginCmd.MarkFlagsMutuallyExclusive("endpoint-ca-certificate", "insecure-skip-tls-verify")

	loginCmd.Example = `
    # Login to Tanzu
    tanzu login

    # Login to Tanzu using non-default endpoint
    tanzu login --endpoint "https://login.example.com"

    # Login to Tanzu by using the provided CA Bundle for TLS verification
    tanzu login --endpoint https://test.example.com[:port] --endpoint-ca-certificate /path/to/ca/ca-cert

    # Login to Tanzu by explicit request to skip TLS verification (this is insecure)
    tanzu login --endpoint https://test.example.com[:port] --insecure-skip-tls-verify

    Note:
       To login to Tanzu an API Key is optional. If provided using the TANZU_API_TOKEN environment
       variable, it will be used. Otherwise, the CLI will attempt to log in interactively to the user's default Cloud Services
       organization. You can override or choose a custom organization by setting the TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID
       environment variable with the custom organization ID value. More information regarding organizations in Cloud Services
       and how to obtain the organization ID can be found at
       https://docs.vmware.com/en/VMware-Cloud-services/services/Using-VMware-Cloud-Services/GUID-CF9E9318-B811-48CF-8499-9419997DC1F8.html
       Also, more information on logging into Tanzu Platform Platform for Kubernetes and using
       interactive login in terminal based hosts (without browser) can be found at
       https://github.com/vmware-tanzu/tanzu-cli/blob/main/docs/quickstart/quickstart.md#logging-into-tanzu-platform-for-kubernetes
`
	return loginCmd
}

func login(_ *cobra.Command, _ []string) (err error) {
	// assign the loginEndpoint to context endpoint option variable
	endpoint = loginEndpoint

	// generate random context name to skip the prompts and later update the
	// context name with organization name acquired after successful authentication
	ctxName = uuid.New().String()
	ctx, err := createContextUsingContextType(contextTanzu)
	if err != nil {
		return err
	}

	err = globalTanzuLogin(ctx, prepareTanzuContextName)
	if err != nil {
		return err
	}

	// if user performs re-login having an existing context with active resource set to project/space/clustergroup
	// update the kubeconfig because "globalTanzuLogin" updates the kubeconfig to point to organization only,
	if err := updateKubeConfigForContext(ctx); err != nil {
		return nil
	}

	// save the context since "ClusterOpts.Context" (kubecontext) in the CLI context could be modified.
	err = config.SetContext(ctx, false)
	if err != nil {
		return errors.Wrap(err, "failed updating the context %q after kubeconfig update")
	}

	// TODO: uncomment the below context plugin sync call once context scope plugin support
	//       is implemented for tanzu context(Tanzu Platform for Kubernetes)
	// Sync all required plugins
	// _ = syncContextPlugins(cmd, ctx.ContextType, ctx.Name)

	return nil
}

func updateKubeConfigForContext(c *configtypes.Context) error {
	projNameStr := getString(c.AdditionalMetadata[config.ProjectNameKey])
	projIDStr := getString(c.AdditionalMetadata[config.ProjectIDKey])
	spaceNameStr := getString(c.AdditionalMetadata[config.SpaceNameKey])
	clusterGroupNameNameStr := getString(c.AdditionalMetadata[config.ClusterGroupNameKey])

	return updateTanzuContextKubeconfig(c, projNameStr, projIDStr, spaceNameStr, clusterGroupNameNameStr)
}

func getString(data interface{}) string {
	if _, ok := data.(string); !ok {
		return ""
	}
	return data.(string)
}

// prepareTanzuContextName returns the context name given organization name,endpoint and staging details
// pre-req orgName and endpoint is non-empty string
func prepareTanzuContextName(orgName, endpoint string, isStaging bool) string {
	contextName := strings.Replace(orgName, " ", "_", -1)
	if isStaging {
		contextName += "-staging"
	}

	if endpoint != defaultTanzuEndpoint {
		// append just 8 chars of sha to the context name
		contextName += fmt.Sprintf("-%s", hashString(endpoint)[:8])
	}
	return contextName
}

func hashString(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
