#!/bin/bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# This script accepts a file directory path which has ginkgo generated test reports json files as an argument and processes the json files and generates a github flavored markdown table of Test results summary

# Read the data from the file
echo "loading data from $1"
file_data=$(cat $1)

echo "done loading data from $1"

# Function to remove ASCII escape sequences
remove_escape_sequences_a() {
    echo "$1" | sed -E 's/\x1B\[[0-9;]*[a-zA-Z]//g'
}

# Function to remove ASCII escape sequences from a string
remove_escape_sequences() {
    echo -e "$1" | sed -E 's/\x1B\[([0-9]{1,2}(;[0-9]{1,2})*)?[mGK]//g'
}

# Function to extract numeric value from a line
extract_numeric_value() {
    echo "$1" | grep -oE '[0-9]+'
}

# Print the table header
echo "| :memo: Test Suite And Path | Status | Total Tests | Passed | Failed | Pending | Skipped | Coverage"
echo "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |"

# Process the data and extract the required information
processed_data=""
coverageInfo=""
coverage=""
suite_path=""
suite_coverage_per=""
while read -r line; do
    # Remove ASCII escape sequences
    #cleaned_line=$(remove_escape_sequences "$line")
    # Extract the "Output" value from each line
    output=$(echo "$line" | jq -r '.Output')
    testSuite=$(echo "$line" | jq -r '.Test')

    # Remove escape sequences from the "Output" value
    cleaned_line=$(remove_escape_sequences "$output")
    if echo "$cleaned_line" | grep -qE "coverage: [0-9.]+% of statements"; then
        suite_full_path=$(echo "$cleaned_line" | grep -oE "\t.*\tcoverage: [0-9.]+% of statements" | cut -f 2)
        suite_coverage_per=$(echo "$cleaned_line" | grep -oE "coverage: [0-9.]+% of statements" | cut -d ' ' -f 2)
        #coverageInfo="Coverage Path: $coverage_path, Coverage: $coverage\n"
        # Check if suite_full_path starts with "github.com/vmware-tanzu/tanzu-cli"
        if [[ "$suite_full_path" == github.com/vmware-tanzu/tanzu-cli* ]]; then
            suite_path="${suite_full_path#github.com/vmware-tanzu/tanzu-cli}"
        else
            suite_path="$suite_full_path"
        fi
    fi
    # Check for lines with SUCCESS!, Passed, Failed, and Pending
    if echo "$cleaned_line" | grep -qE "SUCCESS!.*Passed.*Failed.*Pending"; then
        passed=$(extract_numeric_value "$cleaned_line" | sed -n '1p')
        failed=$(extract_numeric_value "$cleaned_line" | sed -n '2p')
        pending=$(extract_numeric_value "$cleaned_line" | sed -n '3p')
        skipped=$(extract_numeric_value "$cleaned_line" | sed -n '4p')
        #test_name=$(echo "$cleaned_line" | grep -oE '"Test":"[^"]+"' | cut -d '"' -f 4)
        #echo "Test: $test_name, Status: SUCCESS!, Passed: $passed, Failed: $failed, Pending: $pending\n"
        processed_data+="Test: $test_name, Status: SUCCESS!, Passed: $passed, Failed: $failed, Pending: $pending\n"
        if [ -n "$coverageInfo" ]; then
          #echo $coverageInfo
          processed_data+=$coverageInfo
        fi
        coverageInfo=""
        tt=$((passed + failed + pending + skipped))
        echo "| "$testSuite\n$suite_path" | "SUCCESS!" | $tt | $passed | $failed | $pending | $skipped | $suite_coverage_per"
    elif echo "$cleaned_line" | grep -qE "FAIL!.*Passed.*Failed.*Pending"; then
        passed=$(extract_numeric_value "$cleaned_line" | sed -n '1p')
        failed=$(extract_numeric_value "$cleaned_line" | sed -n '2p')
        pending=$(extract_numeric_value "$cleaned_line" | sed -n '3p')
        skipped=$(extract_numeric_value "$cleaned_line" | sed -n '4p')
        #test_name=$(echo "$cleaned_line" | grep -oE '"Test":"[^"]+"' | cut -d '"' -f 4)
        #echo "Test: $test_name, Status: FAILED!, Passed: $passed, Failed: $failed, Pending: $pending, Skipped: $skipped\n"
        processed_data+="Test: $test_name, Status: FAILED!, Passed: $passed, Failed: $failed, Pending: $pending, Skipped: $skipped\n"
        if [ -n "$coverageInfo" ]; then
          #echo $coverageInfo
          processed_data+=$coverageInfo
        fi
        coverageInfo=""
        tt=$((passed + failed + pending + skipped))
        echo "| "$testSuite\n$suite_path" | :x: "FAILED!" | $tt | $passed | $failed | $pending | $skipped | $suite_coverage_per"
    fi
done <<< "$file_data"

# Print the final processed data
#echo -e "$processed_data"
