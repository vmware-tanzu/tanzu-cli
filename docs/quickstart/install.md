# Installing the Tanzu CLI

The Tanzu CLI can be installed either from pre-built binary releases from
its GitHub project, or via popular package managers like Homebrew, apt and yum/dnf.

_Note: The following instructions assume the user does not have any legacy
version of the Tanzu CLI installed. The use of package managers for
installation typically means they would manage the installation of the CLI
binary to a path specific to the package managers. There is a chance in some
cases that the path of the binary could conflict with an already installed
legacy `tanzu` binary of the same name. Hence, should the user wish to retain
the use of the legacy CLI, the user should take the necessary steps to maintain
a separate copy of the binary and adjust the paths to these multiple binaries
accordingly._

## From the Binary Releases in the GitHub project

Every [release](https://github.com/vmware-tanzu/tanzu-cli/releases) of the
Tanzu CLI provides separate binary releases for a variety of OS's and machine
architectures. You only need to install the CLI itself. Note that the other
assets present in each release provide access to the administrative plugins;
however, the CLI can install these plugins directly without having to manually
download these particular assets, which are only provided for convenience.

1. Download the [desired version](https://github.com/vmware-tanzu/tanzu-cli/releases). We recommend picking the latest release of the major version of the CLI you want to use. You should choose the asset of the form `tanzu-cli-OS-ARCH.tar.gz`.
2. Unpack it (e.g. `tar -zxvf tanzu-cli-darwin-amd64.tar.gz`)
3. Find the `tanzu` binary in the unpacked directory, move it to its desired location in your $PATH
   destination (`mv v0.90.0/tanzu-cli-darwin_amd64 /usr/local/bin/tanzu`), provide it executable permission (`chmod u+x /usr/local/bin/tanzu`) if necessary.
4. Verify the correct version of CLI is properly installed: `tanzu version`

To uninstall the binary: `rm /usr/local/bin/tanzu`

## Via Package Managers

A recent, supported version of the Tanzu CLI is also available for install
through the following package managers:

### Homebrew (MacOS)

```console
brew update
brew install vmware-tanzu/tanzu/tanzu-cli
```

To uninstall: `brew uninstall tanzu-cli`

#### Installing a Specific Version

At the time of writing, Homebrew only officially supported installing the
latest version of a formula, however the following workaround allows to install
a specific version by first extracting it to a local tap:

```console
brew tap-new local/tap
brew extract --version=0.90.1 vmware-tanzu/tanzu/tanzu-cli local/tap
brew install tanzu-cli@0.90.1

# To uninstall such an installation
brew uninstall tanzu-cli@0.90.1
```

#### Installing a Pre-Release

Pre-releases of the Tanzu CLI are made available to get early feedback before
a new version is released.  Pre-releases are available through Homebrew
using a different package name: `tanzu-cli-unstable`.

**Note**: Just like installing a new version, installing a pre-release will
replace the `tanzu` binary of any previous installation.

```console
brew update
brew install vmware-tanzu/tanzu/tanzu-cli-unstable --overwrite

# To uninstall such an installation
brew uninstall tanzu-cli-unstable
```

### Chocolatey (Windows)

```console
choco install tanzu-cli
```

Note that the Chocolatey package is part of the main
[Chocolatey community repository](https://community.chocolatey.org/packages).
This means that when a new `tanzu-cli` version is released, the chocolatey package may
not be available immediately as it needs to be approved by a Chocolatey maintainer.
If the above installation instructions install an older version of the Tanzu CLI,
you can explicitly specify the version you want to install using the `--version` flag:

```console
choco install tanzu-cli --version <version>
# example: choco install tanzu-cli --version 0.90.0
```

To uninstall: `choco uninstall tanzu-cli`

#### Installing a Specific Version

You can also use the `--version` flag to install any specific version of the Tanzu CLI.

```console
choco install tanzu-cli --version <version>
```

To uninstall: `choco uninstall tanzu-cli`

#### Installing a Pre-Release

Pre-releases of the Tanzu CLI are made available to get early feedback before
a new version is released.  Pre-releases are available through Chocolatey
using a different package name: `tanzu-cli-unstable`.  Notice also the need to use
the `--pre` flag.

**Note**: Just like installing a new version, installing a pre-release will
replace the `tanzu.exe` binary of any previous installation.

```console
choco install tanzu-cli-unstable --pre
```

To uninstall: `choco uninstall tanzu-cli-unstable`

### Apt (Debian/Ubuntu)

```console
sudo apt update
sudo apt install -y ca-certificates curl gpg
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://packages.vmware.com/tools/keys/VMWARE-PACKAGING-GPG-RSA-KEY.pub | sudo gpg --dearmor -o /etc/apt/keyrings/tanzu-archive-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/tanzu-archive-keyring.gpg] https://storage.googleapis.com/tanzu-cli-os-packages/apt tanzu-cli-jessie main" | sudo tee /etc/apt/sources.list.d/tanzu.list
sudo apt update
sudo apt install -y tanzu-cli
```

To uninstall: `sudo apt remove tanzu-cli`

#### Installing a Specific Version

To install a specific version of the Tanzu CLI you can specify the version explicitly:

```console
# To list the available versions
sudo apt list tanzu-cli -a

# To install a specific version (notice the '=' before the version)
sudo apt install tanzu-cli=0.90.1
```

To uninstall: `sudo apt remove tanzu-cli`

#### Installing a Pre-Release

Pre-releases of the Tanzu CLI are made available to get early feedback before
a new version is released.  Pre-releases are available through `apt` using the
same repository configuration steps described above, but the installation
command uses a different package name: `tanzu-cli-unstable`.

**Note**: Just like installing a new version, installing a pre-release will
replace the `tanzu` binary of any previous installation.

```console
# Remove any installed tanzu cli binary
sudo apt remove tanzu-cli

# First setup the repository if it is not setup already, then run:
sudo apt install tanzu-cli-unstable
```

To uninstall: `sudo apt remove tanzu-cli-unstable`

### From yum/dnf (RHEL)

```console
cat << EOF | sudo tee /etc/yum.repos.d/tanzu-cli.repo
[tanzu-cli]
name=Tanzu CLI
baseurl=https://storage.googleapis.com/tanzu-cli-os-packages/rpm/tanzu-cli
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.vmware.com/tools/keys/VMWARE-PACKAGING-GPG-RSA-KEY.pub
EOF

sudo yum install -y tanzu-cli # dnf install can also be used
```

To uninstall: `sudo yum remove tanzu-cli`

#### Installing a Specific Version

To install a specific version of the Tanzu CLI you can specify the version explicitly:

```console
# To list the available versions
sudo yum list tanzu-cli --showduplicates

# To install a specific version (notice the '-' before the version)
sudo yum install tanzu-cli-0.90.1
```

To uninstall: `sudo yum remove tanzu-cli`

#### Installing a Pre-Release

Pre-releases of the Tanzu CLI are made available to get early feedback before
a new version is released.  Pre-releases are available through `yum/dnf` using the
same repository configuration steps described above, but the installation
command uses a different package name: `tanzu-cli-unstable`.

**Note**: Just like installing a new version, installing a pre-release will
replace the `tanzu` binary of any previous installation.

```console
# Remove any installed tanzu cli binary
sudo yum remove tanzu-cli

# First setup the repository if it is not setup already, then run:
sudo yum install tanzu-cli-unstable
```

To uninstall: `sudo yum remove tanzu-cli-unstable`

### Note on installation paths

Package managers have opinions on locations to which binaries are installed.
If the location of the CLI (e.g `$(brew --prefix)/bin/tanzu` by HomeBrew,
`/usr/bin/tanzu` by apt) conflicts with an existing CLI that needs to be
retained for any reason, the existing binary should be moved to another
location before installation with package managers.

In addition, to ensure the CLI installed is used, ensure that the PATH setting
is such that it is picked up by default. Commands like `which tanzu` and
`tanzu version` will be useful to check that the right CLI binary is being used.

## Automatic Prompts, and Potential Mitigations

At the first suitable opportunity, (and on subsequent CLI use until the EULA is
accepted), the CLI will present the following prompts to solicit inputs from
the CLI user:

****EULA Prompt**** :
The Tanzu CLI prompts the user to review and agree to the VMware General Terms.
Agreeing to the terms is a prerequisite to being able to install or update plugins
available in the default central plugin repository.

Note: to review the Terms and decide again on the acceptance, this same
prompt can also be explicitly invoked with `tanzu config eula show`.

****Customer Experience Improvement Program (CEIP) Prompt**** :
The Tanzu CLI prompts the user to accept participation in CEIP or not.

Systems and users (via automation scripts, for instance) that expect
non-interactive use of the CLI can avoid being prompted with the
above by running the following before any other CLI commands:

- `tanzu config eula accept` to accept the EULA.
- To set the CEIP participation status for automation, the environment variable `TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER` can be set to `No` or `Yes`.

### Essentials plugin group

Introducing a plugin group named `vmware-tanzucli/essentials`, which includes all the necessary prerequisite plugins for the tanzu cli experience.
The initial offering of `vmware-tanzucli/essentials` will include the “telemetry” plugin but the list could grow in the future.

Essential plugins will be installed on the system to enhance the overall tanzu CLI experience.
For instance, the Telemetry plugin should be consistently installed to enable the collection and transmission of telemetry data.

Essentials plugins installed when user accepts EULA

```shell
> tanzu plugin list

? You must agree to the VMware General Terms in order to download, install, or
use software from this registry via Tanzu CLI. Acceptance of the VMware General
Terms covers all software installed via the Tanzu CLI during any Session.
“Session” means the period from acceptance until any of the following occurs:
(1) a change to VMware General Terms, (2) a new major release of the Tanzu CLI
is installed, (3) software is accessed in a separate software distribution
registry, or (4) re-acceptance of the General Terms is prompted by VMware.

To view the VMware General Terms, please see https://www.vmware.com/vmware-general-terms.html

If you agree, the essentials plugins will be installed that is necessary for tanzu cli experience

Do you agree to the VMware General Terms?
 > Yes

[i] The tanzu cli essential plugins have not been installed and are being installed now. The install may take a few seconds.

Standalone Plugins
  NAME       DESCRIPTION              TARGET  VERSION  STATUS
  telemetry  telemetry functionality  global  v1.1.0   installed
```

With each update of this essential plugin (or any future one added to the group), a new version of this plugin group will be released.

By default, the CLI will try to install the most recent version of the essential plugin group.

Essentials plugins being updated when a new version of plugin is available

```shell

> tanzu plugin list
[i] The tanzu cli essential plugins are outdated and are being updated now. The update may take a few seconds.

Standalone Plugins
  NAME       DESCRIPTION              TARGET  VERSION  STATUS
  telemetry  telemetry functionality  global  v1.1.0   installed
```

However, it also allows for the installation of specific versions of the essential plugin group through the environment variable

``` shell
export TANZU_CLI_ESSENTIALS_PLUGIN_GROUP_VERSION=v0.0.2
```

Essential plugin group name can be customized using an env variable

``` shell
export TANZU_CLI_ESSENTIALS_PLUGIN_GROUP_NAME=vmware-tanzucli/essentials
```

When the Tanzu CLI binary is executed it will automatically install or update the essential plugin group, if required.
If a specific version is specified using env `TANZU_CLI_ESSENTIALS_PLUGIN_GROUP_VERSION` only specified version will be installed without upgrading to the latest version.

```shell
export TANZU_CLI_ESSENTIALS_PLUGIN_GROUP_VERSION=v0.0.1
```

To manually install the essentials plugin group.

To install latest version

```shell
tanzu plugin install all -–group vmware-tanzucli/essentials
```

To install specific version

```shell
tanzu plugin install all -–group vmware-tanzucli/essentials:v1.0.0
```

## Installing and using the Tanzu CLI in internet-restricted environments

### Installing the Tanzu CLI

You can install the Tanzu CLI in internet-restricted environments by
downloading the Tanzu CLI Binary from a Github Release and copy it to the
internet-restricted environment. Once copied follow the steps mentioned
[here](#from-the-binary-releases-in-the-github-project) to install the Tanzu
CLI.

There are also different solutions to enabling the use of Package Managers
within internet-restricted environments. The advantage of leveraging the
Package Managers is that upgrades of the CLI itself will be more
straightforward. Since these solutions are somewhat specific to the Package
Management system and user's environments, please contact your system
administrator to find out if any exist.

### Installing Tanzu CLI plugins in internet-restricted environments

The Tanzu CLI allows users to install and run CLI plugins in internet-restricted
environments (which means air-gapped environments, with no physical connection
to the Internet). To run the Tanzu CLI in internet-restricted environments a
private Docker-compatible container registry such as
[Harbor](https://goharbor.io/), [Docker](https://docs.docker.com/registry/), or
[Artifactory](https://jfrog.com/artifactory/) is required.

Once the private registry is set up, the operator of the private registry can
migrate plugins from the publicly available registry to the private registry
using the below-mentioned steps:

1. Download the plugin-inventory image along with all selected plugin images
as a `tar.gz` file on the local disk of a machine which has internet access
using the `tanzu plugin download-bundle` command.
2. Copy this `tar.gz` file to the air-gapped network (using a USB drive or
other mechanism).
3. Upload the plugin bundle `tar.gz` to the air-gapped private registry using
the `tanzu plugin upload-bundle` command.

**Note**: The Tanzu CLI supports uploading plugins to private registries that
require authentication. However, such private registries must allow pulling the
resulting images without authentication.

#### Downloading plugin bundle

You can download all plugins within the default central repository by running
the following command:

```sh
tanzu plugin download-bundle --to-tar /tmp/plugin_bundle_complete.tar.gz
```

However, if you only want to migrate plugins within a specific plugin group version
(e.g. `vmware-tkg/default:v2.1.0`) you can run the following command to download
the plugin bundle containing only the plugins from specified group version:

```sh
tanzu plugin download-bundle --group vmware-tkg/default:v2.1.0 --to-tar /tmp/plugin_bundle_tkg_v2_1_0.tar.gz
```

Note that multiple group versions can be specified at once.  For example:

```sh
tanzu plugin download-bundle --group vmware-tkg/default:v2.1.0,vmware-tkg/default:v1.6.0 --to-tar /tmp/plugin_bundle_tkg_v2_1_0.tar.gz

# or, if you prefer repeating the --group flag
tanzu plugin download-bundle --group vmware-tkg/default:v2.1.0 --group vmware-tkg/default:v1.6.0 --to-tar /tmp/plugin_bundle_tkg_v2_1_0.tar.gz
```

If you do not specify a group's version, the latest version available for the group will be used:

```sh
tanzu plugin download-bundle --group vmware-tkg/default --to-tar /tmp/plugin_bundle_tkg_latest.tar.gz
```

To migrate plugins from a specific plugin repository and not use the default
plugin repository you can provide a `--image` flag with the above command, for example:

```sh
tanzu plugin download-bundle
                  --image custom.repo.example.com/tanzu-cli/plugins/plugin-inventory:latest
                  --group vmware-tkg/default:v2.1.0
                  --to-tar /tmp/plugin_bundle_tkg_v2_1_0.tar.gz
```

#### Uploading plugin bundle to the private registry

Once you download the plugin bundle as a `tar.gz` file and copy the file to the
air-gapped network, you can run the following command to migrate plugins to the
private registry (e.g. `registry.example.com/tanzu-cli/plugin`).

If the private registry requires authentication to upload images to the registry
run `docker login` or `crane auth login` to setup authentication with the registry
before running the `tanzu plugin upload-bundle` command.

Note: If the private registry is using self-signed certificates please configure
certs for the registry as mentioned [here](#interacting-with-a-central-repository-hosted-on-a-registry-with-self-signed-ca-or-with-expired-ca).

```sh
tanzu plugin upload-bundle --tar /tmp/plugin_bundle_complete.tar.gz --to-repo `registry.example.com/tanzu-cli/plugin`
```

The above-mentioned command uploads the plugin bundle to the provided private
repository location with the image name `plugin-inventory:latest`. So for the
above example, the plugin inventory image will be published to
`registry.example.com/tanzu-cli/plugin/plugin-inventory:latest`.

Please note that `tanzu plugin upload-bundle` uploads the plugins by adding them
to any plugin-inventory already present in the private registry. That means if you have already uploaded
any plugins to the specified private repository, it will keep the existing
plugins and append new plugins from the plugin bundle provided.

You can use this image and configure the default discovery source to point to
this image by running the following command:

```sh
tanzu plugin source update default --uri registry.example.com/tanzu-cli/plugin/plugin-inventory:latest
```

Now, the Tanzu CLI should be able to discover plugins from this newly configured
private plugin discovery source. Verify that plugins are discoverable by
running the `tanzu plugin search`, `tanzu plugin group search`, and
`tanzu plugin install` commands.

### Interacting with a central repository hosted on a registry with self-signed CA or with expired CA

If a user has configured a central repository on a custom registry (e.g. air-gaped environment) with a self-signed CA or
if the
registry CA
certificate is expired, the user can execute the `tanzu config cert` family of commands to configure the certificate for
the registry host.

```shell

    # If the registry host is self-signed add CA certificate for the registry
    tanzu config cert add --host test.registry.com --ca-certificate path/to/ca/cert

    # If the registry is self-signed and is serving on non-default port add CA certificate for the registry
    tanzu config cert add --host test.registry.com:8443 --ca-certificate path/to/ca/cert

    # If the registry is self-signed or CA cert is expired, add cert configuration for the registry host with
    # skip-cert-verify option
    tanzu config cert add --host test.registry.com  --skip-cert-verify true

    # Set to allow insecure (http) connection while interacting with host
    tanzu config cert add --host test.registry.com  --insecure true

```

The CLI uses the certificate configuration added for the registry host (using `tanzu config cert add` command ) while
interacting with the registry.

Users can update or delete the certificate configuration using the `tanzu config cert update`
and `tanzu config cert delete` commands.
Also, users can list the certificate configuration using the `tanzu config cert list` command.

#### Proxy CA certificate

If the user configured a proxy between the Tanzu CLI and the central repository and if the proxy certificate
needs to be configured, the user should set the environment variable `PROXY_CA_CERT` with base64 value of
proxy CA certificate.
