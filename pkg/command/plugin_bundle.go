// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/pkg/airgapped"
	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
)

type downloadPluginBundleOptions struct {
	pluginDiscoveryOCIImage string
	tarFile                 string
	groups                  []string
}

var dpbo downloadPluginBundleOptions

func newDownloadBundlePluginCmd() *cobra.Command {
	var downloadBundleCmd = &cobra.Command{
		Use:   "download-bundle",
		Short: "Download plugin bundle to the local system",
		Long:  "Download plugin bundle to the local system",
		RunE: func(cmd *cobra.Command, args []string) error {
			options := airgapped.DownloadPluginBundleOptions{
				PluginInventoryImage: dpbo.pluginDiscoveryOCIImage,
				ToTar:                dpbo.tarFile,
				Groups:               dpbo.groups,
				ImageProcessor:       carvelhelpers.NewImageOperationsImpl(),
			}
			return options.DownloadPluginBundle()
		},
	}

	f := downloadBundleCmd.Flags()
	f.StringVarP(&dpbo.pluginDiscoveryOCIImage, "image", "", "", "URI of the plugin discovery image providing the plugins")
	f.StringVarP(&dpbo.tarFile, "to-tar", "", "", "local tar file path to store the plugin images")
	f.StringArrayVarP(&dpbo.groups, "group", "", []string{}, "only download the plugins specified in the plugin group")

	_ = downloadBundleCmd.MarkFlagRequired("image")
	_ = downloadBundleCmd.MarkFlagRequired("to-tar")

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
		Long:  "Upload plugin bundle to a repository",
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
	f.StringVarP(&upbo.sourceTar, "tar", "", "", "source tar file")
	f.StringVarP(&upbo.destinationRepo, "to-repo", "", "", "destination repository for publishing plugins")

	_ = uploadBundleCmd.MarkFlagRequired("tar")
	_ = uploadBundleCmd.MarkFlagRequired("to-repo")

	return uploadBundleCmd
}
