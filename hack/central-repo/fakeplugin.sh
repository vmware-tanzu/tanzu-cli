#!/bin/bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# Minimally Viable Dummy Tanzu CLI 'Plugin'

info() {
   cat << EOF
{
  "name": "__NAME__",
  "target": "__TARGET__",
  "description": "__NAME__ functionality",
  "version": "__VERSION__",
  "buildSHA": "01234567",
  "group": "System",
  "hidden": false,
  "aliases": [],
  "completionType": 0
}
EOF
  exit 0
}

case "$1" in
    info)  $1 "$@";;
    env) env;;
    *) cat << EOF
Stub plugin "__NAME__" "__VERSION__" for "__TARGET__"
EOF
       ;;
esac
