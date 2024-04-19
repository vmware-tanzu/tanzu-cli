# CLI Architecture

This document aims to provide a general overview of the Tanzu CLI architecture.

## Definition

**_Plugin_** - The CLI consists of plugins, each being a cmd developed in Go and conforming to Cobra CLI standard.

**_Context_** - An isolated scope of relevant client-side configurations for a combination of user identity and server identity.

**_Context-Type_** - Type of control plane or cluster or service that the user connects to.  The currently supported context types are: “kubernetes” (e.g., TKG, a vanilla workload cluster, TAP-enabled workload cluster, etc…), “mission-control” (for TMC SaaS endpoints), "tanzu" (for Tanzu control plane endpoint)

**_Target_** - Represents a group of Tanzu CLI Plugins that are of the same category (generally talking to similar endpoints) and are represented as the `tanzu <target-name> <plugin-name>` command.

**_DiscoverySource_** - Represents a group of plugin artifacts and their distribution details that are installable by the Tanzu CLI.

**_Catalog_** - A catalog holds the information of all currently installed plugins on a host OS.

## Plugins

The CLI is based on a plugin architecture. This architecture enables teams to build, own, and release their piece of functionality as well as enable external partners to integrate with the system.

## Plugin Discovery

A plugin discovery points to a group of plugin artifacts that are installable by the Tanzu CLI. It uses an interface to fetch the list of available plugins, their supported versions, and how to download them.

To install a plugin, the CLI uses an OCI discovery source, which contains
the inventory of plugins including metadata for each plugin. This plugin inventory
also includes the location from which a plugin's binary can be downloaded.

More details about the centralized plugin discovery can be found [here](../dev/centralized_plugin_discovery.md).

The `tanzu plugin source` command is used for configuring the discovery sources.

Listing available discovery sources:

```sh
tanzu plugin source list
```

Update a discovery source:

```sh
# Update the discovery source for an air-gapped scenario. The URI must be an OCI image.
tanzu plugin source update default --uri registry.example.com/tanzu/plugin-inventory:latest
```

Sample tanzu configuration file after adding discovery:

```yaml
apiVersion: config.tanzu.vmware.com/v1alpha1
cli:
  ceipOptIn: "false"
  eulaStatus: accepted
  discoverySources:
    - oci:
        name: default
        image: registry.example.com/tanzu/plugin-inventory:latest
```

To list all the available plugins that are getting discovered:

```sh
tanzu plugin search
```

To install a plugin:

```sh
tanzu plugin install <plugin-name>
```

To describe a plugin use:

```sh
tanzu plugin describe <plugin-name>
```

To see specific plugin information:

```sh
tanzu <plugin> info
```

To uninstall a plugin:

```sh
tanzu plugin uninstall <plugin-name>
```

## Context

Context is an isolated scope of relevant client-side configurations for a combination of user identity and server identity.
There can be multiple contexts for the same combination of `(user, server)`.
Going forward we shall refer to them as `Context` to be explicit. Also, the context can be managed in one place using the `tanzu context` command.

- Each `Context` has a type associated with it which is specified with `ContextType`.

Create a new context:

```sh
# Deprecated: Login to the TKG management cluster by using the kubeconfig path and context for the management cluster
tanzu login --kubeconfig path/to/kubeconfig --context context-name --name mgmt-cluster

# New Command
tanzu context create --kubeconfig path/to/kubeconfig --kubecontext context-name --name mgmt-cluster --type kubernetes
```

List known contexts:

```sh
# New Command
tanzu context list
```

Delete a context:

```sh
# New Command
tanzu context delete demo-cluster
```

Use a context:

```sh
# Deprecated
tanzu login mgmt-cluster

# New Command
tanzu context use mgmt-cluster
```

## Context-Type

Context Type represents a type of control plane or service that the user connects to.

The Tanzu CLI supports three context types: `kubernetes` (e.g., TKG, a vanilla workload cluster, TAP-enabled workload cluster, etc), `mission-control` (for TMC SaaS endpoints), and `tanzu` (for Tanzu control plane endpoint).

Plugins use Tanzu Plugin Runtime API to find the kubeconfig or other credentials to connect to the endpoint.

When creating a new context with Tanzu CLI, the user can pass the optional `--type` flag along with the `tanzu context create` command to specify the type of the context.

```sh
# Create a TKG management cluster context using endpoint and type (--type is optional, if not provided the CLI will infer the type from the endpoint)
tanzu context create mgmt-cluster --endpoint https://k8s.example.com[:port] --type k8s

# Create a Tanzu context with the default endpoint (--type is not necessary for the default endpoint)
tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com

# Create a Tanzu context (--type is needed for a non-default endpoint)
tanzu context create mytanzu --endpoint https://non-default.tanzu.endpoint.com --type tanzu
```

To list the available contexts, the user can run the `tanzu context list` command and it will show the following details:

```sh
tanzu context list
  NAME        ISACTIVE  TYPE              ENDPOINT                            KUBECONFIGPATH             KUBECONTEXT            PROJECT  SPACE
  tkg-mc      false     kubernetes                                            /Users/abc/.kube/config
  tmc-ctx-1   false     mission-control   https://tmc.cloud.vmware.com        n/a                        n/a                    n/a      n/a
  tanzu-ctx-1 true      tanzu             https://api.tanzu.cloud.vmware.com  /Users/abc/.kube/config    tanzu-cli-tanzu-ctx-1
```

To make a context active, the user can run the `tanzu context use <context-name>` command.

Only one context of any type can be active at a time. So, if the user runs `tanzu context use tkg-mc`, it will make `tkg-mc` as an active context and mark `tanzu-ctx-1` inactive.
Note: For backward compatibility reasons, one `mission-control` context can be active along with other contexts.

## Target

Target represents a group of Tanzu CLI Plugins that are of the same category (generally talking to similar endpoints) and are represented as the `tanzu <target-name> <plugin-name>` command.
  
The Tanzu CLI supports four targets: `global`, `kubernetes` (alias `k8s`), `mission-control` (alias `tmc`) and `operations` (alias `ops`).

Each plugin is associated with one of the above targets. `global` is a special target that links the plugin under the root tanzu cli command.

Below are some examples of plugin invocation commands formed for different targets:

- PluginName: `foo`, Target: `kubernetes`, Command: `tanzu kubernetes foo`
- PluginName: `bar`, Target: `mission-control`, Command: `tanzu mission-control bar`
- PluginName: `baz`, Target: `global`, Command: `tanzu baz`
- PluginName: `qux`, Target: `operations`, Command: `tanzu operations qux`

For backward compatibility reasons, the plugins with the `kubernetes` target are also available under the root `tanzu` command along with the `tanzu kubernetes` command.

To list TKG workload clusters using the TKG cluster plugin which is associated with the `kubernetes` target:

```sh
# Without target grouping (a TKG management cluster is set as the current active server)
tanzu cluster list

# With target grouping
tanzu kubernetes cluster list
```

To list TMC workload clusters using the TMC cluster plugin which is associated with the `mission-control` target:

```sh
# With target grouping
tanzu mission-control cluster list
```

## Catalog

A catalog holds the information of all currently installed plugins on a host OS. Plugins are currently stored in $XDG_DATA_HOME/tanzu-cli. Plugins are self-describing and every plugin automatically implements a set of default hidden commands.

```sh
tanzu cluster info
```

Will output the descriptor for that plugin in json format, eg:

```json
{"name":"cluster","description":"Kubernetes cluster operations","version":"v0.0.1","buildSHA":"7e9e562-dirty","group":"Run"}
```

The catalog gets built while installing or upgrading any plugins by executing the info command on the binaries.

## Execution

When the root `tanzu` command is executed it gathers the plugin descriptors from the catalog for all the installed plugins and builds cobra commands for each one.

When these plugin-specific commands are invoked, Core CLI simply executes the plugin binary for the associated plugins and passes along stdout/in/err and any environment variables.

## Versioning

By default, versioning is handled by the git tags for the repo in which the plugins are located. Versions can be overridden by setting the version field in the plugin descriptor.

All versions for a given plugin can be found by running:

```sh
tanzu plugin describe <name>
```

When installing or updating plugins a specific version can be supplied:

```sh
tanzu plugin install <name> --version v1.2.3
```

## Groups

With `tanzu --help` command, Plugins are displayed within groups. This enables the user to easily identify what functionality they may be looking for as plugins proliferate.

Currently, updating plugin groups is not available to end users as new groups must be added to Core CLI directly. This was done to improve consistency but may want to be revisited in the future.

## Testing

Every plugin requires a test that the compiler enforces. Plugin tests are a nested binary under the plugin which should implement the test framework.

Plugin tests can be run by installing the admin test plugin, which provides the ability to run tests for any of the currently installed plugins. It will fetch the test binaries for each plugin from its respective repo.

Execute the test plugin:

```sh
tanzu test plugin <name>
```

For more details go to  [Plugin Development Guide](../plugindev/README.md).

## Docs

Every plugin requires a README.md file at the top level of its directory which is enforced by the compiler. This file should serve as a guide for how the plugin is to be used.

In the future, we should have every plugin implement a `docs` command which outputs the generated cobra docs.

## Builder

The builder admin plugin is a means to build Tanzu CLI plugins. Builder provides a set of commands to bootstrap plugin repositories, add commands to them, and compile them into an artifacts directory

Initialize a plugin repo:

```sh
tanzu builder init
```

Add a cli command:

```sh
tanzu builder cli add-plugin <name>
```

## Release

Plugins are first compiled into an artifact directory (local discovery source) using the builder plugin and then pushed up to their production discovery source.

## Default Plugin Commands

All plugins get several commands bundled with the plugin system, to provide a common set of commands:

- _Lint_: Lints the cobra command structure for flag and command names and shortcuts.
- _Docs_: Every plugin gets the ability to generate its cobra command structure.
- _Describe, Info, Version_: Get the basic details about any plugin.
