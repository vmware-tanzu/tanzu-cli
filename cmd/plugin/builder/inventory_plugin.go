// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/imgpkg"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/inventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// newInventoryPluginCmd creates a new command for plugin inventory operations.
func newInventoryPluginCmd() *cobra.Command {
	var inventoryPluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Plugin Inventory Operations",
	}

	inventoryPluginCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	inventoryPluginCmd.AddCommand(
		newInventoryPluginAddCmd(),
		newInventoryPluginActivateCmd(),
		newInventoryPluginDeactivateCmd(),
	)

	return inventoryPluginCmd
}

type inventoryPluginAddFlags struct {
	Repository        string
	InventoryImageTag string
	ManifestFile      string
	Publisher         string
	Vendor            string
	DeactivatePlugins bool
	ValidateOnly      bool
}

func newInventoryPluginAddCmd() *cobra.Command {
	var ipaFlags = &inventoryPluginAddFlags{}

	var pluginAddCmd = &cobra.Command{
		Use:          "add",
		Short:        "Add the plugin to the inventory database available on the remote repository",
		SilenceUsage: true,
		Example:      ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			paOptions := inventory.InventoryPluginUpdateOptions{
				Repository:        ipaFlags.Repository,
				InventoryImageTag: ipaFlags.InventoryImageTag,
				ManifestFile:      ipaFlags.ManifestFile,
				Vendor:            ipaFlags.Vendor,
				Publisher:         ipaFlags.Publisher,
				DeactivatePlugins: ipaFlags.DeactivatePlugins,
				ValidateOnly:      ipaFlags.ValidateOnly,
				ImgpkgOptions:     imgpkg.NewImgpkgCLIWrapper(),
			}
			return paOptions.PluginAdd()
		},
	}

	pluginAddCmd.Flags().StringVarP(&ipaFlags.Repository, "repository", "", "", "repository to publish plugin inventory image")
	pluginAddCmd.Flags().StringVarP(&ipaFlags.InventoryImageTag, "plugin-inventory-image-tag", "", "latest", "tag to which plugin inventory image needs to be published")
	pluginAddCmd.Flags().StringVarP(&ipaFlags.ManifestFile, "manifest", "", "", "manifest file specifying plugin details that needs to be processed")
	pluginAddCmd.Flags().StringVarP(&ipaFlags.Vendor, "vendor", "", "", "name of the vendor")
	pluginAddCmd.Flags().StringVarP(&ipaFlags.Publisher, "publisher", "", "", "name of the publisher")
	pluginAddCmd.Flags().BoolVarP(&ipaFlags.DeactivatePlugins, "deactivate", "", false, "mark plugins as deactivated")
	pluginAddCmd.Flags().BoolVarP(&ipaFlags.ValidateOnly, "validate", "", false, "validate whether plugins already exists in the plugin inventory or not")

	_ = pluginAddCmd.MarkFlagRequired("repository")
	_ = pluginAddCmd.MarkFlagRequired("vendor")
	_ = pluginAddCmd.MarkFlagRequired("publisher")
	_ = pluginAddCmd.MarkFlagRequired("manifest")

	return pluginAddCmd
}

type inventoryPluginActivateDeactivateFlags struct {
	Repository        string
	InventoryImageTag string
	ManifestFile      string
	Publisher         string
	Vendor            string
}

func newInventoryPluginActivateCmd() *cobra.Command {
	pluginActivateCmd, flags := getActivateDeactivateBaseCmd()
	pluginActivateCmd.Use = "activate" // nolint:goconst
	pluginActivateCmd.Short = "Activate the existing plugin in the inventory database available on the remote repository"
	pluginActivateCmd.Example = ""
	pluginActivateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		piOptions := inventory.InventoryPluginUpdateOptions{
			Repository:        flags.Repository,
			InventoryImageTag: flags.InventoryImageTag,
			ManifestFile:      flags.ManifestFile,
			Vendor:            flags.Vendor,
			Publisher:         flags.Publisher,
			DeactivatePlugins: false,
			ImgpkgOptions:     imgpkg.NewImgpkgCLIWrapper(),
		}
		return piOptions.UpdatePluginActivationState()
	}
	return pluginActivateCmd
}

func newInventoryPluginDeactivateCmd() *cobra.Command {
	pluginDeactivateCmd, flags := getActivateDeactivateBaseCmd()
	pluginDeactivateCmd.Use = "deactivate" // nolint:goconst
	pluginDeactivateCmd.Short = "Deactivate the existing plugin in the inventory database available on the remote repository"
	pluginDeactivateCmd.Example = ""
	pluginDeactivateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		piOptions := inventory.InventoryPluginUpdateOptions{
			Repository:        flags.Repository,
			InventoryImageTag: flags.InventoryImageTag,
			ManifestFile:      flags.ManifestFile,
			Vendor:            flags.Vendor,
			Publisher:         flags.Publisher,
			DeactivatePlugins: true,
			ImgpkgOptions:     imgpkg.NewImgpkgCLIWrapper(),
		}
		return piOptions.UpdatePluginActivationState()
	}
	return pluginDeactivateCmd
}

func getActivateDeactivateBaseCmd() (*cobra.Command, *inventoryPluginActivateDeactivateFlags) { // nolint:dupl
	var flags = &inventoryPluginActivateDeactivateFlags{}

	var activateDeactivateCmd = &cobra.Command{}
	activateDeactivateCmd.SilenceUsage = true

	activateDeactivateCmd.Flags().StringVarP(&flags.Repository, "repository", "", "", "repository to publish plugin inventory image")
	activateDeactivateCmd.Flags().StringVarP(&flags.InventoryImageTag, "plugin-inventory-image-tag", "", "latest", "tag to which plugin inventory image needs to be published")
	activateDeactivateCmd.Flags().StringVarP(&flags.ManifestFile, "manifest", "", "", "manifest file specifying plugin details that needs to be processed")
	activateDeactivateCmd.Flags().StringVarP(&flags.Vendor, "vendor", "", "", "name of the vendor")
	activateDeactivateCmd.Flags().StringVarP(&flags.Publisher, "publisher", "", "", "name of the publisher")

	_ = activateDeactivateCmd.MarkFlagRequired("repository")
	_ = activateDeactivateCmd.MarkFlagRequired("vendor")
	_ = activateDeactivateCmd.MarkFlagRequired("publisher")
	_ = activateDeactivateCmd.MarkFlagRequired("manifest")

	return activateDeactivateCmd, flags
}
