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
)
