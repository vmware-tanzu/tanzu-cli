#!/bin/bash

# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# This script accepts a file directory path which has ginkgo generated test reports json files as an argument and processes the json files and generates a github flavored markdown table of Test results summary

# Save the current time in seconds
start_time=$(date +%s)

# Read the data from the file
file_data=$(cat $1)

# Function to remove ASCII escape sequences from a string
remove_escape_sequences() {
    echo -e "$1" | sed -E 's/\x1B\[([0-9]{1,2}(;[0-9]{1,2})*)?[mGK]//g'
}

# Function to extract numeric value from a line
extract_numeric_value() {
    echo "$1" | grep -oE '[0-9]+'
}

# Print the table header
#echo "| :memo: Test Suite And Path | Status | Total Tests | Passed | Failed | Pending | Skipped | Coverage"
#echo "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |"

# Process the data and extract the required information
ginkgo_suites=""
suite_coverage_per=""
suite_full_path=""
coverage_data=""
input_data=""

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

    # If the $cleaned_line has the coverage info then capture it.
    # input: "github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder	coverage: 6.6% of statements"
    # output: "6.6%"
    if echo "$cleaned_line" | grep -qE "coverage: [0-9.]+% of statements"; then
        input_data+=$cleaned_line
        suite_full_path=$(echo "$cleaned_line" | grep -oE "\t.*\tcoverage: [0-9.]+% of statements" | cut -f 2)
        suite_coverage_per=$(echo "$cleaned_line" | grep -oE "coverage: [0-9.]+% of statements" | cut -d ' ' -f 2)
        coverage_data+="converageInfo: package:$suite_full_path, coverage:$suite_coverage_per \n"
    fi

    # if the $cleaned_line has the "SUCCESS!" or "FAIL!" tests info then capture it
    # Check for lines with SUCCESS!, Passed, Failed, and Pending
    # eg: "SUCCESS! -- 23 Passed | 0 Failed | 0 Pending | 0 Skipped"
    if echo "$cleaned_line" | grep -qE "SUCCESS!.*Passed.*Failed.*Pending"; then
        input_data+=$cleaned_line
        passed=$(extract_numeric_value "$cleaned_line" | sed -n '1p')
        failed=$(extract_numeric_value "$cleaned_line" | sed -n '2p')
        pending=$(extract_numeric_value "$cleaned_line" | sed -n '3p')
        skipped=$(extract_numeric_value "$cleaned_line" | sed -n '4p')
        ginkgo_suites+="ginkgo-suite: package:$suite_full_path, Status: SUCCESS!, Passed: $passed, Failed: $failed, Pending: $pending\n"
    elif echo "$cleaned_line" | grep -qE "FAIL!.*Passed.*Failed.*Pending"; then
        input_data+=$cleaned_line
        passed=$(extract_numeric_value "$cleaned_line" | sed -n '1p')
        failed=$(extract_numeric_value "$cleaned_line" | sed -n '2p')
        pending=$(extract_numeric_value "$cleaned_line" | sed -n '3p')
        skipped=$(extract_numeric_value "$cleaned_line" | sed -n '4p')
        ginkgo_suites+="ginkgo-suite: package:$suite_full_path, Status: FAILED!, Passed: $passed, Failed: $failed, Pending: $pending, Skipped: $skipped\n"
    fi
    
done <<< "$file_data"

# Calculate the time take to process in seconds
end_time=$(date +%s)
elapsed_time=$((end_time - start_time))
minutes=$((elapsed_time / 60))
seconds=$((elapsed_time % 60))
echo "Time taken: $minutes minutes and $seconds seconds"


# Print the final processed data
echo -e "$ginkgo_suites"
echo -e "$coverage_data"
echo -e "$input_data"