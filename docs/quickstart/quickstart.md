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
Kubernetes endpoint, `mission-control`, which applies to Tanzu Mission
Control service endpoints, and `operations`, to support Kubernetes operations
for Tanzu Application Platform. A plugin can be associated with one or zero targets.
A plugin associated with no target implies its functionality can be used
regardless of what endpoint the CLI may be connecting to.

*Context* : Represents a connection to an endpoint with which the Tanzu CLI can
interact. A context can be established via creating a Tanzu Management
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

If not already done, you must enable autocompletion for your shell environment.
This is independent of the Tanzu CLI autocompletion and relevant instructions
can be found on the internet.

Once general autocompletion is setup and if you have installed the Tanzu CLI
using a package manager, you have nothing more to do: autocompletion will
automatically work for the Tanzu CLI.

If you have not used a package manager to install the CLI, you can find out more
about how to set up autocompletion for your specific shell, by running:

`tanzu completion --help`

For instance, to enable autocompletion for the Tanzu CLI for all new sessions of
zsh:

```console
  ## Load for all new sessions:
  echo "autoload -U compinit; compinit" >> ~/.zshrc
  tanzu completion zsh > "${fpath[1]}/_tanzu"
```

### List plugin groups found in the configured central repository

```console
$ tanzu plugin group search
  GROUP                DESCRIPTION                             LATEST
  vmware-tap/default   Plugins for Tanzu Application Platform  v1.4.0
  vmware-tkg/default   Plugins for Tanzu Kubernetes Grid       v2.1.0
  vmware-tmc/tmc-user  Plugins for Tanzu Mission-Control       v0.0.1
```

### List the plugins of a plugin group

```console
$ tz plugin group get vmware-tkg/default:v2.1.0
Plugins in Group:  vmware-tkg/default:v2.1.0
  NAME                TARGET      LATEST
  cluster             kubernetes  v0.28.0
  feature             kubernetes  v0.28.0
  isolated-cluster    global      v0.28.0
  kubernetes-release  kubernetes  v0.28.0
  management-cluster  kubernetes  v0.28.0
  package             kubernetes  v0.28.0
  pinniped-auth       global      v0.28.0
  secret              kubernetes  v0.28.0
  telemetry           kubernetes  v0.28.0
```

Note that you can omit the version if you are interested in the contents of the latest version of a group:

```console
$ tz plugin group get vmware-tkg/default
Plugins in Group:  vmware-tkg/default:v2.1.0
[...]
```

### Install all plugins in a group

```console
$ tanzu plugin install --group vmware-tkg/default:v2.1.0
[i] Installing plugin 'isolated-cluster:v0.28.0' with target 'global'
[i] Installing plugin 'management-cluster:v0.28.0' with target 'kubernetes'
[i] Installing plugin 'package:v0.28.0' with target 'kubernetes'
[i] Installing plugin 'pinniped-auth:v0.28.0' with target 'global'
[i] Installing plugin 'secret:v0.28.0' with target 'kubernetes'
[i] Installing plugin 'telemetry:v0.28.0' with target 'kubernetes'
[ok] successfully installed all plugins from group 'vmware-tkg/default:v2.1.0'
```

The above command fetches, validates and installs a set of plugins defined by
the `vmware-tkg/default:v2.1.0` group version, which in turn is required for
using the TKG 2.1.0 product.

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

### Logging into Tanzu Application Platform

To log in to the Tanzu Application Platform and create a context of type tanzu, use the following command:

```console
tanzu login
```

This command will log in to the Tanzu Application Platform and creates a context associated
with the organization's name that you are logging in to. Once the context is created,
you can manage it using the `tanzu context` command.

After logging in, you can use the tanzu login command again to update the authentication aspects
of the existing context, while keeping the rest of the context data intact.

#### Tanzu login

User can log in using interactive login (default mechanism) or by utilizing an API Token.

##### Interactive Login

By default, CLI uses interactive login. The CLI opens the browser for the user to log in.
CLI will attempt to log in interactively to the user's default Cloud Services organization. You can override or choose a
custom organization by setting the `TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID` environment variable with the custom
organization ID value. More information regarding organizations in Cloud Services and how to obtain the organization ID
can be
found [here](https://docs.vmware.com/en/VMware-Cloud-services/services/Using-VMware-Cloud-Services/GUID-CF9E9318-B811-48CF-8499-9419997DC1F8.html).

After successful authentication, a context of type "tanzu" is created.

Example command for interactive login:

```console
tanzu login
```

Notes:

- For terminal hosts without a browser, users can set the `TANZU_CLI_OAUTH_LOCAL_LISTENER_PORT` environment variable
  with a chosen port number. Then, run the `tanzu login` command.
  The CLI will show an OAuth URL link in the console and start a local listener on the specified port.
  Users can use SSH port forwarding to forward the port on their machine to the terminal/server machine where the
  local listener is running. Once the port forwarding is initiated, the user can open the OAuth URL in their local
  machine browser to complete the login and create a Tanzu context.

- Alternatively, users can run the `tanzu login` command in the
  terminal host. The CLI will display the OAuth URL link in the console and provide an option for the user to paste
  the Auth code from the browser URL([Image reference](./images/interactive_login_copy_authcode.png)) to the
  console.

##### API Token

Example command to log in using an API token:

```console
TANZU_API_TOKEN=<APIToken> tanzu login
```

Users can persist the environment variable in the CLI configuration file, which will be used for each CLI command
invocation:

```console
tanzu config set env.TANZU_API_TOKEN <api_token>
tanzu login
```

### Creating and connecting to a new context

```console
tanzu context create --kubeconfig cluster.kubeconfig --kubecontext tkg-mgmt-vc-admin@tkg-mgmt-vc --name tkg-mgmt-vc
```

This command will create and associate a Context with some endpoint that the
CLI can target. There are various ways to create Contexts, such as providing an
endpoint to the Tanzu Mission Control service, or providing a kubeconfig to
an existing Tanzu Cluster as shown above.

#### Creating a Tanzu Context

The context of type "tanzu" can be created using interactive login (default mechanism) or by utilizing an API Token

##### Interactive Login

By default, CLI uses interactive login to create a tanzu context. The CLI opens the browser for the user to log in.
CLI will attempt to log in interactively to the user's default Cloud Services organization. You can override or choose a
custom organization by setting the `TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID` environment variable with the custom
organization ID value. More information regarding organizations in Cloud Services and how to obtain the organization ID
can be
found [here](https://docs.vmware.com/en/VMware-Cloud-services/services/Using-VMware-Cloud-Services/GUID-CF9E9318-B811-48CF-8499-9419997DC1F8.html).

After successful authentication, a context of type "tanzu" is created.

Example command for interactive login:

```console
tanzu context create <context-name> --type tanzu --endpoint https://api.tanzu.cloud.vmware.com
```

Notes:

- For terminal hosts without a browser, users can set the `TANZU_CLI_OAUTH_LOCAL_LISTENER_PORT` environment variable
  with a chosen port number. Then, run the `tanzu context create --type tanzu --endpoint <endpoint>` command.
  The CLI will show an OAuth URL link in the console and start a local listener on the specified port.
  Users can use SSH port forwarding to forward the port on their machine to the terminal/server machine where the
  local listener is running. Once the port forwarding is initiated, the user can open the OAuth URL in their local
  machine browser to complete the login and create a Tanzu context.

- Alternatively, users can run the `tanzu context create --type tanzu --endpoint <endpoint>` command in the
  terminal host. The CLI will display the OAuth URL link in the console and provide an option for the user to paste
  the Auth code from the browser URL([Image reference](./images/interactive_login_copy_authcode.png)) to the
  console.

##### API Token

Example command for creating a tanzu context using an API token:

```console
TANZU_API_TOKEN=<APIToken> tanzu context create <context-name> --type tanzu --endpoint https://api.tanzu.cloud.vmware.com
```

Users can persist the environment variable in the CLI configuration file, which will be used for each CLI command
invocation:

```console
tanzu config set env.TANZU_API_TOKEN <api_token>
tanzu context create <context-name> --type tanzu --endpoint https://api.tanzu.cloud.vmware.com
```

To create other context types see the [tanzu context create](../cli/commands/tanzu_context_create.md)
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
  NAME                DESCRIPTION                                                        TARGET      INSTALLED  RECOMMENDED  STATUS
  cluster             Kubernetes cluster operations                                      kubernetes  v0.28.0    v0.28.0      installed
  feature             Operate on features and featuregates                               kubernetes  v0.28.0    v0.28.0      installed
  kubernetes-release  Kubernetes release operations                                      kubernetes  v0.28.0    v0.28.0      installed
  isolated-cluster    Prepopulating images/bundle for internet-restricted environments   global      v0.28.0                 installed
  pinniped-auth       Pinniped authentication operations (usually not directly invoked)  global      v0.28.0                 installed
  management-cluster  Kubernetes management cluster operations                           kubernetes  v0.28.0                 installed
  package             Tanzu package management                                           kubernetes  v0.28.0                 installed
  secret              Tanzu secret management                                            kubernetes  v0.28.0                 installed
  telemetry           configure cluster-wide settings for vmware tanzu telemetry         kubernetes  v0.28.0                 installed

```

The way this type of plugin installation works is that the context you
create or activate can recommend a list of plugins and their versions.
When you create or activate a context, the Tanzu CLI fetches the list of
recommended plugins and automatically tries to install these plugins to
your machine. A `RECOMMENDED` column is shown as part of the
`tanzu plugin list` output that specifies which plugins are recommended
by the active context.

If for some reason the auto-installation of the recommended plugins fails while
creating or activating a context or an existing active context starts recommending
some newer versions of the plugins, you can check that with the `tanzu plugin list`
output with the `STATUS` field mentioning `update needed` and a newer version in
the `RECOMMENDED` column. You can run the `tanzu plugin sync` command
to automatically install the recommended version of the plugins.

To learn more about the plugins installed from context, please refer to
[context recommended plugin installation](../full/context-recommended-plugins.md).

## Notes to users of previous versions of the Tanzu CLI

The Tanzu CLI provided by this project is independently installable as a successor
to the legacy versions of the Tanzu CLI. It can be used to run any version of
CLI plugins that have been released along with those legacy CLI versions thus far,
and thus should be usable as a drop-in replacement of the legacy CLI.
However there are some changes to how plugins are discovered, installed and
updated.

Below is the summary of the changes to expect:

### Deprecated tanzu login Command

The `tanzu login` command has undergone significant changes.

- **Formerly a Plugin:** Initially, this command was provided by the login plugin
  and would appear in the list generated by `tanzu plugin list`. However, it has
  now been transitioned to a core CLI command.

- **Deprecation**: The core `tanzu login` command was deprecated in favor of
  the `tanzu context create` or `tanzu context use` commands for managing contexts.

- **Repurposing:** The deprecated `tanzu login` core command has been repurposed
  to facilitate logging into the Tanzu Application Platform. It is no longer
  considered deprecated.

- **For Legacy CLI Users:** If you still require the use of any legacy Tanzu CLI,
  it is advisable to retain the `login plugin`. Once you no longer need the legacy CLI,
  you can uninstall the plugin using `tanzu plugin uninstall login`.

### tanzu config server

The `tanzu config server` group of commands has been removed in favor of
`tanzu context list` and `tanzu context delete`.

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

### Discovery sources

Existing discovery sources configured using previous versions of the CLI will be
ignored since the default plugin repository will be consulted.

### Using the legacy version of the CLI alongside the new Tanzu CLI

If the user wishes to retain the use of the legacy CLI for whatever reason, it
is the user's responsibility to ensure that the new CLI installation does not
overwrite the existing one, and ensure that the correct binary is used as
needed.

If an existing, older version of the Tanzu CLI is already installed on the
system, existing plugins from previous installation will continue to
be visible to and usable by the new CLI.

However, reinstallation of these plugins using the new CLI will result in plugins
being securely pulled from new default plugin repository.

Should the legacy CLI need to be used for any reason, note that all
functionality available to the legacy CLI will function as before.

Note that running `tanzu plugin sync` with the legacy CLI may undo certain
plugin installation actions performed with the new CLI

#### Limitation for global plugins when using a legacy version with a new version of the CLI

Legacy versions of the CLI cannot invoke a **_global_** plugin that was installed with a new
version of the Tanzu CLI (`>= v0.90.0`).
This implies that if such plugins were installed with a legacy version of the CLI but then
installed again using a new version of the CLI, those plugins will no longer be accessible
through the legacy version of the CLI.

Only two such plugins are both global and applicable to a legacy version of the CLI:

1. `isolated-cluster`
2. `pinniped-auth`

##### Workaround

If a global plugin needs to be used with a legacy version of the CLI, it should only be installed through that legacy CLI.
Those plugin installations can then be used with both legacy and new CLIs, if required.

### Configuring the Registry and Proxy CA certificate

Tanzu CLI will not honor the environment variables `TKG_CUSTOM_IMAGE_REPOSITORY_CA_CERTIFICATE`
and `TKG_PROXY_CA_CERT`.
If the user is interacting with a central repository hosted on a registry with self-signed CA, please
refer to the section `Interacting with a central repository hosted on a registry with self-signed CA or with expired CA`
in [installation doc](install.md)
for configuration steps
