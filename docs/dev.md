# Development

## Building

default target to update dependencies, build test and lint the CLI:

```sh
make all
```

NOTE: Until tanzu-plugin-runtime is public, to avoid checksum issues when accessing
said dependency, run this prior to build:

```sh
go env -w GOPRIVATE=github.com/vmware-tanzu/tanzu-plugin-runtime
```

## Source Code Changes

### Default Directory Locations

The names of the directories for the plugins, catalog cache and local
plugin discovery (`<XDG_DATA_HOME>/_tanzu-cli, $HOME/.cache/_tanzu,
$HOME/.config/_tanzu-plugins`) are all directories prefixed with '_' for
now, so as not to conflict with their nonprefixed counterparts

## Source Code Structure

### Tests
