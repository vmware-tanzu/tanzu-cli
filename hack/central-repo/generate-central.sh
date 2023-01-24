#!/usr/bin/env bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

ROOT_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)

# Start a registry
make -C $ROOT_DIR/../.. start-test-central-repo
rm -f /tmp/central.db
$ROOT_DIR/upload-plugins.sh --fast localhost:9876/tanzu-cli/plugins/central:small 
$ROOT_DIR/upload-plugins.sh localhost:9876/tanzu-cli/plugins/central:large

# Stop the registry
make -C $ROOT_DIR/../.. stop-test-central-repo
