// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clientauthv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	commonauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/csp"
	tanzuauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tanzu"
	tkgauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tkg"
	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/uaa"
	kubecfg "github.com/vmware-tanzu/tanzu-cli/pkg/auth/utils/kubeconfig"
	wcpauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/wcp"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

var (
	stderrOnly, forceCSP, staging, onlyCurrent, skipTLSVerify, showAllColumns, shortCtx    bool
	ctxName, endpoint, apiToken, kubeConfig, kubeContext, getOutputFmt, endpointCACertPath string

	tanzuHubEndpoint, tanzuTMCEndpoint, tanzuUCPEndpoint, tanzuAuthEndpoint string

	projectStr, projectIDStr, spaceStr, clustergroupStr string
	contextTypeStr                                      string
)

const (
	knownGlobalHost    = "cloud.vmware.com"
	isPinnipedEndpoint = "isPinnipedEndpoint"

	contextNotExistsForContextType      = "The provided context '%v' does not exist or is not active for the given context type '%v'"
	noActiveContextExistsForContextType = "There is no active context for the given context type '%v'"
	contextNotActiveOrNotExists         = "The provided context '%v' is not active or does not exist"
	contextForContextTypeSetInactive    = "The context '%v' of type '%v' has been set as inactive"
	deactivatingPlugin                  = "Deactivating plugin '%v:%v' for context '%v'"

	invalidTargetErrorForContextCommands = "invalid target specified. Please specify a correct value for the `--target` flag from 'kubernetes[k8s]/mission-control[tmc]'"
	invalidContextType                   = "invalid context type specified. Please specify a correct value for the `--type/-t` flag from 'kubernetes[k8s]/mission-control[tmc]/tanzu'"
	invalidIdpType                       = "invalid IDP type found."
)

// constants that define context creation types
const (
	contextMissionControl     ContextCreationType = "Mission Control"
	contextK8SClusterEndpoint ContextCreationType = "Kubernetes (Cluster Endpoint)"
	contextLocalKubeconfig    ContextCreationType = "Kubernetes (Local Kubeconfig)"
	contextTanzu              ContextCreationType = "Tanzu"
)

type ContextCreationType string

const NA = "n/a"

func newContextCmd() *cobra.Command {
	contextCmd := &cobra.Command{
		Use:     "context",
		Short:   "Configure and manage contexts for the Tanzu CLI",
		Aliases: []string{"ctx", "contexts"},
		Annotations: map[string]string{
			"group": string(plugin.SystemCmdGroup),
		},
	}

	contextCmd.SetUsageFunc(cli.SubCmdUsageFunc)
	contextCmd.AddCommand(
		newCreateCtxCmd(),
		newListCtxCmd(),
		newGetCtxCmd(),
		newCurrentCtxCmd(),
		newDeleteCtxCmd(),
		newUseCtxCmd(),
		newUnsetCtxCmd(),
		newGetCtxTokenCmd(),
		newUpdateCtxCmd(),
	)

	return contextCmd
}

func newCreateCtxCmd() *cobra.Command {
	var createCtxCmd = &cobra.Command{
		Use:               "create CONTEXT_NAME",
		Short:             "Create a Tanzu CLI context",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeCreateCtx,
		RunE:              createCtx,
		Example: `
    # Create a TKG management cluster context using endpoint and type (--type is optional, if not provided the CLI will infer the type from the endpoint)
    tanzu context create mgmt-cluster --endpoint https://k8s.example.com[:port] --type k8s

    # Create a TKG management cluster context using endpoint
    tanzu context create mgmt-cluster --endpoint https://k8s.example.com[:port]

    # Create a TKG management cluster context using kubeconfig path and context
    tanzu context create mgmt-cluster --kubeconfig path/to/kubeconfig --kubecontext kubecontext

    # Create a TKG management cluster context by using the provided CA Bundle for TLS verification:
    tanzu context create mgmt-cluster --endpoint https://k8s.example.com[:port] --endpoint-ca-certificate /path/to/ca/ca-cert

    # Create a TKG management cluster context by explicit request to skip TLS verification, which is insecure:
    tanzu context create mgmt-cluster --endpoint https://k8s.example.com[:port] --insecure-skip-tls-verify

    # Create a TKG management cluster context using default kubeconfig path and a kubeconfig context
    tanzu context create mgmt-cluster --kubecontext kubecontext

    # Create a TMC(mission-control) context using endpoint and type 
    tanzu context create mytmc --endpoint tmc.example.com:443 --type tmc

    # Create a Tanzu context with the default endpoint (--type is not necessary for the default endpoint)
    tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com

    # Create a Tanzu context (--type is needed for a non-default endpoint)
    tanzu context create mytanzu --endpoint https://non-default.tanzu.endpoint.com --type tanzu

    # Create a Tanzu context by using the provided CA Bundle for TLS verification:
    tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com  --endpoint-ca-certificate /path/to/ca/ca-cert

    # Create a Tanzu context but skip TLS verification (this is insecure):
    tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com --insecure-skip-tls-verify

    Notes: 
    1. TMC context: To create Mission Control (TMC) context an API Key is required. It can be provided using the
       TANZU_API_TOKEN environment variable or entered during context creation.
    2. Tanzu context: To create Tanzu context an API Key is optional. If provided using the TANZU_API_TOKEN environment
       variable, it will be used. Otherwise, the CLI will attempt to log in interactively to the user's default Cloud Services
       organization. You can override or choose a custom organization by setting the TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID
       environment variable with the custom organization ID value. More information regarding organizations in Cloud Services
       and how to obtain the organization ID can be found at
       https://docs.vmware.com/en/VMware-Cloud-services/services/Using-VMware-Cloud-Services/GUID-CF9E9318-B811-48CF-8499-9419997DC1F8.html
       Also, more information on creating tanzu context and using interactive login in terminal based hosts (without browser) can be found at
       https://github.com/vmware-tanzu/tanzu-cli/blob/main/docs/quickstart/quickstart.md#creating-and-connecting-to-a-new-context

    [*] : Users have two options to create a kubernetes cluster context. They can choose the control
    plane option by providing 'endpoint', or use the kubeconfig for the cluster by providing
    'kubeconfig' and 'context'. If only '--context' is set and '--kubeconfig' is not, the
    $KUBECONFIG env variable will be used and, if the $KUBECONFIG env is also not set, the default
    kubeconfig file ($HOME/.kube/config) will be used.`,
	}

	createCtxCmd.Flags().StringVar(&ctxName, "name", "", "name of the context")
	utils.PanicOnErr(createCtxCmd.Flags().MarkDeprecated("name", "it has been replaced by using an argument to the command"))

	createCtxCmd.Flags().StringVar(&endpoint, "endpoint", "", "endpoint to create a context for")
	utils.PanicOnErr(createCtxCmd.RegisterFlagCompletionFunc("endpoint", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cobra.AppendActiveHelp(nil, "Please enter the endpoint for which to create the context"), cobra.ShellCompDirectiveNoFileComp
	}))

	createCtxCmd.Flags().StringVar(&apiToken, "api-token", "", "API token for the SaaS context")
	utils.PanicOnErr(createCtxCmd.RegisterFlagCompletionFunc("api-token", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cobra.AppendActiveHelp(nil, fmt.Sprintf("Please enter your api-token (you can instead set the variable %s)", config.EnvAPITokenKey)), cobra.ShellCompDirectiveNoFileComp
	}))

	// Shell completion for this flag is the default behavior of doing file completion
	createCtxCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "path to the kubeconfig file; valid only if user doesn't choose 'endpoint' option.(See [*])")

	createCtxCmd.Flags().StringVar(&kubeContext, "kubecontext", "", "the context in the kubeconfig to use; valid only if user doesn't choose 'endpoint' option.(See [*]) ")
	utils.PanicOnErr(createCtxCmd.RegisterFlagCompletionFunc("kubecontext", completeKubeContext))

	createCtxCmd.Flags().BoolVar(&stderrOnly, "stderr-only", false, "send all output to stderr rather than stdout")
	createCtxCmd.Flags().BoolVar(&forceCSP, "force-csp", false, "force the context to use CSP auth")
	createCtxCmd.Flags().BoolVar(&staging, "staging", false, "use CSP staging issuer")

	// Shell completion for this flag is the default behavior of doing file completion
	createCtxCmd.Flags().StringVar(&endpointCACertPath, "endpoint-ca-certificate", "", "path to the endpoint public certificate")
	createCtxCmd.Flags().BoolVar(&skipTLSVerify, "insecure-skip-tls-verify", false, "skip endpoint's TLS certificate verification")
	createCtxCmd.Flags().StringVarP(&contextTypeStr, "type", "t", "", "type of context to create (kubernetes[k8s]/mission-control[tmc]/tanzu)")
	utils.PanicOnErr(createCtxCmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sContextType, compTanzuContextType, compTMCContextType}, cobra.ShellCompDirectiveNoFileComp
	}))

	utils.PanicOnErr(createCtxCmd.Flags().MarkHidden("api-token"))
	utils.PanicOnErr(createCtxCmd.Flags().MarkHidden("stderr-only"))
	utils.PanicOnErr(createCtxCmd.Flags().MarkHidden("force-csp"))
	utils.PanicOnErr(createCtxCmd.Flags().MarkHidden("staging"))

	createCtxCmd.MarkFlagsMutuallyExclusive("endpoint", "kubecontext")
	createCtxCmd.MarkFlagsMutuallyExclusive("endpoint", "kubeconfig")
	createCtxCmd.MarkFlagsMutuallyExclusive("endpoint-ca-certificate", "insecure-skip-tls-verify")

	return createCtxCmd
}

func createCtx(cmd *cobra.Command, args []string) (err error) {
	// The context name is an optional argument to allow for the prompt to be used
	if len(args) > 0 {
		if ctxName != "" {
			return fmt.Errorf("cannot specify the context name as an argument and with the --name flag at the same time")
		}
		ctxName = args[0]
	}
	if err := validateContextCreateFlagValues(); err != nil {
		return err
	}
	if !configtypes.IsValidContextType(contextTypeStr) {
		return errors.New(invalidContextType)
	}

	ctx, err := createNewContext()
	if err != nil {
		return err
	}
	if ctx.ContextType == configtypes.ContextTypeK8s {
		err = k8sLogin(ctx)
	} else if ctx.ContextType == configtypes.ContextTypeTanzu {
		// Tanzu control plane login
		err = globalTanzuLogin(ctx, nil)
	} else {
		err = globalLogin(ctx)
	}

	if err != nil {
		return err
	}

	// TODO: update the below conditional check (and in login command) after context scope plugin support
	//       is implemented for tanzu context(Tanzu Platform for Kubernetes)
	// Sync all required plugins
	if ctx.ContextType != configtypes.ContextTypeTanzu {
		if err := syncContextPlugins(cmd, ctx.ContextType, ctxName); err != nil {
			log.Warningf("unable to automatically sync the plugins recommended by the new context. Please run 'tanzu plugin sync' to sync plugins manually, error: '%v'", err.Error())
		}
	}
	return nil
}

func validateContextCreateFlagValues() error {
	if contextTypeStr == string(configtypes.ContextTypeTanzu) && kubeConfig != "" {
		return fmt.Errorf("the '–-kubeconfig' flag is not applicable when creating a context of type 'tanzu'")
	}
	if contextTypeStr == string(configtypes.ContextTypeTanzu) && kubeContext != "" {
		return fmt.Errorf("the '–-kubecontext' flag is not applicable when creating a context of type 'tanzu'")
	}
	return nil
}

// syncContextPlugins syncs the plugins for the given context type
func syncContextPlugins(cmd *cobra.Command, contextType configtypes.ContextType, ctxName string) error {
	disablePluginSync, _ := strconv.ParseBool(os.Getenv(constants.SkipAutoInstallOfContextRecommendedPlugins))
	if disablePluginSync {
		return nil
	}

	plugins, err := pluginmanager.DiscoverPluginsForContextType(contextType)
	if err != nil {
		return err
	}

	if len(plugins) == 0 {
		log.Success("No recommended plugins found.")
		return nil
	}

	// update plugins installation status
	pluginmanager.UpdatePluginsInstallationStatus(plugins)

	// sort the plugins based on the plugin name
	sort.Sort(discovery.DiscoveredSorter(plugins))

	pluginsNeedToBeInstalled := []discovery.Discovered{}
	for idx := range plugins {
		if plugins[idx].Status == common.PluginStatusNotInstalled || plugins[idx].Status == common.PluginStatusUpdateAvailable {
			pluginsNeedToBeInstalled = append(pluginsNeedToBeInstalled, plugins[idx])
		}
	}

	if len(pluginsNeedToBeInstalled) == 0 {
		log.Success("All recommended plugins are already installed and up-to-date.")
		return nil
	}

	errList := make([]error, 0)
	log.Infof("Installing the following plugins recommended by context '%s':", ctxName)
	displayToBeInstalledPluginsAsTable(plugins, cmd.ErrOrStderr())
	for i := range pluginsNeedToBeInstalled {
		err = pluginmanager.InstallStandalonePlugin(pluginsNeedToBeInstalled[i].Name, pluginsNeedToBeInstalled[i].RecommendedVersion, pluginsNeedToBeInstalled[i].Target)
		if err != nil {
			errList = append(errList, err)
		}
	}
	err = kerrors.NewAggregate(errList)
	if err == nil {
		log.Success("Successfully installed all recommended plugins.")
	}

	return err
}

// displayToBeInstalledPluginsAsTable takes a list of plugins and displays the plugin info as a table
func displayToBeInstalledPluginsAsTable(plugins []discovery.Discovered, writer io.Writer) {
	outputPlugins := component.NewOutputWriterWithOptions(writer, outputFormat, []component.OutputWriterOption{}, "Name", "Target", "Current", "Installing")
	outputPlugins.MarkDynamicKeys("Current")
	for i := range plugins {
		if plugins[i].Status == common.PluginStatusNotInstalled || plugins[i].Status == common.PluginStatusUpdateAvailable {
			outputPlugins.AddRow(plugins[i].Name, plugins[i].Target, plugins[i].InstalledVersion, plugins[i].RecommendedVersion)
		}
	}
	outputPlugins.Render()
}

func isGlobalContext(endpoint string) bool {
	if strings.Contains(endpoint, knownGlobalHost) {
		return true
	}
	if forceCSP {
		return true
	}
	return false
}

func getPromptOpts() []component.PromptOpt {
	var promptOpts []component.PromptOpt
	if stderrOnly {
		// This uses stderr because it needs to work inside the kubectl exec plugin flow where stdout is reserved.
		promptOpts = append(promptOpts, component.WithStdio(os.Stdin, os.Stderr, os.Stderr))
	}
	// Add default validations, required
	promptOpts = append(promptOpts, component.WithValidator(survey.Required), component.WithValidator(component.NoOnlySpaces))

	return promptOpts
}

func createNewContext() (context *configtypes.Context, err error) {
	var ctxCreationType ContextCreationType
	contextType := getContextType()

	if (contextType == configtypes.ContextTypeTanzu) || (endpoint != "" && isTanzuPlatformSaaSEndpoint(endpoint)) {
		ctxCreationType = contextTanzu
	} else if (contextType == configtypes.ContextTypeTMC) || (endpoint != "" && isGlobalContext(endpoint)) {
		ctxCreationType = contextMissionControl
	} else if endpoint != "" {
		// user provided command line option endpoint is provided that is not globalTanzu or GlobalContext=> it is Kubernetes(Cluster Endpoint) type
		ctxCreationType = contextK8SClusterEndpoint
	} else if kubeContext != "" {
		// user provided command line option kubeContext is provided => it is Kubernetes(Local Kubeconfig) type
		ctxCreationType = contextLocalKubeconfig
	} else if contextType == configtypes.ContextTypeK8s {
		// If user provided only command line option type as "kubernetes" without any other flags to infer
		// ask user for kubernetes context type("Cluster Endpoint" or "Local Kubeconfig")
		ctxCreationType, err = promptKubernetesContextType()
		if err != nil {
			return context, err
		}
	}

	// if user not provided command line options to infer cluster creation type, prompt user
	if ctxCreationType == "" {
		ctxCreationType, err = promptContextType()
		if err != nil {
			return context, err
		}
	}

	return createContextUsingContextType(ctxCreationType)
}

func createContextUsingContextType(ctxCreationType ContextCreationType) (context *configtypes.Context, err error) {
	var ctxCreateFunc func() (*configtypes.Context, error)
	switch ctxCreationType {
	case contextMissionControl:
		ctxCreateFunc = createContextWithTMCEndpoint
	case contextK8SClusterEndpoint:
		ctxCreateFunc = createContextWithClusterEndpoint
	case contextLocalKubeconfig:
		ctxCreateFunc = createContextWithKubeconfig
	case contextTanzu:
		ctxCreateFunc = createContextWithTanzuEndpoint
	}
	return ctxCreateFunc()
}

func createContextWithKubeconfig() (context *configtypes.Context, err error) {
	promptOpts := getPromptOpts()
	if kubeConfig == "" && kubeContext == "" {
		err = component.Prompt(
			&component.PromptConfig{
				Message: "Enter path to kubeconfig (if any)",
			},
			&kubeConfig,
			promptOpts...,
		)
		if err != nil {
			return context, err
		}
	} else if kubeConfig == "" {
		kubeConfig = kubecfg.GetDefaultKubeConfigFile()
	}
	kubeConfig = strings.TrimSpace(kubeConfig)

	if kubeConfig != "" && kubeContext == "" {
		err = component.Prompt(
			&component.PromptConfig{
				Message: "Enter kube context to use",
			},
			&kubeContext,
			promptOpts...,
		)
		if err != nil {
			return context, err
		}
	}
	kubeContext = strings.TrimSpace(kubeContext)

	if ctxName == "" {
		err = component.Prompt(
			&component.PromptConfig{
				Message: "Give the context a name",
			},
			&ctxName,
			promptOpts...,
		)
		if err != nil {
			return context, err
		}
	}
	ctxName = strings.TrimSpace(ctxName)
	exists, err := config.ContextExists(ctxName)
	if err != nil {
		return context, err
	}
	if exists {
		err = fmt.Errorf("context %q already exists", ctxName)
		return context, err
	}

	context = &configtypes.Context{
		Name:        ctxName,
		ContextType: configtypes.ContextTypeK8s,
		ClusterOpts: &configtypes.ClusterServer{
			Path:                kubeConfig,
			Context:             kubeContext,
			Endpoint:            endpoint,
			IsManagementCluster: true,
		},
	}
	return context, err
}

func createContextWithTMCEndpoint() (context *configtypes.Context, err error) {
	if endpoint == "" {
		endpoint, err = promptEndpoint("")
		if err != nil {
			return context, err
		}
	}
	if ctxName == "" {
		ctxName, err = promptContextName("")
		if err != nil {
			return context, err
		}
	}
	exists, err := config.ContextExists(ctxName)
	if err != nil {
		return context, err
	}
	if exists {
		err = fmt.Errorf("context %q already exists", ctxName)
		return context, err
	}

	if os.Getenv(constants.E2ETestEnvironment) != "true" && (strings.HasPrefix(endpoint, "https:") || strings.HasPrefix(endpoint, "http:")) {
		return nil, errors.Errorf("TMC endpoint URL %s should not contain http or https scheme. It should be of the format host[:port]", endpoint)
	}

	context = &configtypes.Context{
		Name:        ctxName,
		ContextType: configtypes.ContextTypeTMC,
		GlobalOpts:  &configtypes.GlobalServer{Endpoint: sanitizeEndpoint(endpoint)},
	}

	return context, err
}

// createContextWithClusterEndpoint creates context for cluster endpoint with pinniped auth
func createContextWithClusterEndpoint() (context *configtypes.Context, err error) {
	if endpoint == "" {
		endpoint, err = promptEndpoint("")
		if err != nil {
			return context, err
		}
	}
	if ctxName == "" {
		ctxName, err = promptContextName("")
		if err != nil {
			return context, err
		}
	}
	exists, err := config.ContextExists(ctxName)
	if err != nil {
		return context, err
	}
	if exists {
		err = fmt.Errorf("context %q already exists", ctxName)
		return context, err
	}

	// TKGKubeconfigFetcher would detect the endpoint is TKGm/TKGs and then fetch the pinniped kubeconfig to create a context
	tkf := NewTKGKubeconfigFetcher(endpoint, endpointCACertPath, skipTLSVerify)
	kubeConfig, kubeContext, err = tkf.GetPinnipedKubeconfig()
	if err != nil {
		return context, err
	}

	context = &configtypes.Context{
		Name:        ctxName,
		ContextType: configtypes.ContextTypeK8s,
		ClusterOpts: &configtypes.ClusterServer{
			Path:                kubeConfig,
			Context:             kubeContext,
			Endpoint:            endpoint,
			IsManagementCluster: true,
		},
		AdditionalMetadata: map[string]interface{}{
			isPinnipedEndpoint: true,
		},
	}
	return context, err
}

// Unless forced to use CSP, use a heuristic based on the matching endpoint
// with known patterns to determined the IDP used for authentication
func getIDPType(endpoint string) config.IdpType {
	if isTanzuPlatformSaaSEndpoint(endpoint) {
		return config.CSPIdpType
	}

	return config.UAAIdpType
}

func createContextWithTanzuEndpoint() (context *configtypes.Context, err error) {
	if endpoint == "" {
		endpoint, err = promptEndpoint(centralconfig.DefaultTanzuPlatformEndpoint)
		if err != nil {
			return context, err
		}
	}
	if ctxName == "" {
		ctxName, err = promptContextName("")
		if err != nil {
			return context, err
		}
	}

	exists, err := config.ContextExists(ctxName)
	if err != nil {
		return context, err
	}
	if exists {
		err = fmt.Errorf("context %q already exists", ctxName)
		return context, err
	}

	// At this point, we can assume that `endpoint` variable has been configured based on the user input
	// So, use that as tanzuPlatform endpoint and setup other service endpoint variables
	err = configureTanzuPlatformServiceEndpoints(endpoint)
	if err != nil {
		return context, err
	}

	// Tanzu context would have both CSP(GlobalOpts) auth details and kubeconfig(ClusterOpts),
	context = &configtypes.Context{
		Name:        ctxName,
		ContextType: configtypes.ContextTypeTanzu,
		GlobalOpts:  &configtypes.GlobalServer{Endpoint: tanzuUCPEndpoint},
		ClusterOpts: &configtypes.ClusterServer{},
		AdditionalMetadata: map[string]interface{}{
			config.TanzuMissionControlEndpointKey: tanzuTMCEndpoint,
			config.TanzuHubEndpointKey:            tanzuHubEndpoint,
			config.TanzuIdpTypeKey:                getIDPType(endpoint),
		},
	}
	if tanzuAuthEndpoint != "" {
		context.AdditionalMetadata[config.TanzuAuthEndpointKey] = tanzuAuthEndpoint
	}
	return context, err
}

func globalLogin(c *configtypes.Context) (err error) {
	apiTokenValue, apiTokenExists := os.LookupEnv(config.EnvAPITokenKey)
	if apiTokenExists {
		log.Info("API token env var is set")
	} else {
		fmt.Fprintln(os.Stderr)
		log.Info("The API key can be provided by setting the TANZU_API_TOKEN environment variable")
		apiTokenValue, err = promptAPIToken("TMC")
		if err != nil {
			return err
		}
	}
	_, err = doCSPAPITokenAuthAndUpdateContext(c, apiTokenValue)
	if err != nil {
		return err
	}

	err = config.AddContext(c, true)
	if err != nil {
		return err
	}

	// format
	fmt.Fprintln(os.Stderr)
	log.Success("successfully created a TMC context")
	return nil
}

func globalTanzuLogin(c *configtypes.Context, generateContextNameFunc func(orgName, endpoint string, isStaging bool) string) error {
	if c.AdditionalMetadata[config.TanzuIdpTypeKey] == config.CSPIdpType {
		return globalTanzuLoginCSP(c, generateContextNameFunc)
	} else if c.AdditionalMetadata[config.TanzuIdpTypeKey] == config.UAAIdpType {
		return globalTanzuLoginUAA(c, generateContextNameFunc)
	}
	return errors.New(invalidIdpType)
}

func globalTanzuLoginUAA(c *configtypes.Context, generateContextNameFunc func(orgName, endpoint string, isStaging bool) string) error {
	uaaEndpoint := c.AdditionalMetadata[config.TanzuAuthEndpointKey].(string)
	log.V(7).Infof("Login to UAA endpoint: %s", uaaEndpoint)

	claims, err := doInteractiveLoginAndUpdateContext(c, uaaEndpoint)
	if err != nil {
		return err
	}

	// UAA-based authentication does not provide org id or name yet
	orgName := ""

	if err := updateContextOnTanzuLogin(c, generateContextNameFunc, claims, orgName); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr)
	log.Successf("Successfully logged in to '%s' and created a tanzu context", endpoint)

	return nil
}

func globalTanzuLoginCSP(c *configtypes.Context, generateContextNameFunc func(orgName, endpoint string, isStaging bool) string) error {
	log.V(7).Infof("Login to CSP endpoint: %s", endpoint)
	claims, err := doCSPAuthentication(c)
	if err != nil {
		return err
	}

	orgName, err := getCSPOrganizationName(c, claims)
	if err != nil {
		return err
	}

	if err := updateContextOnTanzuLogin(c, generateContextNameFunc, claims, orgName); err != nil {
		return err
	}

	// format
	fmt.Fprintln(os.Stderr)
	log.Successf("Successfully logged into '%s' organization and created a tanzu context", orgName)

	// log warning message if the Tanzu Platform for Kubernetes scopes are retrieved successfully and validation failed
	valid, err := validateTokenForTAPScopes(claims, nil)
	if err == nil && !valid {
		logTanzuInvalidOrgWarningMessage(orgName)
	}
	return nil
}

func updateContextOnTanzuLogin(c *configtypes.Context, generateContextNameFunc func(orgName, endpoint string, isStaging bool) string, claims *commonauth.Claims, orgName string) error {
	// update the context name using the context name generator
	if generateContextNameFunc != nil {
		c.Name = generateContextNameFunc(orgName, tanzuUCPEndpoint, staging)
	}

	// update the context metadata
	if err := updateTanzuContextMetadata(c, claims.OrgID, orgName); err != nil {
		return err
	}

	// Fetch the tanzu kubeconfig and update context
	if err := updateContextWithTanzuKubeconfig(c, tanzuUCPEndpoint, claims.OrgID, endpointCACertPath, skipTLSVerify); err != nil {
		return err
	}

	// Add the context to configuration
	if err := config.AddContext(c, true); err != nil {
		return err
	}

	// update the current context in the kubeconfig file after creating the context
	if c.ClusterOpts != nil {
		err := syncCurrentKubeContext(c)
		if err != nil {
			return errors.Wrap(err, "unable to update current kube context")
		}
	}

	return nil
}

func logTanzuInvalidOrgWarningMessage(orgName string) {
	warnMsg := `WARNING: While authenticated to organization '%s', there are insufficient permissions to access
the Tanzu Platform for Kubernetes service. Please ensure correct organization authentication and access permissions
`
	fmt.Fprintln(os.Stderr)
	log.Warningf(warnMsg, orgName)
	fmt.Fprintln(os.Stderr)
}

// updateTanzuContextMetadata updates the context additional metadata
func updateTanzuContextMetadata(c *configtypes.Context, orgID, orgName string) error {
	exists, err := config.ContextExists(c.Name)
	if err != nil {
		return err
	}
	if !exists {
		c.AdditionalMetadata[config.OrgIDKey] = orgID
		c.AdditionalMetadata[config.OrgNameKey] = orgName
		c.AdditionalMetadata[config.TanzuHubEndpointKey] = tanzuHubEndpoint
		c.AdditionalMetadata[config.TanzuMissionControlEndpointKey] = tanzuTMCEndpoint
		return nil
	}
	// This is possible only for contexts created using "tanzu login" command because
	// "tanzu context create" command doesn't allow user to create duplicate contexts
	existingContext, err := config.GetContext(c.Name)
	if err != nil {
		return err
	}
	// If the context exists with the same name, honor the users current context additional metadata
	// which includes the org/project/space details.
	c.AdditionalMetadata = existingContext.AdditionalMetadata
	c.AdditionalMetadata[config.TanzuHubEndpointKey] = tanzuHubEndpoint
	c.AdditionalMetadata[config.TanzuMissionControlEndpointKey] = tanzuTMCEndpoint

	return nil
}

// getCSPOrganizationName returns the CSP Org name using the orgID from the claims.
// It will return empty string if API fails
func getCSPOrganizationName(c *configtypes.Context, claims *commonauth.Claims) (string, error) {
	issuer := csp.GetIssuer(staging)
	if c.GlobalOpts == nil {
		return "", errors.New("invalid context %q. Missing authorization fields")
	}
	orgName, err := csp.GetOrgNameFromOrgID(claims.OrgID, c.GlobalOpts.Auth.AccessToken, issuer)
	if err != nil {
		return "", err
	}
	return orgName, nil
}

func updateContextWithTanzuKubeconfig(c *configtypes.Context, ep, orgID, epCACertPath string, skipTLSVerify bool) error {
	kubeCfg, kubeCtx, orgEndpoint, err := tanzuauth.GetTanzuKubeconfig(c, ep, orgID, epCACertPath, skipTLSVerify)
	if err != nil {
		return err
	}
	if c.ClusterOpts == nil {
		c.ClusterOpts = &configtypes.ClusterServer{}
	}
	c.ClusterOpts.Path = kubeCfg
	c.ClusterOpts.Context = kubeCtx
	// for "tanzu" context ClusterOpts.Endpoint would always be pointing to UCP organization endpoint
	c.ClusterOpts.Endpoint = orgEndpoint

	return nil
}

func doCSPAuthentication(c *configtypes.Context) (*commonauth.Claims, error) {
	apiTokenValue, apiTokenExists := os.LookupEnv(config.EnvAPITokenKey)
	// Use API Token login flow if TANZU_API_TOKEN environment variable is set, else fall back to default interactive login flow
	if apiTokenExists {
		log.Info("API token env var is set")
		return doCSPAPITokenAuthAndUpdateContext(c, apiTokenValue)
	}

	issuer := csp.GetIssuer(staging)
	return doInteractiveLoginAndUpdateContext(c, issuer)
}

func doInteractiveLoginAndUpdateContext(c *configtypes.Context, issuerURL string) (claims *commonauth.Claims, err error) {
	var token *commonauth.Token

	logCSPOrgIDEnvVariableUsage()
	cspOrgIDValue, cspOrgIDExists := os.LookupEnv(constants.CSPLoginOrgID)
	var loginOptions []commonauth.LoginOption
	if cspOrgIDExists && cspOrgIDValue != "" {
		loginOptions = append(loginOptions, commonauth.WithOrgID(cspOrgIDValue))
	}
	// If user chooses to use a specific local listener port, use it
	loginOptions = append(loginOptions, commonauth.WithListenerPortFromEnv(constants.TanzuCLIOAuthLocalListenerPort))

	idpType := c.AdditionalMetadata[config.TanzuIdpTypeKey].(config.IdpType)
	if idpType == config.CSPIdpType {
		token, err = csp.TanzuLogin(issuerURL, loginOptions...)
	} else if idpType == config.UAAIdpType {
		token, err = uaa.TanzuLogin(issuerURL, loginOptions...)
	} else {
		return nil, errors.New(invalidIdpType)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the token")
	}

	claims, err = commonauth.ParseToken(&oauth2.Token{AccessToken: token.AccessToken}, idpType)
	if err != nil {
		return nil, err
	}

	a := configtypes.GlobalServerAuth{}
	a.Issuer = issuerURL
	a.UserName = claims.Username
	a.Permissions = claims.Permissions
	a.AccessToken = token.AccessToken
	a.IDToken = token.IDToken
	a.RefreshToken = token.RefreshToken
	a.Type = token.TokenType
	expiresAt := time.Now().Local().Add(time.Second * time.Duration(token.ExpiresIn))
	a.Expiration = expiresAt
	c.GlobalOpts.Auth = a
	if c.AdditionalMetadata == nil {
		c.AdditionalMetadata = make(map[string]interface{})
	}

	return claims, nil
}

func logCSPOrgIDEnvVariableUsage() {
	// The environment variable can be set using "tanzu config set env.<env variable name>" or by exporting
	// the environment variable name. If both are set, exported environment variable value has priority
	cspOrgIDValueFromCLIConfigEnv, cliConfigEnvErr := config.GetEnv(constants.CSPLoginOrgID)
	if cliConfigEnvErr == nil {
		cspOrgIDValueFromEnv, cspOrgIDExists := os.LookupEnv(constants.CSPLoginOrgID)
		if cspOrgIDExists && cspOrgIDValueFromEnv != "" && cspOrgIDValueFromEnv == cspOrgIDValueFromCLIConfigEnv {
			log.Infof("This tanzu context is being created using organization ID %s as set in the tanzu configuration (to unset, use `tanzu config unset env.TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID`).", cspOrgIDValueFromCLIConfigEnv)
			return
		}
	}

	cspOrgIDValueFromEnv, cspOrgIDExists := os.LookupEnv(constants.CSPLoginOrgID)
	if cspOrgIDExists && cspOrgIDValueFromEnv != "" {
		log.Infof("This tanzu context is being created using organization ID %s as set in the TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID environment variable.", cspOrgIDValueFromEnv)
	}
}

func doCSPAPITokenAuthAndUpdateContext(c *configtypes.Context, apiTokenValue string) (claims *commonauth.Claims, err error) {
	issuer := csp.GetIssuer(staging)
	token, err := csp.GetAccessTokenFromAPIToken(apiTokenValue, issuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the token from CSP")
	}
	claims, err = commonauth.ParseToken(&oauth2.Token{AccessToken: token.AccessToken}, config.CSPIdpType)
	if err != nil {
		return nil, err
	}

	a := configtypes.GlobalServerAuth{}
	a.Issuer = issuer
	a.UserName = claims.Username
	a.Permissions = claims.Permissions
	a.AccessToken = token.AccessToken
	a.IDToken = token.IDToken
	a.RefreshToken = apiTokenValue
	a.Type = commonauth.APITokenType
	expiresAt := time.Now().Local().Add(time.Second * time.Duration(token.ExpiresIn))
	a.Expiration = expiresAt
	c.GlobalOpts.Auth = a
	if c.AdditionalMetadata == nil {
		c.AdditionalMetadata = make(map[string]interface{})
	}

	return claims, nil
}

func promptContextType() (ctxCreationType ContextCreationType, err error) {
	ctxCreationTypeStr := ""
	promptOpts := getPromptOpts()

	err = component.Prompt(
		&component.PromptConfig{
			Message: "Select context creation type",
			Options: []string{string(contextTanzu), string(contextMissionControl), string(contextK8SClusterEndpoint), string(contextLocalKubeconfig)},
			Default: string(contextTanzu),
		},
		&ctxCreationTypeStr,
		promptOpts...,
	)
	if err != nil {
		return
	}

	return stringToContextCreationType(ctxCreationTypeStr), nil
}

func stringToContextCreationType(ctxCreationTypeStr string) ContextCreationType {
	if ctxCreationTypeStr == string(contextMissionControl) {
		return contextMissionControl
	} else if ctxCreationTypeStr == string(contextTanzu) {
		return contextTanzu
	} else if ctxCreationTypeStr == string(contextK8SClusterEndpoint) {
		return contextK8SClusterEndpoint
	} else if ctxCreationTypeStr == string(contextLocalKubeconfig) {
		return contextLocalKubeconfig
	}

	return ""
}

func promptKubernetesContextType() (ctxCreationType ContextCreationType, err error) {
	ctxCreationTypeStr := ""
	promptOpts := getPromptOpts()
	err = component.Prompt(
		&component.PromptConfig{
			Message: "Select the kubernetes context type",
			Options: []string{string(contextLocalKubeconfig), string(contextK8SClusterEndpoint)},
			Default: string(contextLocalKubeconfig),
		},
		&ctxCreationTypeStr,
		promptOpts...,
	)
	if err != nil {
		return
	}
	return stringToContextCreationType(ctxCreationTypeStr), nil
}

func promptEndpoint(defaultEndpoint string) (ep string, err error) {
	promptOpts := getPromptOpts()
	err = component.Prompt(
		&component.PromptConfig{
			Message: "Enter control plane endpoint",
			Default: defaultEndpoint,
		},
		&ep,
		promptOpts...,
	)
	if err != nil {
		return
	}
	ep = strings.TrimSpace(ep)
	return
}
func promptContextName(defaultCtxName string) (cname string, err error) {
	promptOpts := getPromptOpts()
	err = component.Prompt(
		&component.PromptConfig{
			Message: "Give the context a name",
			Default: defaultCtxName,
		},
		&cname,
		promptOpts...,
	)
	if err != nil {
		return
	}
	cname = strings.TrimSpace(cname)
	return
}

// Interactive way to create a TMC context. User will be prompted for CSP API token.
func promptAPIToken(endpointType string) (apiToken string, err error) {
	hostVal := "console.cloud.vmware.com"
	if staging {
		hostVal = "console-stg.cloud.vmware.com"
	}
	consoleURL := url.URL{
		Scheme:   "https",
		Host:     hostVal,
		Path:     "/csp/gateway/portal/",
		Fragment: "/user/tokens",
	}
	// The below message is applicable for TMC
	msg := fmt.Sprintf("If you don't have an API token, visit the VMware Cloud Services console, select your organization, and create an API token with the %s service roles:\n  %s\n",
		endpointType, consoleURL.String())
	// format
	fmt.Println()
	log.Infof(msg)

	promptOpts := getPromptOpts()

	// format
	fmt.Println()
	err = component.Prompt(
		&component.PromptConfig{
			Message:   "API Token",
			Sensitive: true,
		},
		&apiToken,
		promptOpts...,
	)
	apiToken = strings.TrimSpace(apiToken)
	return apiToken, err
}

func k8sLogin(c *configtypes.Context) error {
	if c != nil && c.ClusterOpts != nil && c.ClusterOpts.Path != "" && c.ClusterOpts.Context != "" {
		_, err := tkgauth.GetServerKubernetesVersion(c.ClusterOpts.Path, c.ClusterOpts.Context)
		if err != nil {
			err := fmt.Errorf("failed to create context %q for a kubernetes cluster, %v", c.Name, err)
			log.Error(err, "")
			return err
		}
		err = config.AddContext(c, true)
		if err != nil {
			return err
		}

		// update the current context in the kubeconfig file after creating the context
		err = syncCurrentKubeContext(c)
		if err != nil {
			return errors.Wrap(err, "unable to update current kube context")
		}
		log.Successf("successfully created a kubernetes context using the kubeconfig %s", c.ClusterOpts.Path)
		return nil
	}

	return fmt.Errorf("not yet implemented")
}

func sanitizeEndpoint(endpoint string) string {
	if len(strings.Split(endpoint, ":")) == 1 {
		return fmt.Sprintf("%s:443", endpoint)
	}
	return endpoint
}

func getDiscoveryHTTPClient(tlsConfig *tls.Config) *http.Client {
	tr := &http.Transport{
		TLSClientConfig:     tlsConfig,
		Proxy:               http.ProxyFromEnvironment,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	return &http.Client{Transport: tr}
}

func vSphereSupervisorLogin(endpoint string) (mergeFilePath, currentContext string, err error) {
	port := 443
	kubeCfg, kubeCtx, err := tkgauth.KubeconfigWithPinnipedAuthLoginPlugin(endpoint, nil,
		tkgauth.DiscoveryStrategy{DiscoveryPort: &port, ClusterInfoConfigMap: wcpauth.SupervisorVIPConfigMapName}, endpointCACertPath, skipTLSVerify)
	if err != nil {
		err := fmt.Errorf("error creating kubeconfig with tanzu pinniped-auth login plugin: %v", err)
		log.Error(err, "")
		return "", "", err
	}
	return kubeCfg, kubeCtx, err
}

func newListCtxCmd() *cobra.Command {
	var listCtxCmd = &cobra.Command{
		Use:               "list",
		Short:             "List contexts",
		ValidArgsFunction: noMoreCompletions,
		RunE:              listCtx,
	}

	listCtxCmd.Flags().StringVarP(&targetStr, "target", "", "", "list only contexts associated with the specified target (kubernetes[k8s]/mission-control[tmc])")
	utils.PanicOnErr(listCtxCmd.RegisterFlagCompletionFunc("target", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sContextType, compTMCContextType}, cobra.ShellCompDirectiveNoFileComp
	}))

	listCtxCmd.Flags().StringVarP(&contextTypeStr, "type", "t", "", "list only contexts associated with the specified context-type (kubernetes[k8s]/mission-control[tmc]/tanzu)")
	utils.PanicOnErr(listCtxCmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sContextType, compTanzuContextType, compTMCContextType}, cobra.ShellCompDirectiveNoFileComp
	}))

	listCtxCmd.Flags().BoolVar(&onlyCurrent, "current", false, "list only current active contexts")
	listCtxCmd.Flags().BoolVar(&showAllColumns, "wide", false, "display additional columns for the contexts")
	listCtxCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format: table|yaml|json")
	utils.PanicOnErr(listCtxCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))

	msg := "this was done in the v1.1.0 release, it will be removed following the deprecation policy (6 months). Use the --type flag instead.\n" //nolint:goconst
	utils.PanicOnErr(listCtxCmd.Flags().MarkDeprecated("target", msg))

	return listCtxCmd
}

func listCtx(cmd *cobra.Command, _ []string) error {
	cfg, err := config.GetClientConfig()
	if err != nil {
		return err
	}

	if !configtypes.IsValidContextType(contextTypeStr) {
		return errors.New(invalidContextType)
	}

	if !configtypes.IsValidTarget(targetStr, false, true) {
		return errors.New(invalidTargetErrorForContextCommands)
	}

	if isTableOutputFormat() {
		displayContextListOutputWithDynamicColumns(cfg, cmd.OutOrStdout(), showAllColumns)
	} else {
		displayContextListOutputListView(cfg, cmd.OutOrStdout())
	}

	return nil
}

func newGetCtxCmd() *cobra.Command {
	var getCtxCmd = &cobra.Command{
		Use:               "get CONTEXT_NAME",
		Short:             "Display a context from the config",
		ValidArgsFunction: completeAllContexts,
		RunE:              getCtx,
	}
	getCtxCmd.Flags().StringVarP(&getOutputFmt, "output", "o", "yaml", "output format: yaml|json")
	utils.PanicOnErr(getCtxCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))
	return getCtxCmd
}

func getCtx(cmd *cobra.Command, args []string) error {
	var ctx *configtypes.Context
	var err error
	if len(args) == 0 {
		ctx, err = promptCtx()
		if err != nil {
			return err
		}
	} else {
		ctx, err = config.GetContext(args[0])
		if err != nil {
			return err
		}
	}

	op := component.NewObjectWriter(cmd.OutOrStdout(), getOutputFmt, ctx)
	op.Render()
	return nil
}

func promptCtx() (*configtypes.Context, error) {
	cfg, err := config.GetClientConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil || len(cfg.KnownContexts) == 0 {
		return nil, errors.New("no contexts found")
	}
	return getCtxPromptMessage(cfg.KnownContexts)
}

// promptActiveCtx prompts with active list of contexts for user selection.
func promptActiveCtx() (*configtypes.Context, error) {
	currentCtxMap, err := config.GetAllActiveContextsMap()
	if err != nil {
		return nil, err
	}
	if len(currentCtxMap) == 0 {
		return nil, errors.New("no active contexts found")
	}
	return getCtxPromptMessage(getValues(currentCtxMap))
}

// getCtxPromptMessage prompts with a given list of contexts for user selection.
func getCtxPromptMessage(ctxs []*configtypes.Context) (*configtypes.Context, error) {
	promptOpts := getPromptOpts()
	contexts := make(map[string]*configtypes.Context)
	for _, ctx := range ctxs {
		info, err := config.EndpointFromContext(ctx)
		if err != nil {
			return nil, err
		}
		if info == "" && ctx.ContextType == configtypes.ContextTypeK8s && ctx.ClusterOpts != nil {
			info = fmt.Sprintf("%s:%s", ctx.ClusterOpts.Path, ctx.ClusterOpts.Context)
		}

		ctxKey := rpad(ctx.Name, 20)
		ctxKey = fmt.Sprintf("%s(%s)", ctxKey, info)
		contexts[ctxKey] = ctx
	}

	ctxKeys := getKeys(contexts)
	ctxKey := ctxKeys[0]
	err := component.Prompt(
		&component.PromptConfig{
			Message: "Select a context",
			Options: ctxKeys,
			Default: ctxKey,
		},
		&ctxKey,
		promptOpts...,
	)
	if err != nil {
		return nil, err
	}
	return contexts[ctxKey], nil
}

func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

func getKeys(m map[string]*configtypes.Context) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func getValues(m map[configtypes.ContextType]*configtypes.Context) []*configtypes.Context {
	values := make([]*configtypes.Context, 0, len(m))
	for _, value := range m {
		values = append(values, value)
	}
	return values
}

func newCurrentCtxCmd() *cobra.Command {
	var currentCtxCmd = &cobra.Command{
		Use:               "current",
		Short:             "Display the current context",
		Args:              cobra.NoArgs,
		ValidArgsFunction: noMoreCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			currentCtxMap, err := config.GetAllActiveContextsMap()
			if err != nil {
				return err
			}

			if len(currentCtxMap) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "There is no active context")
				return nil
			}

			ignoreTMCCtx := false
			if len(currentCtxMap) > 1 {
				// If there are multiple contexts, which means 2 of them, ignore the TMC context
				// and prioritize the tanzu or k8s context (only one of those two will be present)
				ignoreTMCCtx = true
			}

			for ctxType, ctx := range currentCtxMap {
				if ignoreTMCCtx && ctxType == configtypes.ContextTypeTMC {
					continue
				}

				if shortCtx {
					printShortContext(cmd.OutOrStdout(), ctx)
				} else {
					printContext(cmd.OutOrStdout(), ctx)
				}
			}
			return nil
		},
	}

	currentCtxCmd.Flags().BoolVarP(&shortCtx, "short", "", false, "prints the context in compact form")

	return currentCtxCmd
}

func printShortContext(writer io.Writer, ctx *configtypes.Context) {
	if ctx == nil {
		return
	}

	var ctxStr strings.Builder
	ctxStr.WriteString(ctx.Name)

	// For a tanzu context, print the project, space, and cluster group
	if ctx.ContextType == configtypes.ContextTypeTanzu {
		resources, err := config.GetTanzuContextActiveResource(ctx.Name)
		if err == nil {
			if resources.ProjectName != "" {
				ctxStr.WriteString(fmt.Sprintf(":%s", resources.ProjectName))
				if resources.SpaceName != "" {
					ctxStr.WriteString(fmt.Sprintf(":%s", resources.SpaceName))
				} else if resources.ClusterGroupName != "" {
					ctxStr.WriteString(fmt.Sprintf(":%s", resources.ClusterGroupName))
				}
			}
		}
	}
	fmt.Fprintln(writer, ctxStr.String())
}

func printContext(writer io.Writer, ctx *configtypes.Context) {
	if ctx == nil {
		return
	}

	// Use a ListTable format to get nice alignment
	columns := []string{"Name", "Type"}
	row := []interface{}{ctx.Name, string(ctx.ContextType)}

	if ctx.ContextType == configtypes.ContextTypeTanzu {
		resources, err := config.GetTanzuContextActiveResource(ctx.Name)
		if err == nil {
			if resources.OrgID != "" {
				columns = append(columns, "Organization")
				row = append(row, fmt.Sprintf("%s (%s)", resources.OrgName, resources.OrgID))
			}

			columns = append(columns, "Project")
			if resources.ProjectName != "" {
				row = append(row, fmt.Sprintf("%s (%s)", resources.ProjectName, resources.ProjectID))
			} else {
				row = append(row, "none set")
			}

			if resources.SpaceName != "" {
				columns = append(columns, "Space")
				row = append(row, resources.SpaceName)
			} else if resources.ClusterGroupName != "" {
				columns = append(columns, "Cluster Group")
				row = append(row, resources.ClusterGroupName)
			}
		}
	}

	if ctx.ContextType != configtypes.ContextTypeTMC {
		var kubeconfig, kubeCtx string
		if ctx.ClusterOpts != nil {
			kubeconfig = ctx.ClusterOpts.Path
			kubeCtx = ctx.ClusterOpts.Context
		}
		columns = append(columns, "Kube Config")
		row = append(row, kubeconfig)
		columns = append(columns, "Kube Context")
		row = append(row, kubeCtx)
	}

	outputWriter := component.NewOutputWriterWithOptions(writer, string(component.ListTableOutputType), []component.OutputWriterOption{}, columns...)
	outputWriter.AddRow(row...)
	outputWriter.Render()
}

func newDeleteCtxCmd() *cobra.Command {
	var deleteCtxCmd = &cobra.Command{
		Use:               "delete CONTEXT_NAME",
		Short:             "Delete a context from the config",
		ValidArgsFunction: completeAllContexts,
		RunE:              deleteCtx,
	}
	deleteCtxCmd.Flags().BoolVarP(&unattended, "yes", "y", false, "delete the context entry without confirmation")
	return deleteCtxCmd
}

func deleteCtx(_ *cobra.Command, args []string) error {
	var name string
	if len(args) == 0 {
		ctx, err := promptCtx()
		if err != nil {
			return err
		}
		name = ctx.Name
	} else {
		name = args[0]
	}

	ctx, err := config.GetContext(name)
	if err != nil {
		return err
	}

	if !unattended {
		isAborted := component.AskForConfirmation("Deleting the context entry from the config will remove it from the list of tracked contexts. " +
			"You will need to use `tanzu context create` to re-create this context. Are you sure you want to continue?")
		if isAborted != nil {
			return nil
		}
	}

	err = config.RemoveContext(name)
	if err != nil {
		return err
	}

	deleteKubeconfigContext(ctx)
	log.Successf("Successfully deleted context %q", name)
	return nil
}

func deleteKubeconfigContext(ctx *configtypes.Context) {
	// Note: currently cleaning up the kubeconfig for tanzu context types only.
	// (Since the kubernetes context type can have kube context provided by the user, it may not be
	// desired outcome for user if CLI deletes/cleanup kubeconfig provided by the user.)
	if ctx.ContextType == configtypes.ContextTypeTanzu || isPinnipedEndpointContext(ctx) {
		if ctx.ClusterOpts == nil {
			return
		}
		log.Infof("Deleting kubeconfig context '%s' from the file '%s'", ctx.ClusterOpts.Context, ctx.ClusterOpts.Path)
		if err := kubecfg.DeleteContextFromKubeConfig(ctx.ClusterOpts.Path, ctx.ClusterOpts.Context); err != nil {
			log.Warningf("Failed to delete the kubeconfig context '%s' from the file '%s'", ctx.ClusterOpts.Context, ctx.ClusterOpts.Path)
		}
	}
}

func isPinnipedEndpointContext(ctx *configtypes.Context) bool {
	if ctx.ContextType != configtypes.ContextTypeK8s || ctx.AdditionalMetadata == nil ||
		ctx.AdditionalMetadata[isPinnipedEndpoint] == nil {
		return false
	}
	isPinnipedEP, valid := (ctx.AdditionalMetadata[isPinnipedEndpoint]).(bool)
	if valid && isPinnipedEP {
		return true
	}
	return false
}

func newUseCtxCmd() *cobra.Command {
	var useCtxCmd = &cobra.Command{
		Use:               "use CONTEXT_NAME",
		Short:             "Set the context to be used by default",
		ValidArgsFunction: completeAllContexts,
		RunE:              useCtx,
	}

	return useCtxCmd
}

func useCtx(cmd *cobra.Command, args []string) error { //nolint:gocyclo
	var ctx *configtypes.Context
	var err error

	if len(args) == 0 {
		ctx, err := promptCtx()
		if err != nil {
			return err
		}
		ctxName = ctx.Name
	} else {
		ctxName = args[0]
	}

	ctx, err = config.GetContext(ctxName)
	if err != nil {
		return err
	}

	if ctx.ClusterOpts != nil {
		err = syncCurrentKubeContext(ctx)
		if err != nil {
			return errors.Wrap(err, "unable to update current kube context")
		}
	}

	err = config.SetActiveContext(ctxName)
	if err != nil {
		return err
	}

	suffixString := fmt.Sprintf("Type: %s", ctx.ContextType)
	if ctx.ContextType == configtypes.ContextTypeTanzu {
		if project, exists := ctx.AdditionalMetadata[config.ProjectNameKey]; exists && project != "" {
			suffixString += fmt.Sprintf(", Project: %s", project)
		}
		// expectation is user/plugin would set both project name and project ID together
		if projectID, exists := ctx.AdditionalMetadata[config.ProjectIDKey]; exists && projectID != "" {
			suffixString += fmt.Sprintf(" (%s)", projectID)
		}
		if space, exists := ctx.AdditionalMetadata[config.SpaceNameKey]; exists && space != "" {
			suffixString += fmt.Sprintf(", Space: %s", space)
		}
		if clustergroup, exists := ctx.AdditionalMetadata[config.ClusterGroupNameKey]; exists && clustergroup != "" {
			suffixString += fmt.Sprintf(", ClusterGroup: %s", clustergroup)
		}
	}
	if suffixString != "" {
		suffixString = "(" + suffixString + ")"
	}

	log.Infof("Successfully activated context '%s' %s ", ctxName, suffixString)

	// TODO: update the below conditional check (and in login command) after context scope plugin support
	//       is implemented for tanzu context(Tanzu Platform for Kubernetes SaaS)
	// Sync all required plugins
	if ctx.ContextType != configtypes.ContextTypeTanzu {
		if err := syncContextPlugins(cmd, ctx.ContextType, ctxName); err != nil {
			log.Warningf("unable to automatically sync the plugins recommended by the active context. Please run 'tanzu plugin sync' to sync plugins manually, error: '%v'", err.Error())
		}
	}
	return nil
}

// pre-reqs context.ClusterOpts is not nil
func syncCurrentKubeContext(ctx *configtypes.Context) error {
	if skipSync, _ := strconv.ParseBool(os.Getenv(constants.SkipUpdateKubeconfigOnContextUse)); skipSync {
		return nil
	}
	return kubecfg.SetCurrentContext(ctx.ClusterOpts.Path, ctx.ClusterOpts.Context)
}

func newUnsetCtxCmd() *cobra.Command {
	var unsetCtxCmd = &cobra.Command{
		Use:               "unset CONTEXT_NAME",
		Short:             "Unset the active context so that it is not used by default",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeActiveContexts,
		RunE:              unsetCtx,
	}

	unsetCtxCmd.Flags().StringVarP(&targetStr, "target", "", "", "unset active context associated with the specified target (kubernetes[k8s]|mission-control[tmc])")
	utils.PanicOnErr(unsetCtxCmd.RegisterFlagCompletionFunc("target", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sContextType, compTMCContextType}, cobra.ShellCompDirectiveNoFileComp
	}))
	unsetCtxCmd.Flags().StringVarP(&contextTypeStr, "type", "t", "", "unset active context associated with the specified context-type (kubernetes[k8s]|mission-control[tmc]|tanzu)")
	utils.PanicOnErr(unsetCtxCmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sContextType, compTanzuContextType, compTMCContextType}, cobra.ShellCompDirectiveNoFileComp
	}))

	msg := "this was done in the v1.1.0 release, it will be removed following the deprecation policy (6 months). Use the --type flag instead.\n"
	utils.PanicOnErr(unsetCtxCmd.Flags().MarkDeprecated("target", msg))

	return unsetCtxCmd
}

func unsetCtx(_ *cobra.Command, args []string) error {
	var name string
	if !configtypes.IsValidContextType(contextTypeStr) {
		return errors.New(invalidContextType)
	}
	if !configtypes.IsValidTarget(targetStr, false, true) {
		return errors.New(invalidTargetErrorForContextCommands)
	}
	contextType := getContextType()
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" && contextType == "" {
		ctx, err := promptActiveCtx()
		if err != nil {
			return err
		}
		name = ctx.Name
	}
	return unsetGivenContext(name, contextType)
}

func unsetGivenContext(name string, contextType configtypes.ContextType) error {
	var unset bool
	currentCtxMap, err := config.GetAllActiveContextsMap()
	if contextType != "" && name != "" {
		ctx, ok := currentCtxMap[contextType]
		if ok && ctx.Name == name {
			err = config.RemoveActiveContext(contextType)
			unset = true
		} else {
			return errors.Errorf(contextNotExistsForContextType, name, contextType)
		}
	} else if contextType != "" {
		ctx, ok := currentCtxMap[contextType]
		if ok {
			name = ctx.Name
			err = config.RemoveActiveContext(contextType)
			unset = true
		} else {
			log.Warningf(noActiveContextExistsForContextType, contextType)
		}
	} else if name != "" {
		for ct, ctx := range currentCtxMap {
			if ctx.Name == name {
				contextType = ct
				err = config.RemoveActiveContext(contextType)
				unset = true
				break
			}
		}
		if !unset {
			return errors.Errorf(contextNotActiveOrNotExists, name)
		}
	}
	if err != nil {
		return err
	} else if unset {
		log.Outputf(contextForContextTypeSetInactive, name, contextType)
	}
	return nil
}

func displayContextListOutputListView(cfg *configtypes.ClientConfig, writer io.Writer) {
	contextType := getContextType()

	// switching to use the new OutputWriter because we want to render the
	// additional metadata map correctly in their native JSON/YAML form
	opts := []component.OutputWriterOption{}
	op := component.NewOutputWriterWithOptions(writer, outputFormat, opts, "Name", "Type", "IsManagementCluster", "IsCurrent", "Endpoint", "KubeConfigPath", "KubeContext", "AdditionalMetadata")
	ctxToList := cfg.KnownContexts

	// sort the contexts by name amd then by target
	sort.Sort(configtypes.ContextSorter(ctxToList))
	for _, ctx := range ctxToList {
		if contextType != "" && ctx.ContextType != contextType {
			continue
		}
		isMgmtCluster := ctx.IsManagementCluster()
		isCurrent := ctx.Name == cfg.CurrentContext[ctx.ContextType]
		if onlyCurrent && !isCurrent {
			continue
		}

		var ep, path, context string
		switch ctx.ContextType {
		case configtypes.ContextTypeTMC:
			if ctx.GlobalOpts != nil {
				ep = ctx.GlobalOpts.Endpoint
			}
		default:
			if ctx.ClusterOpts != nil {
				ep = ctx.ClusterOpts.Endpoint
				path = ctx.ClusterOpts.Path
				context = ctx.ClusterOpts.Context
			}
		}

		op.AddRow(ctx.Name, ctx.ContextType, strconv.FormatBool(isMgmtCluster), strconv.FormatBool(isCurrent), ep, path, context, ctx.AdditionalMetadata)
	}
	op.Render()
}

// getContextsToDisplay returns a filtered list of contexts, and a boolean on
// whether the contexts include some with tanzu context type fields to display
func getContextsToDisplay(cfg *configtypes.ClientConfig, contextType configtypes.ContextType, onlyCurrent bool) ([]*configtypes.Context, bool) {
	var contextOutputList []*configtypes.Context
	var hasTanzuFields bool

	for _, ctx := range cfg.KnownContexts {
		if contextType != "" && ctx.ContextType != contextType {
			continue
		}
		isCurrent := ctx.Name == cfg.CurrentContext[ctx.ContextType]
		if onlyCurrent && !isCurrent {
			continue
		}
		// could be fine-tuned to check for non-empty values as well
		if ctx.ContextType == configtypes.ContextTypeTanzu {
			hasTanzuFields = true
		}
		contextOutputList = append(contextOutputList, ctx)
	}
	return contextOutputList, hasTanzuFields
}

type ContextListOutputRow struct {
	Name           string
	IsActive       string
	Type           string
	Project        string
	ProjectID      string
	Space          string
	ClusterGroup   string
	Endpoint       string
	KubeconfigPath string
	KubeContext    string
}

func displayContextListOutputWithDynamicColumns(cfg *configtypes.ClientConfig, writer io.Writer, showAllColumns bool) { //nolint:funlen
	ct := getContextType()
	ctxs, _ := getContextsToDisplay(cfg, ct, onlyCurrent)
	sort.Sort(configtypes.ContextSorter(ctxs))

	opts := []component.OutputWriterOption{}
	rows := []ContextListOutputRow{}

	tanzuContextExists := false

	for _, ctx := range ctxs {
		ep := NA
		path := NA
		context := NA
		project := NA
		projectID := NA
		space := NA
		clustergroup := NA

		isCurrent := ctx.Name == cfg.CurrentContext[ctx.ContextType]

		switch ctx.ContextType {
		case configtypes.ContextTypeTMC:
			if ctx.GlobalOpts != nil {
				ep = ctx.GlobalOpts.Endpoint
			}
		case configtypes.ContextTypeTanzu:
			tanzuContextExists = true
			project = ""
			projectID = ""
			space = ""
			clustergroup = ""
			ep = ""
			path = ""
			context = ""
			if ctx.ClusterOpts != nil {
				ep = ctx.ClusterOpts.Endpoint
				path = ctx.ClusterOpts.Path
				context = ctx.ClusterOpts.Context
			}
			if ctx.AdditionalMetadata[config.ProjectNameKey] != nil {
				project = ctx.AdditionalMetadata[config.ProjectNameKey].(string)
			}
			if ctx.AdditionalMetadata[config.ProjectIDKey] != nil {
				projectID = ctx.AdditionalMetadata[config.ProjectIDKey].(string)
			}
			if ctx.AdditionalMetadata[config.SpaceNameKey] != nil {
				space = ctx.AdditionalMetadata[config.SpaceNameKey].(string)
			}
			if ctx.AdditionalMetadata[config.ClusterGroupNameKey] != nil {
				clustergroup = ctx.AdditionalMetadata[config.ClusterGroupNameKey].(string)
			}
		default:
			if ctx.ClusterOpts != nil {
				ep = ctx.ClusterOpts.Endpoint
				path = ctx.ClusterOpts.Path
				context = ctx.ClusterOpts.Context
			}
		}
		row := ContextListOutputRow{ctx.Name, strconv.FormatBool(isCurrent), string(ctx.ContextType), project, projectID, space, clustergroup, ep, path, context}
		rows = append(rows, row)
	}

	requiredColumns := []string{"Name", "IsActive", "Type"}
	dynamicColumns := []string{}
	if tanzuContextExists {
		requiredColumns = append(requiredColumns, "Project", "Space")
		dynamicColumns = append(dynamicColumns, "ClusterGroup")
	}
	if showAllColumns {
		if tanzuContextExists {
			dynamicColumns = append(dynamicColumns, "ProjectID")
		}
		requiredColumns = append(requiredColumns, "Endpoint", "KubeconfigPath", "KubeContext")
		requiredColumns = append(requiredColumns, dynamicColumns...)
	}
	renderDynamicTable(rows, component.NewOutputWriterWithOptions(writer, outputFormat, opts, "NAME", "ISACTIVE", "TYPE"), requiredColumns, dynamicColumns)

	if !showAllColumns {
		fmt.Println()
		log.Info("Use '--wide' to view additional columns.")
	}
}

func newGetCtxTokenCmd() *cobra.Command {
	var getCtxTokenCmd = &cobra.Command{
		Use:               "get-token CONTEXT_NAME",
		Short:             "Get the valid CSP token for the given tanzu context",
		Args:              cobra.ExactArgs(1),
		Hidden:            true,
		ValidArgsFunction: completeTanzuContexts,
		RunE:              getToken,
	}
	return getCtxTokenCmd
}

func getToken(cmd *cobra.Command, args []string) error {
	name := args[0]
	ctx, err := config.GetContext(name)
	if err != nil {
		return err
	}
	if ctx.ContextType != configtypes.ContextTypeTanzu {
		return errors.Errorf("context %q is not of type tanzu", name)
	}
	if ctx.GlobalOpts == nil {
		return errors.Errorf("invalid context %q . Missing the authorization fields in the context", name)
	}

	if commonauth.IsExpired(ctx.GlobalOpts.Auth.Expiration) {
		val, ok := ctx.AdditionalMetadata[config.TanzuIdpTypeKey].(string)
		idpType := config.IdpType(val)
		if !ok {
			idpType = config.CSPIdpType
		}
		tokenGetter := uaa.GetTokens
		if idpType == config.CSPIdpType {
			tokenGetter = csp.GetTokens
		}

		_, err := commonauth.GetToken(&ctx.GlobalOpts.Auth, tokenGetter, idpType)
		if err != nil {
			return errors.Wrap(err, "failed to refresh the token")
		}
		if err = config.SetContext(ctx, false); err != nil {
			return errors.Wrap(err, "failed updating the context after token refresh")
		}
	}
	token := ctx.GlobalOpts.Auth.AccessToken
	expTime := ctx.GlobalOpts.Auth.Expiration

	return printTokenToStdout(cmd, token, expTime)
}

func printTokenToStdout(cmd *cobra.Command, token string, expTime time.Time) error {
	et := metav1.NewTime(expTime).Rfc3339Copy()
	cred := clientauthv1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExecCredential",
			APIVersion: "client.authentication.k8s.io/v1",
		},
		Status: &clientauthv1.ExecCredentialStatus{
			Token:               token,
			ExpirationTimestamp: &et,
		},
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(cred)
}

// newUpdateCtxCmd to update an aspect of a context
//
// NOTE!!: This command is EXPERIMENTAL and subject to change in future
func newUpdateCtxCmd() *cobra.Command {
	var updateCtxCmd = &cobra.Command{
		Use:    "update",
		Short:  "Update an aspect of a context (subject to change)",
		Hidden: true,
	}
	updateCtxCmd.AddCommand(
		newTanzuActiveResourceCmd(),
	)
	return updateCtxCmd
}

// tanzuActiveResourceCmd updates the tanzu active resource referenced by tanzu context
//
// NOTE!!: This command is EXPERIMENTAL and subject to change in future
func newTanzuActiveResourceCmd() *cobra.Command {
	tanzuActiveResourceCmd := &cobra.Command{
		Use:               "tanzu-active-resource CONTEXT_NAME",
		Short:             "Updates the tanzu active resource for the given context of type tanzu (subject to change)",
		Hidden:            true,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeTanzuContexts,
		RunE:              setTanzuCtxActiveResource,
	}
	tanzuActiveResourceCmd.Flags().StringVarP(&projectStr, "project", "", "", "project name to be set as active")
	tanzuActiveResourceCmd.Flags().StringVarP(&projectIDStr, "project-id", "", "", "project ID to be set as active")
	tanzuActiveResourceCmd.Flags().StringVarP(&spaceStr, "space", "", "", "space name to be set as active")
	tanzuActiveResourceCmd.Flags().StringVarP(&clustergroupStr, "clustergroup", "", "", "clustergroup name to be set as active")

	return tanzuActiveResourceCmd
}

func setTanzuCtxActiveResource(_ *cobra.Command, args []string) error {
	name := args[0]

	if err := validateActiveResourceOptions(); err != nil {
		return err
	}

	ctx, err := config.GetContext(name)
	if err != nil {
		return err
	}
	if ctx.ContextType != configtypes.ContextTypeTanzu {
		return errors.Errorf("context %q is not of type tanzu", name)
	}
	if ctx.AdditionalMetadata == nil {
		ctx.AdditionalMetadata = make(map[string]interface{})
	}
	ctx.AdditionalMetadata[config.ProjectNameKey] = projectStr
	ctx.AdditionalMetadata[config.ProjectIDKey] = projectIDStr
	ctx.AdditionalMetadata[config.SpaceNameKey] = spaceStr
	ctx.AdditionalMetadata[config.ClusterGroupNameKey] = clustergroupStr
	err = updateTanzuContextKubeconfig(ctx, projectStr, projectIDStr, spaceStr, clustergroupStr)
	if err != nil {
		return errors.Wrap(err, "failed to update the tanzu context kubeconfig")
	}
	err = config.SetContext(ctx, false)
	if err != nil {
		return errors.Wrap(err, "failed updating the context %q with the active tanzu resource")
	}

	return nil
}

// getProjectValueForKubeconfig return the project value to be used for UCP kubeconfig generation.
//
// Note: This method can be removed for official release as projectID would be used for kubeconfig generation.
func getProjectValueForKubeconfig(projectName, projectID string) string {
	// TODO (prkalle): Adding this fallback logic to support the backward compatibility. This should be updated to use projectID for official release
	// If the projectIDStr is set it would be used for kubeconfig generation, else use the project name
	if projectID != "" {
		return projectID
	}
	return projectName
}

func validateActiveResourceOptions() error {
	if spaceStr != "" && clustergroupStr != "" {
		return errors.Errorf("either space or clustergroup can be set as active resource. Please provide either --space or --clustergroup option")
	}

	// TODO(prkalle): Need to update the checks to make project ID and project Name mandatory for official release
	if (projectStr == "" && projectIDStr == "") && spaceStr != "" {
		// TODO(prkalle): update the error message later for official release to use --project and --project-id options to set the project
		return errors.Errorf("space cannot be set without project. Please set the project")
	}

	if (projectStr == "" && projectIDStr == "") && clustergroupStr != "" {
		// TODO(prkalle): update the error message later for official release to use --project and --project-id options to set the project
		return errors.Errorf("clustergroup cannot be set without project. Please set the project")
	}

	return nil
}

func updateTanzuContextKubeconfig(cliContext *configtypes.Context, projectName, projectID, spaceName, clustergroupName string) error {
	if cliContext.ClusterOpts == nil {
		return errors.New("invalid context. Kubeconfig details are missing in the context")
	}

	kcfg, err := clientcmd.LoadFromFile(cliContext.ClusterOpts.Path)
	if err != nil {
		return errors.Wrap(err, "unable to load kubeconfig")
	}

	projectVal := getProjectValueForKubeconfig(projectName, projectID)
	newServerURL := prepareClusterServerURL(cliContext, projectVal, spaceName, clustergroupName)

	// Manage context based on environment variable
	useStableKubeContextName, _ := strconv.ParseBool(os.Getenv(constants.UseStableKubeContextNameForTanzuContext))
	if useStableKubeContextName {
		if err := updateContextInPlace(kcfg, cliContext, newServerURL); err != nil {
			return err
		}
	} else {
		newContextName := prepareKubeContextName(cliContext, projectName, spaceName, clustergroupName)
		if err := updateContextWithNewName(kcfg, cliContext, newContextName, newServerURL); err != nil {
			return err
		}
	}

	err = clientcmd.WriteToFile(*kcfg, cliContext.ClusterOpts.Path)
	if err != nil {
		return errors.Wrap(err, "failed to update the context kubeconfig file")
	}
	return nil
}

func getKubeContextAndCluster(kcfg *api.Config, cliContext *configtypes.Context) (*api.Context, *api.Cluster, error) {
	kubeCtx := kcfg.Contexts[cliContext.ClusterOpts.Context]
	if kubeCtx == nil {
		return nil, nil, errors.Errorf("kubecontext %q doesn't exist", cliContext.ClusterOpts.Context)
	}
	kubeCluster := kcfg.Clusters[kubeCtx.Cluster]
	if kubeCluster == nil {
		return nil, nil, errors.Errorf("kubecluster %q doesn't exist", kubeCtx.Cluster)
	}
	return kubeCtx, kubeCluster, nil
}
func updateContextInPlace(kcfg *api.Config, cliContext *configtypes.Context, newServerURL string) error {
	_, kubeCluster, err := getKubeContextAndCluster(kcfg, cliContext)
	if err != nil {
		return err
	}
	kubeCluster.Server = newServerURL
	return nil
}

func updateContextWithNewName(kcfg *api.Config, cliContext *configtypes.Context, newKubeContextName, newServerURL string) error {
	kubeCtx, kubeCluster, err := getKubeContextAndCluster(kcfg, cliContext)
	if err != nil {
		return err
	}

	newKubeCluster := kubeCluster.DeepCopy()
	newKubeContext := kubeCtx.DeepCopy()

	delete(kcfg.Clusters, kubeCtx.Cluster)
	delete(kcfg.Contexts, cliContext.ClusterOpts.Context)

	newKubeCluster.Server = newServerURL
	// use the same name for kubecontext and cluster
	newKubeContext.Cluster = newKubeContextName
	kcfg.Contexts[newKubeContextName] = newKubeContext
	kcfg.Clusters[newKubeContext.Cluster] = newKubeCluster

	// if the existing kubecontext is current update the current-context to point to new kubecontext
	if kcfg.CurrentContext == cliContext.ClusterOpts.Context {
		kcfg.CurrentContext = newKubeContextName
	}

	// update the CLI context to point to new context name
	cliContext.ClusterOpts.Context = newKubeContextName

	return nil
}

// pre-reqs context.ClusterOpts is not nil
func prepareClusterServerURL(context *configtypes.Context, project, spaceName, clustergroupName string) string {
	serverURL := context.ClusterOpts.Endpoint
	if project == "" {
		return serverURL
	}
	serverURL = serverURL + "/project/" + project

	if spaceName != "" {
		return serverURL + "/space/" + spaceName
	}
	if clustergroupName != "" {
		return serverURL + "/clustergroup/" + clustergroupName
	}
	return serverURL
}

// pre-reqs context.ClusterOpts is not nil
func prepareKubeContextName(context *configtypes.Context, project, spaceName, clustergroupName string) string {
	contextName := "tanzu-cli-" + context.Name
	if project == "" {
		return contextName
	}
	contextName += fmt.Sprintf(":%s", project)

	if spaceName != "" {
		return contextName + fmt.Sprintf(":%s", spaceName)
	}
	if clustergroupName != "" {
		return contextName + fmt.Sprintf(":%s", clustergroupName)
	}

	return contextName
}

func getContextType() configtypes.ContextType {
	if contextTypeStr != "" {
		return configtypes.StringToContextType(contextTypeStr)
	} else if targetStr != "" {
		return configtypes.ConvertTargetToContextType(getTarget())
	}
	return ""
}

// ====================================
// Shell completion functions
// ====================================
func completeAllContexts(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.GetClientConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ct := getContextType()

	var allCtxs []*configtypes.Context
	for _, ctx := range cfg.KnownContexts {
		if ct == "" || ct == ctx.ContextType {
			allCtxs = append(allCtxs, ctx)
		}
	}
	return completionFormatCtxs(allCtxs), cobra.ShellCompDirectiveNoFileComp
}

func completeTanzuContexts(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.GetClientConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var tanzuCtxs []*configtypes.Context
	for _, ctx := range cfg.KnownContexts {
		if ctx.ContextType == configtypes.ContextTypeTanzu {
			tanzuCtxs = append(tanzuCtxs, ctx)
		}
	}
	return completionFormatCtxs(tanzuCtxs), cobra.ShellCompDirectiveNoFileComp
}

func completeActiveContexts(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	currentCtxMap, err := config.GetAllActiveContextsMap()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ct := getContextType()

	var allCtxs []*configtypes.Context
	for _, ctx := range currentCtxMap {
		if ct == "" || ct == ctx.ContextType {
			allCtxs = append(allCtxs, ctx)
		}
	}
	return completionFormatCtxs(allCtxs), cobra.ShellCompDirectiveNoFileComp
}

// Setup shell completion for the kube-context flag
func completeKubeContext(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	if kubeConfig == "" {
		kubeConfig = kubecfg.GetDefaultKubeConfigFile()
	}

	cobra.CompDebugln("About to get the different kube-contexts", false)

	kubeclient, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfig},
		&clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var comps []string
	for name, context := range kubeclient.Contexts {
		comps = append(comps, fmt.Sprintf("%s\t%s@%s", name, context.AuthInfo, context.Cluster))
	}
	// Sort the completion to make testing easier
	sort.Strings(comps)
	return comps, cobra.ShellCompDirectiveNoFileComp
}

func completeCreateCtx(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		comps := cobra.AppendActiveHelp(nil, "Please specify a name for the context")
		return comps, cobra.ShellCompDirectiveNoFileComp
	}

	if endpoint == "" && kubeContext == "" {
		// The user must provide more info by using flags.
		// Note that those flags are not marked as mandatory
		// because the prompt mechanism can be used instead.
		comps := []string{"--"}
		return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}

	// The user has provided enough information
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func completionFormatCtxs(ctxs []*configtypes.Context) []string {
	var comps []string
	for _, ctx := range ctxs {
		info, _ := config.EndpointFromContext(ctx)

		if info == "" && ctx.ContextType == configtypes.ContextTypeK8s && ctx.ClusterOpts != nil {
			info = fmt.Sprintf("%s:%s", ctx.ClusterOpts.Path, ctx.ClusterOpts.Context)
		}

		comps = append(comps, fmt.Sprintf("%s\t%s", ctx.Name, info))
	}

	// Sort the completion to make testing easier
	sort.Strings(comps)
	return comps
}

// renderDynamicTable writes the data in table format dynamically by
// - showing all required columns
// - optionally showing the dynamic columns if the data exists
func renderDynamicTable(slices interface{}, tableWriter component.OutputWriter, requiredColumns, dynamicColumns []string) {
	// Check if the input is a slice
	valueOf := reflect.ValueOf(slices)
	if valueOf.Kind() == reflect.Slice && valueOf.Len() > 0 {
		// Collect header and column data
		header := []string{}
		isColumnFilled := make(map[int]bool)
		showColumn := make(map[int]bool)

		for i := 0; i < valueOf.Len(); i++ {
			elem := valueOf.Index(i)
			elemValue := reflect.ValueOf(elem.Interface())

			// Determine which columns are filled for this element
			for j := 0; j < elemValue.NumField(); j++ {
				field := elemValue.Field(j)
				fieldValue := field.Interface()
				isNA := reflect.DeepEqual(fieldValue, NA)
				isEmpty := reflect.DeepEqual(fieldValue, "")
				if !isNA && !isEmpty {
					isColumnFilled[j] = true
				}
			}
		}

		// Build the header based on the first element
		elem := valueOf.Index(0)
		elemValue := reflect.ValueOf(elem.Interface())
		for j := 0; j < elemValue.NumField(); j++ {
			isRequiredColumn := utils.ContainsString(requiredColumns, elemValue.Type().Field(j).Name)
			isFilledDynamicColumn := utils.ContainsString(dynamicColumns, elemValue.Type().Field(j).Name) && isColumnFilled[j]
			showColumn[j] = isRequiredColumn || isFilledDynamicColumn
		}

		for j := 0; j < elemValue.NumField(); j++ {
			if showColumn[j] {
				fieldName := elemValue.Type().Field(j).Name
				header = append(header, fieldName)
			}
		}
		// Set header
		tableWriter.SetKeys(header...)

		// Build the data rows and add them to tablewriter
		for i := 0; i < valueOf.Len(); i++ {
			elem := valueOf.Index(i)
			elemValue := reflect.ValueOf(elem.Interface())
			dataRow := []interface{}{}
			for j := 0; j < elemValue.NumField(); j++ {
				if showColumn[j] {
					field := elemValue.Field(j)
					fieldValue := field.Interface()
					dataRow = append(dataRow, fmt.Sprintf("%v", fieldValue))
				}
			}
			tableWriter.AddRow(dataRow...)
		}
	}
	tableWriter.Render()
}

func isTableOutputFormat() bool {
	return outputFormat == "" || outputFormat == string(component.TableOutputType)
}
