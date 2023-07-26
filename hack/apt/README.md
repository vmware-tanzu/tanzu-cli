# Using APT to install the Tanzu CLI

APT uses Debian packages for installation. This document describes how to build
such packages for the Tanzu CLI, how to push them to a public repository and
how to install the CLI from that repository.

There are two package names that can be built:

1. "tanzu-cli" for official releases
2. "tanzu-cli-unstable" for pre-releases

The name of the packages built is chosen automatically based on the version
used; a version with a `-` in it is considered a pre-release and will use the
`tanzu-cli-unstable` package name, while other versions will use the
official `tanzu-cli` package name.

## Building the Debian package

Executing the `hack/apt/build_package.sh` script will build the Debian packages
under `hack/apt/_output`. The `hack/apt/build_package.sh` script is meant to
be run on a Linux machine that has `apt` installed. This can be done in
docker.

Then the `hack/apt/build_package_repo.sh` needs to be executed to build the
Debian repository containing the packages.

To facilitate this operation, the `apt-package` Makefile target can be used;
this Makefile target will first start a docker container and then
run the `hack/apt/build_package*.sh` scripts.

The remote location of the existing repository can be overridden by setting
the variable `DEB_METADATA_BASE_URI`.  For example, the default value for
this variable is currently `https://storage.googleapis.com/tanzu-cli-os-packages`

### Pre-requisite

`make cross-build` should be run first to build the `tanzu` binary for linux
in the location expected by the scripts.  To save time, the shorter
`make build-cli-linux-amd64` target can be used.

```bash
cd tanzu-cli
make apt-package
```

Note that two packages will be built, one for AMD64 and one for ARM64. Also, a
repository will be generated as a directory called `_output/apt` which will
contain the two built packages. Please see the section on publishing the
repository for more details.

## Using fedora

The package build scripts support building on a fedora machine.
This can be done using `make apt-package APT_BUILDER_IMAGE=fedora`.

## Testing the installation of the Tanzu CLI locally

We can install the Tanzu CLI using the newly built Debian repository locally on
a Linux machine with `apt` installed or using a docker container. For example:

```bash
$ cd tanzu-cli
$ docker run --rm -it -v $(pwd)/hack/apt/_output/apt:/tmp/apt ubuntu
echo "deb file:///tmp/apt tanzu-cli-jessie main" | tee /etc/apt/sources.list.d/tanzu.list
apt-get update --allow-insecure-repositories
apt install -y tanzu-cli --allow-unauthenticated
tanzu
```

Note that when building locally, the repository isn't signed, so you may see warnings
during installation.

### Testing a pre-release installation

To install a pre-release package, the same procedure must be used as described above
but for the actual installation the command should be:

```bash
apt install -y tanzu-cli-unstable --allow-unauthenticated
```

## Publishing the package to GCloud

The GCloud bucket dedicated to hosting the Tanzu CLI OS packages is
gs://tanzu-cli-os-packages`.

Building the Debian repository incrementally means that we create the
repository metadata for the new package version *and* for any existing packages on
the bucket without downloading the older packages.  This implies that we *must* not
delete the older packages from the bucket but instead we must just upload the
built `hack/apt/_output/apt` on top of the existing bucket's `apt` directory.
This can be done using the `gcloud` CLI:

```bash
gcloud storage cp -r hack/apt/_output/apt gs://tanzu-cli-os-packages
```

This will effectively:

1. upload the new packages to the bucket under `apt/pool/main/t/tanzu-cli/`
2. replace the entire repodata directory located on the bucket at `apt/dists/`

If we want to publish to a brand new bucket, we need to build the repo with
`DEB_METADATA_BASE_URI=new` then upload the entire `apt`
directory located locally at `hack/apt/_output/apt` to the root of the *new* bucket.
Note that it is the second `apt` directory that must be uploaded. You can do this manually.
Once uploaded, the Tanzu CLI can be installed publicly as described in the next section.

The above procedure applies just as much to the `tanzu-cli-unstable` packages.

## Installing the Tanzu CLI

```bash
$ docker run --rm -it ubuntu
apt update
apt install -y ca-certificates curl gpg
mkdir -p /etc/apt/keyrings
curl -fsSL https://packages.vmware.com/tools/keys/VMWARE-PACKAGING-GPG-RSA-KEY.pub | gpg --dearmor -o /etc/apt/keyrings/tanzu-archive-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/tanzu-archive-keyring.gpg] https://storage.googleapis.com/tanzu-cli-os-packages/apt tanzu-cli-jessie main" | tee /etc/apt/sources.list.d/tanzu.list
apt update
apt install -y tanzu-cli
```

### Installing a pre-release

To install a pre-release package, the same procedure must be used as described above
but for the actual installation the command should be:

```bash
apt install -y tanzu-cli-unstable
```
