# Using Chocolatey to install the Tanzu CLI

This document describes how to build a Chocolatey package for the Tanzu CLI and
how to install it.

There are two package names that can be built:

1. "tanzu-cli" for official releases
2. "tanzu-cli-unstable" for pre-releases

The name of the packages built is chosen automatically based on the version
used; a version with a `-` in it is considered a pre-release and will use the
`tanzu-cli-unstable` package name, while other versions will use the
official `tanzu-cli` package name.

## Building the Chocolatey package

Executing the `hack/choco/build_package.sh` script will build the Chocolatey
package under `hack/choco/_output/choco`.

The `hack/choco/build_package.sh` script is meant to be run on a Linux machine
that has `choco` installed.  This is most easily done using docker. Note that
currently, the docker images for Chocolatey only support an `amd64`
architecture. To facilitate building the package, the new `choco-package`
Makefile target has been added; this Makefile target will first start a docker
container and then run the `hack/choco/build_package.sh` script.

NOTE: This docker image can ONLY be run on an AMD64 machine (chocolatey crashes
when running an AMD64 image on an ARM64 arch).

```bash
cd tanzu-cli
make choco-package
```

Note that the `hack/choco/build_package.sh` script automatically fetches the
required SHA for the CLI binary from the appropriate Github release.  If the
Github release is not public yet, it is possible to provide the SHA manually
through the environment variable `SHA_FOR_CHOCO` as shown below:

```bash
cd tanzu-cli
SHA_FOR_CHOCO=12345678901234567 make choco-package
```

Note: It is not possible to publish the Chocolatey package before the release
is public on github because the publication testing done on the Chocolatey
community repo will fail when trying to install the package.

### Content of Chocolatey package

Currently, we build a Chocolatey package without including the actual Tanzu CLI
binary. Instead, when the package is installed, Chocolatey will download the
CLI binary from Github. This has to do with distribution rights as we will
probably publish the Chocolatey package in the community package repository.

## Testing the built Chocolatey package locally

Installing the Tanzu CLI using the newly built Chocolatey package can be done
on a Windows machine having `choco` installed. First, the Chocolatey package must
be manually uploaded to the Windows machine.

For example, if we upload the package to the Windows machine under
`$HOME\tanzu-cli.0.90.1.nupkg`, we can then simply do:

```bash
choco install -s="$HOME" tanzu-cli
```

It is also possible to configure a local repository containing the local package:

```bash
choco source add -n=local -s="file://$HOME"
choco install -s=local tanzu-cli
```

### Installing a pre-release

To install a pre-release package, the same procedure must be used as described above
but for the actual installation the command should be:

```bash
choco install -s="$HOME" tanzu-cli-unstable --pre
```

or

```bash
choco source add -n=local -s="file://$HOME"
choco install -s=local tanzu-cli-unstable --pre
```

## Uninstalling the Tanzu CLI

To uninstall the Tanzu CLI after it has been installed with Chocolatey:

```bash
choco uninstall tanzu-cli
```

or for pre-releases:

```bash
choco uninstall tanzu-cli-unstable
```

## Publishing the package

The Tanzu CLI Chocolatey package is published to the main Chocolatey
community package repository under the `vmware-tanzu` user account.
This step currently needs to be done manually by running the following
commands.  Note that the docker `chocolatey/choco:v1.4.0` needs to be
run under AMD64 for the `choco` command to work.

```bash
$ docker run --rm -it chocolatey/choco:v1.4.0
curl -sLO <URL of Choco package build>/chocorepo.tgz
tar xf chocorepo.tgz
# For final releases
choco push --source https://push.chocolatey.org/ --api-key <api-key> choco/tanzu-cli.<version>.nupkg
# or for the tanzu-cli-unstable package
choco push --source https://push.chocolatey.org/ --api-key <api-key> choco/tanzu-cli-unstable.<version>.nupkg
```

The package will almost immediately become available for installation using
the `--version` flag.  However, for the package to become public and be installed
by default without the use of the `--version` flag, it may take a couple of days
as the package must be approved by a human maintainer of the Chocolatey project.

Progress can be monitored at the following URL (note that you need to be logged
in as vmware-tanzu):

```bash
https://community.chocolatey.org/profiles/vmware-tanzu
```

## Testing after publication

As soon as the `choco push` command is done, the package should be available
using the `--version` flag.  To install it please refer to
[the installation documentation](../../docs/quickstart/install.md#chocolatey-windows).
