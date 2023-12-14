#!/usr/bin/env bash

# Copyright 2024 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e
set -x

if [ -z "$CRD_PACKAGE_IMAGE" ]; then
  echo "skip building of CRD package because CRD_PACKAGE_IMAGE is not set"
  exit 0
fi

BASE_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)
PACKAGE_DIR=${BASE_DIR}/../../package/cliplugin.cli.tanzu.vmware.com
BINDIR=${BASE_DIR}/../tools/bin

KCTRL=${KCTRL:-$BINDIR/kctrl}
YQ=${YQ:-$BINDIR/yq}

pushd ${PACKAGE_DIR}

$YQ e -i ".spec.template.spec.export[0].imgpkgBundle.image=\"${CRD_PACKAGE_IMAGE}\"" ./package-build.yml

PATH=${BINDIR}:$PATH ${KCTRL} package release --yes

popd

