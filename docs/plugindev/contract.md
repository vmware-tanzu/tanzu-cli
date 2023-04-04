# Plugin Contract

For the plugin binary to be usable as a Tanzu CLI plugin, it must meet a set of
requirements. These requirements are also referred to as the plugin contract.

Since the primary means through which the CLI interacts with plugins is via
plugin command invocation. The contract that each plugin has to satisfy can be
summarized as a set of commands it is expected to implement.

## Plugin Contract Commands

### `version`

This command provides version information about the plugin. The CLI user may
use this command to obtain the plugin's version as well.

### `info`

This command  provides metadata about the plugin that the CLI will use when
presenting information about plugins or when performing lifecycle operations on
them.

The output of the command is a JSON structure, like this:

```json
{
  "name": "builder",
  "description": "Build Tanzu components",
  "target": "global",
  "version": "v0.90.0",
  "buildSHA": "3531c92c",
  "digest": "",
  "group": "Admin",
  "docURL": "",
  "completionType": 0,
  "pluginRuntimeVersion": "v0.82.0"
}
```

### `post-install`

This command provides a means for a plugin to _optionally_ implement some logic
to be invoked right after a plugin is installed. To provide a customized
post-install behavior, plugin developers should provide a PostInstallHook as
part of the PluginDescriptor. Said function will be called when the
post-install command is invoked (which happens every time a plugin is
installed).

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
the purpose of the plugin contract and are thus unavailable for implementing
plugin-specific functionality.

## Satisfying the contract

The plugin contract can be met with minimal effort by integrating with the
tanzu-plugin-runtime library. This is accomplished by instantiating a new
Plugin object and supply some plugin-specific metadata along with it. For
more details, see the "bootstrapping a plugin project" section of the
[plugin developer guide](README.md)
