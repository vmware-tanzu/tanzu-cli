#!/usr/bin/env bash

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e

if [ $(uname) != "Linux" ]; then
   echo "This script must be run on a Linux system"
   exit 1
fi

# Use DNF and if it is not installed fallback to YUM
DNF=$(command -v dnf || command -v yum || true)
if [ -z "$DNF" ]; then
   echo "This script requires the presence of either DNF or YUM package manager"
   exit 1
fi

# VERSION should be set when calling this script
if [ -z "${VERSION}" ]; then
   echo "\$VERSION must be set before calling this script"
   exit 1
fi

# Strip 'v' prefix as an rpm package version must start with an integer
VERSION=${VERSION#v}

BASE_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)
OUTPUT_DIR=${BASE_DIR}/_output/rpm/tanzu-cli
ROOT_DIR=${BASE_DIR}/../..

# Install build dependencies
if ! command -v rpmlint &> /dev/null; then
   $DNF install -y rpmdevtools rpmlint createrepo rpm-build rpm-sign
fi

rpmlint ${BASE_DIR}/tanzu-cli.spec

# We must create the sources directory ourselves in the below location
mkdir -p ${HOME}/rpmbuild/SOURCES

# Create the .rpm packages
rm -rf ${OUTPUT_DIR}
mkdir -p ${OUTPUT_DIR}
cd ${ROOT_DIR}

# RPM does not like - in the its package version
PACKAGE_VERSION=${VERSION//-/_}
rpmbuild --define "package_version ${PACKAGE_VERSION}" --define "release_version ${VERSION}" -bb ${BASE_DIR}/tanzu-cli.spec --target x86_64
mv ${HOME}/rpmbuild/RPMS/x86_64/* ${OUTPUT_DIR}/

rpmbuild --define "package_version ${PACKAGE_VERSION}" --define "release_version ${VERSION}" -bb ${BASE_DIR}/tanzu-cli.spec --target aarch64
mv ${HOME}/rpmbuild/RPMS/aarch64/* ${OUTPUT_DIR}/

if [[ ! -z "${RPM_SIGNER}" ]]; then
  ${RPM_SIGNER} ${OUTPUT_DIR}/tanzu-cli*aarch64.rpm
  ${RPM_SIGNER} ${OUTPUT_DIR}/tanzu-cli*x86_64.rpm
else
  echo skip rpmsigning packages
fi

# Create the repository metadata
createrepo ${OUTPUT_DIR}

if [[ ! -z "${RPM_SIGNER}" ]]; then
  # instead of ... gpg --detach-sign --armor repodata/repomd.xml
  ${RPM_SIGNER} ${OUTPUT_DIR}/repodata/repomd.xml
else
  echo skip rpmsigning repo
fi
