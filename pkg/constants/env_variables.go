// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package constants

// environment variables for http proxy
const (
	ProxyCACert = "PROXY_CA_CERT"
)

const (
	AllowedRegistries                                 = "ALLOWED_REGISTRY"
	ConfigVariableAdditionalDiscoveryForTesting       = "TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY"
	ConfigVariableAdditionalPrivateDiscoveryImages    = "TANZU_CLI_PRIVATE_PLUGIN_DISCOVERY_IMAGES"
	ConfigVariableIncludeDeactivatedPluginsForTesting = "TANZU_CLI_INCLUDE_DEACTIVATED_PLUGINS_TEST_ONLY"
	ConfigVariableStandaloneOverContextPlugins        = "TANZU_CLI_STANDALONE_OVER_CONTEXT_PLUGINS"
	// PluginDiscoveryImageSignatureVerificationSkipList is a comma separated list of discovery image urls
	PluginDiscoveryImageSignatureVerificationSkipList = "TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST"
	PublicKeyPathForPluginDiscoveryImageSignature     = "TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH"
	SuppressSkipSignatureVerificationWarning          = "TANZU_CLI_SUPPRESS_SKIP_SIGNATURE_VERIFICATION_WARNING"
	CEIPOptInUserPromptAnswer                         = "TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER"
	EULAPromptAnswer                                  = "TANZU_CLI_EULA_PROMPT_ANSWER"
	// Environment variable to indicate that the CLI is running in E2E test environment
	E2ETestEnvironment                = "TANZU_CLI_E2E_TEST_ENVIRONMENT"
	ShowTelemetryConsoleLogs          = "TANZU_CLI_SHOW_TELEMETRY_CONSOLE_LOGS"
	TelemetrySuperColliderEnvironment = "TANZU_CLI_SUPERCOLLIDER_ENVIRONMENT"

	// TanzuCLIEssentialsPluginGroupName is used to override and customize the default essentials plugin group name
	TanzuCLIEssentialsPluginGroupName = "TANZU_CLI_ESSENTIALS_PLUGIN_GROUP_NAME"

	// TanzuCLIEssentialsPluginGroupVersion is used to override and customize what version of essentials plugin group should be installed
	TanzuCLIEssentialsPluginGroupVersion = "TANZU_CLI_ESSENTIALS_PLUGIN_GROUP_VERSION"

	// TanzuCLIShowPluginInstallationLogs is used to enable or disable the logs for essential plugin group installation process
	// Possible values "True" or "False"
	// by default logs are enabled
	TanzuCLIShowPluginInstallationLogs = "TANZU_CLI_SHOW_PLUGIN_INSTALLATION_LOGS"

	// Suppress updating the current context of the kubeconfig referenced in
	// the CLI Context being activated.
	SkipUpdateKubeconfigOnContextUse = "TANZU_CLI_SKIP_UPDATE_KUBECONFIG_ON_CONTEXT_USE"

	// Control the different ActiveHelp options
	// If set to 0, Cobra will automatically deactivate all ActiveHelp for the CLI
	ConfigVariableActiveHelp = "TANZU_ACTIVE_HELP"

	// Change the default value of the plugin inventory cache TTL
	ConfigVariablePluginDBCacheTTLSeconds = "TANZU_CLI_PLUGIN_DB_CACHE_TTL_SECONDS"

	// ConfigVariablePluginDBCacheRefreshThresholdSeconds Change the default value of db cache refresh threshold
	ConfigVariablePluginDBCacheRefreshThresholdSeconds = "TANZU_CLI_PLUGIN_DB_CACHE_REFRESH_THRESHOLD_SECONDS"

	// ConfigVariableRecommendVersionDelayDays Change the default value of the delay between printing a recommended version message
	ConfigVariableRecommendVersionDelayDays = "TANZU_CLI_RECOMMEND_VERSION_DELAY_DAYS"

	// CSPLoginOrgID overrides the CSP default OrgID to which the user logs into, using CLI interactive login flow
	// Note: More information regarding the CSP organizations can be found at
	// https://docs.vmware.com/en/VMware-Cloud-services/services/Using-VMware-Cloud-Services/GUID-CF9E9318-B811-48CF-8499-9419997DC1F8.html
	CSPLoginOrgID = "TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID"

	// TanzuCLIOAuthLocalListenerPort is the port to be used by local listener for OAuth authorization flow
	TanzuCLIOAuthLocalListenerPort = "TANZU_CLI_OAUTH_LOCAL_LISTENER_PORT"

	// TanzuPluginDiscoveryPathforTanzuContext specifies the custom endpoint path to use with the kubeconfig when talking
	// to the tanzu context to get the recommended plugins by querying CLIPlugin resources
	// If environment variable 'TANZU_CLI_PLUGIN_DISCOVERY_PATH_FOR_TANZU_CONTEXT' is not configured
	// default discovery endpoint configured with TanzuContextPluginDiscoveryEndpointPath will be used
	TanzuPluginDiscoveryPathforTanzuContext = "TANZU_CLI_PLUGIN_DISCOVERY_PATH_FOR_TANZU_CONTEXT"

	// SkipPluginGroupVerificationOnPublish skips the plugin group verification of whether the plugins specified
	// in the plugin-group are available in the database or not.
	// Note: THIS SHOULD ONLY BE USED FOR TEST AND NON PRODUCTION ENVIRONMENTS.
	SkipPluginGroupVerificationOnPublish = "TANZU_CLI_SKIP_PLUGIN_GROUP_VERIFICATION_ON_PUBLISH"

	// SkipAutoInstallOfContextRecommendedPlugins skips the auto-installation of the context recommended plugins
	// on `tanzu context create` or `tanzu context use`
	SkipAutoInstallOfContextRecommendedPlugins = "TANZU_CLI_SKIP_CONTEXT_RECOMMENDED_PLUGIN_INSTALLATION"

	// SkipTAPScopesValidationOnTanzuContext skips the TAP scopes validation on the token acquired while creating "tanzu"
	// context using tanzu login or tanzu context create command
	SkipTAPScopesValidationOnTanzuContext = "TANZU_CLI_SKIP_TAP_SCOPES_VALIDATION_ON_TANZU_CONTEXT"

	// AuthenticatedRegistry provides a comma separated list of registry hosts that requires authentication
	// to pull images. Tanzu CLI will use default docker auth to communicate to these registries
	AuthenticatedRegistry = "TANZU_CLI_AUTHENTICATED_REGISTRY"

	// UseStableKubeContextNameForTanzuContext uses the stable kube context name associated with tanzu context.
	// CLI would not change the context name when the TAP resource pointed by the CLI context is changed.
	UseStableKubeContextNameForTanzuContext = "TANZU_CLI_USE_STABLE_KUBE_CONTEXT_NAME"

	// ActivatePluginsOnPluginGroupPublish activates all the plugins specified within the plugin group
	// as part of the plugin group publishing
	ActivatePluginsOnPluginGroupPublish = "TANZU_CLI_ACTIVATE_PLUGINS_ON_PLUGIN_GROUP_PUBLISH"

	// UseTanzuCSP uses the Tanzu CSP while login/context creation
	UseTanzuCSP = "TANZU_CLI_USE_TANZU_CLOUD_SERVICE_PROVIDER"

	// TPKubernetesOpsEndpoint specifies kubernetes ops endpoint for the Tanzu Platform
	// This will be used as part of `tanzu login`
	TPKubernetesOpsEndpoint = "TANZU_CLI_K8S_OPS_ENDPOINT"
	// TPHubEndpoint specifies hub endpoint for the Tanzu Platform
	// This will be used as part of `tanzu login`
	TPHubEndpoint = "TANZU_CLI_HUB_ENDPOINT"
	// TPUCPEndpoint specifies UCP endpoint for the Tanzu Platform
	// This will be used as part of `tanzu login`
	TPUCPEndpoint = "TANZU_CLI_UCP_ENDPOINT"
)
