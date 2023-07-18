# End-to-End testing in Tanzu CLI Core

## E2E Framework and Tools

The CLI End-to-End (E2E) test framework provides basic tooling
and utility functions to write E2E test cases for CLI functionality
and also to write e2e tests for any CLI based plugin functionality, so
this framework does helps for CLI developers and plugin developers to
implement e2e test cases.

This e2e test framework supports below functionalities:

- Executing any CLI Core command-line functionalities
- Creating and operating k8s KIND clusters
- Executing any unix or command-line commands
- Performing CLI Core commands and processing their outputs
- Framework is extensible to more functionalities
- Framework does implements e2e test cases for CLI functionalities
- Plugin developers can use this framework to write their e2e tests for their plugin functionality

CLI Core developers, the CLI E2E test cases are written and implemented using
the Ginkgo Test Framework. Therefore, before writing CLI Core E2E test cases, one should be
familiar with the Ginkgo testing framework and CLI Core E2E test framework to use
existing tooling and functionalities.

Plugin developers, can use your own test framework to implement your e2e tests
by using the CLI E2E framework, if you see any gaps or needed improvements
let us know we can improve the E2E framework.

For more details about the **E2E framework functionalities** expand below section

<details>
    <summary>CLI E2E Framework functionalities</summary>

The CLI E2E framework is a separate [CLI E2E Framework module](https://github.com/vmware-tanzu/tanzu-cli/tree/main/test/e2e)
in [CLI Core repository](https://github.com/vmware-tanzu/tanzu-cli), so to use the CLI E2E framework
need to import the module "github.com/vmware-tanzu/tanzu-cli/tree/main/test/e2e"

The CLI Core E2E framework has a struct type called `Framework` which provides
all the interfaces and utility functions to implement E2E test cases,
E2E test implementer need to create Framework object `framework.NewFramework()`
to utilize framework functionalities like, execute CLI commands like
end user calling from command line prompt.

``` go
// Framework has all CLI Core commands lifecycle operations and helper functions to write CLI e2e test cases
type Framework struct {
CliOps
Config       ConfigLifecycleOps
KindCluster  ClusterOps      // performs KIND cluster operations
PluginCmd    PluginCmdOps    // performs plugin command operations
PluginHelper PluginHelperOps // helper (pre-setup) for plugin cmd operations
ContextCmd   ContextCmdOps
}

```

To customize the choice of the Tanzu binary for usage, you have the option to indicate it by utilizing the environment variable `TANZU_CLI_E2E_TEST_BINARY_PATH`.
Additionally, within the Tests, the `WithTanzuBinary()` helper method allows you to personalize the Tanzu binary selection.

Below are the major interfaces defined and implemented as part of the E2E
Framework (which are part of the `Framework` struct type). These interfaces are
used to write E2E test cases using the Ginkgo test framework. The interfaces
are self-explanatory:

To execute unix commands:

``` go
// CmdOps performs the Command line exec operations
type CmdOps interface {
    Exec(command string) (stdOut, stdErr *bytes.Buffer, err error)
    ExecContainsString(command, contains string) error
    ExecContainsAnyString(command string, contains []string) error
    ExecContainsErrorString(command, contains string) error
    ExecNotContainsStdErrorString(command, contains string) error
    ExecNotContainsString(command, contains string) error
}
```

To perform tanzu plugin command operations:

``` go
type PluginCmdOps interface {
    PluginBasicOps
    PluginSourceOps
    PluginGroupOps
}

// PluginBasicOps helps to perform the plugin command operations
type PluginBasicOps interface {
    // ListPlugins lists all plugins by running 'tanzu plugin list' command
    ListPlugins() ([]*PluginInfo, error)
    // ListInstalledPlugins lists all installed plugins
    ListInstalledPlugins() ([]*PluginInfo, error)
    // ListPluginsForGivenContext lists all plugins for a given context and either installed only or all
    ListPluginsForGivenContext(context string, installedOnly bool) ([]*PluginInfo, error)
    // SearchPlugins searches all plugins for given filter (keyword|regex) by running 'tanzu plugin search' command
    SearchPlugins(filter string) ([]*PluginInfo, error)
    // InstallPlugin installs given plugin and flags
    InstallPlugin(pluginName, target, versions string) error
    // Sync performs sync operation
    Sync() (string, error)
    // DescribePlugin describes given plugin and flags
    DescribePlugin(pluginName, target string) (string, error)
    // UninstallPlugin uninstalls/deletes given plugin
    UninstallPlugin(pluginName, target string) error
    // DeletePlugin deletes/uninstalls given plugin
    DeletePlugin(pluginName, target string) error
    // ExecuteSubCommand executes specific plugin sub-command
    ExecuteSubCommand(pluginWithSubCommand string) (string, error)
    // CleanPlugins executes the plugin clean command to delete all existing plugins
    CleanPlugins() error
}
// PluginSourceOps helps 'plugin source' commands
type PluginSourceOps interface {
    // UpdatePluginDiscoverySource updates plugin discovery source, and returns stdOut and error info
    UpdatePluginDiscoverySource(discoveryOpts *DiscoveryOptions) (string, error)

    // DeletePluginDiscoverySource removes the plugin discovery source, and returns stdOut and error info
    DeletePluginDiscoverySource(pluginSourceName string) (string, error)

    // ListPluginSources returns all available plugin discovery sources
    ListPluginSources() ([]*PluginSourceInfo, error)

    // InitPluginDiscoverySource initializes the plugin source to its default value, and returns stdOut and error info
    InitPluginDiscoverySource(opts ...E2EOption) (string, error)
}
type PluginGroupOps interface {
    // SearchPluginGroups performs plugin group search
    // input: flagsWithValues - flags and values if any
    SearchPluginGroups(flagsWithValues string) ([]*PluginGroup, error)

    // GetPluginGroup performs plugin group get
    // input: flagsWithValues - flags and values if any
    GetPluginGroup(groupName string, flagsWithValues string, opts ...E2EOption) ([]*PluginGroupGet, error)

    // InstallPluginsFromGroup a plugin or all plugins from the given plugin group
    InstallPluginsFromGroup(pluginNameORAll, groupName string) error
}
```

To perform cluster specific operations:

``` go
// ClusterOps has helper operations to perform on cluster
type ClusterOps interface {
    // CreateCluster creates the cluster with given name
    CreateCluster(clusterName string) (output string, err error)
    // DeleteCluster deletes the cluster with given name
    DeleteCluster(clusterName string) (output string, err error)
    // ClusterStatus checks the status of the cluster for given cluster name
    ClusterStatus(clusterName string) (output string, err error)
    // GetClusterEndpoint returns the cluster endpoint for the given cluster name
    GetClusterEndpoint(clusterName string) (endpoint string, err error)
    // GetClusterContext returns the given cluster kubeconfig context
    GetClusterContext(clusterName string) string
    // GetKubeconfigPath returns the default kubeconfig path
    GetKubeconfigPath() string
    // ApplyConfig applies the given configFilePath on to the given contextName cluster context
    ApplyConfig(contextName, configFilePath string) error
}

// KindCluster performs k8s KIND cluster operations
type KindCluster interface {
    ClusterOps
}
```

To perform tanzu config command and CLI lifecycle operations:

``` go
// ConfigLifecycleOps performs "tanzu config" command operations
type ConfigLifecycleOps interface {
    // ConfigSetFeatureFlag sets the tanzu config feature flag
    ConfigSetFeatureFlag(path, value string) error
    // ConfigGetFeatureFlag gets the tanzu config feature flag
    ConfigGetFeatureFlag(path string) (string, error)
    // ConfigUnsetFeature un-sets the tanzu config feature flag
    ConfigUnsetFeature(path string) error
    // ConfigInit performs "tanzu config init"
    ConfigInit() error
    // GetConfig gets the tanzu config
    GetConfig() (*configapi.ClientConfig, error)
    // ConfigServerList returns the server list
    ConfigServerList() ([]*Server, error)
    // ConfigServerDelete deletes given server from tanzu config
    ConfigServerDelete(serverName string) error
    // DeleteCLIConfigurationFiles deletes cli configuration files
    DeleteCLIConfigurationFiles() error
    // IsCLIConfigurationFilesExists checks the existence of cli configuration files
    IsCLIConfigurationFilesExists() bool
}

// CliOps performs basic cli operations
type CliOps interface {
    CliInit() error
    CliVersion() (string, error)
    InstallCLI(version string) error
    UninstallCLI(version string) error
}
```

</details>

## What is CLI Core E2E tests

End-to-End (E2E) test cases validate the Tanzu CLI Core product functionality
in an environment that resembles a real production environment. They also
validate the backward compatibility of plugins that are developed with
versions of the Tanzu Plugin Runtime library older than the one used by the
current CLI Core. The CLI Core project has unit and integration test cases
to cover isolated functionalities (white box testing). Still, the E2E tests
perform validation from an end-user perspective and test the product as
a whole in a production-like environment.

E2E tests ensure the consistent and reliable behavior of the CLI Core code
base. CLI Core E2E CI pipelines are the final signal to ensure that the CLI
Core product is functional according to product specifications.

## Use cases covered in CLI E2E test cases implementation

### End user operations

E2E tests are written to validate all CLI Core functionalities from the end-user perspective.
They cover all CLI Core commands lifecycle operations. Below is the list of CLI Core commands
or functionalities covered by the E2E tests:

- CLI lifecycle operations, like build and install the CLI in all possible ways and on all platforms (TODO)
- CLI Config command lifecycle operations, like init, get, set, unset and server related operations
- CLI Plugin command lifecycle operations, like search, group, group search, plugin install/upgrade/delete, list and source operations
- CLI Context command lifecycle operations, like create, get, list, delete and use context operations,
  including target (k8s and TMC) specific use cases
- Other CLI commands lifecycle operations, like update (TODO), version, completion and init

### Plugin compatibility/coexistence tests

The E2E framework tests plugin compatibility by using the current version of
the CLI Core and performing basic plugin operations (add/list/delete plugins,
and invoke plugin basic commands such as info, help) on plugins built using
older versions of the CLI Plugin Runtime library. This ensures that
the current CLI supports all older plugins and all plugins
(which are built with supported CLI Runtime version's) can coexists.

We have created a github repository `https://github.com/chandrareddyp/tanzu-cli-test-plugins` to host
test plugins with different CLI Runtime versions which supported by current CLI Core, and
these test plugins are published manually to gcr.io/eminent-nation-87317/tanzu-cli/test/v1/plugins/plugin-inventory:latest
using publish tooling, these test plugins used during e2e plugin compatibility test execution.

## How and when E2E tests are executed

E2E tests are executed as Github runner CI pipelines. The CLI Core E2E test
CI pipelines will be executed for every PR created on the CLI Core repository.
The E2E tests are organized a list of CLI commands/use cases and
plugin compatibility tests in Github CI pipelines, it does shows the test cases results also.

## E2E Test results

You can check most recent E2E test cases execution results at
[![Tanzu CLI Core E2E Tests](https://github.com/vmware-tanzu/tanzu-cli/actions/workflows/cli_core_e2e_test.yaml/badge.svg?branch=main&event=push)](https://github.com/vmware-tanzu/tanzu-cli/actions/workflows/cli_core_e2e_test.yaml?query=event:push+branch:main)

Below table shows list of functionalities and number of test cases or use cases
being executed in E2E test framework on every pull request to main branch:

|       Functionality       | Number of use cases or test cases |
|:-------------------------:|:---------------------------------:|
|   Plugin Compatibility    |                 7                 |
|     Plugin lifecycle      |                28                 |
|   Plugin sync lifecycle   |                28                 |
|       CLI lifecycle       |                 2                 |
|      Config command       |                11                 |
| Context for k8s use cases |                11                 |
| Context for TMC use cases |                10                 |

## What is not covered in E2E tests

CLI Core E2E tests do not execute any test cases to validate
specific plugin functionalities. For example, for a plugin name `Cluster`,
the CLI Core has test cases to validate to discovery and installation of
the plugin, but does not test actual functionality of the `Cluster` plugin itself.
