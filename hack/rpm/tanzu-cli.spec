Name:       tanzu-cli
Version:    %{rpm_package_version}
Release:    %{rpm_release_version}
License:    Apache 2.0
URL:        https://github.com/vmware-tanzu/tanzu-cli
Vendor:     VMware
Summary:    The core Tanzu CLI
Provides:   tanzu-cli
Obsoletes:  tanzu-cli  < %{rpm_package_version}

%ifarch x86_64
%define arch amd64
%endif

%ifarch aarch64
# TODO For now, we use the amd64 build for arm64
%define arch amd64
%endif

Source0:    %{expand:%%(pwd)/artifacts/linux/%{arch}/cli/core/%{cli_version}/tanzu-cli-linux_%{arch}}

%description
VMware Tanzu is a modular, cloud native application platform that enables vital DevSecOps outcomes
in a multi-cloud world.  The Tanzu CLI allows you to control VMware Tanzu from the command-line.

# Go does not generate a build-id compatible with RPM, so we disable the need for a build-id
# See https://github.com/rpm-software-management/rpm/issues/367
%global _missing_build_ids_terminate_build 0

# This is required to avoid some missing debug file errors
%define debug_package %nil

%install
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT%{_bindir}
cp -af %{SOURCEURL0} $RPM_BUILD_ROOT%{_bindir}/tanzu

%post
# Setup bash completion
mkdir -p /usr/share/bash-completion/completions
tanzu completion bash > /usr/share/bash-completion/completions/tanzu
chmod a+r /usr/share/bash-completion/completions/tanzu

# Setup zsh completion
mkdir -p /usr/local/share/zsh/site-functions
tanzu completion zsh > /usr/local/share/zsh/site-functions/_tanzu
chmod a+r /usr/local/share/zsh/site-functions/_tanzu

# Setup fish completion
mkdir -p /usr/share/fish/vendor_completions.d
tanzu completion fish > /usr/share/fish/vendor_completions.d/tanzu.fish
chmod a+r /usr/share/fish/vendor_completions.d/tanzu.fish

%files
%{_bindir}/tanzu
