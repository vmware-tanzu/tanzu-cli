Name:       tanzu-cli
Version:    %{package_version}
Release:    1
License:    Apache 2.0
URL:        https://github.com/vmware-tanzu/tanzu-cli
Vendor:     VMware
Summary:    The core Tanzu CLI
Provides:   tanzu-cli
Obsoletes:  tanzu-cli  < %{package_version}

%ifarch amd64
%define arch amd64
%endif

%ifarch aarch64
# TODO For now, we use the amd64 build for arm64
%define arch amd64
%endif

%undefine _disable_source_fetch
Source0:    https://github.com/vmware-tanzu/tanzu-cli/releases/download/v%{release_version}/tanzu-cli-linux-%{arch}.tar.gz

%description
VMware Tanzu is a modular, cloud native application platform that enables vital DevSecOps outcomes
in a multi-cloud world.  The Tanzu CLI allows you to control VMware Tanzu from the command-line.

# Go does not generate a build-id compatible with RPM, so we disable the need for a build-id
# See https://github.com/rpm-software-management/rpm/issues/367
%global _missing_build_ids_terminate_build 0

# This is required to avoid some missing debug file errors
%define debug_package %nil

%prep
%setup -q -n v%{release_version}

%build
# Nothing to build

%install
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/%{_bindir}
mv tanzu-cli-linux_%{arch} $RPM_BUILD_ROOT/%{_bindir}/tanzu

%files
%{_bindir}/tanzu
