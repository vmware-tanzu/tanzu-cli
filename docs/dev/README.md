# Tanzu CLI Development

This section provides useful information about developing or building the
tanzu-cli project.

## Building

default target to update dependencies, build test, and lint the CLI:

```sh
make all
```

## Source Code Changes

### Default Directory Locations

The location of the directories used by the CLI are:

1. to store plugin binaries: `<XDG_DATA_HOME>/tanzu-cli`
1. to store the plugin catalog as well as the plugin inventory DB cache: `$HOME/.cache/tanzu`
1. to store configuration files: `$HOME/.config/tanzu`
1. to store the telemetry DB: `$HOME/.config/tanzu-cli-telemetry`

## Source Code Structure

`cmd/plugin/`: code location for various plugins

Run the `make build-all` will build the CLI and any plugins in this directory
unlike `make build` which only builds the CLI

`cmd/plugin/builder`: code location for the builder plugin

Note on `builder init`:
The generated project's Makefile expects the TZBIN to be set to the name
of the CLI binary located in the user's path. Its default value is
currently set to 'tz'. This convention will allow the CLI under
development to coexist with the released tanzu CLI typically named 'tanzu'.
We should continue to adopt said convention until the CLI under
development is released as a backward-compatible replacement of the
existing CLI.

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

## Deprecation of existing functionality

Any changes aimed to remove functionality in the CLI (e.g. commands, command
flags) have to follow the deprecation policy, For more details on the
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
