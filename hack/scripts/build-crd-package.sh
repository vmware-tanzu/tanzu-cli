#!/usr/bin/env bash

# Copyright 2024 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e
set -x

BASE_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)
PACKAGE_DIR=${BASE_DIR}/../../package
BINDIR=${BASE_DIR}/../tools/bin

KCTRL=${KCTRL:-$BINDIR/kctrl}

pushd ${PACKAGE_DIR}

yq e -i ".spec.template.spec.export[0].imgpkgBundle.image=\"${CRD_PACKAGE_IMAGE}\"" ./package-build.yml

PATH=${BINDIR}:$PATH ${KCTRL} package release --yes

popd

