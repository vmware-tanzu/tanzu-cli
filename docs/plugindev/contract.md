# Plugin Contract

plugin autocompletion integration

- plugin expected to provide autocompletion support for its own commands

main cli will capture and passthrough the arguments it receives
plugin post-install

--------------------------

More than any arbitrary executable binary, the plugin binary has to satisfy a set of for
usable as a Tanzu CLI plugin. These requirements are also referred to as the
plugin contract.

Since the primary means through which the CLI interacts with plugins is via
plugin command invocation. The contract that each plugin has to satisfy can be
summarized as a set of commands it is expected to implement:

## Commands to implement

### `version`

This command provides basic version information about the plugin, likely the
only common command of broad use to the CLI user.

### `info`

This command  provides metadata about the plugin that the CLI will use when
presenting information about plugins or when performing lifecycle operations on
them.

The output of the command is a json structure, like this:

```json
{
  "name": "builder",
  "description": "Build Tanzu components",
  "target": "global",
  "version": "v0.1.0-dev-18-g3531c92c",
  "buildSHA": "3531c92c",
  "digest": "",
  "group": "Admin",
  "docURL": "",
  "completionType": 0,
  "pluginRuntimeVersion": "v0.0.2-0.20230321210330-330c29284da6"
}
```

### `post-install`

This command provides a means for a plugin to optionally implement some logic
to be invoked right after a plugin is installed. To provide a customized
post-install behavior, plugin developers should provide a PostInstallHook as
part of the PluginDescriptor. Said function will be called when the
post-install command is invoked (every time a plugin is installed).

### `generate-docs`

This command generates a tree of markdown documentation files for the commands
the plugin provides. It is typically used by the CLI's `generate-all-docs`
command to produce command documentation for all installed plugins.

### `lint`

Validate the command name and arguments to flag any new terms unaccounted for
in the CLI [taxonomy document](taxonomy.md). Authors are highly encouraged to
follow the [CLI Style Guide](style_guide.md) and adhere to the taxonomy where
possible to minimize violations flagged by this command.

### Notes

It should be apparent that the above-mentioned command names are reserved for
the purpose of the plugin contract and are thus unevailable for implementing
plugin-specific functionality.

## Satisfying the contract

By integrating the the tanzu-plugin-runtime library, the plugin contract can be
satisfied with minimal effort. This is accomplished via instantiating a new
Plugin object and supplying some plugin-specific metadata along with it. For
more details, see the "bootstrapping a plugin project" section of the [plugin developer guide](README.md)
