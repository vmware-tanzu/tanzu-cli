#!/bin/bash

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# Change directories to the parent directory of the one in which this
# script is located.
cd "$(dirname "${BASH_SOURCE[0]}")/../.."

# mdlint rules with common errors and possible fixes can be found here:
# https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md
# Additional configuration can be found in the .markdownlintrc file at
# the root of the repo.
docker run --rm -v "$(pwd)":/build \
  gcr.io/eminent-nation-87317/tanzu-cli/ci-images/mdlint:0.23.2 /md/lint \
  -i docs/cli/commands \
  -i **/CHANGELOG.md .
