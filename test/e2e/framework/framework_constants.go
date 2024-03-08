// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package framework defines the integration and end-to-end test case for cli core
package framework

import (
	"time"
)

const (
	CliCore = "[CLI-Core]"

	TargetList = "kubernetes[k8s]/mission-control[tmc]/operations[ops]/global"

	InitCmd          = "%s init"
	VersionCmd       = "%s version"
	CompletionCmd    = "%s completion"
	CobraCompleteCmd = "%s __complete ''"
	TanzuPrefix      = "tanzu"
	TzPrefix         = "tz"

	// Config commands
	ConfigCmd        = "%s config"
	ConfigSet        = "%s config set "
	ConfigGet        = "%s config get"
	ConfigUnset      = "%s config unset "
	ConfigInit       = "%s config init"
	ConfigCertAdd    = "%s config cert add --host %s --ca-cert %s --skip-cert-verify %s --insecure %s"
	ConfigCertDelete = "%s config cert delete %s"
	ConfigCertList   = "%s config cert list -o json"

	// Plugin commands
	UpdatePluginSource                  = "%s plugin source update %s --uri %s"
	ListPluginSourcesWithJSONOutputFlag = "%s plugin source list -o json"
	DeletePluginSource                  = "%s plugin source delete %s"
	InitPluginDiscoverySource           = "%s plugin source init"
	ListPluginsCmdWithJSONOutputFlag    = "%s plugin list -o json"
	SearchPluginsCmd                    = "%s plugin search"
	SearchPluginGroupsCmd               = "%s plugin group search"
	GetPluginGroupCmd                   = "%s plugin group get %s"
	InstallPluginCmd                    = "%s plugin install %s"
	InstallPluginFromGroupCmd           = "%s plugin install %s --group %s"
	InstallAllPluginsFromGroupCmd       = "%s plugin install --group %s"
	DescribePluginCmd                   = "%s plugin describe %s"
	UninstallPLuginCmd                  = "%s plugin uninstall %s --yes"
	CleanPluginsCmd                     = "%s plugin clean"
	pluginSyncCmd                       = "%s plugin sync"
	PluginDownloadBundleCmd             = "%s plugin download-bundle"
	PluginUploadBundleCmd               = "%s plugin upload-bundle"
	PluginCmdWithOptions                = "%s plugin "
	JSONOutput                          = " -o json"
	TestPluginsPrefix                   = "test-plugin-"
	PluginSubCommand                    = "%s %s"
	PluginKey                           = "%s_%s_%s" // Plugins - Name_Target_Versions
	LegacyPluginKey                     = "%s_%s"    // Plugins - Name_Target_Versions or Name_Version_Status for legacy cli plugins

	// Central repository
	TanzuCliE2ETestCentralRepositoryURL                                             = "TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL"
	TanzuCliE2ETestLocalCentralRepositoryURL                                        = "TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL"
	TanzuCliE2ETestLocalCentralRepositoryPluginDiscoveryImageSignaturePublicKeyPath = "TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH"
	TanzuCliPluginDiscoveryImageSignaturePublicKeyPath                              = "TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH"
	TanzuCliE2ETestLocalCentralRepositoryHost                                       = "TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST"
	TanzuCliE2ETestLocalCentralRepositoryCACertPath                                 = "TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH"
	TanzuCliE2ETestAirgappedRepo                                                    = "TANZU_CLI_E2E_AIRGAPPED_REPO"
	TanzuCliE2ETestAirgappedRepoWithAuth                                            = "TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH"
	TanzuCliE2ETestAirgappedRepoWithAuthUsername                                    = "TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_USERNAME"
	TanzuCliE2ETestAirgappedRepoWithAuthPassword                                    = "TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_PASSWORD"
	TanzuCliPluginDiscoverySignatureVerificationSkipList                            = "TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST"

	// CLI Coexistence
	CLICoexistenceLegacyTanzuCLIInstallationPath = "TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR"
	CLICoexistenceNewTanzuCLIInstallationPath    = "TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR"
	CLICoexistenceLegacyTanzuCLIVersion          = "TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_VERSION"
	CLICoexistenceNewTanzuCLIVersion             = "TANZU_CLI_BUILD_VERSION"
	CLICoexistenceTanzuCEIPParticipation         = "TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER"
	CLIEulaParticipation                         = "TANZU_CLI_EULA_PROMPT_ANSWER"

	// This skips hardcoding HTTPS in CLI Core when the E2E tests mock the TMC endpoint
	CLIE2ETestEnvironment = "TANZU_CLI_E2E_TEST_ENVIRONMENT"

	// General constants
	True             = "true"
	Installed        = "installed"
	UpdateAvailable  = "update available"
	NotInstalled     = "not installed"
	RecommendInstall = "install recommended"
	RecommendUpdate  = "update recommended"
	JSONOtuput       = "-o json"

	// Context commands
	CreateContextWithEndPoint              = "%s context create --endpoint %s %s"
	CreateContextWithEndPointStaging       = "%s context create --endpoint %s --staging %s"
	CreateContextWithKubeconfigFile        = "%s context create --kubeconfig %s --kubecontext %s %s"
	CreateContextWithDefaultKubeconfigFile = "%s context create --kubecontext %s %s"
	UseContext                             = "%s context use %s"
	UnsetContext                           = "%s context unset"
	GetContext                             = "%s context get %s"
	ListContextOutputInJSON                = "%s context list -o json"
	DeleteContext                          = "%s context delete %s --yes"
	TanzuAPIToken                          = "TANZU_API_TOKEN" //nolint:gosec
	TanzuCliTmcUnstableURL                 = "TANZU_CLI_TMC_UNSTABLE_URL"

	// context specific
	ContextShouldNotExists              = "the context %s should not exists"
	ContextShouldExistsAsCreated        = "the context %s should exists as its been created"
	ContextNotExistsForTarget           = "The provided context '%v' does not exist or is not active for the given context type '%v'"
	NoActiveContextExistsForContextType = "There is no active context for the given context type '%v'"
	ContextNotActiveOrNotExists         = "The provided context '%v' is not active or does not exist"
	ContextForContextTypeSetInactive    = "The context '%v' of type '%v' has been set as inactive"
	DeactivatingPlugin                  = "Deactivating plugin '%v:%v' for context '%v'"

	KindClusterCreate = "kind create cluster --name %s"
	KindClusterStatus = "kubectl cluster-info --context %s"
	KindClusterDelete = "kind delete cluster --name %s"
	KindClusterGet    = "kind get clusters "
	KindClusterInfo   = "kubectl cluster-info --context %s"
	KubectlApply      = "kubectl --context %s apply -f %s"
	KubectlWait       = "kubectl --context %s wait %s"

	// specific plugin custom resource file name cr_<pluginName>_<target>_<versions>.yaml to apply on kind cluster
	PluginCRFileName = "cr_%s_%s_%s.yaml"

	KindCreateCluster = "kind create cluster --name "
	DockerInfo        = "docker info"
	StartDockerUbuntu = "sudo systemctl start docker"
	StopDockerUbuntu  = "sudo systemctl stop docker"

	TMC                         = "tmc"
	TKG                         = "tkg"
	SourceType                  = "oci"
	GlobalTarget                = "global"
	KubernetesTarget            = "kubernetes"
	MissionControlTarget        = "mission-control"
	TMCPluginGroupPrefix        = "vmware-tmc"
	K8SPluginGroupPrefix        = "vmware-tkg"
	EssentialsPluginGroupPrefix = "vmware-tanzucli"

	// log info
	ExecutingCommand = "Executing command: %s"
	FileContent      = "file: %s , content: %s"

	// Error messages
	UnableToFindPluginForTarget                   = "unable to find plugin '%s' matching version '%s'"
	UnableToFindPluginWithVersionForTarget        = "unable to find plugin '%s' matching version '%s' for target '%s'"
	UnableToFindPlugin                            = "unable to find plugin '%s'"
	InvalidTargetSpecified                        = "invalid target specified. Please specify a correct value for the `--target` flag from '" + TargetList + "'"
	InvalidTargetGlobal                           = "invalid target for plugin: global"
	DiscoverySourceNotFound                       = "discovery %q does not exist"
	ErrorLogForCommandWithErrStdErrAndStdOut      = "error while executing command:'%s', error:'%s' stdErr:'%s' stdOut: '%s'"
	FailedToConstructJSONNodeFromOutputAndErrInfo = "failed to construct json node from output:'%s' error:'%s' "
	FailedToConstructJSONNodeFromOutput           = "failed to construct json node from output:'%s'"
	NoErrorForPluginGroupSearch                   = "should not get any error for plugin group search"
	NoErrorForPluginGroupGet                      = "should not get any error for plugin group get"
	NoErrorForPluginSearch                        = "should not get any error for plugin search"
	PluginSearchOutputShouldBeSortedByName        = "plugin search output should be sorted by name"
	UnableToSync                                  = "unable to automatically sync the plugins recommended by the active context. Please run 'tanzu plugin sync' command to sync plugins manually"
	PluginDescribeShouldNotThrowErr               = "should not get any error for plugin describe"
	PluginDescShouldExist                         = "there should be one plugin description"
	PluginNameShouldMatch                         = "plugin name should be same as input value"

	CompletionWithoutShell        = "shell not specified, choose one of: bash, zsh, fish, powershell"
	CompletionOutputForBash       = "bash completion V2 for tanzu"
	CompletionOutputForZsh        = "zsh completion for tanzu"
	CompletionOutputForFish       = "fish completion for tanzu"
	CompletionOutputForPowershell = "powershell completion for tanzu"
	FailedToRunCompletionCmd      = "failed to run completion command: %s, stdout: %s"
	FailedToRunCmd                = "failed to run command: %s, stdout: %s"

	// config related constants
	FailedToCreateContext           = "failed to create context"
	FailedToCreateContextWithStdout = FailedToCreateContext + ", stdout:%s"
	ContextCreated                  = "context %s created successfully"
	ContextDeleted                  = "context %s deleted successfully"
	FailedToDeleteContext           = "failed to delete context"
	ContextPrefixK8s                = "plugin-sync-k8s-"
	ContextPrefixTMC                = "plugin-sync-tmc-"

	// TestDir is the directory under $HOME, created during framework initialization, and the $HOME updated as $HOME/$TestDir, to create all Tanzu CLI specific files
	// and not to disturb any existing Tanzu CLI files
	TestDir = ".tanzu-cli-e2e"

	// TestPluginsDir is the directory under $HOME/$TestDir, to store test plugins for E2E tests
	TestPluginsDir = ".e2e-test-plugins"

	// TempDirInTestDirPath is the directory under $HOME/$TestDir, to create temporary files (if any) for E2E test execution
	TempDirInTestDirPath = "temp"

	ConfigFolder                = ".config"
	TanzuFolder                 = "tanzu"
	TanzuPluginsFolder          = "tanzu-plugins"
	ConfigFile                  = "config.yaml"
	ConfigNGFile                = "config-ng.yaml"
	K8SCRDFileName              = "cli.tanzu.vmware.com_cliplugins.yaml"
	Config                      = "config"
	TanzuCLIE2ETestBinaryPath   = "TANZU_CLI_E2E_TEST_BINARY_PATH"
	WiredMockHTTPServerStartCmd = "docker run --rm -d -p 8080:8080 -p 8443:8443 --name %s -v %s:/home/wiremock rodolpheche/wiremock:2.25.1"
	HTTPMockServerStopCmd       = "docker container stop %s"
	HTTPMockServerName          = "wiremock"
	defaultTimeout              = 5 * time.Second

	TMCEndpointForPlugins        = "/v1alpha1/system/binaries/plugins"
	TMCMockServerEndpoint        = "http://localhost:8080"
	TMCPluginsMockServerEndpoint = "http://localhost:8080/v1alpha1/system/binaries/plugins"
	HTTPContentType              = "application/json; charset=utf-8"

	// k8s CRD file
	K8SCRDFilePath = "../../config_data/cli.tanzu.vmware.com_cliplugins.yaml"
)
