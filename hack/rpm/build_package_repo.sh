#!/usr/bin/env bash

# Copyright 2024 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# This script expects the RPM packages to already be present in the _output/rpm/tanzu-cli directory
# It will create a repository in the same directory and sign it with the provided key
# If the RPM_SIGNER environment variable is not set, the repository will not be signed

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
# Directory where the packages are stored
PKG_DIR=${OUTPUT_DIR}
ROOT_DIR=${BASE_DIR}/../..

# Install build dependencies
if ! command -v createrepo &> /dev/null; then
   $DNF install -y createrepo yum-utils
fi

cd ${ROOT_DIR}

######################
# Build the repository
######################

# Prepare the existing repository info so we can sync from it
RPM_METADATA_BASE_URI=${RPM_METADATA_BASE_URI:=https://storage.googleapis.com/tanzu-cli-installer-packages}
RPM_REPO_GPG_PUBLIC_KEY_URI=${RPM_REPO_GPG_PUBLIC_KEY_URI:=https://storage.googleapis.com/tanzu-cli-installer-packages/keys/TANZU-PACKAGING-GPG-RSA-KEY.gpg}
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
gpgkey=${RPM_REPO_GPG_PUBLIC_KEY_URI}
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

   # Allow replacing an existing package version with a new one; this
   # has proven useful when a package was not signed properly.
   #
   # These RPM versions are set when building the packages
   RPM_RELEASE_VERSION=1
   # For unstable versions, the rpm version is different
   if [[ ${VERSION} == *-* ]]; then
       RPM_RELEASE_VERSION="0.1_${VERSION#*-}"
       RPM_RELEASE_VERSION=${RPM_RELEASE_VERSION//-/_}
   fi

   # Remove the current version's packages, in case they are already
   # part of the existing repo, and create the metadata without the
   # current packages.  This allows to cleanup the metadata.
   # We need to do this because we use the --skip-stat flag when building
   # the repo, which implies that if the package is already in the repo,
   # it won't be replaced with a newer version.
   # Therefore, we first remove the current packages, then we add them back
   # after the metadata has been cleaned up.
   # If the current packages are not part of the existing repo, then
   # this step is a no-op.
   mv ${PKG_DIR}/*${VERSION}-${RPM_RELEASE_VERSION}*.rpm ${OUTPUT_DIR}/..
   createrepo --update --skip-stat ${OUTPUT_DIR}

   # Add back the current packages so we can build the final repo metadata
   mv ${OUTPUT_DIR}/../*${VERSION}-${RPM_RELEASE_VERSION}*.rpm ${PKG_DIR}
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
