# Tanzu CLI User Guide

This section provides more details on how the Tanzu CLI works, and important
functionality and behaviors that the user should be aware of.

## Important Concepts

## Plugin Architecture

The CLI is based on a plugin mechanism where CLI command functionality can
be delivered through independently developed plugin binaries.

While the CLI provides some core functionality like CLI configuration, a
unified command tree and plugin management, much of its power comes from the
plugins that integrates with it.

## Command Groups and Targets

Tanzu CLI commands are organized into command groups. To list available command
groups, run `tanzu`.

In this example, several command groups are shown, such as cluster, builder, package, etc
Commands within a command group can be explored via `tanzu <command group>`,
and invocable via the tanzu command, e.g.  `tanzu cluster create ....`

The list of available command groups differs based on what Tanzu CLI plugins
are installed on your machine, and on what endpoints the CLI is currently
configured to connect to.

While not always the case, commands falling with a single command group are
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

A Target refers to a category or tier of control planes that the CLI can interact with. There are currently two supported targets : kuberbetes (or k8s) and mission-control (or tmc) which corresponds to Kubernetes cluster endpoint type and Tanzu Mission Control endpoint types. A context is associated with one of the supported targets. Plugins are not necessarily and can often be associated with one of the targets as well. The see plugins that apply to a particular Target and not any others, run `tanzu <target>'.

Similarly, commands from plugins that are associated with a target are unambiguously invoked by prefixing the command group with the target, like so:

```console
tanzu mission-control data-protection ...
or
tanzu k8s management-cluster ...
```

Note that today, the CLI supports omitting the target for historical reasons, but such omission only applies command for the k8s target.  (So `tanzu management-cluster ...` variant of the above example is valid, but not `tanzu data-protection ...`). The CLI team is exploring making the 'assumed target' configurable.

## Interaction between CLI and its plugins

The Core CLI plays several roles in plug architecture:

### Central, consistent point of interaction

All commands provided by the CLI are invocable via the tanzu binary, which in turn dispatches the command to the appropriate plugin, capturing output and errors from the latter in the process to return back to the user.

Commands producing meaningful output consistently provide alternative output such as JSON, YAML or tabular formats using the `-o {json|yaml}` flag.

### Plugin discovery and lifecycle management

The CLI is configured to use a default plugin repository. Through various commands like `tanzu plugin search`, `tanzu plugin install`, `tanzu plugin group install`, the CLI provides various means to discover, then securely install or update plugins to serve specific needs of the user.

Certain context endpoints will require that a specific set of plugins be installed so as enable proper interaction with said endpoints. For these, establish a connection with these endpoints will lead to the discovery and automatic installation of the additional plugins. Plugins installed through this mean of discovery are referred to as "context-scoped" plugins.

For an overview on some of these plugin lifecycle commands, see the [Quickstart Guide](../quickstart/quickstart.md)
For more details on these commands, see the [command reference](../cli/commands/tanzu_plugin.md)

### Context management

The CLI maintains a list of Contexts and an active Context for each Target type. A command from a plugin for use with a particular Target type will always be able to access the Context information necessary to interacted with the endpoint associated with the Context.

## CLI Configuration

The Tanzu CLI configuration, stored in .config/tanzu/ of your home directory,

* Names, contexts, and kubeconfig locations for the servers that the CLI knows about, and which contexts are the active ones
* Global and plugin-specific configuration options, or features
* Sources for CLI plugin discovery

You can use the tanzu config set PATH VALUE and tanzu config unset PATH
commands to customize your CLI configuration, as described in the table below.

Running these commands updates the ~/.config/tanzu/config.yaml file.

| Path|Value|Description |
|:---------------------:|:------:|:-------:|
| env.VARIABLE | Your variable value; for example, Standard_D2s_v3|This path sets or unsets global environment variables for the Tanzu CLI. For example, tanzu config set env.AZURE_NODE_MACHINE_TYPE Standard_D2s_v3. Variables set by running tanzu config set persist until you unset them with tanzu config unset.  For a list of variables that you can set, see Configuration File Variable Reference.
features.global.FEATURE | true or false | This path activates or deactivates global features in your CLI configuration. Use only if you want to change or restore the defaults. For example, tanzu config set features.global.context-aware-cli-for-plugins true. |
| features.PLUGIN.FEATURE | true or false | This path activates or deactivates plugin-specific features in your CLI configuration. Use only if you want to change or restore the defaults; some of these features are experimental and intended for evaluation and test purposes only. For example, running tanzu config set features.cluster.dual-stack-ipv4-primary true sets the dual-stack-ipv4-primary feature of the cluster CLI plugin to true. By default, only production-ready plugin features are set to true in the CLI. |

Features

To activate a CLI feature:

To activate a global feature, run:

tanzu config set features.global.FEATURE true
Where FEATURE is the name of the feature that you want to activate.

To activate a plugin feature, run:

tanzu config set features.PLUGIN.FEATURE true
Where:

PLUGIN is the name of the CLI plugin. For example, cluster or
management-cluster.
FEATURE is the name of the feature that you want to activate.
To deactivate a CLI feature:

To deactivate a global feature, run:

tanzu config set features.global.FEATURE false
Where FEATURE is the name of the feature that you want to deactivate.

To deactivate a plugin feature, run:

tanzu config set features.PLUGIN.FEATURE false
Where:

PLUGIN is the name of the CLI plugin. For example, cluster or
management-cluster.

FEATURE is the name of the feature that you want to deactivate.

### Plugin Discovery Source

VVV This section is mostly incorrect or irrelevant for the user guide.
VVV how much to include, if at all.

Discovery is the interface to fetch the list of available plugins, their
supported versions and how to download them either standalone or scoped to a
context(server). E.g., the CLIPlugin resource in a management cluster, OCI
based plugin discovery for standalone plugins, a similar REST API etc. provides
the list of available plugins and details about the supported versions. Having
a separate interface for discovery helps to decouple discovery (which is
usually tied to a server or user identity) from distribution (which can be
shared).

Plugins can be of two different types:

  1. Standalone plugins: independent of the CLI context and are discovered using standalone discovery source

      This type of plugins are not associated with the `tanzu login` workflow and are available to the Tanzu CLI independent of the CLI context.

  2. Context(server) scoped plugins: scoped to one or more contexts and are discovered using kubernetes or other server associated discovery source

      This type of plugins are associated with the `tanzu login` workflow and are discovered from the management-cluster or global server endpoint.
      In terms of management-clusters, this type of plugins are mostly associated with the installed packages.

      Example:

      As a developer of a `velero` package, I would like to create a Tanzu CLI
plugin that can be used to configure and manage installed `velero` package
configuration.

      This usecase can be handled with context scoped plugins by installing
`CLIPlugin` CR related to `velero` plugin on the management-cluster as part of
`velero` package installation.

      ```sh
      # Login to a management-cluster
      $ tanzu login

      # Installs velero package to the management-cluster along with `velero` CLIPlugin resource
      $ tanzu package install velero-pkg --package-name velero.tanzu.vmware.com

      # Plugin list should show a new `velero` plugin available
      $ tanzu plugin list
        NAME     DESCRIPTION                    SCOPE       DISCOVERY          VERSION    STATUS
        velero   Backup and restore operations  Context     cluster-default    v0.1.0     not installed

      # Install velero plugin
      $ tanzu plugin install velero
      ```

The default standalone plugins discovery source automatically gets added to the tanzu config files and plugins from this discovery source are automatically discovered.

```sh
$ tanzu plugin list
  NAME                DESCRIPTION                                 SCOPE       DISCOVERY             VERSION      STATUS
  login               Login to the platform                       Standalone  default               v0.11.0-dev  not installed
  management-cluster  Kubernetes management-cluster operations    Standalone  default               v0.11.0-dev  not installed
```

To add a plugin discovery source the command `tanzu plugin source add` should
be used. For example, assuming the admin plugin's manifests are released as a
carvel-package at OCI image
`projects.registry.vmware.com/tkg/tanzu-plugins/admin-plugins:v0.11.0-dev` then
we use the following command to add that discovery source to the tanzu
configuration.

```sh
 tanzu plugin source add --name admin --type oci --uri projects.registry.vmware.com/tkg/tanzu-plugins/admin-plugins:v0.11.0-dev
```

We can check the newly added discovery source with

```sh
$ tanzu plugin source list
  NAME     TYPE  SCOPE
  default  oci   Standalone
  admin    oci   Standalone
```

## Common plugin commands

There is a small set of commands that every plugin provides. These commands are
typically not invoked directly by CLI users; some are in fact hidden for that
reason. Below is a brief summary of these commands

`version`: provides basic version information about the plugin, likely the only common command of broad use to the CLI user.

`info`: provides metadata about the plugin that the CLI will use when presenting information about plugins or when performing lifecycle operations on them.

`post-install`: provide a means for a plugin to optionally implement some logic to be invoked right after a plugin is installed.

`generate-docs`: generate a tree of documentation markdown files for the commands the plugin provides, typically used by the CLI's generate-all-docs command to produce command documentation for all installed plugins

`lint`: validate the command name and arguments to flag any new terms unaccounted for in the CLI taxonomy document

More information about these commands are available in the [plugin contract](../plugindev/contract.md) section of the plugin development guide.
