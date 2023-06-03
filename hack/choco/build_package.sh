#!/usr/bin/env bash

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e

if [ "$(command -v choco)" = "" ]; then
   echo "This script must be run on a system that has 'choco' installed"
   exit 1
fi

# VERSION should be set when calling this script
if [ -z "${VERSION}" ]; then
   echo "\$VERSION must be set before calling this script"
   exit 1
fi

# Strip 'v' prefix to be consistent with our other package names
VERSION=${VERSION#v}

BASE_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)
OUTPUT_DIR=${BASE_DIR}/_output/choco

mkdir -p ${OUTPUT_DIR}/
# Remove the nupkg file made by `choco pack` in the working dir
rm -f ${OUTPUT_DIR}/*.nupkg

# Obtain SHA if it is not already specified in the variable SHA_FOR_CHOCO
if [ -z "$SHA_FOR_CHOCO" ]; then
   SHA_FOR_CHOCO=$(curl -sL https://github.com/vmware-tanzu/tanzu-cli/releases/download/v${VERSION}/tanzu-cli-binaries-checksums.txt | grep tanzu-cli-windows-amd64 |cut -f1 -d" ")
   if [ -z "$SHA_FOR_CHOCO" ]; then
      echo "Unable to determine SHA for package of version $VERSION"
      exit 1
   fi
fi

echo "Using SHA: ${SHA_FOR_CHOCO}"

# Prepare install script
sed -e s,__CLI_VERSION__,v${VERSION}, -e s,__CLI_SHA__,${SHA_FOR_CHOCO}, \
   ${BASE_DIR}/chocolateyInstall.ps1.tmpl > ${OUTPUT_DIR}/chocolateyInstall.ps1
chmod a+x ${OUTPUT_DIR}/chocolateyInstall.ps1

# Bundle the powershell scripts and nuspec into a nupkg file
# Passing a variable ("cliVersion") wasn't working with chocolatey 2.0.0 but worked with 1.4.0
# Also, chocolatey (nuspec) does not like a `.` in the pre-release part of the version.
# For example if we have v0.90.0-beta.0 we need to remove the last `.`
# First let's make sure we are dealing with a version that has a pre-release part
if [[ ${VERSION} == *-* ]];then
   mainVersion=${VERSION%%-*}  # this is the part before the `-` (e.g., v0.90.0)
   preVersion=${VERSION#*-}  # this is the part after the `-` (e.g, beta.0)
   preVersion=${preVersion//./}  # remove all the `.` in the pre-version part
   finalVersion=${mainVersion}-${preVersion}  # reconstruct the version without the `.`
else
   # Not a pre-release, so we can use the version directly
   finalVersion=$VERSION
fi
choco pack ${BASE_DIR}/tanzu-cli-release.nuspec --out ${OUTPUT_DIR} "cliVersion=${finalVersion}"
# For unofficial builds.  This needs to be adapted to know which of the two packages to use
# choco pack ${BASE_DIR}/tanzu-cli-release-unofficial.nuspec --out ${OUTPUT_DIR} "cliVersion=${finalVersion}"

# Upload the nupkg file to the registry
# Do this by hand until we have proper automation
# choco push --source https://push.chocolatey.org/ --api-key ....... ${OUTPUT_DIR}/tanzu-cli.${finalVersion}.nupkg

# For the unofficial builds
# choco push --source https://push.chocolatey.org/ --api-key ....... ${OUTPUT_DIR}/tanzu-cli-unofficial.${finalVersion}.nupkg
