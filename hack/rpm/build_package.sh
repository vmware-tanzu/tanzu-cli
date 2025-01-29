#!/usr/bin/env bash

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# This script builds an RPM package for the Tanzu CLI for each supported architecture.
# It can be called more than once to build multiple packages with different names.
# This is useful if we want to use a different signing key for different packages.
# Once all the packages are built, the final repository can be built using the
# build_package_repo.sh script.
set -e
set -x

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
# Set the default package name if not provided
RPM_PACKAGE_NAME=${RPM_PACKAGE_NAME:-"tanzu-cli"}

BASE_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)
OUTPUT_DIR=${BASE_DIR}/_output/rpm/tanzu-cli
# Directory where the packages will be stored
PKG_DIR=${OUTPUT_DIR}
ROOT_DIR=${BASE_DIR}/../..

# Install build dependencies
if ! command -v rpmlint &> /dev/null; then
   $DNF install -y rpmlint rpm-build
fi

rpmlint ${BASE_DIR}/tanzu-cli.spec

# We must create the sources directory ourselves in the below location
mkdir -p ${HOME}/rpmbuild/SOURCES

# We support building multiple packages by calling the script multiple times
# and specifying a different RPM_PACKAGE_NAME. This is useful if we want
# to use a different signing key for different packages.
# So, we only want to clean the output directory if we have built
# the final repository, which indicates that this is an old build.
if [ -d ${OUTPUT_DIR}/repodata ]; then
   rm -rf ${OUTPUT_DIR}
fi 
mkdir -p ${OUTPUT_DIR}
mkdir -p ${PKG_DIR}
cd ${ROOT_DIR}

# Transform the CLI version into RPM-compatible package version and release numbers.
if [[ ${VERSION} == *-* ]]; then 
   # If the version contains a - character, we are dealing with an unstable version
   # so we should append -unstable to the package name
   RPM_PACKAGE_NAME=${RPM_PACKAGE_NAME}-unstable

   # When there is a - in the version, we are dealing with a pre-release
   # e.g., 1.0.0-dev, 1.0.0-alpha.0, 1.0.0.rc.1, etc
   # Such versions should be marked as RPM pre-releases by using a package release
   # number of the form 0.1...
   # See https://serverfault.com/a/867567

   # For the package version 1.0.0-rc-1 becomes 1.0.0
   RPM_PACKAGE_VERSION="${VERSION%%-*}"
   # while the release version becomes 0.1-rc-1
   RPM_RELEASE_VERSION="0.1_${VERSION#*-}"
   # RPM does not like having the - character in the version of the package
   RPM_RELEASE_VERSION=${RPM_RELEASE_VERSION//-/_}
   # For a full version of 1.0.0-0.1_rc_1
else
   RPM_PACKAGE_VERSION="${VERSION}"
   RPM_RELEASE_VERSION="1"
fi

######################
# Build the packages
######################

# Build the package for each architecture
for arch in x86_64 aarch64; do
   rpmbuild --define "rpm_package_name ${RPM_PACKAGE_NAME}" \
            --define "rpm_package_version ${RPM_PACKAGE_VERSION}" \
            --define "rpm_release_version ${RPM_RELEASE_VERSION}" \
            --define "cli_version v${VERSION}" \
            -bb ${BASE_DIR}/tanzu-cli.spec \
            --target ${arch}

   # Sign the package before moving it to the common output directory
   if [[ ! -z "${RPM_SIGNER}" ]]; then
     ${RPM_SIGNER} ${HOME}/rpmbuild/RPMS/${arch}/tanzu-cli*${arch}.rpm
   else
     echo skip rpmsigning packages for ${arch}
   fi

   # Move the signed package to the output directory where the other packages
   # also reside, so that we can build the repository at the very end 
   mv ${HOME}/rpmbuild/RPMS/${arch}/* ${PKG_DIR}/
done
