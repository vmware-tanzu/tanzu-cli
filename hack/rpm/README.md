# Using YUM/DNF to install the Tanzu CLI

YUM and DNF (the replacement for YUM) use RPM packages for installation. This
document describes how to build such packages for the Tanzu CLI, how to push
them to a public repository and how to install the CLI from that repository.

There are two package names that can be built by default:

1. "tanzu-cli" for official releases
2. "tanzu-cli-unstable" for pre-releases

The name of the packages built is chosen automatically based on the version
used; a version with a `-` in it is considered a pre-release and will use the
`tanzu-cli-unstable` package name, while other versions will use the
official `tanzu-cli` package name.

It is possible to specify the name of the package by setting the `RPM_PACKAGE_NAME`
environment variable to replace the default name of `tanzu-cli`.  This should not
normally be used as the package name is what the end-users will install and the
default `tanzu-cli` name is the one users are familiar with.

## Building the RPM package

Executing the `hack/rpm/build_package.sh` script will build the RPM packages
under `hack/rpm/_output`. The `hack/rpm/build_package.sh` script is meant to
be run on a Linux machine that has `dnf` or `yum` installed.
This can be done in docker using the `fedora` image.
Once the packages are built, the `hack/rpm/build_package_repo.sh` script should
be invoked to build the repository that will contain the packages.
To facilitate this double operation, the `rpm-package` Makefile target can be used;
this Makefile target will first start a docker container and then run the
appropriate scripts.

The remote location of the existing repository can be overridden by setting
the variable `RPM_METADATA_BASE_URI`.  For example, the default value for
this variable is currently `https://storage.googleapis.com/tanzu-cli-installer-packages`

### Pre-requisite

`make cross-build` should be run first to build the `tanzu` binary for linux
in the location expected by the scripts.  To save time, the shorter
`make build-cli-linux-amd64` target can be used.

```bash
cd tanzu-cli
make rpm-package
```

Note that two packages will be built, one for AMD64 and one for ARM64. Also, a
repository will be generated as a directory called `_output/rpm` which will
contain the two built packages as well as some metadata. Please see the section
on publishing the repository for more details.

## Testing the installation of the Tanzu CLI locally

We can install the Tanzu CLI using the newly built RPM repository locally on a
Linux machine with `yum` or `dnf` installed or using a docker container. For
example, using `yum`:

```bash
$ cd tanzu-cli
$ docker run --rm -it -v $(pwd)/hack/rpm/_output/rpm:/tmp/rpm fedora
cat << EOF | sudo tee /etc/yum.repos.d/tanzu-cli.repo
[tanzu-cli]
name=Tanzu CLI
baseurl=file:///tmp/rpm/tanzu-cli
enabled=1
gpgcheck=0
EOF
yum install -y tanzu-cli
tanzu
```

Note that when building locally, the repository isn't signed, so you may see warnings during installation.

### Testing a pre-release installation

To install a pre-release package, the same procedure must be used as described above
but for the actual installation the command should be:

```bash
yum install -y tanzu-cli-unstable
```

## Publishing the package to GCloud

The GCloud bucket dedicated to hosting the Tanzu CLI OS packages is
gs://tanzu-cli-installer-packages`.

Building the RPM repository incrementally means that we create the
repository metadata for the new package version *and* for any existing packages on
the bucket without downloading the older packages.  This implies that we *must* not
delete the older packages from the bucket but instead we must just upload the
built `hack/rpm/_output/rpm` on top of the existing bucket's `rpm` directory.
This can be done using the `gcloud` CLI:

```bash
gcloud storage cp -r hack/rpm/_output/rpm gs://tanzu-cli-installer-packages
```

This will effectively:

1. upload the new packages to the bucket under `rpm/tanzu-cli/`
2. replace the entire repodata directory located on the bucket at `rpm/tanzu-cli/repodata/`

If we want to publish to a brand new bucket, we need to build the repo with
`RPM_METADATA_BASE_URI=new` then upload the entire `rpm`
directory located locally at `hack/rpm/_output/rpm` to the root of the *new* bucket.
Note that it is the second `rpm` directory that must be uploaded. You can do this manually.
Once uploaded, the Tanzu CLI can be installed publicly as described in the next section.

The above procedure applies just as much to the `tanzu-cli-unstable` packages.

## Installing the Tanzu CLI

```bash
$ docker run --rm -it fedora
cat << EOF | sudo tee /etc/yum.repos.d/tanzu-cli.repo
[tanzu-cli]
name=Tanzu CLI
baseurl=https://storage.googleapis.com/tanzu-cli-installer-packages/rpm/tanzu-cli
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://storage.googleapis.com/tanzu-cli-installer-packages/keys/TANZU-PACKAGING-GPG-RSA-KEY.gpg
EOF
yum install -y tanzu-cli
```

### Installing a pre-release

To install a pre-release package, the same procedure must be used as described above
but for the actual installation the command should be:

```bash
yum install -y tanzu-cli-unstable
```
