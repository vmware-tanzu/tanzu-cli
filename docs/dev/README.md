# Tanzu CLI Development

This section provides useful information about developing or building the
tanzu-cli project.

## Building

default target to update dependencies, build test and lint the CLI:

```sh
make all
```

## Source Code Changes

### Default Directory Locations

The names of the directories for the plugins, catalog cache and local
plugin discovery (`<XDG_DATA_HOME>/_tanzu-cli, $HOME/.cache/_tanzu,$HOME/.config/_tanzu-plugins`)
are all directories prefixed with '_' for now, so as not to conflict with their nonprefixed counterparts.

This setup is temporary. We intend to unify the locations of various
directories with those used by existing CLI installations once the new CLI
releases can serve as drop-in replacements for the existing ones.

## Source Code Structure

cmd/plugin/ : code location for various plugins

`make build-all` will build the CLI and any plugins in this directory
unlike `make build` which only builds the CLI

cmd/plugin/builder : code location for builder plugin

Note on `builder init`:
The generated project's Makefile expects the TZBIN to be set to the name
of the CLI binary located in the user's path. Its default value is
currently set to 'tz'. This convention will allow the CLI under
development to coexist with the released tanzu CLI typically name 'tanzu'.
We should continue to adopt said convention until the CLI under
development is released as a backward-compatible replacement of the
existing CLI.

### Tests

```sh
make test
```

## Deprecation of existing functionality

Any changes aimed to remove functionality in the CLI (e.g. commands, command
flags) have to follow the deprecation policy, For more details on the
deprecation policy and process please refer to the [Deprecation
document](../dev/deprecation.md).
