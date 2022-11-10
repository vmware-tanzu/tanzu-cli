#!/usr/bin/env bash

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0


set -o errexit
set -o nounset
set -o pipefail


REPO_PATH="$(git rev-parse --show-toplevel)"

MISSPELL_LOC="${REPO_PATH}/hack/tools/bin"

# Spell checking
# misspell check Project - https://github.com/client9/misspell
misspellignore_files="${REPO_PATH}/hack/check/.misspellignore"
ignore_files=$(cat "${misspellignore_files}")
git ls-files | grep -v "${ignore_files}" | xargs "${MISSPELL_LOC}/misspell" | grep "misspelling" && echo "Please fix the listed misspell errors and verify using 'make misspell'" && exit 1 || echo "misspell check passed!"
