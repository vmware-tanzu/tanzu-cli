# Tanzu CLI Quick Start Guide

This guide covers how to quickly get started using the Tanzu CLI.

## Installing Tanzu CLI

This guide assumes that the Tanzu CLI is installed on your system. See
[installation instructions](install.md) to install the CLI if you haven't
already done so.

## Terminology

*Plugin* : An executable binary which packages a group of CLI commands. A basic
unit to extend the functionality of the Tanzu CLI.

*Plugin Group* : Defines a list of plugin/version combinations that are
applicable together. Multiple products and services use the Tanzu CLI to deliver
CLI functionality to their users by requiring a particular set of plugins to be
installed. Plugin groups is the means to facilitate an efficient installation of
such sets of plugins.

*Target* : A class of control-plane endpoints that a plugin may interact with.
The list of targets supported thus far are `kubernetes`, which applies to
Kubernetes clusters, and `mission-control`, which applies to Tanzu Mission
Control service endpoints. A plugin can be associated with one or zero targets.
A plugin associated with no target implies its functionality can be used
regardless of what endpoint the CLI may be connecting to.

*Context* : Represents a connection to an endpoint with which the Tanzu CLI can
interact.  A context can be established via creating a Tanzu Management
Cluster, associating a valid kubeconfig for a cluster with the CLI, or
configuring a connection to a Tanzu Mission Control URL.

## Using the Tanzu CLI

With the CLI installed, a user can choose to install one or more sets of plugins
for the product or service the user wants to interact with using the CLI.
The recommended flow a user will typically go through is described below.

### Install autocompletion scripts for your shell

Command completion support is built into the CLI, but it requires an initial
setup for the command shell in use. The shells with autocompletion support are
`bash`, `zsh`, `fish` and `powershell`.

For instance, to enable autocompletion for the Tanzu CLI for all new sessions of
zsh:

```console
  ## Load for all new sessions:
  echo "autoload -U compinit; compinit" >> ~/.zshrc
  tanzu completion zsh > "${fpath[1]}/_tanzu"
```

To find out more about how to set up autocompletion for your specific shell, run:

`tanzu completion --help`

### List plugin groups found in the local test central repository

```console
$ tanzu plugin group search
  GROUP
  vmware-tap/v1.4.0
  vmware-tkg/v2.1.0
  vmware-tmc/v0.0.1
  vmware-tzcli/admin:v0.90.0
```

### Install all plugins in a group

```console
$ tanzu plugin install all --group vmware-tkg/v2.1.0
[i] Installing plugin 'isolated-cluster:v0.28.0' with target 'global'
[i] Installing plugin 'management-cluster:v0.28.0' with target 'kubernetes'
[i] Installing plugin 'package:v0.28.0' with target 'kubernetes'
[i] Installing plugin 'pinniped-auth:v0.28.0' with target 'global'
[i] Installing plugin 'secret:v0.28.0' with target 'kubernetes'
[i] Installing plugin 'telemetry:v0.28.0' with target 'kubernetes'
[ok] successfully installed all plugins from group 'vmware-tkg/v2.1.0'
```

The above command fetches, validates and installs a set of plugins defined by
the `vmware-tkg/v2.1.0` group, which in turn is required for using the TKG 2.1.0
product.

### Plugins are now installed and available for use

```console
$ tanzu plugin list
Standalone Plugins
  NAME                DESCRIPTION                                                        TARGET      VERSION  STATUS
  isolated-cluster    Prepopulating images/bundle for internet-restricted environments   global      v0.28.0  installed
  pinniped-auth       Pinniped authentication operations (usually not directly invoked)  global      v0.28.0  installed
  management-cluster  Kubernetes management cluster operations                           kubernetes  v0.28.0  installed
  package             Tanzu package management                                           kubernetes  v0.28.0  installed
  secret              Tanzu secret management                                            kubernetes  v0.28.0  installed
  telemetry           configure cluster-wide settings for vmware tanzu telemetry         kubernetes  v0.28.0  installed

$ tanzu package -h
Tanzu package management (available, init, install, installed, release, repository)

Usage:
  tanzu package [flags]
  tanzu package [command]

Available Commands:
  available     Manage available packages (get, list)
  init          Initialize Package (experimental)
  install       Install package
  installed     Manage installed packages (create, delete, get, kick, list, pause, status, update)
  release       Release package (experimental)
  repository    Manage package repositories (add, delete, get, kick, list, release, update)

Flags:
      --column strings              Filter to show only given columns
      --debug                       Include debug output
  -h, --help                        help for package
      --kube-api-burst int          Set Kubernetes API client burst limit (default 1000)
      --kube-api-qps float32        Set Kubernetes API client QPS limit (default 1000)
      --kubeconfig string           Path to the kubeconfig file ($TANZU_KUBECONFIG)
      --kubeconfig-context string   Kubeconfig context override ($TANZU_KUBECONFIG_CONTEXT)
      --kubeconfig-yaml string      Kubeconfig contents as YAML ($TANZU_KUBECONFIG_YAML)
  -y, --yes                         Assume yes for any prompt

Use "tanzu package [command] --help" for more information about a command.
```

### Installing an individual plugin

Individual plugins can also be explicitly installed using:

```console
tanzu plugin install <plugin> [--version <version>] [--target <target>]
```

### Upgrading an individual plugin

```console
tanzu plugin upgrade <plugin> [--target <target>]
```

This command will update the specified plugin to the recommendedVersion
associated with this plugin's entry found in the plugin repository.

### Creating and connecting to a new context

```console
tanzu context create --kubeconfig cluster.kubeconfig --kubecontext tkg-mgmt-vc-admin@tkg-mgmt-vc --name tkg-mgmt-vc
```

This command will create and associate a Context with some endpoint that the
CLI can target. There are various ways to create Contexts, such as providing an
endpoint to the Tanzu Mission Control service, or providing a kubeconfig to
an existing Tanzu Cluster as shown above.

See the [tanzu context create](../cli/commands/tanzu_context_create.md)
reference for more detail.

Other plugins, such as the Tanzu `management-cluster` plugin, can create a
context as part of creating a Tanzu cluster.

### Plugins can also be discovered and installed when connecting to a Context

Switching to another context:

```console
> tanzu context use tkg-mgmt-vc
[i] Checking for required plugins...
[i] Installing plugin 'cluster:v0.28.0' with target 'kubernetes'
[i] Installing plugin 'feature:v0.28.0' with target 'kubernetes'
[i] Installing plugin 'kubernetes-release:v0.28.0' with target 'kubernetes'
[i] Successfully installed all required plugins

> tanzu plugin list
Standalone Plugins
  NAME                DESCRIPTION                                                        TARGET      VERSION  STATUS
  isolated-cluster    Prepopulating images/bundle for internet-restricted environments   global      v0.28.0  installed
  pinniped-auth       Pinniped authentication operations (usually not directly invoked)  global      v0.28.0  installed
  management-cluster  Kubernetes management cluster operations                           kubernetes  v0.28.0  installed
  package             Tanzu package management                                           kubernetes  v0.28.0  installed
  secret              Tanzu secret management                                            kubernetes  v0.28.0  installed
  telemetry           configure cluster-wide settings for vmware tanzu telemetry         kubernetes  v0.28.0  installed

Plugins from Context:  tkg-mgmt-vc
  NAME                DESCRIPTION                           TARGET      VERSION  STATUS
  cluster             Kubernetes cluster operations         kubernetes  v0.28.0  installed
  feature             Operate on features and featuregates  kubernetes  v0.28.0  installed
  kubernetes-release  Kubernetes release operations         kubernetes  v0.28.0  installed
```

To learn more about the plugins installed from context, please refer to
[context-scoped plugin installation](../full/context-scoped-plugins.md).

## Notes to users of previous versions of the Tanzu CLI

The Tanzu CLI provided by this project is independently installable as a successor
to the legacy versions of the Tanzu CLI.  It can be used to run any version of
CLI plugins that have been released along with those legacy CLI versions thus far.
However there are some changes to how plugins are discovered, installed and
updated.

Below is the summary of the changes to expect:

### tanzu plugin sync

Given the every-growing number of discoverable plugins in the default plugin
repository. It is no longer practical to install all of these plugins in a
single sync command.

The `plugin sync` command will now only install plugins as "recommended" by an
active context.

Recall, however, that when the active context changes, all plugin versions
specified by the new context (through the “CLIPlugin” custom-resource or REST
endpoint) will be installed automatically. If one such plugin is already
installed but with a incompatible version, the new version will be installed
instead.

Given that these recommended plugin versions are synched automatically, the
`plugin sync` command will normally not need to be used manually by CLI users.

### tanzu plugin list

`tanzu plugin list` will no longer list all available plugins, but only
installed ones. To find other plugins available for installation, use
`tanzu plugin search` instead.

### discovery sources

Existing discovery sources configured using previous versions of the CLI will be
ignored since the default plugin repository will be consulted.
