// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/csp"
	tkgauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tkg"
	wcpauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/wcp"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

var (
	stderrOnly, forceCSP, staging, onlyCurrent, selfManaged, skipTLSVerify                 bool
	ctxName, endpoint, apiToken, kubeConfig, kubeContext, getOutputFmt, endpointCACertPath string
)

const (
	knownGlobalHost                        = "cloud.vmware.com"
	apiTokenType                           = "api-token"
	idTokenType                            = "id-token"
	controlPlaneEndPointTypeSelfManagedTMC = "self-managed-tmc"

	contextNotExistsForTarget      = "The provided context %v does not exist or is not active for the given target %v"
	noActiveContextExistsForTarget = "There is no active context for the given target %v"
	contextNotActiveOrNotExists    = "The provided context %v is not active or does not exist"
	contextForTargetSetInactive    = "The context %v for the target %v has been set as inactive"

	invalidTarget = "invalid target specified. Please specify a correct value for the `--target/-t` flag from 'kubernetes[k8s]/mission-control[tmc]"
)

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
	)

	initCreateCtxCmd()

	listCtxCmd.Flags().StringVarP(&targetStr, "target", "t", "", "list only contexts associated with the specified target (kubernetes[k8s]/mission-control[tmc])")
	listCtxCmd.Flags().BoolVar(&onlyCurrent, "current", false, "list only current active contexts")
	listCtxCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format: table|yaml|json")

	getCtxCmd.Flags().StringVarP(&getOutputFmt, "output", "o", "yaml", "output format: yaml|json")

	deleteCtxCmd.Flags().BoolVarP(&unattended, "yes", "y", false, "delete the context entry without confirmation")

	unsetCtxCmd.Flags().StringVarP(&targetStr, "target", "t", "", "unset active context associated with the specified target (kubernetes[k8s]|mission-control[tmc])")
}

var createCtxCmd = &cobra.Command{
	Use:   "create CONTEXT_NAME",
	Short: "Create a Tanzu CLI context",
	Args:  cobra.MaximumNArgs(1),
	RunE:  createCtx,
	Example: `
    # Create a TKG management cluster context using endpoint
    tanzu context create mgmt-cluster --endpoint "https://k8s.example.com[:port]"

    # Create a TKG management cluster context using kubeconfig path and context
    tanzu context create mgmt-cluster --kubeconfig path/to/kubeconfig --kubecontext kubecontext

    # Create a TKG management cluster context by using the provided CA Bundle for TLS verification:
    tanzu context create --endpoint https://k8s.example.com[:port] --endpoint-ca-certificate /path/to/ca/ca-cert

    # Create a TKG management cluster context by explicit request to skip TLS verification, which is insecure:
    tanzu context create --endpoint https://k8s.example.com[:port] --insecure-skip-tls-verify

    # Create a TKG management cluster context using default kubeconfig path and a kubeconfig context
    tanzu context create mgmt-cluster --kubecontext kubecontext

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
	createCtxCmd.Flags().StringVar(&apiToken, "api-token", "", "API token for the SaaS context")
	createCtxCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "path to the kubeconfig file; valid only if user doesn't choose 'endpoint' option.(See [*])")
	createCtxCmd.Flags().StringVar(&kubeContext, "kubecontext", "", "the context in the kubeconfig to use; valid only if user doesn't choose 'endpoint' option.(See [*]) ")
	createCtxCmd.Flags().BoolVar(&stderrOnly, "stderr-only", false, "send all output to stderr rather than stdout")
	createCtxCmd.Flags().BoolVar(&forceCSP, "force-csp", false, "force the context to use CSP auth")
	createCtxCmd.Flags().BoolVar(&staging, "staging", false, "use CSP staging issuer")
	createCtxCmd.Flags().StringVar(&endpointCACertPath, "endpoint-ca-certificate", "", "path to the endpoint public certificate")
	createCtxCmd.Flags().BoolVar(&skipTLSVerify, "insecure-skip-tls-verify", false, "skip endpoint's TLS certificate verification")

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

	controlPlaneEPType := os.Getenv(constants.ControlPlaneEndpointType)
	if controlPlaneEPType != "" {
		if strings.EqualFold(controlPlaneEPType, controlPlaneEndPointTypeSelfManagedTMC) {
			selfManaged = true
		}
	}
	ctx, err := createNewContext()
	if err != nil {
		return err
	}
	if ctx.Target == configtypes.TargetK8s {
		err = k8sLogin(ctx)
	} else if selfManaged {
		err = selfManagedTMCLogin(ctx)
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
	// user provided command line options to create a context using kubeconfig[optional] and context
	if kubeContext != "" {
		return createContextWithKubeconfig()
	}
	// user provided command line options to create a context using endpoint
	if endpoint != "" || selfManaged {
		return createContextWithEndpoint()
	}
	promptOpts := getPromptOpts()

	var ctxCreationType string

	err = component.Prompt(
		&component.PromptConfig{
			Message: "Select context creation type",
			Options: []string{"Control plane endpoint", "Local kubeconfig"},
			Default: "Control plane endpoint",
		},
		&ctxCreationType,
		promptOpts...,
	)
	if err != nil {
		return context, err
	}

	if ctxCreationType == "Control plane endpoint" {
		return createContextWithEndpoint()
	}

	return createContextWithKubeconfig()
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

func createContextWithEndpoint() (context *configtypes.Context, err error) {
	promptOpts := getPromptOpts()
	if endpoint == "" {
		err = component.Prompt(
			&component.PromptConfig{
				Message: "Enter control plane endpoint",
			},
			&endpoint,
			promptOpts...,
		)
		if err != nil {
			return
		}
	}
	endpoint = strings.TrimSpace(endpoint)

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

	if isGlobalContext(endpoint) || selfManaged {
		context = &configtypes.Context{
			Name:       ctxName,
			Target:     configtypes.TargetTMC,
			GlobalOpts: &configtypes.GlobalServer{Endpoint: sanitizeEndpoint(endpoint)},
		}
	} else {
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
	}
	return context, err
}

func globalLogin(c *configtypes.Context) (err error) {
	apiTokenValue, apiTokenExists := os.LookupEnv(config.EnvAPITokenKey)

	issuer := csp.ProdIssuer
	if staging {
		issuer = csp.StgIssuer
	}
	if apiTokenExists {
		log.Info("API token env var is set")
	} else {
		apiTokenValue, err = promptAPIToken()
		if err != nil {
			return err
		}
	}
	token, err := csp.GetAccessTokenFromAPIToken(apiTokenValue, issuer)
	if err != nil {
		return err
	}
	claims, err := csp.ParseToken(&oauth2.Token{AccessToken: token.AccessToken})
	if err != nil {
		return err
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

	err = config.AddContext(c, true)
	if err != nil {
		return err
	}

	// format
	fmt.Println()
	log.Success("successfully created a TMC context")
	return nil
}

func selfManagedTMCLogin(c *configtypes.Context) (err error) {
	issuer, err := getIssuerURLForTMCEndPoint(c.GlobalOpts.Endpoint)
	if err != nil {
		return err
	}
	refreshToken := ""
	token, err := csp.GetAccessTokenFromSelfManagedIDP(refreshToken, issuer)
	if err != nil {
		return err
	}
	if token == nil {
		return errors.Errorf("token issuer %s returned nil token", issuer)
	}

	a := configtypes.GlobalServerAuth{}
	a.Issuer = issuer
	// TODO: parse claims from ID token to get org-tenant ID info
	// a.UserName = claims.Username
	// a.Permissions = claims.Permissions
	a.AccessToken = token.AccessToken
	a.IDToken = token.IDToken
	a.RefreshToken = token.RefreshToken
	a.Type = idTokenType
	expiresAt := time.Now().Add(time.Second * time.Duration(token.ExpiresIn))
	a.Expiration = expiresAt
	c.GlobalOpts.Auth = a

	err = config.AddContext(c, true)
	if err != nil {
		return err
	}

	// format
	fmt.Println()
	log.Success("successfully created a TMC self-managed context")
	return nil
}

// Instead of the end user having to know the OIDC token issuer URL
// we will derive the token issuer URL based on the TMC endpoint for the customer's organization.
// The issuer URL and the TMC endpoint are both expected to share the same DNS zone and will only
// differ in-terms of the domain of the URL and the path.
func getIssuerURLForTMCEndPoint(tmcEP string) (string, error) {
	tmcEP = strings.TrimSpace(tmcEP)
	// the empty string is successfully parsed
	// so add a special check to ensure the tmc endpoint is not an empty string.
	if tmcEP == "" {
		return "", errors.Errorf("cannot get issuer URL for empty TMC endpoint")
	}

	// assume that the host in the tmc endpoint will always look like
	// tmc.my-domain.com:443 or <TMC SELF MANAGED DNS ZONE>:443
	tmcEPHost, _, err := net.SplitHostPort(tmcEP)
	if err != nil {
		return "", errors.Wrapf(err, "TMC endpoint URL %s should be of the format host:port", tmcEP)
	}
	if tmcEPHost == "" {
		return "", errors.Errorf("TMC endpoint URL %s should be of the format host:port", tmcEP)
	}
	u := url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s.%s", csp.PinnipedSupervisorDomain, tmcEPHost),
		Path:   csp.FederationDomainPath,
	}

	return u.String(), nil
}

// Interactive way to create a TMC context. User will be prompted for CSP API token.
func promptAPIToken() (apiToken string, err error) {
	consoleURL := url.URL{
		Scheme:   "https",
		Host:     "console.cloud.vmware.com",
		Path:     "/csp/gateway/portal/",
		Fragment: "/user/tokens",
	}

	// format
	fmt.Println()
	log.Infof(
		"If you don't have an API token, visit the VMware Cloud Services console, select your organization, and create an API token with the TMC service roles:\n  %s\n",
		consoleURL.String(),
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
	return
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
	Use:   "list",
	Short: "List contexts",
	RunE:  listCtx,
}

func listCtx(cmd *cobra.Command, _ []string) error {
	cfg, err := config.GetClientConfig()
	if err != nil {
		return err
	}

	if !common.IsValidTarget(targetStr, false, true) {
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
	Use:   "get CONTEXT_NAME",
	Short: "Display a context from the config",
	RunE:  getCtx,
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

func getValues(m map[string]*configtypes.Context) []*configtypes.Context {
	values := make([]*configtypes.Context, 0, len(m))
	for _, value := range m {
		values = append(values, value)
	}
	return values
}

var deleteCtxCmd = &cobra.Command{
	Use:   "delete CONTEXT_NAME",
	Short: "Delete a context from the config",
	RunE:  deleteCtx,
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

	log.Infof("Deleting entry for cluster %s", name)
	err := config.RemoveContext(name)
	if err != nil {
		return err
	}

	return nil
}

var useCtxCmd = &cobra.Command{
	Use:   "use CONTEXT_NAME",
	Short: "Set the context to be used by default",
	RunE:  useCtx,
}

func useCtx(_ *cobra.Command, args []string) error {
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

	err := config.SetCurrentContext(name)
	if err != nil {
		return err
	}

	// Sync all required plugins
	_ = syncContextPlugins()

	return nil
}

var unsetCtxCmd = &cobra.Command{
	Use:   "unset CONTEXT_NAME",
	Short: "Unset the active context so that it is not used by default.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  unsetCtx,
}

func unsetCtx(_ *cobra.Command, args []string) error {
	var name string
	if !common.IsValidTarget(targetStr, false, true) {
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

func unsetGivenContext(name, target string) error {
	var err error
	var unset bool
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
	}
	return nil
}

func displayContextListOutputListView(cfg *configtypes.ClientConfig, writer io.Writer) {
	target := getTarget()

	op := component.NewOutputWriter(writer, outputFormat, "Name", "Type", "IsManagementCluster", "IsCurrent", "Endpoint", "KubeConfigPath", "KubeContext")
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
		op.AddRow(ctx.Name, ctx.Target, isMgmtCluster, isCurrent, ep, path, context)
	}
	op.Render()
}

func displayContextListOutputSplitViewTarget(cfg *configtypes.ClientConfig, writer io.Writer) {
	target := getTarget()

	outputWriterK8sTarget := component.NewOutputWriter(writer, outputFormat, "Name", "IsActive", "Endpoint", "KubeConfigPath", "KubeContext")
	outputWriterTMCTarget := component.NewOutputWriter(writer, outputFormat, "Name", "IsActive", "Endpoint")
	for _, ctx := range cfg.KnownContexts {
		if target != configtypes.TargetUnknown && ctx.Target != target {
			continue
		}
		isCurrent := ctx.Name == cfg.CurrentContext[ctx.Target]
		if onlyCurrent && !isCurrent {
			continue
		}

		var ep, path, context string
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

			outputWriterK8sTarget.AddRow(ctx.Name, isCurrent, ep, path, context)
		}
	}

	cyanBold := color.New(color.FgCyan).Add(color.Bold)
	cyanBoldItalic := color.New(color.FgCyan).Add(color.Bold, color.Italic)
	if target == configtypes.TargetUnknown || target == configtypes.TargetK8s {
		_, _ = cyanBold.Println("Target: ", cyanBoldItalic.Sprintf("%s", configtypes.TargetK8s))
		outputWriterK8sTarget.Render()
	}
	if target == configtypes.TargetUnknown || target == configtypes.TargetTMC {
		_, _ = cyanBold.Println("Target: ", cyanBoldItalic.Sprintf("%s", configtypes.TargetTMC))
		outputWriterTMCTarget.Render()
	}
}
