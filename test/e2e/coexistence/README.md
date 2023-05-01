# Tanzu CLI Coexistence Testing

## Overview

The purpose of this testing is to test the unification of Tanzu CLI to be able to use both new and legacy Tanzu CLI versions.

Tanzu CLI Coexistence testing ensures that both the legacy Tanzu CLI and the new Tanzu CLI can work together. To learn more about this, visit to [Using Legacy Tanzu CLI along with new Tanzu CLI](https://github.com/vmware-tanzu/tanzu-cli/blob/main/docs/quickstart/quickstart.md#using-the-legacy-version-of-the-cli-alongside-the-new-tanzu-cli)

Tanzu CLI Coexistence tests are implemented to test the new Tanzu CLI and legacy Tanzu CLI to interoperate when both CLIs exist

Tests are added using legacy Tanzu CLI v0.28 and new Tanzu CLI latest code.

### How to run the tests ?

 The tests are run in a docker environment

- Install docker on your local machine
- Build the docker image using

```shell
make build-cli-coexistence
```

Run all tests use the below command to run the tests in docker

```shell
make cli-coexistence-tests
```

### Design

- `tanzu` - refers to legacy Tanzu CLI v0.28 and new Tanzu CLI (when in override state i.e. overriding the legacy Tanzu CLI v0.28)
- `tz` - refers to new Tanzu CLI. For testing purpose `tz` is the convention used to disambiguate from the legacy Tanzu CLI `tanzu` when both CLI coexist on the machine.
