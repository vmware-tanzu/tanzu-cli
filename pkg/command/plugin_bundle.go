// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/pkg/airgapped"
	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

type downloadPluginBundleOptions struct {
	pluginDiscoveryOCIImage string
	tarFile                 string
	groups                  []string
	dryRun                  bool
}

var dpbo downloadPluginBundleOptions

func newDownloadBundlePluginCmd() *cobra.Command {
	var downloadBundleCmd = &cobra.Command{
		Use:   "download-bundle",
		Short: "Download plugin bundle to the local system",
		Long: `Download a plugin bundle to the local file system to be used when migrating plugins
to an internet-restricted environment. Please also see the "upload-bundle" command.`,
		Example: `
    # Download a plugin bundle for a specific group version from the default discovery source
    tanzu plugin download-bundle --to-tar /tmp/plugin_bundle_vmware_tkg_default_v1.0.0.tar.gz --group vmware-tkg/default:v1.0.0

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
				DryRun:               dpbo.dryRun,
				ImageProcessor:       carvelhelpers.NewImageOperationsImpl(),
			}
			return options.DownloadPluginBundle()
		},
	}

	f := downloadBundleCmd.Flags()
	f.StringVarP(&dpbo.pluginDiscoveryOCIImage, "image", "", constants.TanzuCLIDefaultCentralPluginDiscoveryImage, "URI of the plugin discovery image providing the plugins")
	utils.PanicOnErr(downloadBundleCmd.RegisterFlagCompletionFunc("image", cobra.NoFileCompletions))

	// Shell completion for this flag is the default behavior of doing file completion
	f.StringVarP(&dpbo.tarFile, "to-tar", "", "", "local tar file path to store the plugin images")
	f.StringSliceVarP(&dpbo.groups, "group", "", []string{}, "only download the plugins specified in the plugin-group version (can specify multiple)")
	utils.PanicOnErr(downloadBundleCmd.RegisterFlagCompletionFunc("group", completeGroupsAndVersion))

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
		ValidArgsFunction: cobra.NoFileCompletions,
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
	utils.PanicOnErr(uploadBundleCmd.RegisterFlagCompletionFunc("to-repo", cobra.NoFileCompletions))

	_ = uploadBundleCmd.MarkFlagRequired("tar")
	_ = uploadBundleCmd.MarkFlagRequired("to-repo")

	return uploadBundleCmd
}

func completeDownloadBundle(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if !dpbo.dryRun && dpbo.tarFile == "" {
		// The user must provide more info by using flags.
		// Note that those flags are not marked as mandatory
		// because only one of the two --to-tar and --dry-run are required
		comps := []string{"--"}
		return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}

	// The user has provided enough information
	return nil, cobra.ShellCompDirectiveNoFileComp
}
