# Using YUM/DNF to install the Tanzu CLI

YUM and DNF (the replacement for YUM) use RPM packages for installation. This
document describes how to build such packages for the Tanzu CLI, how to push
them to a public repository and how to install the CLI from that repository.

## Building the RPM package

Executing the `hack/rpm/build_package.sh` script will build the RPM packages
under `hack/rpm/_output`. The `hack/rpm/build_package.sh` script is meant to
be run on a Linux machine that has `dnf` or `yum` installed.
This can be done in docker using the `fedora` image. To facilitate this
operation, the new `rpm-package` Makefile target has been added; this Makefile
target will first start a docker container and then run the
`hack/rpm/build_package.sh` script.

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

## Publishing the package to GCloud

The GCloud bucket dedicated to hosting the Tanzu CLI OS packages is
gs://tanzu-cli-os-packages`.

To publish the repository containing the new rpm packages for the Tanzu CLI, we
must upload the entire `rpm` directory located at `tanzu-cli/hack/rpm/_output/rpm`
to the root of the bucket.  Note that it is the second `rpm` directory that must be
uploaded. You can do this manually. Once uploaded, the Tanzu CLI
can be installed publicly as described in the next section.

## Installing the Tanzu CLI

```bash
$ docker run --rm -it fedora
cat << EOF | sudo tee /etc/yum.repos.d/tanzu-cli.repo
[tanzu-cli]
name=Tanzu CLI
baseurl=https://storage.googleapis.com/tanzu-cli-os-packages/rpm/tanzu-cli
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.vmware.com/tools/keys/VMWARE-PACKAGING-GPG-RSA-KEY.pub
EOF
yum install -y tanzu-cli
```
