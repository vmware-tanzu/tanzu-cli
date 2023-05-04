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
	ConfigVariableIncludeDeactivatedPluginsForTesting = "TANZU_CLI_INCLUDE_DEACTIVATED_PLUGINS_TEST_ONLY"
	ConfigVariableStandaloneOverContextPlugins        = "TANZU_CLI_STANDALONE_OVER_CONTEXT_PLUGINS"
	// PluginDiscoveryImageSignatureVerificationSkipList is a comma separated list of discovery image urls
	PluginDiscoveryImageSignatureVerificationSkipList = "TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST"
	PublicKeyPathForPluginDiscoveryImageSignature     = "TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH"
	SuppressSkipSignatureVerificationWarning          = "TANZU_CLI_SUPPRESS_SKIP_SIGNATURE_VERIFICATION_WARNING"
	CEIPOptInUserPromptAnswer                         = "TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER"
)
