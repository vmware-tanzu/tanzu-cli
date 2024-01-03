// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"net/http"
	"net/url"
	"os"
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

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/csp"
	tanzuauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tanzu"
	tkgauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tkg"
	kubecfg "github.com/vmware-tanzu/tanzu-cli/pkg/auth/utils/kubeconfig"
	wcpauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/wcp"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

var (
	stderrOnly, forceCSP, staging, onlyCurrent, skipTLSVerify                              bool
	ctxName, endpoint, apiToken, kubeConfig, kubeContext, getOutputFmt, endpointCACertPath string

	projectStr, spaceStr, clustergroupStr string
	contextTypeStr                        string
)

const (
	knownGlobalHost      = "cloud.vmware.com"
	defaultTanzuEndpoint = "https://api.tanzu.cloud.vmware.com"
	isPinnipedEndpoint   = "isPinnipedEndpoint"

	contextNotExistsForContextType      = "The provided context '%v' does not exist or is not active for the given context type '%v'"
	noActiveContextExistsForContextType = "There is no active context for the given context type '%v'"
	contextNotActiveOrNotExists         = "The provided context '%v' is not active or does not exist"
	contextForContextTypeSetInactive    = "The context '%v' of type '%v' has been set as inactive"
	deactivatingPlugin                  = "Deactivating plugin '%v:%v' for context '%v'"

	invalidTargetErrorForContextCommands = "invalid target specified. Please specify a correct value for the `--target` flag from 'kubernetes[k8s]/mission-control[tmc]'"
	invalidContextType                   = "invalid context type specified. Please specify a correct value for the `--type/-t` flag from 'kubernetes[k8s]/mission-control[tmc]/tanzu'"
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

var contextCmd = &cobra.Command{
	Use:     "context",
	Short:   "Configure and manage contexts for the Tanzu CLI",
	Aliases: []string{"ctx", "contexts"},
	Annotations: map[string]string{
		"group": string(plugin.SystemCmdGroup),
	},
}

func init() {
	contextCmd.SetUsageFunc(cli.SubCmdUsageFunc)
	contextCmd.AddCommand(
		createCtxCmd,
		listCtxCmd,
		getCtxCmd,
		deleteCtxCmd,
		useCtxCmd,
		unsetCtxCmd,
		getCtxTokenCmd,
		newUpdateCtxCmd(),
	)

	initCreateCtxCmd()

	listCtxCmd.Flags().StringVarP(&targetStr, "target", "", "", "list only contexts associated with the specified target (kubernetes[k8s]/mission-control[tmc])")
	utils.PanicOnErr(listCtxCmd.RegisterFlagCompletionFunc("target", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sContextType, compTMCContextType}, cobra.ShellCompDirectiveNoFileComp
	}))

	listCtxCmd.Flags().StringVarP(&contextTypeStr, "type", "t", "", "list only contexts associated with the specified context-type (kubernetes[k8s]/mission-control[tmc]/tanzu)")
	utils.PanicOnErr(listCtxCmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sContextType, compTanzuContextType, compTMCContextType}, cobra.ShellCompDirectiveNoFileComp
	}))

	listCtxCmd.Flags().BoolVar(&onlyCurrent, "current", false, "list only current active contexts")
	listCtxCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format: table|yaml|json")
	utils.PanicOnErr(listCtxCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))

	getCtxCmd.Flags().StringVarP(&getOutputFmt, "output", "o", "yaml", "output format: yaml|json")
	utils.PanicOnErr(getCtxCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))

	deleteCtxCmd.Flags().BoolVarP(&unattended, "yes", "y", false, "delete the context entry without confirmation")

	unsetCtxCmd.Flags().StringVarP(&targetStr, "target", "", "", "unset active context associated with the specified target (kubernetes[k8s]|mission-control[tmc])")
	utils.PanicOnErr(unsetCtxCmd.RegisterFlagCompletionFunc("target", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sContextType, compTMCContextType}, cobra.ShellCompDirectiveNoFileComp
	}))
	unsetCtxCmd.Flags().StringVarP(&contextTypeStr, "type", "t", "", "unset active context associated with the specified context-type (kubernetes[k8s]|mission-control[tmc]|tanzu)")
	utils.PanicOnErr(unsetCtxCmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sContextType, compTanzuContextType, compTMCContextType}, cobra.ShellCompDirectiveNoFileComp
	}))

	msg := "this was done in the v1.1.0 release, it will be removed following the deprecation policy (6 months). Use the --type flag instead.\n"
	utils.PanicOnErr(listCtxCmd.Flags().MarkDeprecated("target", msg))
	utils.PanicOnErr(unsetCtxCmd.Flags().MarkDeprecated("target", msg))
}

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

    # Create an Tanzu context with the default endpoint (--type is not necessary for the default endpoint)
    tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com

    # Create an Tanzu context (--type is needed for a non-default endpoint)
    tanzu context create mytanzu --endpoint https://non-default.tanzu.endpoint.com --type tanzu

    # Create an Tanzu context by using the provided CA Bundle for TLS verification:
    tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com  --endpoint-ca-certificate /path/to/ca/ca-cert

    # Create an Tanzu context but skipping TLS verification (this is insecure):
    tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com --insecure-skip-tls-verify

    Note: The "tanzu" context type is being released to provide advance support for the development
    and release of new services (and CLI plugins) which extend and combine features provided by
    individual tanzu components.

    Notes: 
    1. TMC context: To create Mission Control (TMC) context an API Key is required. It can be provided using the 
       TANZU_API_TOKEN environment variable or entered during context creation.
    2. Tanzu context: To create Tanzu context an API Key is optional. If provided using the TANZU_API_TOKEN environment
       variable, it will be used. Otherwise, the CLI will attempt to log in interactively to the user's default Cloud Services
       organization. You can override or choose a custom organization by setting the TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID 
       environment variable with the custom organization ID value. More information regarding organizations in Cloud Services
       and how to obtain the organization ID can be found at
       https://docs.vmware.com/en/VMware-Cloud-services/services/Using-VMware-Cloud-Services/GUID-CF9E9318-B811-48CF-8499-9419997DC1F8.html

    [*] : Users have two options to create a kubernetes cluster context. They can choose the control
    plane option by providing 'endpoint', or use the kubeconfig for the cluster by providing
    'kubeconfig' and 'context'. If only '--context' is set and '--kubeconfig' is not, the
    $KUBECONFIG env variable will be used and, if the $KUBECONFIG env is also not set, the default
    kubeconfig file ($HOME/.kube/config) will be used.`,
}

func initCreateCtxCmd() {
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
}

func createCtx(cmd *cobra.Command, args []string) (err error) {
	// The context name is an optional argument to allow for the prompt to be used
	if len(args) > 0 {
		if ctxName != "" {
			return fmt.Errorf("cannot specify the context name as an argument and with the --name flag at the same time")
		}
		ctxName = args[0]
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
		err = globalTanzuLogin(ctx)
	} else {
		err = globalLogin(ctx)
	}

	if err != nil {
		return err
	}

	// Sync all required plugins
	_ = syncContextPlugins(cmd, ctx.ContextType, ctxName, true)

	return nil
}

// syncContextPlugins syncs the plugins for the given context type
// if listPlugins is true, it will list the plugins that will be installed for the given context type
func syncContextPlugins(cmd *cobra.Command, contextType configtypes.ContextType, ctxName string, listPlugins bool) error {
	plugins, err := pluginmanager.DiscoverPluginsForContextType(contextType)
	errList := make([]error, 0)
	if err != nil {
		errList = append(errList, err)
	}

	// update plugins installation status
	pluginmanager.UpdatePluginsInstallationStatus(plugins)

	// list plugins only if listPlugins is true and there are plugins to be installed
	if listPlugins {
		pluginsNeedstoBeInstalled := 0
		for idx := range plugins {
			if plugins[idx].Status == common.PluginStatusNotInstalled || plugins[idx].Status == common.PluginStatusUpdateAvailable {
				pluginsNeedstoBeInstalled++
			}
		}
		if pluginsNeedstoBeInstalled > 0 {
			log.Infof("The following plugins will be installed for context '%s' of contextType '%s': ", ctxName, contextType)
			displayUninstalledPluginsContentAsTable(plugins, cmd.ErrOrStderr())
		}
	}

	err = pluginmanager.InstallDiscoveredContextPlugins(plugins)
	if err != nil {
		errList = append(errList, err)
	}
	err = kerrors.NewAggregate(errList)
	if err != nil {
		log.Warningf("unable to automatically sync the plugins from target context. Please run 'tanzu plugin sync' command to sync plugins manually, error: '%v'", err.Error())
	}
	return err
}

// displayUninstalledPluginsContentAsTable takes a list of plugins and writes the uninstalled plugins as a table
func displayUninstalledPluginsContentAsTable(plugins []discovery.Discovered, writer io.Writer) {
	outputUninstalledPlugins := component.NewOutputWriterWithOptions(writer, outputFormat, []component.OutputWriterOption{}, "Name", "Target", "Version")
	for i := range plugins {
		if plugins[i].Status == common.PluginStatusNotInstalled || plugins[i].Status == common.PluginStatusUpdateAvailable {
			outputUninstalledPlugins.AddRow(plugins[i].Name, plugins[i].Target, plugins[i].RecommendedVersion)
		}
	}
	outputUninstalledPlugins.Render()
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

func isGlobalTanzuEndpoint(endpoint string) bool {
	for _, hostStr := range []string{"api.tanzu.cloud.vmware.com", "api.tanzu-dev.cloud.vmware.com", "api.tanzu-stable.cloud.vmware.com "} {
		if strings.Contains(endpoint, hostStr) {
			return true
		}
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

	if (contextType == configtypes.ContextTypeTanzu) || (endpoint != "" && isGlobalTanzuEndpoint(endpoint)) {
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
			return
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
			return
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
			return
		}
	}
	ctxName = strings.TrimSpace(ctxName)
	exists, err := config.ContextExists(ctxName)
	if err != nil {
		return context, err
	}
	if exists {
		err = fmt.Errorf("context %q already exists", ctxName)
		return
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
			return
		}
	}
	if ctxName == "" {
		ctxName, err = promptContextName("")
		if err != nil {
			return
		}
	}
	exists, err := config.ContextExists(ctxName)
	if err != nil {
		return context, err
	}
	if exists {
		err = fmt.Errorf("context %q already exists", ctxName)
		return
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
			return
		}
	}
	if ctxName == "" {
		ctxName, err = promptContextName("")
		if err != nil {
			return
		}
	}
	exists, err := config.ContextExists(ctxName)
	if err != nil {
		return context, err
	}
	if exists {
		err = fmt.Errorf("context %q already exists", ctxName)
		return
	}

	// TKGKubeconfigFetcher would detect the endpoint is TKGm/TKGs and then fetch the pinniped kubeconfig to create a context
	tkf := NewTKGKubeconfigFetcher(endpoint, endpointCACertPath, skipTLSVerify)
	kubeConfig, kubeContext, err = tkf.GetPinnipedKubeconfig()
	if err != nil {
		return
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

func createContextWithTanzuEndpoint() (context *configtypes.Context, err error) {
	if endpoint == "" {
		endpoint, err = promptEndpoint(defaultTanzuEndpoint)
		if err != nil {
			return
		}
	}
	if ctxName == "" {
		ctxName, err = promptContextName("")
		if err != nil {
			return
		}
	}

	exists, err := config.ContextExists(ctxName)
	if err != nil {
		return context, err
	}
	if exists {
		err = fmt.Errorf("context %q already exists", ctxName)
		return
	}

	// Tanzu context would have both CSP(GlobalOpts) auth details and kubeconfig(ClusterOpts),
	context = &configtypes.Context{
		Name:        ctxName,
		ContextType: configtypes.ContextTypeTanzu,
		GlobalOpts:  &configtypes.GlobalServer{Endpoint: sanitizeEndpoint(endpoint)},
		ClusterOpts: &configtypes.ClusterServer{},
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
	fmt.Println()
	log.Success("successfully created a TMC context")
	return nil
}

func globalTanzuLogin(c *configtypes.Context) error {
	var claims *csp.Claims
	var err error
	apiTokenValue, apiTokenExists := os.LookupEnv(config.EnvAPITokenKey)
	// Use API Token login flow if TANZU_API_TOKEN environment variable is set, else fall back to default interactive login flow
	if apiTokenExists {
		log.Info("API token env var is set")
		claims, err = doCSPAPITokenAuthAndUpdateContext(c, apiTokenValue)
	} else {
		claims, err = doCSPInteractiveLoginAndUpdateContext(c)
	}
	if err != nil {
		return err
	}
	c.AdditionalMetadata[config.OrgIDKey] = claims.OrgID

	kubeCfg, kubeCtx, serverEndpoint, err := tanzuauth.GetTanzuKubeconfig(c, endpoint, claims.OrgID, endpointCACertPath, skipTLSVerify)
	if err != nil {
		return err
	}

	c.ClusterOpts.Path = kubeCfg
	c.ClusterOpts.Context = kubeCtx
	c.ClusterOpts.Endpoint = serverEndpoint

	err = config.AddContext(c, true)
	if err != nil {
		return err
	}
	// update the current context in the kubeconfig file after creating the context
	err = syncCurrentKubeContext(c)
	if err != nil {
		return errors.Wrap(err, "unable to update current kube context")
	}

	// format
	fmt.Println()
	orgName := getCSPOrgName(c, claims)
	// If the orgName fetching API fails(corner case), we only print the tanzu context creation success message
	msg := "Successfully created a tanzu context"
	if orgName != "" {
		msg = fmt.Sprintf("Successfully logged into '%s' organization and created a tanzu context", orgName)
	}
	log.Success(msg)
	return nil
}

// getCSPOrgName returns the CSP Org name using the orgID from the claims.
// It will return empty string if API fails
func getCSPOrgName(c *configtypes.Context, claims *csp.Claims) string {
	issuer := csp.ProdIssuer
	if staging {
		issuer = csp.StgIssuer
	}
	orgName, err := csp.GetOrgNameFromOrgID(claims.OrgID, c.GlobalOpts.Auth.AccessToken, issuer)
	if err != nil {
		return ""
	}
	return orgName
}

func doCSPInteractiveLoginAndUpdateContext(c *configtypes.Context) (claims *csp.Claims, err error) {
	issuer := csp.ProdIssuer
	if staging {
		issuer = csp.StgIssuer
	}
	cspOrgIDValue, cspOrgIDExists := os.LookupEnv(constants.CSPLoginOrgID)
	var options []csp.LoginOption
	if cspOrgIDExists && cspOrgIDValue != "" {
		options = append(options, csp.WithOrgID(cspOrgIDValue))
	}
	token, err := csp.TanzuLogin(issuer, options...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the token from CSP")
	}
	claims, err = csp.ParseToken(&oauth2.Token{AccessToken: token.AccessToken})
	if err != nil {
		return nil, err
	}

	a := configtypes.GlobalServerAuth{}
	a.Issuer = issuer
	a.UserName = claims.Username
	a.Permissions = claims.Permissions
	a.AccessToken = token.AccessToken
	a.IDToken = token.IDToken
	a.RefreshToken = token.RefreshToken
	a.Type = token.TokenType
	expiresAt := time.Now().Local().Add(time.Second * time.Duration(token.ExpiresIn))
	a.Expiration = expiresAt
	c.GlobalOpts.Auth = a
	c.AdditionalMetadata = make(map[string]interface{})

	return claims, nil
}

func doCSPAPITokenAuthAndUpdateContext(c *configtypes.Context, apiTokenValue string) (claims *csp.Claims, err error) {
	issuer := csp.ProdIssuer
	if staging {
		issuer = csp.StgIssuer
	}
	token, err := csp.GetAccessTokenFromAPIToken(apiTokenValue, issuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the token from CSP")
	}
	claims, err = csp.ParseToken(&oauth2.Token{AccessToken: token.AccessToken})
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
	a.Type = csp.APITokenType
	expiresAt := time.Now().Local().Add(time.Second * time.Duration(token.ExpiresIn))
	a.Expiration = expiresAt
	c.GlobalOpts.Auth = a
	c.AdditionalMetadata = make(map[string]interface{})

	return claims, nil
}

func promptContextType() (ctxCreationType ContextCreationType, err error) {
	ctxCreationTypeStr := ""
	promptOpts := getPromptOpts()

	fmt.Print(`
Note: The "tanzu" context type is being released to provide advance support for the development
and release of new services (and CLI plugins) which extend and combine features provided by
individual tanzu components.

`)

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

var listCtxCmd = &cobra.Command{
	Use:               "list",
	Short:             "List contexts",
	ValidArgsFunction: noMoreCompletions,
	RunE:              listCtx,
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

	if outputFormat == "" || outputFormat == string(component.TableOutputType) {
		displayContextListOutputWithDynamicColumns(cfg, cmd.OutOrStdout())
	} else {
		displayContextListOutputListView(cfg, cmd.OutOrStdout())
	}

	return nil
}

var getCtxCmd = &cobra.Command{
	Use:               "get CONTEXT_NAME",
	Short:             "Display a context from the config",
	ValidArgsFunction: completeAllContexts,
	RunE:              getCtx,
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

var deleteCtxCmd = &cobra.Command{
	Use:               "delete CONTEXT_NAME",
	Short:             "Delete a context from the config",
	ValidArgsFunction: completeAllContexts,
	RunE:              deleteCtx,
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

	if !unattended {
		isAborted := component.AskForConfirmation("Deleting the context entry from the config will remove it from the list of tracked contexts. " +
			"You will need to use `tanzu context create` to re-create this context. Are you sure you want to continue?")
		if isAborted != nil {
			return nil
		}
	}
	ctx, err := config.GetContext(name)
	if err != nil {
		return err
	}
	installed, _, _, _ := getInstalledAndMissingContextPlugins() //nolint:dogsled
	log.Infof("Deleting entry for context '%s'", name)
	err = config.RemoveContext(name)
	if err != nil {
		return err
	}
	listDeactivatedPlugins(installed, name)
	deleteKubeconfigContext(ctx)

	return nil
}

func deleteKubeconfigContext(ctx *configtypes.Context) {
	// Note: currently cleaning up the kubeconfig for tanzu context types only.
	// (Since the kubernetes context type can have kube context provided by the user, it may not be
	// desired outcome for user if CLI deletes/cleanup kubeconfig provided by the user.)
	if ctx.ContextType == configtypes.ContextTypeTanzu || isPinnipedEndpointContext(ctx) {
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

var useCtxCmd = &cobra.Command{
	Use:               "use CONTEXT_NAME",
	Short:             "Set the context to be used by default",
	ValidArgsFunction: completeAllContexts,
	RunE:              useCtx,
}

func useCtx(cmd *cobra.Command, args []string) error {
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

	log.Infof("Successfully activated context '%s'", ctxName)

	// Sync all required plugins
	_ = syncContextPlugins(cmd, ctx.ContextType, ctxName, true)

	return nil
}

func syncCurrentKubeContext(ctx *configtypes.Context) error {
	if skipSync, _ := strconv.ParseBool(os.Getenv(constants.SkipUpdateKubeconfigOnContextUse)); skipSync {
		return nil
	}
	return kubecfg.SetCurrentContext(ctx.ClusterOpts.Path, ctx.ClusterOpts.Context)
}

var unsetCtxCmd = &cobra.Command{
	Use:               "unset CONTEXT_NAME",
	Short:             "Unset the active context so that it is not used by default",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeActiveContexts,
	RunE:              unsetCtx,
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
	installed, _, _, _ := getInstalledAndMissingContextPlugins() //nolint:dogsled
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
		listDeactivatedPlugins(installed, name)
	}
	return nil
}

// listDeactivatedPlugins stdout the plugins that are being deactivated
func listDeactivatedPlugins(deactivatedPlugins []discovery.Discovered, ctxName string) {
	for i := range deactivatedPlugins {
		if (deactivatedPlugins)[i].ContextName == ctxName {
			log.Outputf(deactivatingPlugin, (deactivatedPlugins)[i].Name, deactivatedPlugins[i].InstalledVersion, ctxName)
		}
	}
}

func displayContextListOutputListView(cfg *configtypes.ClientConfig, writer io.Writer) {
	contextType := getContextType()

	// switching to use the new OutputWriter because we want to render the
	// additional metadata map correctly in their native JSON/YAML form
	opts := []component.OutputWriterOption{}
	op := component.NewOutputWriterWithOptions(writer, outputFormat, opts, "Name", "Type", "IsManagementCluster", "IsCurrent", "Endpoint", "KubeConfigPath", "KubeContext", "AdditionalMetadata")

	for _, ctx := range cfg.KnownContexts {
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
			ep = ctx.GlobalOpts.Endpoint
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
	Endpoint       string
	KubeconfigPath string
	KubeContext    string
	Project        string
	Space          string
	ClusterGroup   string
}

func displayContextListOutputWithDynamicColumns(cfg *configtypes.ClientConfig, writer io.Writer) {
	ct := getContextType()
	ctxs, _ := getContextsToDisplay(cfg, ct, onlyCurrent)

	opts := []component.OutputWriterOption{}
	rows := []ContextListOutputRow{}
	for _, ctx := range ctxs {
		ep := NA
		path := NA
		context := NA
		project := NA
		space := NA
		clustergroup := NA

		isCurrent := ctx.Name == cfg.CurrentContext[ctx.ContextType]

		switch ctx.ContextType {
		case configtypes.ContextTypeTMC:
			if ctx.GlobalOpts != nil {
				ep = ctx.GlobalOpts.Endpoint
			}
		case configtypes.ContextTypeTanzu:
			project = ""
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
		row := ContextListOutputRow{ctx.Name, strconv.FormatBool(isCurrent), string(ctx.ContextType), ep, path, context, project, space, clustergroup}
		rows = append(rows, row)
	}

	dynamicTableWriter(rows, component.NewOutputWriterWithOptions(writer, outputFormat, opts, "NAME", "ISACTIVE", "TYPE"))
}

var getCtxTokenCmd = &cobra.Command{
	Use:               "get-token CONTEXT_NAME",
	Short:             "Get the valid CSP token for the given tanzu context",
	Args:              cobra.ExactArgs(1),
	Hidden:            true,
	ValidArgsFunction: completeTanzuContexts,
	RunE:              getToken,
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
	if csp.IsExpired(ctx.GlobalOpts.Auth.Expiration) {
		_, err := csp.GetToken(&ctx.GlobalOpts.Auth)
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
	tanzuActiveResourceCmd.Flags().StringVarP(&projectStr, "project", "", "", "project name to be set as active")
	tanzuActiveResourceCmd.Flags().StringVarP(&spaceStr, "space", "", "", "space name to be set as active")
	tanzuActiveResourceCmd.Flags().StringVarP(&clustergroupStr, "clustergroup", "", "", "clustergroup name to be set as active")

	updateCtxCmd.AddCommand(
		tanzuActiveResourceCmd,
	)
	return updateCtxCmd
}

// tanzuActiveResourceCmd updates the tanzu active resource referenced by tanzu context
//
// NOTE!!: This command is EXPERIMENTAL and subject to change in future
var tanzuActiveResourceCmd = &cobra.Command{
	Use:               "tanzu-active-resource CONTEXT_NAME",
	Short:             "Updates the tanzu active resource for the given context of type tanzu (subject to change)",
	Hidden:            true,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeTanzuContexts,
	RunE:              setTanzuCtxActiveResource,
}

func setTanzuCtxActiveResource(_ *cobra.Command, args []string) error {
	name := args[0]

	if spaceStr != "" && clustergroupStr != "" {
		return errors.Errorf("either space or clustergroup can be set as active resource. Please provide either --space or --clustergroup option")
	}
	if projectStr == "" && spaceStr != "" {
		return errors.Errorf("space cannot be set without project name. Please provide project name also using --project option")
	}
	if projectStr == "" && clustergroupStr != "" {
		return errors.Errorf("clustergroup cannot be set without project name. Please provide project name also using --project option")
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
	ctx.AdditionalMetadata[config.SpaceNameKey] = spaceStr
	ctx.AdditionalMetadata[config.ClusterGroupNameKey] = clustergroupStr
	err = config.SetContext(ctx, false)
	if err != nil {
		return errors.Wrap(err, "failed updating the context %q with the active tanzu resource")
	}
	err = updateTanzuContextKubeconfig(ctx, projectStr, spaceStr, clustergroupStr)
	if err != nil {
		return errors.Wrap(err, "failed to update the tanzu context kubeconfig")
	}

	return nil
}

func updateTanzuContextKubeconfig(cliContext *configtypes.Context, projectName, spaceName, clustergroupName string) error {
	kcfg, err := clientcmd.LoadFromFile(cliContext.ClusterOpts.Path)
	if err != nil {
		return errors.Wrap(err, "unable to load kubeconfig")
	}

	kubeContext := kcfg.Contexts[cliContext.ClusterOpts.Context]
	if kubeContext == nil {
		return errors.Errorf("kubecontext %q doesn't exist", cliContext.ClusterOpts.Context)
	}
	cluster := kcfg.Clusters[kubeContext.Cluster]
	cluster.Server = prepareClusterServerURL(cliContext, projectName, spaceName, clustergroupName)
	err = clientcmd.WriteToFile(*kcfg, cliContext.ClusterOpts.Path)
	if err != nil {
		return errors.Wrap(err, "failed to update the context kubeconfig file")
	}
	return nil
}

func prepareClusterServerURL(context *configtypes.Context, projectName, spaceName, clustergroupName string) string {
	serverURL := context.ClusterOpts.Endpoint
	if projectName == "" {
		return serverURL
	}
	serverURL = serverURL + "/project/" + projectName

	if spaceName != "" {
		return serverURL + "/space/" + spaceName
	}
	if clustergroupName != "" {
		return serverURL + "/clustergroup/" + clustergroupName
	}
	return serverURL
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

// dynamicTableWriter writes the data in table format dynamically by hiding column if all related rows for a column is `n/a`
func dynamicTableWriter(slices interface{}, tableWriter component.OutputWriter) {
	// Check if the input is a slice
	valueOf := reflect.ValueOf(slices)
	if valueOf.Kind() == reflect.Slice && valueOf.Len() > 0 {
		// Collect header and column data
		header := []string{}
		isColumnFilled := make(map[int]bool)

		for i := 0; i < valueOf.Len(); i++ {
			elem := valueOf.Index(i)
			elemValue := reflect.ValueOf(elem.Interface())

			// Determine which columns are filled for this element
			for j := 0; j < elemValue.NumField(); j++ {
				field := elemValue.Field(j)
				fieldValue := field.Interface()
				isNA := reflect.DeepEqual(fieldValue, NA)
				if !isNA {
					isColumnFilled[j] = true
				}
			}
		}

		// Build the header based on the first element
		elem := valueOf.Index(0)
		elemValue := reflect.ValueOf(elem.Interface())
		for j := 0; j < elemValue.NumField(); j++ {
			if _, exists := isColumnFilled[j]; exists {
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
				if _, exists := isColumnFilled[j]; exists {
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
