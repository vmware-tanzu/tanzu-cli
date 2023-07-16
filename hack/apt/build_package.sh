#!/usr/bin/env bash

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e
set -x

if [ $(uname) != "Linux" ]; then
   echo "This script must be run on a Linux system"
   exit 1
fi

# Use apt-get, but also support dnf/yum
PKG_MGR=$(command -v apt-get || command -v dnf || command -v yum || true)

if [ -z "$PKG_MGR" ]; then
   echo "This script requires one of the following package managers: apt-get, dnf or yum"
   exit 1
fi

# VERSION should be set when calling this script
if [ -z "${VERSION}" ]; then
   echo "\$VERSION must be set before calling this script"
   exit 1
fi

# Strip 'v' prefix as an apt package version must start with an integer
VERSION=${VERSION#v}

BASE_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)
OUTPUT_DIR=${BASE_DIR}/_output
ARTIFACTS_DIR=${BASE_DIR}/../../artifacts

# Install build dependencies
if ! command -v dpkg-deb &> /dev/null; then
   ${PKG_MGR} update -y
   ${PKG_MGR} install -y dpkg
fi

# Clean any old packages
rm -rf ${OUTPUT_DIR}

# Copy the CLI build from ARTIFACTS_DIR to the package directory
for arch in amd64 arm64; do
   echo "===================================="
   echo "Building debian package for $arch..."
   echo "===================================="

   # For now, we don't have an ARM64 build, so we get the AMD64 one and use it for ARM64.
   # This is for Apple M1 machines which normally have an emulator.
   # TODO: Replace all instances of "${fakeArch}" with "${arch}"
   fakeArch=amd64
   mkdir -p ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}/usr/bin
   cp ${ARTIFACTS_DIR}/linux/${fakeArch}/cli/core/v${VERSION}/tanzu-cli-linux_${fakeArch} ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}/usr/bin/tanzu

   # Create the control file
   mkdir -p ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}/DEBIAN
   echo "Package: tanzu-cli
Version: ${VERSION}
Maintainer: Tanzu CLI project team
Architecture: ${arch}
Section: main
Priority: optional
Homepage: https://github.com/vmware-tanzu/tanzu-cli
Description: The core Tanzu CLI" \
      > ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}/DEBIAN/control

   # Add a postinstall script to setup shell completion
   echo "#!/bin/bash

mkdir -p /usr/share/bash-completion/completions
tanzu completion bash > /usr/share/bash-completion/completions/tanzu
chmod a+r /usr/share/bash-completion/completions/tanzu

mkdir -p /usr/local/share/zsh/site-functions
tanzu completion zsh > /usr/local/share/zsh/site-functions/_tanzu
chmod a+r /usr/local/share/zsh/site-functions/_tanzu

mkdir -p /usr/share/fish/vendor_completions.d
tanzu completion fish > /usr/share/fish/vendor_completions.d/tanzu.fish
chmod a+r /usr/share/fish/vendor_completions.d/tanzu.fish" \
      > ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}/DEBIAN/postinst
   chmod a+x ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}/DEBIAN/postinst

   # Create the .deb package
   dpkg-deb --build -Zgzip ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}

   rm -rf ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}
done

if [[ ! -z "${DEB_SIGNER}" ]]; then
   for deb in `find ${OUTPUT_DIR} -name "*.deb"`; do
      ${DEB_SIGNER} $deb
   done
else
   echo skip debsigning
fi
