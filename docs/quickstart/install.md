# Installing the Tanzu CLI

The Tanzu CLI can be installed either from pre-built binary releases from
its GitHub project, or via popular package managers like Homebrew, apt and yum/dnf.

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

(Available at the first alpha release)

```console
brew update
brew install vmware-tanzu/tanzu/tanzu-cli
```

### Chocolatey (Windows)

The Chocolatey package is not available yet.

```console
choco install tanzu-cli
```

### Apt (Debian/Ubuntu)

(Available at the first alpha release)

Note: The current APT package is not signed and will cause security warnings.
This will be fixed very soon.

```console
sudo apt-get update
sudo apt-get install -y ca-certificates
echo "deb https://storage.googleapis.com/tanzu-cli-os-packages/apt tanzu-cli-jessie main" | sudo tee /etc/apt/sources.list.d/tanzu.list
sudo apt update --allow-insecure-repositories
sudo apt-get install tanzu-cli
```

### From yum/dnf (RHEL)

(Available at the first alpha release)

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
