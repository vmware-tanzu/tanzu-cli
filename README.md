# Tanzu Core CLI

:warning: NOTE: This repository is still under early development. We plan to
make releases available for evaluation in the second quarter of 2023.  Please
watch this page for further updates.

## Overview

VVV: diagram maybe?

The Tanzu CLI provides integrated and unified command-line access to a broad
array of [products and solutions](https://tanzu.vmware.com/get-started) in the
[VMware Tanzu](https://tanzu.vmware.com/) portfolio.
The CLI is based on a plugin architecture where CLI command functionality can
be delivered through independently developed plugin binaries. To support this
architecture, this project provides releases of the core CLI binary that
plugins integrate with. Said binary serves the role of

1. providing discovery, installation and lifecycle management of plugins on the CLI host
1. providing dispatching of CLI command invocation to a specific plugin
1. providing authentication with and managing access to endpoints which certain CLI commands will target

To facilitate plugin development, the Core CLI also provides

1. the ability to scaffold new plugin projects and plugin commands themselves.
1. the capability to build, test, and publish the plugins being developed.

## Installation

For information on how to install the CLI, see the [Installation Guide](docs/quickstart/install.md)

## Documentation

To get a quick start on how to use Tanzu CLI, visit the
[Quick Start guide](docs/quickstart/quickstart.md) or visit the
[Full Documentation](docs/full/main.md) for more details.

## Plugin Development

To learn more about how to develop a Tanzu CLI plugin, see the
[Tanzu plugin development guide](docs/plugindev/main.md).

## Development

Details about how to get started with development for this project can be found
in the [Development Guide](docs/dev/main.md).

## Contributing

Thanks for taking the time to join our community and start contributing! We
welcome pull requests. Feel free to dig through the
[issues](https://github.com/vmware-tanzu/tanzu-cli/issues) and jump in.

### Before you begin

* Check out the [contribution guidelines](CONTRIBUTING.md) to learn more about how to contribute.
