#!/usr/bin/env bash

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

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

BASE_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)
OUTPUT_DIR=${BASE_DIR}/_output/rpm/tanzu-cli
# Directory where the packages will be stored
PKG_DIR=${OUTPUT_DIR}
ROOT_DIR=${BASE_DIR}/../..

# Install build dependencies
if ! command -v rpmlint &> /dev/null; then
   $DNF install -y rpmlint createrepo rpm-build yum-utils
fi

rpmlint ${BASE_DIR}/tanzu-cli.spec

# We must create the sources directory ourselves in the below location
mkdir -p ${HOME}/rpmbuild/SOURCES

# Create the .rpm packages
rm -rf ${OUTPUT_DIR}
mkdir -p ${OUTPUT_DIR}
mkdir -p ${PKG_DIR}
cd ${ROOT_DIR}

UNSTABLE="false"
# Transform the CLI version into RPM-compatible package version and release numbers.
if [[ ${VERSION} == *-* ]]; then 
   UNSTABLE="true"

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
   rpmbuild --define "rpm_package_version ${RPM_PACKAGE_VERSION}" \
            --define "rpm_release_version ${RPM_RELEASE_VERSION}" \
            --define "cli_version v${VERSION}" \
            --define "unstable ${UNSTABLE}" \
            -bb ${BASE_DIR}/tanzu-cli.spec \
            --target ${arch}
   mv ${HOME}/rpmbuild/RPMS/${arch}/* ${PKG_DIR}/

   if [[ ! -z "${RPM_SIGNER}" ]]; then
     ${RPM_SIGNER} ${PKG_DIR}/tanzu-cli*${arch}.rpm
   else
     echo skip rpmsigning packages for ${arch}
   fi
done

######################
# Build the repository
######################

# Prepare the existing repository info so we can sync from it
RPM_METADATA_BASE_URI=${RPM_METADATA_BASE_URI:=https://storage.googleapis.com/tanzu-cli-os-packages}
if [ "${RPM_METADATA_BASE_URI}" = "new" ]; then
   echo
   echo "Building a brand new repository"
   echo
else
   cat << EOF | tee /tmp/tanzu-cli.repo
[tanzu-cli]
name=Tanzu CLI
baseurl=${RPM_METADATA_BASE_URI}/rpm/tanzu-cli
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.vmware.com/tools/keys/VMWARE-PACKAGING-GPG-RSA-KEY.pub
EOF
   
   # Sync the metadata so we can update it
   # Use the --source flag to avoid downloading the actual RPMs
   reposync --repoid=tanzu-cli --download-metadata -p ${OUTPUT_DIR} -c /tmp/tanzu-cli.repo --norepopath --source -y
   # Remove the old signature, which won't be valid anymore
   rm -f ${OUTPUT_DIR}/repodata/repomd.xml.asc
   
   # Now list the existing RPMs so we can pretend to have them locally
   for p in $(reposync --repoid=tanzu-cli -c /tmp/tanzu-cli.repo -u -y | grep ${RPM_METADATA_BASE_URI}); do
      echo "Found package: $p"
      touch ${PKG_DIR}/$(basename $p)
   done
fi

# Create the repository metadata
createrepo --update --skip-stat ${OUTPUT_DIR}

# Now that the repo is created, remove the fake empty packages so they don't
# risk being copied over the real ones in the final repository.
find ${PKG_DIR} -type f -empty -delete

if [[ ! -z "${RPM_SIGNER}" ]]; then
  # instead of ... gpg --detach-sign --armor repodata/repomd.xml
  ${RPM_SIGNER} ${OUTPUT_DIR}/repodata/repomd.xml
else
  echo skip rpmsigning repo
fi
