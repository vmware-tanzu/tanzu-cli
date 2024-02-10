#!/bin/bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# This script accepts a file directory path which has ginkgo generated test reports json files as an argument and processes the json files and generates a github flavored markdown table of Test results summary

if [ $# -eq 0 ]
then
    echo "Usage: $0 ginkgo generated test report json is required"
    exit 1
fi

# Accepts file directory as an argument, which has ginkgo test results files in json format
# Usage:  sh process-ginkgo-test-results.sh ./testresults

# Print the table header
echo "| :memo: Test Suite Description | Total Tests | Passed | Failed | Skipped |"
echo "| --- | ---: | ---: | ---: | ---: |"

# Counters for total tests
total_tests=0
total_passed=0
total_failed=0
total_skipped=0

# Loop through each file in the given directory
for file in `ls $1`; do
  json=$(cat "$1/$file")
  # Loop through each suite
  for suite in $(echo "$json" | jq -r '.[] | @base64'); do
    suite_json=$(echo "$suite" | base64 --decode | jq -r '.')

    # Get suite description
    suite_description=$(echo "$suite_json" | jq -r '.SuiteDescription')

    # Counters for suite tests
    suite_tests=0
    suite_passed=0
    suite_failed=0
    suite_skipped=0

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
      elif [ "$state" == "failed" ]; then
        ((suite_failed++))
        ((total_failed++))
      elif [ "$state" == "skipped" ]; then
        ((suite_skipped++))
        ((total_skipped++))
      fi
    done

    # Print the suite row with color and icon depending on the result
    if [ "$suite_failed" -eq 0 ]; then
      echo "| $suite_description | $suite_tests | $suite_passed | $suite_failed | $suite_skipped |"
    else
      echo "| $suite_description | $suite_tests | $suite_passed | :x: $suite_failed | $suite_skipped |"
    fi

  done
done
# Print the total line with color and icon depending on the result
if [ "$total_failed" -eq 0 ]; then
  echo "| **Total** | **$total_tests** | **$total_passed** | **$total_failed** | **$total_skipped** |"
else
  echo "| **Total** | **$total_tests** | **$total_passed** | :x: **$total_failed** | **$total_skipped** |"
fi
