# Tanzu CLI User Guide

This section provides more details on how the Tanzu CLI works, and important
functionality and behaviors that the user should be aware of.

## Important Concepts

## Plugin Architecture

The CLI is based on a plugin mechanism where CLI command functionality can
be delivered through independently developed plugin binaries.

While the CLI provides some core functionality like CLI configuration, a
unified command tree and plugin management, much of its power comes from the
plugins that integrate with it.

## Command Groups and Targets

Tanzu CLI commands are organized into command groups. To list available command
groups, run `tanzu`.

In this example, several command groups are shown, such as cluster, builder, package, etc
Commands within a command group can be explored via `tanzu <command group>`,
and invocable via the tanzu commands part of the command group, e.g. `tanzu cluster create ....`

The list of available command groups differs based on what Tanzu CLI plugins
are installed on your machine, and on what endpoints/contexts the CLI is currently
configured to connect to.

While not always the case, commands falling within a single command group are
typically delivered by a single plugin.

## Plugin

A *Tanzu CLI Plugin*, or *plugin* for short unless otherwise noted, is an executable
binary, with one or more invocable commands, used to extend the functionality
of the Tanzu CLI. See the next section for the complete requirements on what it
takes to be a Tanzu CLI plugin.

## Plugin Repository

A plugin repository is a location (typically an OCI registry) to which authored
plugins are typically published. It is a source of plugins that the CLI can
query for what plugins are installable.

## Context

A Context is an isolated scope of relevant client-side configurations for a
combination of user identity and server identity. There can be multiple
contexts for the same combination of `(user, server)`.

## Target

A Target refers to a category or tier of control planes that the CLI can interact with. There are currently two supported targets : `kubernetes` (or `k8s`) and `mission-control` (or `tmc`) which corresponds to the Kubernetes cluster endpoint type and Tanzu Mission Control endpoint type respectively. A context is associated with one of the two supported targets. Plugins are generally associated with one of the above mentioned targets but if a plugin doesn't fall into any of the above categories a developer can create a plugin with the 'global' target. A plugin using the `global` target is available as a root Tanzu CLI command. To see plugins that only apply the `kubernetes` Target or only to the `mission-control` Target, run the command `tanzu <target>'.

Similarly, commands from plugins that are associated with a target are unambiguously invoked by prefixing the command group with the target, like so:

```console
tanzu mission-control data-protection ...
or
tanzu k8s management-cluster ...
```

Note that today, the CLI supports omitting the target for historical reasons, but this omission only applies to commands for the k8s target.  (So `tanzu management-cluster ...` is a valid variant of the above example, but not `tanzu data-protection ...`). The CLI team is exploring making the 'assumed target' configurable.

## Interaction between CLI and its plugins

The Core CLI plays several roles in the plugin architecture:

### Central, consistent point of interaction

All commands provided by the CLI are invocable via the `tanzu` binary, which in turn dispatches the command to the appropriate plugin, capturing output and errors from the latter to return back to the user.

Commands producing meaningful output consistently provide alternative output such as JSON, YAML or tabular formats using the `-o {json|yaml}` flag.

### Plugin discovery and lifecycle management

The CLI is configured to use a default plugin repository. Through various commands like `tanzu plugin search`, `tanzu plugin install`, `tanzu plugin install --group <groupName>`, the CLI provides various means to discover, then securely install or update plugins to serve specific needs of the user.

Certain context endpoints will require that a specific set of plugins be installed so as to enable proper interaction with said endpoints. For these, establishing a connection with these endpoints may lead to the discovery and automatic installation of additional plugins. Plugins installed through this mean of discovery are referred to as "context-scoped" plugins. To learn more about the context-scoped plugins, please check the [context-scoped plugin installation](../full/context-scoped-plugins.md) documentation.

For an overview on some of these plugin lifecycle commands, see the [Quickstart Guide](../quickstart/quickstart.md).
For more details on these commands, see the [command reference](../cli/commands/tanzu_plugin.md).

### Context management

The CLI maintains a list of Contexts and an active Context for each Target type. A plugin command with a particular Target type will always be able to access the active context information by using the APIs exposed by the `tanzu-plugin-runtime` library. This will allow plugins to interact with the endpoint associated with the Context.

## CLI Configuration

The Tanzu CLI configuration is stored in `.config/tanzu/` of your home directory.  It contains:

* Names, target, and kubeconfig locations for the contexts that the CLI knows about, and which context is currently the active one for each target type
* Global and plugin-specific configuration options, or feature flags
* Sources for CLI plugin discovery

You can use the `tanzu config set PATH VALUE` and `tanzu config unset PATH`
commands to customize your CLI configuration, as described in the table below.

Running these commands updates the ~/.config/tanzu/config.yaml file.

| Path|Value|Description |
|:---------------------:|:------:|:-------:|
| env.VARIABLE | Your variable value; for example, Standard_D2s_v3|This path sets or unsets global environment variables for the Tanzu CLI. For example, `tanzu config set env.AZURE_NODE_MACHINE_TYPE Standard_D2s_v3`. Variables set by running tanzu config set persist until you unset them with `tanzu config unset`; they will be available as regular environment variables to the CLI and plugins that wish to read them.
features.global.FEATURE | true or false | This path activates or deactivates global features in your CLI configuration. Use only if you want to change or restore the defaults. For example, tanzu config set features.global.context-aware-cli-for-plugins true. |
| features.PLUGIN.FEATURE | true or false | This path activates or deactivates plugin-specific features in your CLI configuration. Use only if you want to change or restore the defaults; some of these features are experimental and intended for evaluation and test purposes only. For example, running tanzu config set features.cluster.dual-stack-ipv4-primary true sets the dual-stack-ipv4-primary feature of the cluster CLI plugin to true. By default, only production-ready plugin features are set to true in the CLI. |

### Features

#### To activate a CLI feature

To activate a global feature, run:

`tanzu config set features.global.FEATURE true`
Where FEATURE is the name of the feature that you want to activate.

To activate a plugin feature, run:

`tanzu config set features.PLUGIN.FEATURE true`
Where PLUGIN is the name of the CLI plugin. For example, cluster or
management-cluster. FEATURE is the name of the feature that you want to activate.

#### To deactivate a CLI feature

To deactivate a global feature, run:

`tanzu config set features.global.FEATURE false`
Where FEATURE is the name of the feature that you want to deactivate.

To deactivate a plugin feature, run:

`tanzu config set features.PLUGIN.FEATURE false`
Where PLUGIN is the name of the CLI plugin. For example, cluster or
management-cluster. FEATURE is the name of the feature that you want to deactivate.

## Common plugin commands

There is a small set of commands that every plugin provides. These commands are
typically not invoked directly by CLI users; some are in fact hidden for that
reason. Below is a brief summary of these commands

`version`: provides basic version information about the plugin, likely the only
common command of broad use to the CLI user.

`info`: provides metadata about the plugin that the CLI will use when presenting
information about plugins or when performing lifecycle operations on them.

`post-install`: provide a means for a plugin to optionally implement some logic
to be invoked right after a plugin is installed.

`generate-docs`: generate a tree of documentation markdown files for the
commands the plugin provides, typically used by the CLI's hidden
`generate-all-docs` command to produce command documentation for all installed
plugins.

`lint`: validate the command name and arguments to flag any new terms unaccounted for in the CLI taxonomy document.

More information about these commands are available in the [plugin contract](../plugindev/contract.md) section of the plugin development guide.

## Secure plugin installation

CLI verifies the identity and integrity of the plugin while installing the plugin
from the repository. You can find more details in the
[secure plugin installation proposal document](../proposals/secure-plugin-installation-design.md)

### User experience

CLI verifies [cosign](https://docs.sigstore.dev/cosign/overview/) signature of
the plugin inventory image present in the repository. If the signature
verification is successful, it would download the plugin inventory image on the
user's machine and caches the verified plugin inventory image to improve the
latency on subsequent plugin installation/search commands. If the signature
verification fails, CLI would throw an error and stops continuing.

Signature verification could fail in the scenarios below:

1. Unplanned key rotation: In this case, user either can update to the latest
   CLI version release with the new key, or users should download the new
   public key posted in a well known secure location[TBD] to their local file
   system and export the path of the public key by setting the environment
   variable `TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH`.
2. Repositories without a signature: If users/developers wants to use their own
   repository without the signature for testing, they can skip the
   validation (not recommended in production) by appending the repository URL to
   the comma-separated list in the environment
   variable `TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST`
   . (e.g. to skip signature validation for 2 plugin test repositories:
   `tanzu config set env.TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST
   "test-registry1.harbor.vmware.com/plugins/plugin-inventory:latest,test-registry2.harbor.vmware.com/plugins/plugin-inventory:latest"`)
   .
   After the repository URL is added to skip list, CLI would show warning message
   that signature verification is skipped for the repository. Users can choose to
   suppress this warning by setting the environment variable `TANZU_CLI_SUPPRESS_SKIP_SIGNATURE_VERIFICATION_WARNING`
   to `true`.
