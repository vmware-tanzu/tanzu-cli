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

UNSTABLE=""
if [[ ${VERSION} == *-* ]]; then 
   UNSTABLE="-unstable"
fi

BASE_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)
OUTPUT_DIR=${BASE_DIR}/_output

# Install dependencies
if ! command -v bzip2 &> /dev/null || ! command -v curl &> /dev/null; then
   ${PKG_MGR} update -y
   ${PKG_MGR} install -y bzip2 curl
fi

# Generate the repo metadata
DIST_DIR=${OUTPUT_DIR}/apt/dists/tanzu-cli-jessie
for arch in amd64 arm64; do
   FINAL_DIR=${DIST_DIR}/main/binary-${arch}

   # Assumes ${OUTPUT_DIR} is populated with the new deb packages from build_package.sh
   PKG="${OUTPUT_DIR}/tanzu-cli${UNSTABLE}_${VERSION}_linux_${arch}.deb"
   if [ ! -f ${PKG} ]; then
      echo "Not found: ${PKG}"
      exit 1
   fi

   # All that is needed from the original repo is to have the original Packages files under:
   #  ${FINAL_DIR}/Packages
   DEB_METADATA_BASE_URI=${DEB_METADATA_BASE_URI:=https://storage.googleapis.com/tanzu-cli-os-packages}
   mkdir -p ${FINAL_DIR}
   if [ "${DEB_METADATA_BASE_URI}" = "new" ]; then
      echo
      echo "Building a brand new repository"
      echo
   else
      # To build a brand new repo, we must pass in DEB_METADATA_BASE_URI=new
      curl -sLo ${FINAL_DIR}/Packages ${DEB_METADATA_BASE_URI}/apt/dists/tanzu-cli-jessie/main/binary-${arch}/Packages
   fi

   # Generate the new entry for the new package and add it to the existing file of packages
   cat << EOF >> ${FINAL_DIR}/Packages
Package: tanzu-cli${UNSTABLE}
Version: ${VERSION}
Maintainer: Tanzu CLI project team
Architecture: ${arch}
Homepage: https://github.com/vmware-tanzu/tanzu-cli
Priority: optional
Section: main
Filename: pool/main/t/tanzu-cli${UNSTABLE}/$(basename ${PKG})
Size: $(ls -l ${PKG} | awk '{print $5}')
SHA256: $(sha256sum ${PKG} | cut -f1 -d' ')
SHA1: $(sha1sum ${PKG} | cut -f1 -d' ')
MD5sum: $(md5sum ${PKG} | cut -f1 -d' ')
SHA512: $(sha512sum ${PKG} | cut -f1 -d' ')
Description: The core Tanzu CLI
EOF

   # Create the two compressed Packages file
   gzip -k -f ${FINAL_DIR}/Packages
   bzip2 -k -f ${FINAL_DIR}/Packages
   # The dists/tanzu-cli-jessie/main/binary-${arch}/Release file does not seem to change
   # so let's create our own instead of having to download it
   cat << EOF > ${FINAL_DIR}/Release
Component: main
Architecture: ${arch}
EOF

   # Move the new package into its final location
   mkdir -p ${OUTPUT_DIR}/apt/pool/main/t/tanzu-cli${UNSTABLE}
   mv ${PKG} ${OUTPUT_DIR}/apt/pool/main/t/tanzu-cli${UNSTABLE}
done

# Finally, re-generate the dists/tanzu-cli-jessie/Release file
cd $DIST_DIR
cat << EOF > Release
Codename: tanzu-cli-jessie
Date: $(TZ=UTC date '+%a, %d %b %Y %T %Z')
Architectures: amd64 arm64
Components: main
MD5Sum:
$(for f in $(find main -type f); do
  size=$(ls -l $f | awk '{print $5}')
  md5sum $f | awk -v size=$size '{print "  "$1" "size" "$2}'
done)
SHA1:
$(for f in $(find main -type f); do
  size=$(ls -l $f | awk '{print $5}')
  sha1sum $f | awk -v size=$size '{print "  "$1" "size" "$2}'
done)
SHA256:
$(for f in $(find main -type f); do
  size=$(ls -l $f | awk '{print $5}')
  sha256sum $f | awk -v size=$size '{print "  "$1" "size" "$2}'
done)
SHA512:
$(for f in $(find main -type f); do
  size=$(ls -l $f | awk '{print $5}')
  sha512sum $f | awk -v size=$size '{print "  "$1" "size" "$2}'
done)
EOF

if [[ ! -z "${DEB_SIGNER}" ]]; then
   # Sign the main release file which has all the checksums
   ${DEB_SIGNER} ${DIST_DIR}/Release ${DIST_DIR}/Release.gpg
else
   echo skip debsigning
fi
