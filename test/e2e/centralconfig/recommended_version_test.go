// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package central_config_e2e_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/recommendedversion"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func deleteCLIDataStoreFile() error {
	homeDir, _ := os.UserHomeDir()
	datastore := filepath.Join(homeDir, framework.ConfigFileDir, ".data-store.yaml")
	_, err := os.Stat(datastore)
	if err == nil {
		if fileErr := os.Remove(datastore); fileErr != nil {
			return fileErr
		}
	}
	return nil
}

// These tests verify the recommended version feature of the CLI.
// This feature should print a notification to the user if the current version
// of the CLI is not the recommended version.
var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Recommended-version]", func() {
	const (
		olderVersion = "v0.9.9"
		newerVersion = "v1.9.9"
	)
	var (
		tf                       *framework.Framework
		recommendedOlderVersions = []recommendedversion.RecommendedVersion{
			{Version: olderVersion},
			{Version: "v0.8.8"},
		}
		recommendedNewerVersions = []recommendedversion.RecommendedVersion{
			{Version: "1.8.8"},
			{Version: newerVersion},
		}
	)
	BeforeEach(func() {
		tf = framework.NewFramework()
	})
	Context("tests for the recommended version feature", func() {
		When("there is no data store", func() {
			BeforeEach(func() {
				// Remove the data store file
				err := deleteCLIDataStoreFile()
				Expect(err).To(BeNil())
			})
			It("print the recommended notification for a newer version", func() {
				// Use a version we are sure is higher than the current CLI version
				updateRecommendedVersions(recommendedNewerVersions)

				// Run any command to trigger the recommended version check
				_, _, errStream, err := tf.PluginCmd.ListPlugins()
				Expect(err).To(BeNil())
				Expect(errStream).To(ContainSubstring("Note: A new version of the Tanzu CLI is available"))
				Expect(errStream).To(ContainSubstring(newerVersion))
			})
			It("do not print the recommended notification when the feature is disabled", func() {
				// Use a version we are sure is higher than the current CLI version
				updateRecommendedVersions(recommendedNewerVersions)

				// Set a 0 delay to disable the feature
				os.Setenv("TANZU_CLI_RECOMMEND_VERSION_DELAY_DAYS", "0")

				_, _, errStream, err := tf.PluginCmd.ListPlugins()
				Expect(err).To(BeNil())
				Expect(errStream).To(BeEmpty())
			})
			It("do not print any recommended notification for an older version", func() {
				// Use a version we are sure is lower than the current CLI version
				updateRecommendedVersions(recommendedOlderVersions)

				// Run any command to trigger the recommended version check
				_, _, errStream, err := tf.PluginCmd.ListPlugins()
				Expect(err).To(BeNil())
				Expect(errStream).To(BeEmpty())
			})
		})
		When("there is a data store", func() {
			It("print the recommended notification for a newer version after the delay", func() {
				// Use a version we are sure is higher than the current CLI version
				updateRecommendedVersions(recommendedNewerVersions)

				// Run any command to trigger the recommended version check
				_, _, _, err := tf.PluginCmd.ListPlugins()
				Expect(err).To(BeNil())
				// Run the command again to see that the notification is not printed again
				_, _, errStream, err := tf.PluginCmd.ListPlugins()
				Expect(err).To(BeNil())
				Expect(errStream).ToNot(ContainSubstring("Note: A new version of the Tanzu CLI is available"))
				Expect(errStream).ToNot(ContainSubstring(newerVersion))

				// Now set a low delay so we can test the notification is printed again.
				// Negative values mean a delay in seconds instead of days.
				os.Setenv("TANZU_CLI_RECOMMEND_VERSION_DELAY_DAYS", "-1")
				time.Sleep(time.Second * 1)

				_, _, errStream, err = tf.PluginCmd.ListPlugins()
				Expect(err).To(BeNil())
				Expect(errStream).To(ContainSubstring("Note: A new version of the Tanzu CLI is available"))
				Expect(errStream).To(ContainSubstring(newerVersion))
			})
		})
		When("the format has changed", func() {
			It("print the recommended notification even with a (backwards-compatible) change in format", func() {
				// Create a new recommended version format
				type newFormat struct {
					Version     string
					Description string
				}
				newFormatVersions := []newFormat{}
				for _, v := range recommendedNewerVersions {
					newFormatVersions = append(newFormatVersions, newFormat{
						Version:     v.Version,
						Description: "This is a new recommended version",
					})
				}
				// Use a version we are sure is higher than the current CLI version
				updateRecommendedVersions(newFormatVersions)

				// Remove the data store file
				err := deleteCLIDataStoreFile()
				Expect(err).To(BeNil())

				// Run any command to trigger the recommended version check
				_, _, errStream, err := tf.PluginCmd.ListPlugins()
				Expect(err).To(BeNil())
				Expect(errStream).To(ContainSubstring("Note: A new version of the Tanzu CLI is available"))
				Expect(errStream).To(ContainSubstring(newerVersion))
			})
		})
	})
})

func updateRecommendedVersions(recommended any) {
	testCentralConfigFile := filepath.Join(framework.TestHomeDir, ".cache", "tanzu", common.PluginInventoryDirName, "default", "central_config.yaml")

	// Read the central config file so we can update it with the new recommended version string
	b, err := os.ReadFile(testCentralConfigFile)
	Expect(err).To(BeNil())

	var content map[string]interface{}
	err = yaml.Unmarshal(b, &content)
	Expect(err).To(BeNil())

	content["cli.core.cli_recommended_versions"] = recommended

	b, err = yaml.Marshal(content)
	Expect(err).To(BeNil())

	// Write the updated content back to the file
	err = os.WriteFile(testCentralConfigFile, b, 0644)
	Expect(err).To(BeNil())
}
