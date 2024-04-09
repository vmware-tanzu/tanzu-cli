// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/pkg/airgapped"
	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

type downloadPluginBundleOptions struct {
	pluginDiscoveryOCIImage string
	tarFile                 string
	groups                  []string
	plugins                 []string
	dryRun                  bool
}

var (
	dpbo downloadPluginBundleOptions
	// Can be overridden for unit testing
	imageProcessorForDownloadBundleComp = carvelhelpers.NewImageOperationsImpl()
)

func newDownloadBundlePluginCmd() *cobra.Command {
	var downloadBundleCmd = &cobra.Command{
		Use:   "download-bundle",
		Short: "Download plugin bundle to the local system",
		Long: `Download a plugin bundle to the local file system to be used when migrating plugins
to an internet-restricted environment. Please also see the "upload-bundle" command.`,
		Example: `
    # Download a plugin bundle for a specific group version from the default discovery source
    tanzu plugin download-bundle --group vmware-tkg/default:v1.0.0 --to-tar /tmp/plugin_bundle_vmware_tkg_default_v1.0.0.tar.gz

    # To download plugin bundle with a specific plugin from the default discovery source
    #     --plugin name                 : Downloads the latest available version of the plugin. (Returns an error if the specified plugin name is available across multiple targets)
    #     --plugin name:version         : Downloads the specified version of the plugin. (Returns an error if the specified plugin name is available across multiple targets)
    #     --plugin name@target:version  : Downloads the specified version of the plugin for the specified target.
    #     --plugin name@target          : Downloads the latest available version of the plugin for the specified target.
    tanzu plugin download-bundle --plugin cluster:v1.0.0 --to-tar /tmp/plugin_bundle_cluster.tar.gz

    # Download a plugin bundle with the entire plugin repository from a custom discovery source
    tanzu plugin download-bundle --image custom.registry.vmware.com/tkg/tanzu-plugins/plugin-inventory:latest --to-tar /tmp/plugin_bundle_complete.tar.gz`,
		ValidArgsFunction: completeDownloadBundle,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dpbo.dryRun && dpbo.tarFile == "" {
				return errors.New("flag '--to-tar' is required")
			}
			options := airgapped.DownloadPluginBundleOptions{
				PluginInventoryImage: dpbo.pluginDiscoveryOCIImage,
				ToTar:                dpbo.tarFile,
				Groups:               dpbo.groups,
				Plugins:              dpbo.plugins,
				DryRun:               dpbo.dryRun,
				ImageProcessor:       carvelhelpers.NewImageOperationsImpl(),
			}
			return options.DownloadPluginBundle()
		},
	}

	f := downloadBundleCmd.Flags()
	f.StringVarP(&dpbo.pluginDiscoveryOCIImage, "image", "", constants.TanzuCLIDefaultCentralPluginDiscoveryImage, "URI of the plugin discovery image providing the plugins")
	utils.PanicOnErr(downloadBundleCmd.RegisterFlagCompletionFunc("image", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cobra.AppendActiveHelp(nil, "Please enter the URI of the plugin discovery image providing the plugins"), cobra.ShellCompDirectiveNoFileComp
	}))

	// Shell completion for this flag is the default behavior of doing file completion
	f.StringVarP(&dpbo.tarFile, "to-tar", "", "", "local tar file path to store the plugin images")
	f.StringSliceVarP(&dpbo.groups, "group", "", []string{}, "only download the plugins specified in the plugin-group version (can specify multiple)")
	utils.PanicOnErr(downloadBundleCmd.RegisterFlagCompletionFunc("group", completeGroupsAndVersionForBundleDownload))

	f.StringSliceVarP(&dpbo.plugins, "plugin", "", []string{}, "only download plugins matching specified pluginID. Format: name/name:version/name@target:version (can specify multiple)")
	utils.PanicOnErr(downloadBundleCmd.RegisterFlagCompletionFunc("plugin", completePluginIDForBundleDownload))

	f.BoolVarP(&dpbo.dryRun, "dry-run", "", false, "perform a dry run by listing the images to download without actually downloading them")
	_ = downloadBundleCmd.Flags().MarkHidden("dry-run")

	// TODO(khouzam): Once using Cobra 1.8, we can use MarkFlagsOneRequired.
	// We can then adjust the shell completion as it will be handled by cobra
	downloadBundleCmd.MarkFlagsMutuallyExclusive("to-tar", "dry-run")

	return downloadBundleCmd
}

type uploadPluginBundleOptions struct {
	sourceTar       string
	destinationRepo string
}

var upbo uploadPluginBundleOptions

func newUploadBundlePluginCmd() *cobra.Command {
	var uploadBundleCmd = &cobra.Command{
		Use:   "upload-bundle",
		Short: "Upload plugin bundle to a repository",
		Long: `Upload a plugin bundle to an alternate container registry for use in an internet-restricted
environment. The plugin bundle is obtained using the "download-bundle" command.`,
		Example: `
    # Upload the plugin bundle to the remote repository
    tanzu plugin upload-bundle --tar /tmp/plugin_bundle_vmware_tkg_default_v1.0.0.tar.gz --to-repo custom.registry.company.com/tanzu-plugins/
    tanzu plugin upload-bundle --tar /tmp/plugin_bundle_complete.tar.gz --to-repo custom.registry.company.com/tanzu-plugins/`,
		ValidArgsFunction: completeUploadBundle,
		RunE: func(cmd *cobra.Command, args []string) error {
			options := airgapped.UploadPluginBundleOptions{
				Tar:             upbo.sourceTar,
				DestinationRepo: upbo.destinationRepo,
				ImageProcessor:  carvelhelpers.NewImageOperationsImpl(),
			}
			return options.UploadPluginBundle()
		},
	}

	f := uploadBundleCmd.Flags()

	// Shell completion for this flag is the default behavior of doing file completion
	f.StringVarP(&upbo.sourceTar, "tar", "", "", "source tar file")
	f.StringVarP(&upbo.destinationRepo, "to-repo", "", "", "destination repository for publishing plugins")
	utils.PanicOnErr(uploadBundleCmd.RegisterFlagCompletionFunc("to-repo", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cobra.AppendActiveHelp(nil, "Please enter the URI of the destination repository for publishing plugins"), cobra.ShellCompDirectiveNoFileComp
	}))

	_ = uploadBundleCmd.MarkFlagRequired("tar")
	_ = uploadBundleCmd.MarkFlagRequired("to-repo")

	return uploadBundleCmd
}

// ====================================
// Shell completion functions
// ====================================
func completeDownloadBundle(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	if !dpbo.dryRun && dpbo.tarFile == "" {
		// The user must provide more info by using flags.
		// Note that those flags are not marked as mandatory
		// because only one of the two --to-tar and --dry-run are required
		comps := []string{"--"}
		return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}

	// The user has provided enough information
	return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
}

func completeUploadBundle(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	if upbo.destinationRepo == "" || upbo.sourceTar == "" {
		// Both flags are required, so completion will be provided for them
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// The user has provided enough information
	return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
}

func completionDownloadInventoryImage() (string, error) {
	// For a download-bundle, we cannot use the DB cache.  This is because
	// the download-bundle does not use the configured plugin sources.  Instead it
	// uses the repo specified by the `--image`` flag, or, without the `--image` flag,
	// it uses the default central repo automatically.

	// We start by downloading the inventory of the required repo.  This is not
	// very fast, but there isn't much we can do about it.

	var err error
	tempDBDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}

	// Download the plugin inventory oci image to tempDBDir
	inventoryFile := filepath.Join(tempDBDir, plugininventory.SQliteDBFileName)
	if err := imageProcessorForDownloadBundleComp.DownloadImageAndSaveFilesToDir(dpbo.pluginDiscoveryOCIImage, filepath.Dir(inventoryFile)); err != nil {
		return "", err
	}

	return inventoryFile, nil
}

func completeGroupVersionsForDownloadBundle(groups []*plugininventory.PluginGroup, id, _ string) []string {
	var group *plugininventory.PluginGroup
	for _, g := range groups {
		if plugininventory.PluginGroupToID(g) == id {
			group = g
			break
		}
	}

	if group == nil {
		return cobra.AppendActiveHelp(nil, fmt.Sprintf("There is no group named: '%s'", id))
	}

	// Since more recent versions are more likely to be
	// useful, we return the list of versions in reverse order
	var versions []string
	for v := range group.Versions {
		versions = append(versions, v)
	}
	// Sort in ascending order
	_ = utils.SortVersions(versions)

	// Create the completions in reverse order
	comps := make([]string, len(versions))
	for i := range versions {
		comps[len(versions)-1-i] = fmt.Sprintf("%s:%s", id, versions[i])
	}
	return comps
}

func completionGetPluginGroupsForBundleDownload() ([]*plugininventory.PluginGroup, error) {
	inventoryFile, err := completionDownloadInventoryImage()
	defer os.RemoveAll(inventoryFile)
	if err != nil {
		return nil, err
	}

	// Read the plugin inventory database to read the plugin groups it contains
	pi := plugininventory.NewSQLiteInventory(inventoryFile, path.Dir(dpbo.pluginDiscoveryOCIImage))
	pluginGroups, err := pi.GetPluginGroups(plugininventory.PluginGroupFilter{IncludeHidden: true}) // Include the hidden plugin groups during plugin migration
	if err != nil {
		return nil, err
	}
	return pluginGroups, err
}

func completeGroupsAndVersionForBundleDownload(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) { //nolint:dupl
	pluginGroups, err := completionGetPluginGroupsForBundleDownload()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if pluginGroups == nil {
		comps := cobra.AppendActiveHelp(nil, fmt.Sprintf("There are no plugin groups in %s", dpbo.pluginDiscoveryOCIImage))
		return comps, cobra.ShellCompDirectiveNoFileComp
	}

	if idx := strings.Index(toComplete, ":"); idx != -1 {
		// The gID is already specified before the :
		// so now we should complete the group version.
		// Since more recent versions are more likely to be
		// useful, we tell the shell to preserve the order
		// using cobra.ShellCompDirectiveKeepOrder
		gID := toComplete[:idx]
		versionToComplete := toComplete[idx+1:]

		return completeGroupVersionsForDownloadBundle(pluginGroups, gID, versionToComplete), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	}

	// Complete plugin group names
	var comps []string
	for _, g := range pluginGroups {
		comps = append(comps, fmt.Sprintf("%s\t%s", plugininventory.PluginGroupToID(g), g.Description))
	}

	// Sort to allow for testing
	sort.Strings(comps)

	// Don't add a space after the group name so the uer can add a : if
	// they want to specify a version.
	return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

func completeVersionInPluginIDForDownloadBundle(plugins []*plugininventory.PluginInventoryEntry, id, _ string) []string {
	var matchingPluginEntries []*plugininventory.PluginInventoryEntry
	for _, p := range plugins {
		if plugininventory.PluginToID(p) == id {
			matchingPluginEntries = append(matchingPluginEntries, p)
			break
		}
	}

	if len(matchingPluginEntries) == 0 {
		return cobra.AppendActiveHelp(nil, fmt.Sprintf("There are no plugins matching: '%s'", id))
	}

	// Since more recent versions are more likely to be
	// useful, we return the list of versions in reverse order
	var versions []string
	for _, pe := range matchingPluginEntries {
		for v := range pe.Artifacts {
			versions = append(versions, v)
		}
	}
	// Sort in ascending order
	_ = utils.SortVersions(versions)

	// Create the completions in reverse order
	comps := make([]string, len(versions))
	for i := range versions {
		comps[len(versions)-1-i] = fmt.Sprintf("%s:%s", id, versions[i])
	}
	return comps
}

func completionGetPluginsForBundleDownload() ([]*plugininventory.PluginInventoryEntry, error) {
	inventoryFile, err := completionDownloadInventoryImage()
	defer os.RemoveAll(inventoryFile)
	if err != nil {
		return nil, err
	}

	// Read the plugin inventory database to read the plugins it contains
	pi := plugininventory.NewSQLiteInventory(inventoryFile, path.Dir(dpbo.pluginDiscoveryOCIImage))
	pluginEntries, err := pi.GetPlugins(&plugininventory.PluginInventoryFilter{IncludeHidden: true}) // Include the hidden plugin groups during plugin migration
	if err != nil {
		return nil, err
	}
	return pluginEntries, err
}

func completePluginIDForBundleDownload(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) { //nolint:dupl
	pluginEntries, err := completionGetPluginsForBundleDownload()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if pluginEntries == nil {
		comps := cobra.AppendActiveHelp(nil, fmt.Sprintf("There are no plugins in %s", dpbo.pluginDiscoveryOCIImage))
		return comps, cobra.ShellCompDirectiveNoFileComp
	}

	if idx := strings.Index(toComplete, ":"); idx != -1 {
		// The pluginID is already specified before the :
		// so now we should complete the plugin version.
		// Since more recent versions are more likely to be
		// useful, we tell the shell to preserve the order
		// using cobra.ShellCompDirectiveKeepOrder
		pluginID := toComplete[:idx]
		versionToComplete := toComplete[idx+1:]

		return completeVersionInPluginIDForDownloadBundle(pluginEntries, pluginID, versionToComplete), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	}

	// Complete pluginIDs
	var comps []string
	for _, p := range pluginEntries {
		comps = append(comps, fmt.Sprintf("%s\t%s", plugininventory.PluginToID(p), p.Description))
	}

	// Sort to allow for testing
	sort.Strings(comps)

	// Don't add a space after the pluginID so the uer can add a : if
	// they want to specify a version.
	return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}
