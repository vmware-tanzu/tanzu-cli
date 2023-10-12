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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
	taert "github.com/vmware-tanzu/tanzu-plugin-runtime/tae"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/csp"
	taeauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tae"
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
	stderrOnly, forceCSP, staging, onlyCurrent, skipTLSVerify                                           bool
	ctxName, endpoint, apiToken, kubeConfig, kubeContext, getOutputFmt, endpointCACertPath, contextType string
	projectStr, spaceStr                                                                                string
)

const (
	knownGlobalHost    = "cloud.vmware.com"
	apiTokenType       = "api-token"
	defaultTAEEndpoint = "https://api.tanzu.cloud.vmware.com"

	contextNotExistsForTarget      = "The provided context %v does not exist or is not active for the given target %v"
	noActiveContextExistsForTarget = "There is no active context for the given target %v"
	contextNotActiveOrNotExists    = "The provided context %v is not active or does not exist"
	contextForTargetSetInactive    = "The context %v for the target %v has been set as inactive"
	deactivatingPlugin             = "Deactivating plugin '%v:%v' for context '%v'"

	invalidTarget = "invalid target specified. Please specify a correct value for the `--target/-t` flag from 'kubernetes[k8s]/mission-control[tmc]/application-engine[tae]"
)

// constants that define context creation types
const (
	ContextMissionControl     ContextCreationType = "Mission Control"
	ContextK8SClusterEndpoint ContextCreationType = "Kubernetes (Cluster Endpoint)"
	ContextLocalKubeconfig    ContextCreationType = "Kubernetes (Local Kubeconfig)"
	ContextApplicationEngine  ContextCreationType = "Application Engine"
)

type ContextCreationType string

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

	listCtxCmd.Flags().StringVarP(&targetStr, "target", "t", "", "list only contexts associated with the specified target (kubernetes[k8s]/mission-control[tmc]/application-engine[tae])")
	utils.PanicOnErr(listCtxCmd.RegisterFlagCompletionFunc("target", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sTarget, compTAETarget, compTMCTarget}, cobra.ShellCompDirectiveNoFileComp
	}))

	listCtxCmd.Flags().BoolVar(&onlyCurrent, "current", false, "list only current active contexts")
	listCtxCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format: table|yaml|json")
	utils.PanicOnErr(listCtxCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))

	getCtxCmd.Flags().StringVarP(&getOutputFmt, "output", "o", "yaml", "output format: yaml|json")
	utils.PanicOnErr(getCtxCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))

	deleteCtxCmd.Flags().BoolVarP(&unattended, "yes", "y", false, "delete the context entry without confirmation")

	unsetCtxCmd.Flags().StringVarP(&targetStr, "target", "t", "", "unset active context associated with the specified target (kubernetes[k8s]|mission-control[tmc]|application-engine[tae])")
	utils.PanicOnErr(unsetCtxCmd.RegisterFlagCompletionFunc("target", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{compK8sTarget, compTAETarget, compTMCTarget}, cobra.ShellCompDirectiveNoFileComp
	}))
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

    # Create an Application Engine (TAE) context with the default endpoint (--type is not necessary for the default endpoint)
    tanzu context create mytae --endpoint https://api.tanzu.cloud.vmware.com

    # Create an Application Engine (TAE) context (--type is needed for a non-default endpoint)
    tanzu context create mytae --endpoint https://non-default.tae.endpoint.com --type application-engine

    # Create an Application Engine (TAE) context by using the provided CA Bundle for TLS verification:
    tanzu context create mytae --endpoint https://api.tanzu.cloud.vmware.com  --endpoint-ca-certificate /path/to/ca/ca-cert

    # Create an Application Engine (TAE) context but skipping TLS verification (this is insecure):
    tanzu context create mytae --endpoint https://api.tanzu.cloud.vmware.com --insecure-skip-tls-verify

    [*] : Users have two options to create a kubernetes cluster context. They can choose the control
    plane option by providing 'endpoint', or use the kubeconfig for the cluster by providing
    'kubeconfig' and 'context'. If only '--context' is set and '--kubeconfig' is not, the
    $KUBECONFIG env variable will be used and, if the $KUBECONFIG env is also not set, the default
    kubeconfig file ($HOME/.kube/config) will be used.`,
}

func initCreateCtxCmd() {
	createCtxCmd.Flags().StringVar(&ctxName, "name", "", "name of the context")
	_ = createCtxCmd.Flags().MarkDeprecated("name", "it has been replaced by using an argument to the command")

	createCtxCmd.Flags().StringVar(&endpoint, "endpoint", "", "endpoint to create a context for")
	utils.PanicOnErr(createCtxCmd.RegisterFlagCompletionFunc("endpoint", cobra.NoFileCompletions))

	createCtxCmd.Flags().StringVar(&apiToken, "api-token", "", "API token for the SaaS context")
	utils.PanicOnErr(createCtxCmd.RegisterFlagCompletionFunc("api-token", cobra.NoFileCompletions))

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
	createCtxCmd.Flags().StringVar(&contextType, "type", "", "type of context to create (kubernetes[k8s]/mission-control[tmc]/application-engine[tae])")
	utils.PanicOnErr(createCtxCmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{
				"tmc\tContext for a Tanzu Mission Control endpoint",
				"tae\tContext for a Tanzu Application Engine endpoint",
				"k8s\tContext for a Kubernetes cluster"},
			cobra.ShellCompDirectiveNoFileComp
	}))

	_ = createCtxCmd.Flags().MarkHidden("api-token")
	_ = createCtxCmd.Flags().MarkHidden("stderr-only")
	_ = createCtxCmd.Flags().MarkHidden("force-csp")
	_ = createCtxCmd.Flags().MarkHidden("staging")
	createCtxCmd.MarkFlagsMutuallyExclusive("endpoint", "kubecontext")
	createCtxCmd.MarkFlagsMutuallyExclusive("endpoint", "kubeconfig")
	createCtxCmd.MarkFlagsMutuallyExclusive("endpoint-ca-certificate", "insecure-skip-tls-verify")
}

func createCtx(_ *cobra.Command, args []string) (err error) {
	// The context name is an optional argument to allow for the prompt to be used
	if len(args) > 0 {
		if ctxName != "" {
			return fmt.Errorf("cannot specify the context name as an argument and with the --name flag at the same time")
		}
		ctxName = args[0]
	}

	ctx, err := createNewContext()
	if err != nil {
		return err
	}
	if ctx.Target == configtypes.TargetK8s {
		err = k8sLogin(ctx)
	} else if ctx.Target == configtypes.TargetTAE {
		err = globalTAELogin(ctx)
	} else {
		err = globalLogin(ctx)
	}

	if err != nil {
		return err
	}

	// Sync all required plugins
	_ = syncContextPlugins()

	return nil
}

func syncContextPlugins() error {
	err := pluginmanager.SyncPlugins()
	if err != nil {
		log.Warningf("unable to automatically sync the plugins from target context. Please run 'tanzu plugin sync' command to sync plugins manually, error: '%v'", err.Error())
	}
	return err
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

func isGlobalTAEEndpoint(endpoint string) bool {
	for _, hostStr := range []string{"api.tanzu.cloud.vmware.com", "api-dev.tanzu.cloud.vmware.com"} {
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

func createNewContext() (context *configtypes.Context, err error) { //nolint:gocyclo
	var ctxCreationType ContextCreationType
	contextType = strings.TrimSpace(contextType)

	if (contextType == "application-engine" || contextType == "tae") || (endpoint != "" && isGlobalTAEEndpoint(endpoint)) {
		ctxCreationType = ContextApplicationEngine
	} else if (contextType == "mission-control" || contextType == "tmc") || (endpoint != "" && isGlobalContext(endpoint)) { //nolint: goconst
		ctxCreationType = ContextMissionControl
	} else if endpoint != "" {
		// user provided command line option endpoint is provided that is not globalTAE or GlobalContext=> it is Kubernetes(Cluster Endpoint) type
		ctxCreationType = ContextK8SClusterEndpoint
	} else if kubeContext != "" {
		// user provided command line option kubeContext is provided => it is Kubernetes(Local Kubeconfig) type
		ctxCreationType = ContextLocalKubeconfig
	} else if contextType == "kubernetes" || contextType == "k8s" {
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
	case ContextMissionControl:
		ctxCreateFunc = createContextWithTMCEndpoint
	case ContextK8SClusterEndpoint:
		ctxCreateFunc = createContextWithClusterEndpoint
	case ContextLocalKubeconfig:
		ctxCreateFunc = createContextWithKubeconfig
	case ContextApplicationEngine:
		ctxCreateFunc = createContextWithTAEEndpoint
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
		kubeConfig = getDefaultKubeconfigPath()
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
		Name:   ctxName,
		Target: configtypes.TargetK8s,
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

	context = &configtypes.Context{
		Name:       ctxName,
		Target:     configtypes.TargetTMC,
		GlobalOpts: &configtypes.GlobalServer{Endpoint: sanitizeEndpoint(endpoint)},
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
		Name:   ctxName,
		Target: configtypes.TargetK8s,
		ClusterOpts: &configtypes.ClusterServer{
			Path:                kubeConfig,
			Context:             kubeContext,
			Endpoint:            endpoint,
			IsManagementCluster: true,
		},
	}
	return context, err
}

func createContextWithTAEEndpoint() (context *configtypes.Context, err error) {
	if endpoint == "" {
		endpoint, err = promptEndpoint(defaultTAEEndpoint)
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

	// TAE context would have both CSP(GlobalOpts) auth details and kubeconfig(ClusterOpts),
	context = &configtypes.Context{
		Name:        ctxName,
		Target:      configtypes.TargetTAE,
		GlobalOpts:  &configtypes.GlobalServer{Endpoint: sanitizeEndpoint(endpoint)},
		ClusterOpts: &configtypes.ClusterServer{},
	}
	return context, err
}
func globalLogin(c *configtypes.Context) (err error) {
	_, err = doCSPAuthAndUpdateContext(c, "TMC")
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

func globalTAELogin(c *configtypes.Context) error {
	claims, err := doCSPAuthAndUpdateContext(c, "TAE")
	if err != nil {
		return err
	}
	c.AdditionalMetadata[taert.OrgIDKey] = claims.OrgID

	kubeCfg, kubeCtx, serverEndpoint, err := taeauth.GetTAEKubeconfig(c, endpoint, claims.OrgID, endpointCACertPath, skipTLSVerify)
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
	log.Success("successfully created a Application Engine(TAE) context")
	return nil
}

func doCSPAuthAndUpdateContext(c *configtypes.Context, endpointType string) (claims *csp.Claims, err error) {
	apiTokenValue, apiTokenExists := os.LookupEnv(config.EnvAPITokenKey)

	issuer := csp.ProdIssuer
	if staging {
		issuer = csp.StgIssuer
	}
	if apiTokenExists {
		log.Info("API token env var is set")
	} else {
		apiTokenValue, err = promptAPIToken(endpointType)
		if err != nil {
			return nil, err
		}
	}
	token, err := csp.GetAccessTokenFromAPIToken(apiTokenValue, issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token from CSP for %s", endpointType)
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
	a.Type = apiTokenType
	expiresAt := time.Now().Local().Add(time.Second * time.Duration(token.ExpiresIn))
	a.Expiration = expiresAt
	c.GlobalOpts.Auth = a
	c.AdditionalMetadata = make(map[string]interface{})

	return claims, nil
}

func promptContextType() (ctxCreationType ContextCreationType, err error) {
	ctxCreationTypeStr := ""
	promptOpts := getPromptOpts()
	err = component.Prompt(
		&component.PromptConfig{
			Message: "Select context creation type",
			Options: []string{string(ContextMissionControl), string(ContextApplicationEngine), string(ContextK8SClusterEndpoint), string(ContextLocalKubeconfig)},
			Default: string(ContextMissionControl),
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
	if ctxCreationTypeStr == string(ContextMissionControl) {
		return ContextMissionControl
	} else if ctxCreationTypeStr == string(ContextApplicationEngine) {
		return ContextApplicationEngine
	} else if ctxCreationTypeStr == string(ContextK8SClusterEndpoint) {
		return ContextK8SClusterEndpoint
	} else if ctxCreationTypeStr == string(ContextLocalKubeconfig) {
		return ContextLocalKubeconfig
	}

	return ""
}

func promptKubernetesContextType() (ctxCreationType ContextCreationType, err error) {
	ctxCreationTypeStr := ""
	promptOpts := getPromptOpts()
	err = component.Prompt(
		&component.PromptConfig{
			Message: "Select the kubernetes context type",
			Options: []string{string(ContextLocalKubeconfig), string(ContextK8SClusterEndpoint)},
			Default: string(ContextLocalKubeconfig),
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

	// format
	fmt.Println()
	log.Infof(
		"If you don't have an API token, visit the VMware Cloud Services console, select your organization, and create an API token with the %s service roles:\n  %s\n",
		endpointType, consoleURL.String(),
	)

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

func getDefaultKubeconfigPath() string {
	kubeConfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	// fallback to default kubeconfig file location if no env variable set
	if kubeConfigPath == "" {
		kubeConfigPath = clientcmd.RecommendedHomeFile
	}
	return kubeConfigPath
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
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              listCtx,
}

func listCtx(cmd *cobra.Command, _ []string) error {
	cfg, err := config.GetClientConfig()
	if err != nil {
		return err
	}

	if !configtypes.IsValidTarget(targetStr, false, true) {
		return errors.New(invalidTarget)
	}

	if outputFormat == "" || outputFormat == string(component.TableOutputType) {
		displayContextListOutputSplitViewTarget(cfg, cmd.OutOrStdout())
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
	currentCtxMap, err := config.GetAllCurrentContextsMap()
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
		if info == "" && ctx.Target == configtypes.TargetK8s && ctx.ClusterOpts != nil {
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

func getValues(m map[configtypes.Target]*configtypes.Context) []*configtypes.Context {
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
	installed, _, _, _ := getInstalledAndMissingContextPlugins() //nolint:dogsled
	log.Infof("Deleting entry for context '%s'", name)
	err := config.RemoveContext(name)
	if err != nil {
		return err
	}
	listDeactivatedPlugins(installed, name)
	return nil
}

var useCtxCmd = &cobra.Command{
	Use:               "use CONTEXT_NAME",
	Short:             "Set the context to be used by default",
	ValidArgsFunction: completeAllContexts,
	RunE:              useCtx,
}

func useCtx(_ *cobra.Command, args []string) error {
	var name string
	var ctx *configtypes.Context
	var err error

	if len(args) == 0 {
		ctx, err := promptCtx()
		if err != nil {
			return err
		}
		name = ctx.Name
	} else {
		name = args[0]
	}

	ctx, err = config.GetContext(name)
	if err != nil {
		return err
	}

	if ctx.ClusterOpts != nil {
		err = syncCurrentKubeContext(ctx)
		if err != nil {
			return errors.Wrap(err, "unable to update current kube context")
		}
	}

	err = config.SetCurrentContext(name)
	if err != nil {
		return err
	}

	// Sync all required plugins
	_ = syncContextPlugins()

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
	Short:             "Unset the active context so that it is not used by default.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeActiveContexts,
	RunE:              unsetCtx,
}

func unsetCtx(_ *cobra.Command, args []string) error {
	var name string
	if !configtypes.IsValidTarget(targetStr, false, true) {
		return errors.New(invalidTarget)
	}
	target := getTarget()
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" && target == "" {
		ctx, err := promptActiveCtx()
		if err != nil {
			return err
		}
		name = ctx.Name
	}
	return unsetGivenContext(name, target)
}

func unsetGivenContext(name string, target configtypes.Target) error {
	var unset bool
	installed, _, _, _ := getInstalledAndMissingContextPlugins() //nolint:dogsled
	currentCtxMap, err := config.GetAllCurrentContextsMap()
	if target != "" && name != "" {
		ctx, ok := currentCtxMap[target]
		if ok && ctx.Name == name {
			err = config.RemoveCurrentContext(target)
			unset = true
		} else {
			return errors.Errorf(contextNotExistsForTarget, name, target)
		}
	} else if target != "" {
		ctx, ok := currentCtxMap[target]
		if ok {
			name = ctx.Name
			err = config.RemoveCurrentContext(target)
			unset = true
		} else {
			log.Warningf(noActiveContextExistsForTarget, target)
		}
	} else if name != "" {
		for t, ctx := range currentCtxMap {
			if ctx.Name == name {
				target = t
				err = config.RemoveCurrentContext(t)
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
		log.Outputf(contextForTargetSetInactive, name, target)
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
	target := getTarget()
	op := component.NewOutputWriter(writer, outputFormat, "Name", "Type", "IsManagementCluster", "IsCurrent", "Endpoint", "KubeConfigPath", "KubeContext", "AdditionalMetadata")
	for _, ctx := range cfg.KnownContexts {
		if target != configtypes.TargetUnknown && ctx.Target != target {
			continue
		}
		isMgmtCluster := ctx.IsManagementCluster()
		isCurrent := ctx.Name == cfg.CurrentContext[ctx.Target]
		if onlyCurrent && !isCurrent {
			continue
		}

		var ep, path, context string
		switch ctx.Target {
		case configtypes.TargetTMC:
			ep = ctx.GlobalOpts.Endpoint
		default:
			if ctx.ClusterOpts != nil {
				ep = ctx.ClusterOpts.Endpoint
				path = ctx.ClusterOpts.Path
				context = ctx.ClusterOpts.Context
			}
		}

		op.AddRow(ctx.Name, ctx.Target, strconv.FormatBool(isMgmtCluster), strconv.FormatBool(isCurrent), ep, path, context, ctx.AdditionalMetadata)
	}
	op.Render()
}

// getContextsToDisplay returns a filtered list of contexts, and a boolean on
// whether the contexts include some with TAE fields to display
func getContextsToDisplay(cfg *configtypes.ClientConfig, target configtypes.Target, onlyCurrent bool) ([]*configtypes.Context, bool) {
	var contextOutputList []*configtypes.Context
	var hasTAEFields bool

	for _, ctx := range cfg.KnownContexts {
		if target != configtypes.TargetUnknown && ctx.Target != target {
			continue
		}
		isCurrent := ctx.Name == cfg.CurrentContext[ctx.Target]
		if onlyCurrent && !isCurrent {
			continue
		}
		// could be fine-tuned to check for non-empty values as well
		if ctx.Target == configtypes.TargetTAE {
			hasTAEFields = true
		}
		contextOutputList = append(contextOutputList, ctx)
	}
	return contextOutputList, hasTAEFields
}

func displayContextListOutputSplitViewTarget(cfg *configtypes.ClientConfig, writer io.Writer) {
	var k8sContextTable component.OutputWriter
	target := getTarget()

	ctxs, showTAEColumns := getContextsToDisplay(cfg, target, onlyCurrent)

	if showTAEColumns {
		k8sContextTable = component.NewOutputWriter(writer, outputFormat, "Name", "IsActive", "Type", "Endpoint", "KubeConfigPath", "KubeContext", "Project", "Space")
	} else {
		k8sContextTable = component.NewOutputWriter(writer, outputFormat, "Name", "IsActive", "Type", "Endpoint", "KubeConfigPath", "KubeContext")
	}
	outputWriterTMCTarget := component.NewOutputWriter(writer, outputFormat, "Name", "IsActive", "Endpoint")
	for _, ctx := range ctxs {
		var ep, path, context string
		var project, space string

		isCurrent := ctx.Name == cfg.CurrentContext[ctx.Target]

		switch ctx.Target {
		case configtypes.TargetTMC:
			if ctx.GlobalOpts != nil {
				ep = ctx.GlobalOpts.Endpoint
			}

			outputWriterTMCTarget.AddRow(ctx.Name, isCurrent, ep)
		default:
			if ctx.ClusterOpts != nil {
				ep = ctx.ClusterOpts.Endpoint
				path = ctx.ClusterOpts.Path
				context = ctx.ClusterOpts.Context
			}
			contextType := "cluster"
			if ctx.Target == configtypes.TargetTAE {
				contextType = "TAE"
			}

			if showTAEColumns {
				if ctx.AdditionalMetadata["taeProjectName"] != nil {
					project = ctx.AdditionalMetadata["taeProjectName"].(string)
				}
				if ctx.AdditionalMetadata["taeSpaceName"] != nil {
					space = ctx.AdditionalMetadata["taeSpaceName"].(string)
				}
				k8sContextTable.AddRow(ctx.Name, isCurrent, contextType, ep, path, context, project, space)
			} else {
				k8sContextTable.AddRow(ctx.Name, isCurrent, contextType, ep, path, context)
			}
		}
	}

	cyanBold := color.New(color.FgCyan).Add(color.Bold)
	cyanBoldItalic := color.New(color.FgCyan).Add(color.Bold, color.Italic)
	if target == configtypes.TargetUnknown || target == configtypes.TargetK8s || target == configtypes.TargetTAE {
		_, _ = cyanBold.Println("Target: ", cyanBoldItalic.Sprintf("%s", configtypes.TargetK8s))
		k8sContextTable.Render()
	}
	if target == configtypes.TargetUnknown || target == configtypes.TargetTMC {
		_, _ = cyanBold.Println("Target: ", cyanBoldItalic.Sprintf("%s", configtypes.TargetTMC))
		outputWriterTMCTarget.Render()
	}
}

var getCtxTokenCmd = &cobra.Command{
	Use:               "get-token CONTEXT_NAME",
	Short:             "Get the valid CSP token for the given TAE context.",
	Args:              cobra.ExactArgs(1),
	Hidden:            true,
	ValidArgsFunction: completeTAEContexts,
	RunE:              getToken,
}

func getToken(cmd *cobra.Command, args []string) error {
	name := args[0]
	ctx, err := config.GetContext(name)
	if err != nil {
		return err
	}
	if ctx.Target != configtypes.TargetTAE {
		return errors.Errorf("context %q is not of type TAE", name)
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
	taeActiveResourceCmd.Flags().StringVarP(&projectStr, "project", "", "", "project name to be set as active")
	taeActiveResourceCmd.Flags().StringVarP(&spaceStr, "space", "", "", "space name to be set as active")

	updateCtxCmd.AddCommand(
		taeActiveResourceCmd,
	)
	return updateCtxCmd
}

// taeActiveResourceCmd updates the TAE(Tanzu Application Engine) active resource referenced by tae context
//
// NOTE!!: This command is EXPERIMENTAL and subject to change in future
var taeActiveResourceCmd = &cobra.Command{
	Use:               "tae-active-resource CONTEXT_NAME",
	Short:             "updates the Tanzu Application Engine(TAE) active resource for the given TAE context (subject to change).",
	Hidden:            true,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeTAEContexts,
	RunE:              setTAECtxActiveResource,
}

func setTAECtxActiveResource(_ *cobra.Command, args []string) error {
	name := args[0]
	if projectStr == "" && spaceStr != "" {
		return errors.Errorf("space cannot be set without project name. Please provide project name also using --project option")
	}
	ctx, err := config.GetContext(name)
	if err != nil {
		return err
	}
	if ctx.Target != configtypes.TargetTAE {
		return errors.Errorf("context %q is not of type TAE", name)
	}
	if ctx.AdditionalMetadata == nil {
		ctx.AdditionalMetadata = make(map[string]interface{})
	}
	ctx.AdditionalMetadata[taert.ProjectNameKey] = projectStr
	ctx.AdditionalMetadata[taert.SpaceNameKey] = spaceStr
	err = config.SetContext(ctx, false)
	if err != nil {
		return errors.Wrap(err, "failed updating the context %q with the active TAE resource")
	}
	err = updateTAEContextKubeconfig(ctx, projectStr, spaceStr)
	if err != nil {
		return errors.Wrap(err, "failed to update the TAE context kubeconfig")
	}

	return nil
}

func updateTAEContextKubeconfig(cliContext *configtypes.Context, projectName, spaceName string) error {
	kcfg, err := clientcmd.LoadFromFile(cliContext.ClusterOpts.Path)
	if err != nil {
		return errors.Wrap(err, "unable to load kubeconfig")
	}

	kubeContext := kcfg.Contexts[cliContext.ClusterOpts.Context]
	if kubeContext == nil {
		return errors.Errorf("kubecontext %q doesn't exist", cliContext.ClusterOpts.Context)
	}
	cluster := kcfg.Clusters[kubeContext.Cluster]
	cluster.Server = prepareClusterServerURL(cliContext, projectName, spaceName)
	err = clientcmd.WriteToFile(*kcfg, cliContext.ClusterOpts.Path)
	if err != nil {
		return errors.Wrap(err, "failed to update the context kubeconfig file")
	}
	return nil
}

func prepareClusterServerURL(context *configtypes.Context, projectName, spaceName string) string {
	serverURL := context.ClusterOpts.Endpoint
	if projectName == "" {
		return serverURL
	}
	serverURL = serverURL + "/project/" + projectName

	if spaceName == "" {
		return serverURL
	}
	return serverURL + "/space/" + spaceName
}

// ====================================
// Shell completion functions
// ====================================
func completeAllContexts(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.GetClientConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	target := getTarget()

	var allCtxs []*configtypes.Context
	for _, ctx := range cfg.KnownContexts {
		if target == configtypes.TargetUnknown || target == ctx.Target {
			allCtxs = append(allCtxs, ctx)
		}
	}
	return completionFormatCtxs(allCtxs), cobra.ShellCompDirectiveNoFileComp
}

func completeTAEContexts(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.GetClientConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var taeCtxs []*configtypes.Context
	for _, ctx := range cfg.KnownContexts {
		if ctx.Target == configtypes.TargetTAE {
			taeCtxs = append(taeCtxs, ctx)
		}
	}
	return completionFormatCtxs(taeCtxs), cobra.ShellCompDirectiveNoFileComp
}

func completeActiveContexts(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	currentCtxMap, err := config.GetAllCurrentContextsMap()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	target := getTarget()

	var allCtxs []*configtypes.Context
	for _, ctx := range currentCtxMap {
		if target == configtypes.TargetUnknown || target == ctx.Target {
			allCtxs = append(allCtxs, ctx)
		}
	}
	return completionFormatCtxs(allCtxs), cobra.ShellCompDirectiveNoFileComp
}

// Setup shell completion for the kube-context flag
func completeKubeContext(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	if kubeConfig == "" {
		kubeConfig = getDefaultKubeconfigPath()
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

		if info == "" && ctx.Target == configtypes.TargetK8s && ctx.ClusterOpts != nil {
			info = fmt.Sprintf("%s:%s", ctx.ClusterOpts.Path, ctx.ClusterOpts.Context)
		}

		comps = append(comps, fmt.Sprintf("%s\t%s", ctx.Name, info))
	}

	// Sort the completion to make testing easier
	sort.Strings(comps)
	return comps
}
