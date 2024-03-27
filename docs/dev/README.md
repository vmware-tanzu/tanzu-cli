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
