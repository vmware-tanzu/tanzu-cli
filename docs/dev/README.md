# Tanzu CLI Development

This section provides useful information about developing or building the
tanzu-cli project.

## Building

The default make target to update dependencies, build, test, and lint the CLI:

```sh
make all
```

However, for day-to-day development, the following individual make targets provide
more flexibility:

```sh
make build
make test
make lint
make gomod
```

## Source Code Changes

### Default Directory Locations

The location of the directories used by the CLI are:

1. to store plugin binaries: `<XDG_DATA_HOME>/tanzu-cli`
1. to store the plugin catalog: `$HOME/.cache/tanzu`
1. to store the plugin inventory DB cache and the central configuration file: `$HOME/.cache/tanzu/plugin_inventory/<discovery>`
1. to store configuration files as well as the data store file: `$HOME/.config/tanzu`
1. to store the telemetry DB: `$HOME/.config/tanzu-cli-telemetry`

## Source Code Structure

`cmd/plugin/`: code location for various plugins

Running `make build-all` will build the CLI and any plugins in this repository
unlike `make build` which only builds the CLI.

`cmd/plugin/builder`: code location for the builder plugin

### Data Store

The CLI has a key/value data store in the form of the yaml file `$HOME/.config/tanzu/.data_store.yaml`.
Unlike the configuration files found in the same directory, the data store is for internal CLI use and
not meant to be seen/modified by the user.  It can be used by the CLI to store certain information
and read it back.  For example the feature that notifies the user of a new release of the CLI uses the data store
to store a timestamp of the last time the user was notified so that such notifications can be limited to once a day.

The `datastore.GetDataStoreValue(key, &value)`, `datastore.SetDataStoreValue(key value)` and
`datastore.DeleteDataStoreValue(key)` API can be used to make use of the CLI data store.

### Tests

To run unit tests within the repository:

```sh
make test
```

To run e2e tests for the repository:

```sh
make e2e-cli-core
```

### Environment variables for testing the CLI

Some test options affecting the CLI are only available through the use of environment
variables.  These variables are only meant to be used during testing and are not subject
to the CLI's backwards-compatibility policy, i.e., they could be changed or removed at any time.
These variables can be set in the shell or using `tanzu config set env.<VAR> <VALUE>`
(see [the documentation about this command](../full/README.md#cli-configuration)).
Below is the list of these variables:

| Environment Variable | Description | Value |
| -------------------- | ----------- | ----- |
| `SQL_STATEMENTS_LOG_FILE` | Specifies a log file where SQL commands will be logged when _modifying_ the plugin inventory database.  This is done when publishing plugins using the `builder` plugin. | A file name with its path |
| `TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY` | Specifies test plugin repositories to use as a supplement to the production Central Repository of plugins. Ignored if `TANZU_CLI_PRIVATE_PLUGIN_DISCOVERY_IMAGES` is set. | Comma-separated list of test plugin repository URIs| |
| `TANZU_CLI_ESSENTIALS_PLUGIN_GROUP_NAME` | Override the default name (`vmware-tanzucli/essentials`) of the Essential Plugins group.  Should not be needed. | Group name |
| `TANZU_CLI_ESSENTIALS_PLUGIN_GROUP_VERSION` | Specify a fixed version to use for the Essential Plugins group instead of the latest.  Should not be needed. | Group version |
| `TANZU_CLI_INCLUDE_DEACTIVATED_PLUGINS_TEST_ONLY` | Instruct the CLI to treat deactivated plugins as if they were active | `1` or `true` to use deactivated plugin, `0`, `false`, `""` or unset not to use them |
| `TANZU_CLI_E2E_TEST_BINARY_PATH` | Specifies the CLI binary to use for E2E tests.  Defaults to `tanzu` as found on `$PATH`. | The path including the binary to the CLI  |
| `TANZU_CLI_PLUGIN_DB_CACHE_REFRESH_THRESHOLD_SECONDS` | Overrides the default threshold at which point the plugin inventory will be automatically refreshed.  Default: 24 hours. | Threshold in seconds |
| `TANZU_CLI_PLUGIN_DB_CACHE_TTL_SECONDS` | Overrides the default 30 minute delay in which the plugin inventory cache is used without checking if it should be refreshed. | Delay in seconds |
| `TANZU_CLI_PLUGIN_DISCOVERY_PATH_FOR_TANZU_CONTEXT` | Allows testing the preliminary context-scoped plugin support for a Tanzu context type. | The path portion of the URI to use for discovery of context-scoped plugins on a Tanzu context |
| `TANZU_CLI_SHOW_PLUGIN_INSTALLATION_LOGS` | Allows to print plugin installation logs during the Essential Plugins installation. |  `1` or `true` to print the logs, `0`, `false`, `""` or unset not to print them |
| `TANZU_CLI_STANDALONE_OVER_CONTEXT_PLUGINS` | Tells the CLI to use any standalone plugins currently installed even if the same plugin is also installed as context-scoped.  This allows to test new plugin versions locally. |  `1` or `true` to use standalone plugins before context-scoped plugins, `0`, `false`, `""` or unset to prioritize context-scoped plugins |
| `TANZU_CLI_SUPERCOLLIDER_ENVIRONMENT` | Specifies the use of the staging super collider environment instead of the production environment. | `"staging"` |
| `TANZU_CLI_TMC_UNSTABLE_URL` | Specifies the endpoint for the TMC cluster to use in E2E tests. | The URI of the endpoint |
| `TANZU_CONFIG` | Use a different `config.yaml` file. | Full path to the new config file |
| `TANZU_CONFIG_METADATA` | Use a different `.config-metadata.yaml` file. | Full path to the new config-metadata file|
| `TANZU_CONFIG_NEXT_GEN` | Use a different `config-ng.yaml` fil.e | Full path to the new config-ng file |
| `TEST_CUSTOM_CATALOG_CACHE_DIR` | Use a different directory for the `catalog.yaml` plugin catalog cache file. | Full path of the directory |
| `TEST_CUSTOM_DATA_STORE_FILE` | Use a different `.data-store.yaml` file | Full path of the new file |
| `TEST_CUSTOM_PLUGIN_COMMAND_TREE_CACHE_DIR` | Use a different directory for the command-tree information used for telemetry. | Full path to the new directory |
| `TEST_TANZU_CLI_USE_DB_CACHE_ONLY` | Always use the plugin inventory cache, never try to refresh it. |  `1` or `true` to always use the cache, `0`, `false`, `""` to properly refresh the cache when needed |
| `TZ_ENFORCE_TEST_PLUGIN` | Prevent the installation of a plugin that does not provide a corresponding `test` plugin. | `1` to require plugins to have a `test` plugin, any other value or unset for the `test` plugin to be optional |

The following other variables are used internally through the `Makefile` for E2E tests:

- `TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR`
- `TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR`
- `TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH`
- `TANZU_CLI_E2E_TEST_ENVIRONMENT`

### Debugging

By default the CLI is built without debug symbols to reduce its binary size;
this has the side-effect of preventing the use of a debugger towards the built
binary.  However, when using an IDE, a different binary is used, one built by
the IDE, and therefore the debugger can be used directly from the IDE.

If you require using a debugger directly on a CLI binary, you have to build
a binary that includes the debug symbols.  You can build such a binary by using
`TANZU_CLI_ENABLE_DEBUG=1` along with your build command.

## Centralized Discovery of Plugins

The Tanzu CLI uses a system of plugins to provide functionality to interact
with different Tanzu products. To install a plugin, the CLI uses an OCI
discovery source, which contains the inventory of plugins. For the
implementation details of the OCI discovery solution, please refer to the
[Centralized Discovery](centralized_plugin_discovery.md) document.

## Central Configuration

The "Central Configuration" refers to an asynchronously updatable, centrally-hosted CLI configuration.
Deployed CLIs regularly read this Central Configuration and can take action on specific changes.
Publishing changes to the Central Configuration allows to modify the behavior of existing CLI
binaries without requiring a new release.

Currently, the CLI uses the Central Configuration to check if a new version of the CLI itself has
been released and in such a case notifies the user.  This implies that upon each new release of the CLI,
the newly released version number is added to the Central Configuration for existing CLIs to detect.

The Central Configuration is stored in a `central_config.yaml` file that is bundled in the same OCI image
as the database of the central repository of plugins.  On the user's machine, the Central Configuration file
benefits from the automatic refresh of this OCI image and the `central_config.yaml` file is stored in the
same location as the database of plugins, e.g., `$HOME/.cache/tanzu/plugin_inventory/default/central_config.yaml`.

### Using the Central Configuration

The Central Configuration is a list of key/value pairs where the key is a string and the value is any structure
that is valid yaml.  To read the Central Configuration the `CentralConfig` interface should be used as follows:

```go
  discoverySource, err := config.GetCLIDiscoverySource("default")
  reader := centralconfig.NewCentralConfigReader(discoverySource)
  var myValue myValueType
  err = reader.GetCentralConfigEntry("myStringKey", &myValue)
```

## Global Initializers

The CLI has a concept of global initializers accessible from the `globalinit` package.  Such initializers can
be used by CLI features to initialize/update certain data required for the proper functioning of the CLI itself.
Such initializers are often useful after an older CLI version was executed and a new CLI version now needs to
properly setup its data.

## Deprecation of existing functionality

Any changes aimed to remove functionality in the CLI (e.g. commands, command
flags) have to follow the deprecation policy. For more details on the
deprecation policy and process please refer to the [Deprecation
document](deprecation.md).

## Shell Completion Support

The Tanzu CLI supports shell completion for a series of shells as provided by
the [Cobra project](https://github.com/spf13/cobra).  Shell completion for commands and
flag names is automatically handled by Cobra.  However, shell completion for
arguments and flag values must be coded in the Tanzu CLI itself.

All core CLI command and flags should provide proper shell completion for the
arguments and flag values they accept.  Whenever a new command or flag is added
the appropriate shell completion code must also be added.  For examples, please
refer to existing `ValidArgsFunction` function implementations and calls to
`RegisterFlagCompletionFunc()`.

### ActiveHelp Support

ActiveHelp are messages printed through shell completion as the program is being used.
The Tanzu CLI provides ActiveHelp in certain situations which should be maintained.
For examples, please refer to calls to `cobra.AppendActiveHelp()`.
The following simple guidelines should be respected:

1. when all arguments for a command have been provided on the command line, the functions `noMoreCompletions` or `activeHelpNoMoreArgs` should be used to provide ActiveHelp to indicate to the user no more arguments are accepted,
1. whenever a command accepts an argument or a flag accepts a value, but that the shell completion code is unable to provide suggestions, an ActiveHelp message should be provided to guide the user,
1. when the shell completion code is unable to provide suggestions due to an error with user input (e.g, an invalid plugin name), an ActiveHelp message should be added to guide the user in realizing what the problem is.
