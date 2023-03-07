# Installing the Tanzu CLI

The Tanzu CLI can be installed either from source, or from pre-built binary
releases from the its GitHub project, or via popular package managers like
Homebrew and apt.

## From the Binary Releases in GitHub project

Every [release](https://github.com/vmware-tanzu/tanzu-cli/releases) of the
Tanzu CLI provides separate binary releases for a variety of OS's and machine
architectures. To install using one of these:

1. Download the [desired version](https://github.com/vmware-tanzu/tanzu-cli/releases). We recommend picking the latest release of the major version of the CLI you want to use.
2. Unpack it (e.g. `tar -zxvf tanzu-cli-darwin-amd64.tar.gz`)
3. Find the `tanzu` binary in the unpacked directory, move it to its desired location in your $PATH
   destination (`mv darwin/arm64/cli/core/v0.1.2/tanzu-cli-darwin_arm64 /usr/local/bin/tanzu`), provide it executable permission (`chmod u+x /usr/local/bin/tanzu`) if necessary.
4. Verify the correct version of CLI is properly installed: `tanzu version`

## Via Package Managers

A recent, supported version of the Tanzu CLI is also available for install
through the following package managers:

### Homebrew (MacOS)

```console
VVV
brew install tanzu-cli
```

### Chocolatey (Windows)

```console
VVV
choco install tanzu-cli
```

### Apt (Debian/Ubuntu)

```console
VVV
curl -1sLf https://github.com/vmware-tanzu/tanzu-cli/signing-keys/tanzu-cli-release-signing-key.asc | sudo gpg --dearmor | sudo tee /usr/share/keyrings/vmware-tanzu-cli.gpg
sudo apt-get install apt-transport-https --yes
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/tanzu-cli.gpg] https://xxxx/tanzu-cli/stable/debian/ all main" | sudo tee /etc/apt/sources.list.d/tanzu-cli-stable-debian.list
sudo apt-get update
sudo apt-get install tanzu-cli
```

### From yum (RHEL)

```console
sudo yum install tanzu-cli
```
