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
	ConfigVariablePluginDBCacheTTL = "TANZU_CLI_PLUGIN_DB_CACHE_TTL_SECONDS"
)
