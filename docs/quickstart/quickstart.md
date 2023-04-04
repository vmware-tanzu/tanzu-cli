# Tanzu CLI Quick Start Guide

This guide covers how to quickly get started using the Tanzu CLI.

## Installing Tanzu CLI

The guide assumes that the Tanzu CLI is installed on your system. See
[installation instructions](install.md) to install the CLI if you haven't
already done so.

## Terminology

*Plugin* : An executable binary which packages a group of CLI commands. A basic
unit to extend the functionality of the Tanzu CLI.

*Plugin Group* : Defines a list of plugin/version combinations that are
applicable together. Multiple products and services uses the Tanzu CLI deliver
CLI functionality to their users by requiring a particular set of plugins to be
installed. Plugin groups is the means to facilate efficient installation of
such sets of plugins.

*Target* : A class of control-plane endpoints that a plugin may interact with.
The list of targets supported thus far are `kubernetes`, which applies to
Kubernetes clusters, and `mission-control`, which applies to Tanzu Mission
Control service endpoints. A plugin can be associated with one or zero targets.
A plugin associated with no target implies its functionality can be used
regardless of what endpoint the CLI may be connecting to.

*Context* : Represents a connection to a an endpoint at which the Tanzu CLI can
interact with.  A context can be established via creating a Tanzu Management
Cluster, associating a valid kubeconfig for a cluster with the CLI, or
configuring a connection to a Tanzu Mission Control URL.

## Using Tanzu CLI

With the CLI installed, user can choose to install one or more sets of plugins
for the product or service the user wants to interact with using the CLI

### Install autocompletion scripts for your shell

Command completion support is built into the CLI, but it requires an initial
setup for the command shell in use. The shells with autocompletion support art
bash, zsh, fish and powershell.

For instance, to enable autocompletion for Tanzu CLI for all new sessions of
zsh

```console
  ## Load for all new sessions:
  echo "autoload -U compinit; compinit" >> ~/.zshrc
  tanzu completion zsh > "${fpath[1]}/_tanzu"
```

To find out more about how to set up autocompletion for your specific shell, run:

`tanzu completion --help`

### List plugin groups found in the local test central repo

```console
$ tanzu plugin group search
  GROUP
  vmware-tkg/v1.0.0
  vmware-tkg/v2.1.0
  vmware-tmc/v1.2.3
  vmware-tmc/v9.0.0
```

### Install all plugins in the group

```console
$ tanzu plugin install all --group vmware-tkg/v2.1.0
ℹ  Installing plugin 'cluster:v2.1.0' with target 'kubernetes'
ℹ  Installing plugin 'feature:v0.9.0' with target 'kubernetes'
ℹ  Installing plugin 'kubernetes-release:v2.1.0' with target 'kubernetes'
ℹ  Installing plugin 'management-cluster:v2.1.0' with target 'kubernetes'
ℹ  Installing plugin 'package:v1.2.3' with target 'kubernetes'
ℹ  Installing plugin 'secret:v1.2.3' with target 'kubernetes'
✔  successfully installed all plugins from group 'vmware-tkg/v2.1.0'
```

The above command fetches, validates and installs a set of plugins defined by
the vmware-tkg/v2.1.0 group, which in turn is required for using the TKG 2.1.0
product.

### Plugins are now installed and available for use

```console
$ tanzu plugin list
Installed Plugins
  NAME                DESCRIPTION                       TARGET      VERSION  STATUS
  cluster             cluster functionality             kubernetes  v2.1.0   installed
  feature             feature functionality             kubernetes  v0.9.0   installed
  kubernetes-release  kubernetes-release functionality  kubernetes  v2.1.0   installed
  management-cluster  management-cluster functionality  kubernetes  v2.1.0   installed
  package             package functionality             kubernetes  v1.2.3   installed
  secret              secret functionality              kubernetes  v1.2.3   installed

$ tanzu package -h
Tanzu package management

Usage:
  tanzu package [command]

Available Commands:
  available     Manage available packages
  install       Install a package
  installed     Manage installed packages
  repository    Repository operations

Flags:
  -h, --help                help for package
      --kubeconfig string   The path to the kubeconfig file, optional
      --verbose int32       Number for the log level verbosity(0-9)

Use "tanzu package [command] --help" for more information about a command.
```

### Installing an individual plugin

Individual plugins can also be explicitly installed as well using:

```console
tanzu plugin install <plugin> [--version <version>] [--target <target>]
```

### Upgrading an individual plugin

```console
tanzu plugin upgrade <plugin> [--target <target>]
```

This command will update the specified plugin to the version specified by the
context, if any, or else to the recommendedVersion associated with this plugin
entry found in the plugin repository.

### Creating and connecting to a new context

```console
tanzu context create --kubeconfig cluster.kubeconfig --kubecontext tkg-mgmt-vc-admin@tkg-mgmt-vc tkg-mgmt-vc
```

This command will create and associate a Context with some endpoint that the CLI can target. There are various ways to create Contexts, such as providing and endpoint to the Tanzu Mission Control service, or providing a kubeconfig to an existing Tanzu Cluster as shown above.

Other plugins, such as the Tanzu `management-cluster` plugin, can create a context as part of creating a Tanzu cluster.

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
  NAME     DESCRIPTION             TARGET  VERSION                  STATUS
  builder  Build Tanzu components  global  v0.1.0-dev-18-g668cb41c  installed
  test     Test the CLI            global  v0.1.0-dev-18-g668cb41c  installed

Plugins from Context:  tkg-mgmt-vc
  NAME                DESCRIPTION                           TARGET      VERSION  STATUS
  cluster             Kubernetes cluster operations         kubernetes  v0.28.0  installed
  feature             Operate on features and featuregates  kubernetes  v0.28.0  installed
  kubernetes-release  Kubernetes release operations         kubernetes  v0.28.0  installed
```

To learn more about the plugins from context, please check [context-scoped plugin installation](../full/context-scoped-plugins.md).

## Notes to users of previous versions of the Tanzu CLI

The Tanzu CLI provided by this project is independently installable successor
to the legacy versions of the Tanzu CLI.  It can be used to run any version of
CLI plugins that have been released along with those CLI versions thus far.
However there are some changes to how plugins are discovered, installed and
updated.

Below is the summary of the changes to expect:

### tanzu plugin sync

Given the every-growing number of discoverable plugins in the default plugin
repository. It is no longer practical to install all of these plugins in a
single sync command.

This command will now only install plugins as "recommended" by an active
context.

Recall, however, that when the active context changes, all plugin versions
specified by the context (through the “CLIPlugin” custom-resource or REST
endpoint) will be installed automatically. If one such plugin is already
installed but with the incompatible version, the new version will be installed
instead.

Given that these recommended  plugins are synched automatically, the `sync`
command will not normally need to be used manually by CLI users.
