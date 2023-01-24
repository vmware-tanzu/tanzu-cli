#!/usr/bin/env bash

ROOT_DIR=$(cd $(dirname "${BASH_SOURCE[0]}"); pwd)

# Start a registry
make -C $ROOT_DIR/../.. start-test-central-repo
rm -f /tmp/central.db
$ROOT_DIR/upload-plugins.sh --fast localhost:9876/tanzu-cli/plugins/central:small 
$ROOT_DIR/upload-plugins.sh localhost:9876/tanzu-cli/plugins/central:large

# Stop the registry
make -C $ROOT_DIR/../.. stop-test-central-repo
