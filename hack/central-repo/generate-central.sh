#!/usr/bin/env bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

ROOT_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)

# Start a registry
make -C $ROOT_DIR/../.. start-test-central-repo
rm -f /tmp/central.db

# Build a run a docker image that contains imgpkg and sqlite3
# to avoid having to install them locally
echo "========================================"
echo "Setting up image with imgpkg and sqlite3"
echo "========================================"
IMAGE=build-central
docker build -t ${IMAGE} ${ROOT_DIR} -f - <<- EOF
   FROM ubuntu
   RUN apt update && \
       apt install -y curl \
                      sqlite3 \
                      libdigest-sha-perl

    RUN mkdir /tmp/carvel/ && \
        curl -L https://carvel.dev/install.sh | K14SIO_INSTALL_BIN_DIR=/tmp/carvel bash && \
        install /tmp/carvel/imgpkg /usr/bin

   WORKDIR /work
   COPY upload-plugins.sh .
   COPY fakeplugin.sh .
EOF

# Generate both the small and large test central repositories
docker run --rm ${IMAGE} ./upload-plugins.sh --fast host.docker.internal:9876/tanzu-cli/plugins/central:small 
docker run --rm ${IMAGE} ./upload-plugins.sh host.docker.internal:9876/tanzu-cli/plugins/central:large

# Stop the registry
make -C $ROOT_DIR/../.. stop-test-central-repo
