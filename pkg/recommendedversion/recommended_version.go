// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package recommendedversion is used to check for
// the currently recommended versions of the Tanzu CLI
// and inform the user if they are using an outdated version.
package recommendedversion

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo"
	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/datastore"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// RecommendedVersion is the data structure of a single recommended version.
// We use a struct so that we can add new fields in the future.
// An array of this struct is the format that must be stored
// in the central configuration and read back.
type RecommendedVersion struct {
	Version string `yaml:"version" json:"version"`
}

// dataStoreLastVersionCheckKey is the data store key used to store the last
// time the version check was done
const (
	centralConfigRecommendedVersionsKey = "cli.core.cli_recommended_versions"
	dataStoreLastVersionCheckKey        = "lastVersionCheck"
	recommendedVersionCheckDelaySeconds = 24 * 60 * 60 // 24 hours
)

// CheckRecommendedCLIVersion checks the recommended versions of the Tanzu CLI
// and prints recommendations to the user if they are using an outdated version.
// Once recommendations are printed to the user, the next check is only done after 24 hours.
func CheckRecommendedCLIVersion(cmd *cobra.Command) {
	if !shouldCheckVersion() {
		return
	}

	// Get the recommended versions from the default central configuration
	var versionStruct []RecommendedVersion
	err := centralconfig.DefaultCentralConfigReader.GetCentralConfigEntry(centralConfigRecommendedVersionsKey, &versionStruct)
	if err != nil {
		log.V(7).Error(err, "error reading recommended versions from central config")
		return
	}

	// Convert to a string array for easier processing since there is nothing else in the struct
	var recommendedVersions []string
	for _, rv := range versionStruct {
		recommendedVersions = append(recommendedVersions, rv.Version)
	}
	recommendedVersions, err = sortRecommendedVersionsDescending(recommendedVersions)
	if err != nil {
		log.V(7).Error(err, "failed to sort recommended versions")
		return
	}

	currentVersion := buildinfo.Version
	includePreReleases := utils.IsPreRelease(currentVersion)
	major := findRecommendedMajorVersion(recommendedVersions, currentVersion, includePreReleases)
	minor := findRecommendedMinorVersion(recommendedVersions, currentVersion, includePreReleases)
	patch := findRecommendedPatchVersion(recommendedVersions, currentVersion, includePreReleases)

	printVersionRecommendations(cmd.ErrOrStderr(), currentVersion, major, minor, patch)
}

// findRecommendedMajorVersion will return the recommended major version from the list of
// recommended versions. If the current version is already at the most recent major version,
// it will return an empty string.
func findRecommendedMajorVersion(recommendedVersions []string, currentVersion string, includePreReleases bool) string {
	for _, newVersion := range recommendedVersions {
		if !includePreReleases && utils.IsPreRelease(newVersion) {
			// Skip pre-release versions
			continue
		}

		// This is the most recent of all versions. If it is the same major
		// as the current version, then the current version is already the correct major version
		if utils.IsSameMajor(newVersion, currentVersion) {
			return ""
		}

		// If it is not a newer version than the current version
		// then there is no recommendation to give
		if !utils.IsNewVersion(newVersion, currentVersion) {
			return ""
		}

		return newVersion
	}
	return ""
}

// findRecommendedMinorVersion will return the recommended minor version from the list of
// recommended versions. If the current version is already at the most recent minor version,
// it will return an empty string.
func findRecommendedMinorVersion(recommendedVersions []string, currentVersion string, includePreReleases bool) string {
	for _, newVersion := range recommendedVersions {
		if !includePreReleases && utils.IsPreRelease(newVersion) {
			// Skip pre-release versions
			continue
		}

		// Since the recommended versions are sorted in descending order,
		// the first version that is the same major version as the current version
		// will be the most recent minor to recommend.
		if utils.IsSameMajor(newVersion, currentVersion) {
			// This is the most recent of version within the same major version.
			// If it is the same minor as the current version, then the current version
			// is already the correct minor version
			if utils.IsSameMinor(newVersion, currentVersion) {
				return ""
			}
			// If it is not a newer version than the current version
			// then there is no recommendation to give
			if !utils.IsNewVersion(newVersion, currentVersion) {
				return ""
			}

			return newVersion
		}
	}
	return ""
}

// findRecommendedPatchVersion will return the recommended patch version from the list of
// recommended versions. If the current version is already at that patch version,
// it will return an empty string.
func findRecommendedPatchVersion(recommendedVersions []string, currentVersion string, includePreReleases bool) string {
	for _, newVersion := range recommendedVersions {
		if !includePreReleases && utils.IsPreRelease(newVersion) {
			// Skip pre-release versions
			continue
		}

		// Since the recommended versions are sorted in descending order,
		// the first version that is the same minor version as the current version
		// will be the most recent patch to recommend.
		if utils.IsSameMinor(newVersion, currentVersion) {
			// If it is not a newer version than the current version
			// then there is no recommendation to give
			if !utils.IsNewVersion(newVersion, currentVersion) {
				return ""
			}

			return newVersion
		}
	}
	return ""
}

// sortRecommendedVersionsDescending will convert the array of recommended
// versions into an array sorted in descending order of semver
func sortRecommendedVersionsDescending(recommendedVersions []string) ([]string, error) {
	// Trim any spaces around the version strings and remove duplicates
	finalVersions := make([]string, 0, len(recommendedVersions))
	alreadyPresent := make(map[string]bool)
	for _, newVersion := range recommendedVersions {
		trimmedVersion := strings.TrimSpace(newVersion)
		if trimmedVersion != "" && !alreadyPresent[trimmedVersion] {
			finalVersions = append(finalVersions, trimmedVersion)
			alreadyPresent[trimmedVersion] = true
		}
	}

	// Now sort the versions, then reverse the order
	err := utils.SortVersions(finalVersions)
	if err != nil {
		return nil, err
	}

	// Reverse the order so it is descending
	for i := len(finalVersions)/2 - 1; i >= 0; i-- {
		opp := len(finalVersions) - 1 - i
		finalVersions[i], finalVersions[opp] = finalVersions[opp], finalVersions[i]
	}
	return finalVersions, err
}

func getRecommendationDelayInSeconds() int {
	delay := recommendedVersionCheckDelaySeconds
	delayOverride := os.Getenv(constants.ConfigVariableRecommendVersionDelayDays)
	if delayOverride != "" {
		delayOverrideValue, err := strconv.Atoi(delayOverride)
		if err == nil {
			if delayOverrideValue >= 0 {
				// Convert from days to seconds
				delay = delayOverrideValue * 24 * 60 * 60
			} else {
				// When the configured delay is negative, it means the value
				// should be in seconds.  This is used for testing purposes.
				delay = -delayOverrideValue
			}
		}
	}
	return delay
}

func shouldCheckVersion() bool {
	delay := getRecommendationDelayInSeconds()
	if delay == 0 {
		// The user has disabled the version check
		return false
	}

	// Get the last time the version check was done
	var lastCheck time.Time
	err := datastore.GetDataStoreValue(dataStoreLastVersionCheckKey, &lastCheck)
	if err != nil {
		return true
	}

	return time.Since(lastCheck) > time.Duration(delay)*time.Second
}

func printVersionRecommendations(writer io.Writer, currentVersion, _, minor, patch string) {
	// Only print the message for minor and patch versions.
	// We don't print anything for major versions because they are breaking changes
	// and it will take a while for the user to upgrade to a new major version.
	if (minor == "" || !utils.IsNewVersion(minor, currentVersion)) &&
		(patch == "" || !utils.IsNewVersion(patch, currentVersion)) {
		// The current version is the best recommended version
		return
	}

	// Put a delimiter before this notification so the user
	// can see it is not part of the command output
	fmt.Fprintln(writer, "\n==")
	fmt.Fprintf(writer, "Note: A new version of the Tanzu CLI is available. You are at version: %s.\n", currentVersion)
	fmt.Fprintln(writer, "To benefit from the latest security and features, please update to a recommended version:")

	if minor != "" {
		// Only print the recommended version if it is a newer version
		if utils.IsNewVersion(minor, currentVersion) {
			fmt.Fprintf(writer, "  - %s\n", minor)
		}
	}
	if patch != "" {
		// Only print the recommended version if it is a newer version
		if utils.IsNewVersion(patch, currentVersion) {
			fmt.Fprintf(writer, "  - %s\n", patch)
		}
	}

	fmt.Fprintf(writer, "\nPlease refer to these instructions for upgrading: https://github.com/vmware-tanzu/tanzu-cli/blob/main/docs/quickstart/install.md.\n")

	delay := getRecommendationDelayInSeconds()
	var delayStr string
	if delay >= 60*60 {
		// If the delay is more than an hour, show the delay in hours
		delayStr = fmt.Sprintf("%d hours", delay/60/60)
	} else {
		delayStr = fmt.Sprintf("%d seconds", delay)
	}
	fmt.Fprintf(writer, "\nThis message will print at most once per %s until you update the CLI.\n"+
		"Set %s to adjust this period (0 to disable).\n",
		delayStr, constants.ConfigVariableRecommendVersionDelayDays)

	// Now that we printed the message to the use, save the time of the last check
	// so that we don't continually print the message at every command
	_ = datastore.SetDataStoreValue(dataStoreLastVersionCheckKey, time.Now())
}
