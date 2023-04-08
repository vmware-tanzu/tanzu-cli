# Tanzu Core CLI

[![Tanzu Core CLI Tests](https://github.com/vmware-tanzu/tanzu-cli/actions/workflows/cli_core_e2e_test.yaml/badge.svg?branch=main&event=push)](https://github.com/vmware-tanzu/tanzu-cli/actions/workflows/cli_core_e2e_test.yaml?query=event:push+branch:main)

:warning: NOTE: This repository is still under early development. We plan to
make releases available for evaluation in the second quarter of 2023. Please
watch this page for further updates.

## Overview

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
[Full Documentation](docs/full/README.md) for more details.

_For users of previous version of the CLI, see
[this section](docs/quickstart/quickstart.md#notes-to-users-of-previous-versions-of-the-tanzu-cli)
of the Quick Start guide for changes to be aware of._

## Plugin Development

To learn more about how to develop a Tanzu CLI plugin, see the
[Tanzu plugin development guide](docs/plugindev/README.md).

### Testing

Plugin developers can use the End-to-End test framework to implement
tests for the functionality of their plugins under a Tanzu CLI installation.
More details found in the
[End-to-End framework and test case implementation](test/e2e/README.md).

## Contributing

Thanks for taking the time to join our community and start contributing! We
welcome pull requests. Feel free to dig through the
[issues](https://github.com/vmware-tanzu/tanzu-cli/issues) and jump in.

### Before you begin

- Check out the [contribution guidelines](CONTRIBUTING.md) to learn more about how to contribute.
- Check out the document [here](docs/community/support.md) about the project's support process.

## Development

Details about how to get started with development for this project can be found
in the [Development Guide](docs/dev/README.md).

## Testing

Unit and Integration tests implementation is part of CLI Core development.
CLI core does have end-to-end test case implementation.
More details found in the [End-to-End framework and test case implementation](test/e2e/README.md).
