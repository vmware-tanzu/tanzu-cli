# End-to-End testing in Tanzu CLI Core

## E2E tests

End-to-End (E2E) test cases validate the Tanzu CLI Core product functionality
in an environment which resembles a real production environment. They also
validate the backward compatibility of plugins which are developed with
versions of the Tanzu Plugin Runtime library older than the one used by the
current CLI Core. The CLI Core project has unit and integration test cases
covering current functionality, but the E2E tests perform validation from an
end user's perspective and test the product as a whole in a production-like
environment.

E2E tests ensure the consistent and reliable behavior of the CLI Core code
base. CLI Core E2E CI pipelines are the final signal to ensure that the CLI
Core product is functional according to product specifications, and ready for
release.

## E2E Framework and Tools

The End-to-End (E2E) test framework provides basic tooling and utility
functions to write E2E test cases. This framework includes: generating and
publishing plugin binaries/bundles, creating k8s clusters, executing unix
commands, and performing CLI core commands and processing their output. Apart
from the basic framework tooling, the test cases are written and executed using
the Ginkgo Test Framework. Before writing E2E tests one should be familiar with
how to write test cases using Ginkgo and how to add logging information using
that framework. One should also be familiar with the E2E framework itself so as
to use the existing tooling instead of potentially re-writing utility
functions.

**E2E Framework Interfaces**:

The CLI Core E2E framework has a struct type called `Framework` which provides
all the interfaces and utility functions mentioned in the previous section. To
write an E2E test, one should create an object of type `Framework` using
`framework.NewFramework()`, then use the object to trigger different CLI core
commands lifecycle operations and access helper functions.

```go
// Framework has all CLI Core commands lifecycle operations and helper functions to write CLI e2e test cases
type Framework struct {
    CliOps
    Config ConfigLifecycleOps
    ClusterOps
    PluginCmd    PluginCmdOps // performs plugin command operations
    PluginHelper PluginHelperOps // helper (pre-setup) for plugin cmd operations
}
```

Below are the major interfaces defined and implemented as part of the E2E
Framework (which are part of the `Framework` struct type). These interfaces are
used to write E2E test cases using the Ginkgo test framework. The interfaces
are self-explanatory:

To execute unix commands:

```go
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

To generate plugins and publish plugin binaries and bundles to a local OCI
repository and TMC (yet to implement)

```go
// GeneratePluginOps helps to generate script-based plugin binaries, and plugin binaries can be used to perform plugin testing
// like, add plugin source, list, and install plugins. And call sub-commands such as info and version.
type GeneratePluginOps interface {
    // GeneratePluginBinaries generates plugin binaries for given plugin metadata and return generated plugin binary file paths
    GeneratePluginBinaries(pluginsMeta []*PluginMeta) ([]string, []error)
}

// PublishPluginOps helps to publish plugin binaries and plugin bundles
type PublishPluginOps interface {
    // PublishPluginBinary publishes the plugin binaries to given registry bucket and returns the plugin distribution urls
    PublishPluginBinary(pluginsInfo []*PluginMeta) (distributionUrls []string, errs []error)
    // GeneratePluginBundle generates plugin bundle in local file system for given plugin metadata
    GeneratePluginBundle(pluginsMeta []*PluginMeta) ([]string, []error)
    // PublishPluginBundle publishes the plugin bundles to given registry bucket and returns the plugins discovery urls
    PublishPluginBundle(pluginsInfo []*PluginMeta) (discoveryUrls []string, errs []error)
}

// PluginHelperOps helps to generate and publish plugins
type PluginHelperOps interface {
    GeneratePluginOps
    PublishPluginOps
}
```

To perform tanzu plugin command operations:

```go
// PluginBasicOps helps to perform the plugin command operations
type PluginBasicOps interface {
    // ListPlugins lists all plugins by running 'tanzu plugin list' command
    ListPlugins() ([]PluginListInfo, error)
    // TODO: more plugin command operations will be added, such as clean, delete, describe, install, sync and upgrade
}

// PluginSourceOps helps 'plugin source' commands
type PluginSourceOps interface {
    // AddPluginDiscoverySource adds plugin discovery source
    AddPluginDiscoverySource(discoveryOpts *DiscoveryOptions) error
}

// PluginCmdOps helps to perform the plugin and its sub-commands lifecycle operations
type PluginCmdOps interface {
    PluginBasicOps
    PluginSourceOps
}
```

To perform cluster specific operations:

```go
// ClusterOps has helper operations to perform on cluster
type ClusterOps interface {
    CreateCluster(name string, args []string) (output string, err error)
    DeleteCluster(name string, args []string) (output string, err error)
    ClusterStatus(name string, args []string) (output string, err error)
}

// KindCluster performs k8s KIND cluster operations
type KindCluster interface {
    ClusterOps
}
```

To perform tanzu config command and CLI lifecycle operations:

```go
// ConfigLifecycleOps performs "tanzu config" command operations
type ConfigLifecycleOps interface {
    ConfigSetFeatureFlag(path, value string) error
    ConfigGetFeatureFlag(path string) (string, error)
    ConfigUnsetFeature(path string) error
    ConfigInit() error
    ConfigServerList() error
    ConfigServerDelete(serverName string) error
    DeleteCLIConfigurationFiles() error
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

## Use cases covered in E2E tests

### End user operations

E2E tests are written to validate all CLI Core functionalities from the end-user perspective. They cover all CLI Core commands lifecycle operations. Below is the list of CLI Core commands or use cases covered by the E2E tests:

- CLI lifecycle operations, like build and install the CLI in all possible ways and on all platforms (TODO)
- CLI Config command lifecycle operations, like init, get, set, unset and server related operations (In-progress)
- CLI Plugin command lifecycle operations, like install, upgrade, list, delete and discovery source operations (In-progress)
- CLI Context command lifecycle operations, like create, get, list, delete and use context operations, including target (k8s and TMC) specific use cases (In-progress)
- Other CLI commands lifecycle operations, like update, version, completion and init (TODO)

### plugin compatibility/coexistence tests

The E2E framework tests plugin compatibility by using the current version of the CLI Core and performing basic plugin operations (add/list/delete plugins, and invoke plugin basic commands such as info, help) on plugins built using older versions of the CLI Plugin Runtime library. This ensures that the current CLI supports all older plugins and all plugins can coexists.

## How and when E2E tests are executed

E2E tests are executed as Github runner CI pipelines. The CLI Core E2E test CI pipelines will be executed for every PR created on the CLI Core repository. The E2E tests are organized a list of CLI commands/use cases and plugin compatibility tests in Github CI pipelines, it does shows the test cases results also.

### What is not covered in E2E tests

CLI Core E2E tests do not execute any test cases to validate specific plugin functionalities. For example, for a plugin name `Cluster`, the CLI Core has test cases to validate to discovery and installation of the plugin, but does not test actual functionality of the `Cluster` plugin itself.
