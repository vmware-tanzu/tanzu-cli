#!/bin/bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# This script accepts a ginkgo generated test reports json file as an argument and processes the json and generates a github flavored markdown table of Test results summary

if [ $# -eq 0 ]
then
    echo "Usage: $0 ginkgo generated test report json is required"
    exit 1
fi

# Accepts file as an argument
# Usage:  sh process-ginkgo-test-results.sh results.json

json=$(cat "$1")

# Print the table header
echo "| :memo: Test Suite Description | Total Tests | Passed | Failed |"
echo "| --- | ---: | ---: | ---: |"

# Counters for total tests
total_tests=0
total_passed=0
total_failed=0

# Loop through each suite
for suite in $(echo "$json" | jq -r '.[] | @base64'); do
  suite_json=$(echo "$suite" | base64 --decode | jq -r '.')

  # Get suite description
  suite_description=$(echo "$suite_json" | jq -r '.SuiteDescription')

  # Counters for suite tests
  suite_tests=0
  suite_passed=0
  suite_failed=0

  # Loop through each spec in the suite
  for spec in $(echo "$suite_json" | jq -r '.SpecReports[] | @base64'); do
    spec_json=$(echo "$spec" | base64 --decode | jq -r '.')
    state=$(echo "$spec_json" | jq -r '.State')

    # Increment counters
    ((suite_tests++))
    ((total_tests++))

    if [ "$state" == "passed" ]; then
      ((suite_passed++))
      ((total_passed++))
    else
      ((suite_failed++))
      ((total_failed++))
    fi
  done

  # Print the suite row with color and icon depending on the result
  if [ "$suite_failed" -eq 0 ]; then
    echo "| $suite_description | $suite_tests | $suite_passed | $suite_failed |"
  else
    echo "| $suite_description | $suite_tests | $suite_passed | :x: $suite_failed |"
  fi

done

# Print the total line with color and icon depending on the result
if [ "$total_failed" -eq 0 ]; then
  echo "| **Total** | **$total_tests** | **$total_passed** | **$total_failed** |"
else
  echo "| **Total** | **$total_tests** | **$total_passed** | :x: **$total_failed** |"
fi
