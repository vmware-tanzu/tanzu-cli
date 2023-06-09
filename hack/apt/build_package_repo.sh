#!/usr/bin/env bash

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e

if [ $(uname) != "Linux" ] || [ -z "$(command -v apt)" ]; then
   echo "This script must be run on a Linux system that uses the APT package manager"
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

# Install build dependencies
if ! command -v reprepro &> /dev/null; then
   apt-get update
   apt-get install -y reprepro
fi

# Assumes ${OUTPUT_DIR} is populated from build_package.sh

# Download the SRP-compliant CLI build from github and copy it to the package directory
for arch in amd64 arm64; do
   echo "===================================="
   echo "Building debian package repo for $arch..."
   echo "===================================="

   # Expects signed file to be present
   if [ ! -f "${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}.deb" ]; then
      echo "Not found: ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}.deb"
      exit 1
   fi

   # Create repository
   reprepro -b ${OUTPUT_DIR}/apt includedeb tanzu-cli-jessie ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}.deb

   # Cleanup
   rm -f ${OUTPUT_DIR}/tanzu-cli_${VERSION}_linux_${arch}.deb
done

if [[ ! -z "${DEB_SIGNER}" ]]; then
   for rel in `find ${OUTPUT_DIR} -name "Release"`; do
      ${DEB_SIGNER} $rel ${rel}.gpg
   done
else
   echo skip debsigning
fi

# Global cleanup
rm -rf ${OUTPUT_DIR}/apt/conf
rm -rf ${OUTPUT_DIR}/apt/db
