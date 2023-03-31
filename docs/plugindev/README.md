# Tanzu CLI Plugin Implementation Guide

## Introduction

The Tanzu CLI was built to be extensible across teams and be cohesive across
SKUs. To this end, the Tanzu CLI provides tools to make creating and compiling
new plugins straightforward.

Before embarking on developing a plugin, the develop should be familiar with
the following materials:

1. The key concepts related to the the Tanzu CLI
2. The CLI plugin architecture
3. The [Tanzu CLI Styleguide](style_guide.md), which describes the user
interaction best practices to be followed. It is highly recommended that anyone
interested in developing a Tanzu CLI plugin be familiarized with the
recommendations.
4. The [build stage](build_stage_styleguide_checklist.md) and [design stage](design_stage_styleguide_checklist.md) style guide checklist are useful resources to refer to so as to maximize UX consistency in the plugin being deveoped.

This document will primarily focus on setting up a development environment to
build and publish plugins.

### Note to developers with existing plugin projects

For plugin projects implemented based on legacy Tanzu CLI codebase, some minor
adjustments will have to be made to account for the new code dependencies and
plugin publishing process. See the [transition guide](migrate.md) for more
details.

## Environment

The Tanzu Plugin Runtime (also referred to as "runtime library" or simply
"runtime") is a library the plugin implementation should integrate with to
implement the [plugin contract](contract.md).

The Tanzu Core CLI and runtime library are written in the Go programming
language. While the plugin architecture tecnically does not prevent the
development of plugins using other programming languages, at the moment the
only supported means of plugin development is that which integrates with a
released version of the runtime library.

The minimum supported Go version is 1.18.

You will need Docker to be installed on your development system.

Note for Mac developers on Apple Silicon machines: Tanzu CLI does not yet
officially support arm64-based binaries.  To develop on these machines ensure
that amd64 versions of the Go toolchain is installed.

------------------------------

### Starting a plugin project

Some CLI functionality essential for plugin development are available as Tanzu
CLI plugins. These are termed "admin plugins" for the rest of the document.

The easiest way to bootstrap a new plugin project is to make use of the
`builder` admin plugin. This plugin provides commands to construct scaffolding
for plugins and commands, and remove much of the need to write boilerplate
code. Use one of the following methods to install the builder plugin.

#### Installing official release of admin plugins

```console
VVV update once a working repo is available
tanzu plugin install --group tzcli/admin:latest
OR
VVV update once a working repo is available
tanzu plugin install builder
```console

VVV test link

For more details on the builder plugin, see the [command reference](../cli/commands/tanzu_builderx.md)

#### Installing local build of admin plugins

```console
#check out the [tanzu-cli](https://github.com/vmware-tanzu/tanzu-cli) repository.

#ensure all the tooling dependencies
make tools

make plugin-build-and-publish-packages
OVERRIDE_INVENTORY_IMAGE=1 make inventory-init && make inventory-plugin-insert
tanzu plugin install builder

tanzu builder --help
```

### Bootstrapping a plugin project

#### 1) create a new plugin repository

```shell
`tanzu builder init <repo-name>`
```

either specify the --repo-type or be prompted to choose between GitLab or
GitHub type repository. The choice will determine the type of skeleton CI
configuration file generated.

#### 2) add main package

```shell
cd <repo-name> && tanzu builder cli add-plugin <plugin-name>
```

will add a `main` package for the new plugin. You should now adjust the
newly created `main` package to implement the functionality of your new plugin.

#### 3) update plugin metadata

You will notice in the generated `main.go` file, that CLI plugins have to instantiate a
[Plugin Descriptor](https://github.com/vmware-tanzu/tanzu-plugin-runtime/blob/main/plugin/types.go#L60)

```go
import (
  "github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

var descriptor = plugin.PluginDescriptor{
    Name:            "helloworld",
    Target:          types.TargetGlobal,
    Description:     "Hello world plugin",
    Group:           plugin.AdminCmdGroup,
    Version:         "v0.0.1",
    BuildSHA:        buildinfo.SHA,
    PostInstallHook: postInstallHook,
}

func main() {
    p, err := plugin.NewPlugin(&descriptor)
    //...
}
```

#### 4) commit the changes to repository

Initialize go module.

```shell
make init
```

Create an initial commit.

```shell
git add -A
git commit -m "Initialize plugin repository"
```

### Building a Plugin

At this point the source repository does not yet have any specific commands
implemented, but yet a fully functional plugin should (with some common
commands included) should already be buildable.

The `builder` plugin also provides functionality to build, install or publish
the plugins. These capabilities are most easily accessed through Makefile
targets already set up for the same purposes.

#### Add your plugin name(s) to the `Makefile`

Add your plugin name to the `PLUGINS` variable in the `Makefile`. This is the
name you used with the `tanzu builder cli add-plugin <plugin-name>` command.

```shell
# Add list of plugins separated by space
PLUGINS ?= "<plugin-name>"
```

#### Build the plugin binary

```sh
make plugin-build-local
```

This will build plugin artifacts under `./artifacts` with plugins organized under: `artifacts/plugins/<OS>/<ARCH>/<TARGET>`

```sh
tanzu plugin install --local ./artifacts/plugins/${HOSTOS}/${HOSTARCH}/<TARGET> [pluginname|all]
```

Your plugin is now available for you through the Tanzu CLI. You can confirm
this by running `tanzu plugin list` which will now show your plugin.

Plugins are installed into `$XDG_DATA_HOME`, (read more about the XDG Base Directory Specification [here.](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)

The next steps are to write the plugin code to implement what the plugin is meant to do.

#### Adding plugin commands

The scaffolded code creates a Plugin object to which additional sub [cobra.Command's](https://pkg.go.dev/github.com/spf13/cobra#Command) can be added to.

#### Tests

Every CLI plugin should have a nested test executable. The executable should
utilize the test framework found in `pkg/v1/test/cli`.

Tests are written to ensure the stability of the commands and are compiled
alongside the plugins. Tests can be run by the admin `test` plugin of the Tanzu
CLI.

#### Docs

Since every plugin is required a README.md document that explains its basic
usage, a basic one is generated in the top-level directory of the repository as
well.

Edit the file as appropriate.

### Publishing a plugin

To publish one or more built plugins to a target repository, one would need to

1. Specify the repository with the env var PLUGIN_PUBLISH_REPOSITORY (the default location is localhost:5001/test/v1/tanzu-cli/plugins), which is where the local test repository is deployed
1. Have push access to said repository location

```sh
make plugin-publish-packages
```

one can also combine the building and publishing of the plugins with

```sh
make plugin-build-and-publish-packages
```

### Updating the plugin inventory of the plugin repository

VVV worth considering providing a single step to e2e build/publish?

```sh
make inventory-init
make inventory-plugin-add
```

will update the inventory database image at the plugin repository with the new plugin entries

### Using the published plugins

Assuming the default location was used in the publishing step.

The following CLI configuration can be set to instruct the CLI to query it

```sh
tanzu config set env.TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY localhost:5001/test/v1/tanzu-cli/plugins/plugin-inventory:latest
```

Once set, plugin lifecycle commands like `tanzu plugin search` will be interact with the test repository as well.

Note: as the configuration variable implies, the
TANZU_CLI_ADDITIONAL_PLUGIN_DISCOVERY_IMAGES_TEST_ONLY setting is meant for
plugin development and testing only. Use of this setting in production setting
is not supported.

------------------------------

## CLI command best practices

### Components

CLI commands should, to the extent possible, utilize the
[Plugin UX component library](https://github.com/vmware-tanzu/tanzu-plugin-runtime/tree/main/component)
for interactive features like prompts or table printing to achieve consistency
across different plugin usage.

### Asynchronous Requests

Commands should be written in such a way as to return as quickly as possible.
When a request is not expected to return immediately, as is often the case with
declarative commands, the command should return immediately with an exit code
indicating the server's response.

The completion notice should include an example of the `get` command the user
would need in order to poll the resource to check the state/status of the
operation.

### Shell Completion

Shell completion (or "command-completion" or "tab completion") is the ability
for the program to automatically fill-in partially typed commands, arguments,
flags and flag values. The Tanzu CLI provides an integrated solution for shell
completion which will automatically take care of completing commands and flags
for your plugin. To make the completions richer, a plugin can add logic to
also provide shell completion for its arguments and flag values; these are
referred to as "custom completions".

Please refer to the Cobra project's documentation on
[Customizing completions](https://github.com/spf13/cobra/blob/main/shell_completions.md#customizing-completions)
to learn how to make your plugin more user-friendly using shell completion.

### Configuration file

The tanzu configuration files reside in
XDG_DATA_HOME/.config/tanzu, which will typically be `~/.config/tanzu/` on most systems.

(See [XDG](https://github.com/adrg/xdg/blob/master/README.md) for more details)

For more details on the APIs available to retrieve or set various CLI configuration settings, refer to
[Plugin UX component library](https://github.com/vmware-tanzu/tanzu-plugin-runtime/tree/main/docs/config.md)

### Other state kept on the CLI machine

Besides XDG_DATA_HOME/.config/tanzu, the following directories are also used
to exclusively store data and artifacts specific to Tanzu CLI:

- [XDG](https://github.com/adrg/xdg/blob/master/README.md)_DATA_HOME/tanzu_cli: where plugins are installed
- _your home directory_/.cache/tanzu: contains the catalog recording information about installed plugins, as well as plugin inventory information.

Cleaning out the above-mentioned directories should restore the CLI to a pristine state.

## Deprecation of existing plugin functionality

It is highly recommended that plugin authors follow the same process used by
the Core CLI to announce and implement deprecation of specific plugin
functionality .

For more details on the deprecation policy and process please refer to the
[Deprecation document](../full/deprecation.md).
