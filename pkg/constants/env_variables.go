// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package constants

// environment variables for http proxy
const (
	ProxyCACert = "PROXY_CA_CERT"
)

const (
	AllowedRegistries                           = "ALLOWED_REGISTRY"
	ConfigVariablePreReleasePluginRepoImage     = "TANZU_CLI_PRE_RELEASE_REPO_IMAGE"
	ConfigVariableAdditionalDiscoveryForTesting = "TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY"
	// PluginDiscoveryImageSignatureVerificationSkipList is a comma separated list of discovery image urls
	PluginDiscoveryImageSignatureVerificationSkipList = "TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST"
	PublicKeyPathForPluginDiscoveryImageSignature     = "TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH"
	SuppressSkipSignatureVerificationWarning          = "TANZU_CLI_SUPPRESS_SKIP_SIGNATURE_VERIFICATION_WARNING"
)
