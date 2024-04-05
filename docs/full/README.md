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

A Target refers to a category of commands or tier of control planes that the CLI
can interact with. There are currently three supported targets : `kubernetes` (or
`k8s`), `mission-control` (or `tmc`) and `operations` (or `ops`). Plugins are
generally associated with one of the above mentioned targets but if a plugin
doesn't fall into any of the above categories a developer can create a plugin
with the `global` target. A plugin using the `global` target is available as a
root Tanzu CLI sub-command. To see plugins that only apply to a specific target,
run the command `tanzu <target>'.

Similarly, commands from plugins that are associated with a target are
unambiguously invoked by prefixing the command group with the target, like so:

```console
tanzu mission-control data-protection ...
or
tanzu k8s management-cluster ...
or
tanzu operations clustergroup ...
```

Note that today, the CLI supports omitting the target for historical reasons,
but this omission only applies to commands for the `k8s` target.
(So `tanzu management-cluster ...` is a valid variant of the above example, but not
`tanzu data-protection ...` or `tanzu clustergroup`).

Note also that until a plugin associated with a target is installed, the target in
question will be hidden from the user.  For example, if no `tmc`/`mission-control`
plugins are installed, the `tanzu tmc`/`tanzu mission-control` sub-command will not
be shown to the user in the help.

## Interaction between CLI and its plugins

The Core CLI plays several roles in the plugin architecture:

### Central, consistent point of interaction

All commands provided by the CLI are invocable via the `tanzu` binary, which in turn dispatches the command to the appropriate plugin, capturing output and errors from the latter to return back to the user.

Commands producing meaningful output consistently provide alternative output such as JSON, YAML or tabular formats using the `-o {json|yaml}` flag.

### Plugin discovery and lifecycle management

The CLI is configured to use a default plugin repository. Through various commands like `tanzu plugin search`, `tanzu plugin install`, `tanzu plugin install --group <groupName>:<groupVersion>`, the CLI provides various means to discover, then securely install or update plugins to serve specific needs of the user.

Certain context endpoints will require that a specific set of plugins be installed so as to enable proper interaction with said endpoints. For these, establishing a connection with these endpoints may lead to the discovery and automatic installation of additional plugins. Plugins installed through this mean of discovery are referred to as "context-scoped" plugins. To learn more about the context-scoped plugins, please check the [context-scoped plugin installation](../full/context-scoped-plugins.md) documentation.

For an overview on some of these plugin lifecycle commands, see the [Quickstart Guide](../quickstart/quickstart.md).
For more details on these commands, see the [command reference](../cli/commands/tanzu_plugin.md).

### Context management

The CLI maintains a list of Contexts and an active Context for each Target type. A plugin command with a particular Target type will always be able to access the active context information by using the APIs exposed by the `tanzu-plugin-runtime` library. This will allow plugins to interact with the endpoint associated with the Context.

## CLI Configuration

The Tanzu CLI configuration is stored in `.config/tanzu/` of your home directory. It contains:

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

### Environment variables affecting the CLI

Some options affecting the CLI are only available through the use of environment
variables.  Below is the list of such variables:

| Environment Variable | Description | Value |
| -------------------- | ----------- | ----- |
| `NO_COLOR` | Turns off color and special formatting in CLI output. May affect other programs. | Any value to activate, `""` or unset to deactivate |
| `PROXY_CA_CERT` | Custom CA certificate for a proxy that needs to be used by the CLI. | Base64 value of the proxy CA certificate |
| `TANZU_ACTIVE_HELP` | Deactivate some ActiveHelp messages. | `0` to deactivate all ActiveHelp messages, `no_short_help` to deactivate the short help string from ActiveHelp, `""` or unset to allow all ActiveHelp messages |
| `TANZU_API_TOKEN` | Specifies the token to be used for the creation of a Tanzu context. If not used, the CLI will attempt to log in interactively using a browser. Also used to specify the token for the creation of TMC contexts. Note that a Tanzu token and a TMC token are not the same value. | Token string |
| `TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER` | Automatically answer the Customer Experience Improvement Program (ceip) prompt. | `Yes` to agree to participate, `No` to decline |
| `TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID` | Specifies the Cloud Services organization to use for the interactive login during the creation of a Tanzu context. | Organization ID string |
| `TANZU_CLI_EULA_PROMPT_ANSWER` | Automatically answer the End User License Agreement prompt. | `Yes` to agree to the terms, `No` to decline |
| `TANZU_CLI_LOG_LEVEL`  | Used to increase the amount of logging during troubleshooting.  This variable is not yet respected by plugins but is respected by the CLI core commands. | `0` to `9` |
| `TANZU_CLI_NO_COLOR` | Turns off color and special formatting in CLI output.  This variable is not respected by all plugins and `NO_COLOR` is currently preferred. | Any value to activate, `""` or unset to deactivate |
| `TANZU_CLI_OAUTH_LOCAL_LISTENER_PORT` | For hosts without a browser, this variable can be used to specify a port to use for a local listener automatically started by the CLI. Users can use SSH port forwarding to forward the port on their own machine to the port of the local listener.  This will allow using the browser of the user's machine. | An unused TCP port number |
| `TANZU_CLI_PINNIPED_AUTH_LOGIN_SKIP_BROWSER` | If set to any value, the browser will not be used when pinniped authentication is triggered. | Any value to activate, `""` or unset to deactivate |
| `TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH` | Override the plugin inventory verification key. Should not be necessary. Will only be used in the very rare case of a change of signature keys which will be specified clearly in the documentation. | The replacement public key provided by VMware |
| `TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST` | Used to skip signature verification of custom discovery URIs when doing plugin discovery/installation.  Its use could put your environment at risk. | Comma-separated list of plugin discovery URIs that should not be verified |
| `TANZU_CLI_PRIVATE_PLUGIN_DISCOVERY_IMAGES` | Deprecated. Specifies private plugin repositories to use as a supplement to the production Central Repository of plugins. | Comma-separated list of private plugin repository URIs |
| `TANZU_CLI_RECOMMEND_VERSION_DELAY_DAYS` | Override the default delay (24 hours) between notifications that a new CLI version is available for upgrade (available since CLI v1.3.0). | Delay in days |
| `TANZU_CLI_SHOW_TELEMETRY_CONSOLE_LOGS` | Print telemetry logs (defaults to off). | `1` or `true` to print, `0`, `false`, `""` or unset not to print |
| `TANZU_CLI_SKIP_UPDATE_KUBECONFIG_ON_CONTEXT_USE` | Do not synchronize the active Kubernetes context when the Tanzu context is changed. | `1` or `true` to skip, `0`, `false`, `""` or unset to do the synchronization |
| `TANZU_CLI_SUPPRESS_SKIP_SIGNATURE_VERIFICATION_WARNING` | Suppress the warning message that some plugin discoveries are not being verified due to the use of `TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_ SIGNATURE_VERIFICATION_SKIP_LIST`.  The use of this variable should be avoided as it can put your environment at risk. | `1`, `true` to suppress, `0`, `false`, `""` or unset to allow the message |
| `TANZU_ENDPOINT` | Specifies the endpoint to login into for the `login` command when the `--server` and `--endpoint` flags are not specified. | Endpoint URI |

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

CLI verifies [cosign](https://docs.sigstore.dev/signing/quickstart/) signature of
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

## Autocompletion Support

The Tanzu CLI supports shell autocompletion for the `bash`, `zsh`, `fish` and `powershell` shells.
Please refer to the [autocompletion quickstart section](../quickstart/quickstart.md#install-autocompletion-scripts-for-your-shell)
to setup autocompletion for your shell.

### ActiveHelp Support

ActiveHelp are messages printed through autocompletions as the program is being used.  Once autocompletion has been set up,
the Tanzu CLI automatically provides ActiveHelp messages to improve the user-experience when the user presses `TAB`.
An example of ActiveHelp messages can be seen below:

```console
bash-5.1$ tanzu context create [tab]
Command help: Create a Tanzu CLI context
Please specify a name for the context
```

To deactivate all ActiveHelp messages you can set the `TANZU_ACTIVE_HELP` environment variable to `0`.

**Note: ActiveHelp messages will only be shown when using the `bash` or `zsh` shells.**

#### Known Issues

When using the `bash` shell, if the `tanzu` command is preceded by some input such as setting a variable, there is an issue
re-printing that extra input when ActiveHelp messages are triggered.  For example:

```bash
bash-5.1$ TANZU_API_TOKEN=12345 tanzu context <TAB>
Command help: Configure and manage contexts for the Tanzu CLI

bash-5.1$ tanzu context
```

Notice that the input `TANZU_API_TOKEN=12345` preceding the `tanzu` command is not re-printed, however
it is still present and will take effect if the command is executed.  Pressing `<TAB>` again will often
correct the situation but when it does not, you can simply refresh your shell display (e.g., `^L`).

## Auto-detection and notification of new CLI releases

The CLI will periodically verify if new releases of the CLI itself are available.
The CLI will print a notification to the user to indicate whenever a new minor or patch release
is available to be installed.  This notification will be printed at most once a day until the
CLI is upgraded.  The interval between such notifications can be changed by setting the
`TANZU_CLI_RECOMMEND_VERSION_DELAY_DAYS` variable to the desired amount of days.  Setting this
variable to `0` will turn off such notifications.

Note that special consideration must be given for this feature to work in an internet-restricted environment.
Please refer to [this section](../quickstart/install.md#updating-the-central-configuration) of the documentation.
