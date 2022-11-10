# Tanzu Core CLI

## Overview

The Tanzu Core CLI project provides the core functionality of the Tanzu CLI.
The CLI is based on a plugin architecture where CLI command functionality can
be delivered through independently developed plugin binaries.  To support this
architecture, this project providing releases of the core CLI binary that
plugins integrate with. Said binary serves the role of

1. providing discovery, installation and lifecycle management of plugins on the CLI host
1. providing dispatching of CLI command invocation to a specific plugin
1. authentication with and access to endpoints at which certain CLI commands will target
