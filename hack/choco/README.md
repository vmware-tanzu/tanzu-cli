# Using Chocolatey to install the Tanzu CLI

This document describes how to build a Chocolatey package for the Tanzu CLI and
how to install it.

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

## Installing the Tanzu CLI using the built Chocolatey package

Installing the Tanzu CLI using the newly built Chocolatey package can be done
on a Windows machine having `choco` installed. First, the Chocolatey package must
be uploaded to the Windows machine.

For example, if we upload the package to the Windows machine under
`$HOME\tanzu-cli.0.90.0-beta0.nupkg`, we can then simply do:

```bash
choco install -f "$HOME\tanzu-cli.0.90.0-beta0.nupkg"
```

It is also possible to configure a local repository containing the local package:

```bash
choco source add -n=local -s="file://$HOME"
choco install tanzu-cli
```

## Uninstalling the Tanzu CLI

To uninstall the Tanzu CLI after it has been installed with Chocolatey:

```bash
choco uninstall tanzu-cli
```

## Publishing the package

The Tanzu CLI Chocolatey package is published to the main Chocolatey
community package repository under the `vmware-tanzu` user account.
This step currently needs to be done manually by running the command:

```bash
choco push --source https://push.chocolatey.org/ --api-key <api-key> hack/choco/_output/choco/tanzu-cli.<version>.nupkg
```

The result of the publication can take a couple of hours as tests are run
on the community repo before the package becomes public.  Progress can be
monitored at the following URL (note that you need to be logged in as vmware-tanzu):

```bash
https://community.chocolatey.org/profiles/vmware-tanzu
```

Once the publication is triggered, it seems to take 1 to 2 hours for the package to pass all tests and become available.
