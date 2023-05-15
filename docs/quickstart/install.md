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

## Via Package Managers

A recent, supported version of the Tanzu CLI is also available for install
through the following package managers:

### Homebrew (MacOS)

```console
brew update
brew install vmware-tanzu/tanzu/tanzu-cli
```

### Chocolatey (Windows)

NOTE: The Chocolatey package is not available yet.

```console
choco install tanzu-cli
```

### Apt (Debian/Ubuntu)

Note: The current APT package is not signed and will cause security warnings.
This will be fixed very soon.

```console
sudo apt-get update
sudo apt-get install -y ca-certificates
echo "deb https://storage.googleapis.com/tanzu-cli-os-packages/apt tanzu-cli-jessie main" | sudo tee /etc/apt/sources.list.d/tanzu.list
sudo apt-get update --allow-insecure-repositories
sudo apt-get install -y tanzu-cli --allow-unauthenticated
```

### From yum/dnf (RHEL)

Note: The current yum/dnf package is not signed and will cause security warnings.
This will be fixed very soon.

```console
cat << EOF | sudo tee /etc/yum.repos.d/tanzu-cli.repo
[tanzu-cli]
name=Tanzu CLI
baseurl=https://storage.googleapis.com/tanzu-cli-os-packages/rpm/tanzu-cli
enabled=1
gpgcheck=0
EOF

sudo yum install tanzu-cli # dnf install can also be used
```

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
non-interactive use of the CLI can avoid being prompted with either of the
above by running the following before any other CLI commands:

- `tanzu config eula accept` to accept the EULA.
- `tanzu ceip-participation set <true/false>` to set the CEIP participation status.

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

Note: If you private registry is using self-signed certificates please configure
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
