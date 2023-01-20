# Tanzu Core CLI

## Overview

The Tanzu Core CLI project provides the core functionality of the Tanzu CLI.
The CLI is based on a plugin architecture where CLI command functionality can
be delivered through independently developed plugin binaries.  To support this
architecture, this project provides releases of the core CLI binary that
plugins integrate with. Said binary serves the role of

1. providing discovery, installation and lifecycle management of plugins on the CLI host
1. providing dispatching of CLI command invocation to a specific plugin
1. providing authentication with and managing access to endpoints which certain CLI commands will target

## Development

Details about how to get started with development for this project can be found in the development [guide](docs/dev.md).

## Contributing

Thanks for taking the time to join our community and start contributing! We
welcome pull requests. Feel free to dig through the [issues](https://github.com/vmware-tanzu/tanzu-cli/issues) and jump in.

### Before you begin

* Check out the [contribution guidelines](CONTRIBUTING.md) to learn more about how to contribute.
