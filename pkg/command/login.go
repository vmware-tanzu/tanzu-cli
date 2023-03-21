// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/csp"
	tkgauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tkg"
	wcpauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/wcp"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginmanager"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

var (
	name, server string
)

var loginCmd = &cobra.Command{
	Use:     "login",
	Short:   "Login to the platform",
	Aliases: []string{"lo", "logins"},
	Annotations: map[string]string{
		"group": string(plugin.SystemCmdGroup),
	},
	RunE: login,
}

func init() {
	loginCmd.Flags().StringVar(&endpoint, "endpoint", "", "endpoint to login to")
	loginCmd.Flags().StringVar(&name, "name", "", "name of the server")
	loginCmd.Flags().StringVar(&apiToken, "apiToken", "", "API token for global login")
	loginCmd.Flags().StringVar(&server, "server", "", "login to the given server")
	loginCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "path to kubeconfig management cluster. Valid only if user doesn't choose 'endpoint' option.(See [*])")
	loginCmd.Flags().StringVar(&kubeContext, "context", "", "the context in the kubeconfig to use for management cluster. Valid only if user doesn't choose 'endpoint' option.(See [*]) ")
	loginCmd.Flags().BoolVar(&stderrOnly, "stderr-only", false, "send all output to stderr rather than stdout")
	loginCmd.Flags().BoolVar(&forceCSP, "force-csp", false, "force the endpoint to be logged in as a csp server")
	loginCmd.Flags().BoolVar(&staging, "staging", false, "use CSP staging issuer")
	loginCmd.Flags().MarkHidden("stderr-only") // nolint
	loginCmd.Flags().MarkHidden("force-csp")   // nolint
	loginCmd.Flags().MarkHidden("staging")     // nolint
	loginCmd.SetUsageFunc(cli.SubCmdUsageFunc)
	msg := fmt.Sprintf("will be removed in version %q. Use the %q command instead.", "1.2.0", "context")
	loginCmd.Deprecated = msg
	loginCmd.Example = `
	# Login to TKG management cluster using endpoint
	tanzu login --endpoint "https://login.example.com"  --name mgmt-cluster

	# Login to TKG management cluster by using kubeconfig path and context for the management cluster
	tanzu login --kubeconfig path/to/kubeconfig --context path/to/context --name mgmt-cluster

	# Login to TKG management cluster by using default kubeconfig path and context for the management cluster
	tanzu login  --context path/to/context --name mgmt-cluster

	# Login to an existing server
	tanzu login --server mgmt-cluster

	[*] : User has two options to login to TKG. User can choose the login endpoint option
	by providing 'endpoint', or user can choose to use the kubeconfig for the management cluster by
	providing 'kubeconfig' and 'context'. If only '--context' is set and '--kubeconfig' is unset
	$KUBECONFIG env variable would be used and, if $KUBECONFIG env is also unset default 
	kubeconfig($HOME/.kube/config) would be used
	`
}

func login(cmd *cobra.Command, args []string) (err error) {
	cfg, err := config.GetClientConfig()
	if err != nil {
		return err
	}

	newServerSelector := "+ new server"
	var serverTarget *configtypes.Server // nolint:staticcheck // Deprecated
	if name != "" {
		serverTarget, err = createNewServer()
		if err != nil {
			return err
		}
	} else if server == "" {
		serverTarget, err = getServerTarget(cfg, newServerSelector)
		if err != nil {
			return err
		}
	} else {
		serverTarget, err = config.GetServer(server) // nolint:staticcheck // Deprecated
		if err != nil {
			return err
		}
	}

	if server == newServerSelector {
		serverTarget, err = createNewServer()
		if err != nil {
			return err
		}
	}

	if serverTarget.Type == configtypes.GlobalServerType { // nolint:staticcheck // Deprecated
		err = globalLoginUsingServer(serverTarget)
	} else {
		err = managementClusterLogin(serverTarget)
	}

	if err != nil {
		return err
	}

	// Sync all required plugins
	if err = pluginmanager.SyncPlugins(); err != nil {
		log.Warning("unable to automatically sync the plugins from target server. Please run 'tanzu plugin sync' command to sync plugins manually")
	}

	return nil
}

func getServerTarget(cfg *configtypes.ClientConfig, newServerSelector string) (*configtypes.Server, error) { // nolint:staticcheck // Deprecated
	promptOpts := getPromptOpts()
	servers := map[string]*configtypes.Server{} // nolint:staticcheck // Deprecated
	for _, server := range cfg.KnownServers {   // nolint:staticcheck // Deprecated
		ep, err := config.EndpointFromServer(server) // nolint:staticcheck // Deprecated
		if err != nil {
			return nil, err
		}

		s := rpad(server.Name, 20)
		s = fmt.Sprintf("%s(%s)", s, ep)
		servers[s] = server
	}
	if endpoint == "" {
		endpoint, _ = os.LookupEnv(config.EnvEndpointKey)
	}
	// If there are no existing servers
	if len(servers) == 0 {
		return createNewServer()
	}
	serverKeys := getKeysFromServerMap(servers)
	serverKeys = append(serverKeys, newServerSelector)
	servers[newServerSelector] = &configtypes.Server{} // nolint:staticcheck // Deprecated
	err := component.Prompt(
		&component.PromptConfig{
			Message: "Select a server",
			Options: serverKeys,
			Default: serverKeys[0],
		},
		&server,
		promptOpts...,
	)
	if err != nil {
		return nil, err
	}
	return servers[server], nil
}

func getKeysFromServerMap(m map[string]*configtypes.Server) []string { // nolint:staticcheck // Deprecated
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isGlobalServer(endpoint string) bool {
	if strings.Contains(endpoint, knownGlobalHost) {
		return true
	}
	if forceCSP {
		return true
	}
	return false
}

func createNewServer() (server *configtypes.Server, err error) { // nolint:staticcheck // Deprecated
	// user provided command line options to create a server using kubeconfig[optional] and context
	if kubeContext != "" {
		return createServerWithKubeconfig()
	}
	// user provided command line options to create a server using endpoint
	if endpoint != "" {
		return createServerWithEndpoint()
	}
	promptOpts := getPromptOpts()

	var loginType string

	err = component.Prompt(
		&component.PromptConfig{
			Message: "Select login type",
			Options: []string{"Server endpoint", "Local kubeconfig"},
			Default: "Server endpoint",
		},
		&loginType,
		promptOpts...,
	)
	if err != nil {
		return server, err
	}

	if loginType == "Server endpoint" {
		return createServerWithEndpoint()
	}

	return createServerWithKubeconfig()
}

func createServerWithKubeconfig() (server *configtypes.Server, err error) { // nolint:staticcheck // Deprecated
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
	}
	kubeConfig = strings.TrimSpace(kubeConfig)
	if kubeConfig == "" {
		kubeConfig = getDefaultKubeconfigPath()
	}

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
	if name == "" {
		err = component.Prompt(
			&component.PromptConfig{
				Message: "Give the server a name",
			},
			&name,
			promptOpts...,
		)
		if err != nil {
			return
		}
	}
	name = strings.TrimSpace(name)
	nameExists, err := config.ServerExists(name) // nolint:staticcheck // Deprecated
	if err != nil {
		return server, err
	}
	if nameExists {
		err = fmt.Errorf("server %q already exists", name)
		return
	}

	endpointType := configtypes.ManagementClusterServerType // nolint:staticcheck // Deprecated

	server = &configtypes.Server{ // nolint:staticcheck // Deprecated
		Name: name,
		Type: endpointType,
		ManagementClusterOpts: &configtypes.ManagementClusterServer{ // nolint:staticcheck // Deprecated
			Path:     kubeConfig,
			Context:  kubeContext,
			Endpoint: endpoint},
	}
	return server, err
}

func createServerWithEndpoint() (server *configtypes.Server, err error) { // nolint:staticcheck // Deprecated
	promptOpts := getPromptOpts()
	if endpoint == "" {
		err = component.Prompt(
			&component.PromptConfig{
				Message: "Enter server endpoint",
			},
			&endpoint,
			promptOpts...,
		)
		if err != nil {
			return
		}
	}
	endpoint = strings.TrimSpace(endpoint)
	if name == "" {
		err = component.Prompt(
			&component.PromptConfig{
				Message: "Give the server a name",
			},
			&name,
			promptOpts...,
		)
		if err != nil {
			return
		}
	}
	name = strings.TrimSpace(name)
	nameExists, err := config.ServerExists(name) // nolint:staticcheck // Deprecated
	if err != nil {
		return server, err
	}
	if nameExists {
		err = fmt.Errorf("server %q already exists", name)
		return
	}
	if isGlobalServer(endpoint) {
		server = &configtypes.Server{ // nolint:staticcheck // Deprecated
			Name:       name,
			Type:       configtypes.GlobalServerType, // nolint:staticcheck // Deprecated
			GlobalOpts: &configtypes.GlobalServer{Endpoint: sanitizeEndpoint(endpoint)},
		}
	} else {
		// While this would add an extra HTTP round trip, it avoids the need to
		// add extra provider specific login flags.
		isVSphereSupervisor, err := wcpauth.IsVSphereSupervisor(endpoint, getDiscoveryHTTPClient())
		// Fall back to assuming non vSphere supervisor.
		if err != nil {
			err := fmt.Errorf("error creating kubeconfig with tanzu pinniped-auth login plugin: %v", err)
			log.Error(err, "")
			return nil, err
		}
		if isVSphereSupervisor {
			log.Info("Detected a vSphere Supervisor being used")
			kubeConfig, kubeContext, err = vSphereSupervisorLogin(endpoint)
			if err != nil {
				err := fmt.Errorf("error logging in to vSphere Supervisor: %v", err)
				log.Error(err, "")
				return nil, err
			}
		} else {
			kubeConfig, kubeContext, err = tkgauth.KubeconfigWithPinnipedAuthLoginPlugin(endpoint, nil, tkgauth.DiscoveryStrategy{ClusterInfoConfigMap: tkgauth.DefaultClusterInfoConfigMap})
			if err != nil {
				err := fmt.Errorf("error creating kubeconfig with tanzu pinniped-auth login plugin: %v", err)
				log.Error(err, "")
				return nil, err
			}
		}

		server = &configtypes.Server{ // nolint:staticcheck // Deprecated
			Name: name,
			Type: configtypes.ManagementClusterServerType, // nolint:staticcheck // Deprecated
			ManagementClusterOpts: &configtypes.ManagementClusterServer{ // nolint:staticcheck // Deprecated
				Path:     kubeConfig,
				Context:  kubeContext,
				Endpoint: endpoint},
		}
	}
	return server, err
}

func globalLoginUsingServer(s *configtypes.Server) (err error) { // nolint:staticcheck // Deprecated
	a := configtypes.GlobalServerAuth{}
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

	a.Issuer = issuer

	a.UserName = claims.Username
	a.Permissions = claims.Permissions
	a.AccessToken = token.AccessToken
	a.IDToken = token.IDToken
	a.RefreshToken = apiTokenValue
	a.Type = "api-token"

	expiresAt := time.Now().Local().Add(time.Second * time.Duration(token.ExpiresIn))
	a.Expiration = expiresAt

	s.GlobalOpts.Auth = a

	err = config.PutServer(s, true) // nolint:staticcheck // Deprecated
	if err != nil {
		return err
	}

	fmt.Println()
	log.Success("successfully logged into global control plane")
	return nil
}

func managementClusterLogin(s *configtypes.Server) error { // nolint:staticcheck // Deprecated
	if s.ManagementClusterOpts.Path != "" && s.ManagementClusterOpts.Context != "" {
		_, err := tkgauth.GetServerKubernetesVersion(s.ManagementClusterOpts.Path, s.ManagementClusterOpts.Context)
		if err != nil {
			err := fmt.Errorf("failed to login to the management cluster %s, %v", s.Name, err)
			log.Error(err, "")
			return err
		}
		err = config.PutServer(s, true) // nolint:staticcheck // Deprecated
		if err != nil {
			return err
		}

		log.Successf("successfully logged in to management cluster using the kubeconfig %s", s.Name)
		return nil
	}

	return fmt.Errorf("not yet implemented")
}
