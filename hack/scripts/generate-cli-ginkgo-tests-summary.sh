#!/bin/bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

#####################################################################
# Script Name: generate-cli-ginkgo-tests-summary.sh
# Description: This script takes output of "make test | go tool test2json" as input, process it and
#     generates ginkgo test summary (if any ginkgo test suites) and code coverage per package
# Created Date: 2023-07-21
#####################################################################

# Usage: make test | go tool test2json | generate-cli-ginkgo-tests-summary.sh

# Dependencies: None

# Save the current time in seconds to calculate the time taken to execute this script.
start_time=$(date +%s)

# Read the data from the file.
# The input file should be the output of "make test | go tool test2json".
# It contains the test cases execution output organized by package.
# The ginkgo tests are treated as one of the test cases, and their summary looks
# like this: SUCCESS! -- 16 Passed | 0 Failed | 0 Pending | 0 Skipped.
# At the end of each package's test execution, there will be coverage information
# like this: ok     github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier    coverage: 86.5% of statements.
# We need to process this $make_test_output per package, first extracting ginkgo tests, then extracting coverage information.
# Each package may have ginkgo tests or may not, but there should always be coverage information.
make_test_output=$(cat $1)

# Function to remove ASCII escape sequences from a string
remove_escape_sequences() {
    echo -e "$1" | sed -E 's/\x1B\[([0-9]{1,2}(;[0-9]{1,2})*)?[mGK]//g'
}

# Function to extract numeric value from a line
extract_numeric_value() {
    echo "$1" | grep -oE '[0-9]+'
}

# Process the data and extract the required information
ginkgo_tests_summary=""
ginkgo_suites=""
suite_coverage_per=""
suite_full_path=""
coverage_data=""

while read -r line; do
    # Extract the "Output" value from each line
    # eg: get Output:
    #     {"Action":"output","Output":"\tgithub.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder\tcoverage: 6.6% of statements\n"}
    #     {"Action":"output","Test":"TestInventorySuite","Output":"\u001b[38;5;10m\u001b[1mSUCCESS!\u001b[0m -- \u001b[38;5;10m\u001b[1m23 Passed\u001b[0m | \u001b[38;5;9m\u001b[1m0 Failed\u001b[0m | \u001b[38;5;11m\u001b[1m0 Pending\u001b[0m | \u001b[38;5;14m\u001b[1m0 Skipped\u001b[0m\n"
    output=$(echo "$line" | jq -r '.Output')

    # Remove ASCII escape sequences
    # input: 1. "\u001b[38;5;10m\u001b[1mSUCCESS!\u001b[0m -- \u001b[38;5;10m\u001b[1m23 Passed\u001b[0m | \u001b[38;5;9m\u001b[1m0 Failed\u001b[0m | \u001b[38;5;11m\u001b[1m0 Pending\u001b[0m | \u001b[38;5;14m\u001b[1m0 Skipped\u001b[0m\n"
    #        2. "\tgithub.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder\tcoverage: 6.6% of statements\n"
    # output:1. "SUCCESS! -- 23 Passed | 0 Failed | 0 Pending | 0 Skipped"
    #        2. "github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder	coverage: 6.6% of statements"
    cleaned_line=$(remove_escape_sequences "$output")

    # if the $cleaned_line has the "SUCCESS!" or "FAIL!" tests info then capture it
    # Check for lines with SUCCESS!, Passed, Failed, and Pending
    # eg: "SUCCESS! -- 23 Passed | 0 Failed | 0 Pending | 0 Skipped"
    if echo "$cleaned_line" | grep -qE "SUCCESS!.*Passed.*Failed.*Pending"; then
        passed=$(extract_numeric_value "$cleaned_line" | sed -n '1p')
        failed=$(extract_numeric_value "$cleaned_line" | sed -n '2p')
        pending=$(extract_numeric_value "$cleaned_line" | sed -n '3p')
        skipped=$(extract_numeric_value "$cleaned_line" | sed -n '4p')
        ginkgo_tests_summary="Status: SUCCESS!, Passed: $passed, Failed: $failed, Pending: $pending\n"
    elif echo "$cleaned_line" | grep -qE "FAIL!.*Passed.*Failed.*Pending"; then
        passed=$(extract_numeric_value "$cleaned_line" | sed -n '1p')
        failed=$(extract_numeric_value "$cleaned_line" | sed -n '2p')
        pending=$(extract_numeric_value "$cleaned_line" | sed -n '3p')
        skipped=$(extract_numeric_value "$cleaned_line" | sed -n '4p')
        ginkgo_tests_summary="Status: FAILED!, Passed: $passed, Failed: $failed, Pending: $pending, Skipped: $skipped\n"
    fi
    
    # If the $cleaned_line has the code coverage information then capture code coverage and package path
    # input: "ok  	github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder	coverage: 6.6% of statements"
    # capture "6.6%" and "github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder" as code coverage and package path
    if echo "$cleaned_line" | grep -qE "^ok[[:space:]]+.*coverage: [0-9.]+% of statements"; then
        suite_full_path=$(echo "$cleaned_line" | awk '{print $2}')
        suite_coverage_per=$(echo "$cleaned_line" | grep -oE "coverage: [0-9.]+% of statements" | cut -d ' ' -f 2)
        coverage_data+="converageInfo: package:$suite_full_path, coverage:$suite_coverage_per \n"
        # $ginkgo_tests_summary is not empty means, there are ginkgo tests executed in this package
        # capture ginkgo tests execution summary
        if [ -n "$ginkgo_tests_summary" ]; then
            ginkgo_suites+="ginkgo-suite: package:$suite_full_path, $ginkgo_tests_summary"
        fi
        ginkgo_tests_summary=""
        suite_full_path=""
    fi

done <<< "$make_test_output"

# Calculate the time take to process in seconds
end_time=$(date +%s)
elapsed_time=$((end_time - start_time))
minutes=$((elapsed_time / 60))
seconds=$((elapsed_time % 60))
echo "Time taken: $minutes minutes and $seconds seconds"

# Print the final processed data
echo -e "$ginkgo_suites"
echo -e "$coverage_data"