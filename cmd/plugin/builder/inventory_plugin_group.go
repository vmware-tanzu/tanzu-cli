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
func newInventoryPluginGroupCmd() *cobra.Command {
	var inventoryPluginCmd = &cobra.Command{
		Use:   "plugin-group",
		Short: "Plugin-Group Inventory Operations",
	}

	inventoryPluginCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	inventoryPluginCmd.AddCommand(
		newInventoryPluginGroupAddCmd(),
		newInventoryPluginGroupActivateCmd(),
		newInventoryPluginGroupDeactivateCmd(),
	)

	return inventoryPluginCmd
}

type inventoryPluginGroupAddFlags struct {
	GroupName             string
	Repository            string
	InventoryImageTag     string
	ManifestFile          string
	Publisher             string
	Vendor                string
	DeactivatePluginGroup bool
	Override              bool
}

func newInventoryPluginGroupAddCmd() *cobra.Command {
	var ipgaFlags = &inventoryPluginGroupAddFlags{}

	var pluginGroupAddCmd = &cobra.Command{
		Use:          "add",
		Short:        "Add the plugin-group to the inventory database available on the remote repository",
		SilenceUsage: true,
		Example:      ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			pgaOptions := inventory.InventoryPluginGroupUpdateOptions{
				GroupName:               ipgaFlags.GroupName,
				Repository:              ipgaFlags.Repository,
				InventoryImageTag:       ipgaFlags.InventoryImageTag,
				PluginGroupManifestFile: ipgaFlags.ManifestFile,
				Vendor:                  ipgaFlags.Vendor,
				Publisher:               ipgaFlags.Publisher,
				DeactivatePluginGroup:   ipgaFlags.DeactivatePluginGroup,
				Override:                ipgaFlags.Override,
				ImgpkgOptions:           imgpkg.NewImgpkgCLIWrapper(),
			}
			return pgaOptions.PluginGroupAdd()
		},
	}

	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.GroupName, "name", "", "", "name of the plugin group")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.Repository, "repository", "", "", "repository to publish plugin inventory image")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.InventoryImageTag, "plugin-inventory-image-tag", "", "latest", "tag to which plugin inventory image needs to be published")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.ManifestFile, "manifest", "", "", "manifest file specifying plugin-group details that needs to be processed")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.Vendor, "vendor", "", "", "name of the vendor")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.Publisher, "publisher", "", "", "name of the publisher")
	pluginGroupAddCmd.Flags().BoolVarP(&ipgaFlags.DeactivatePluginGroup, "deactivate", "", false, "mark plugin-group as deactivated")
	pluginGroupAddCmd.Flags().BoolVarP(&ipgaFlags.Override, "override", "", false, "override the plugin-group if already exists")

	_ = pluginGroupAddCmd.MarkFlagRequired("name")
	_ = pluginGroupAddCmd.MarkFlagRequired("repository")
	_ = pluginGroupAddCmd.MarkFlagRequired("vendor")
	_ = pluginGroupAddCmd.MarkFlagRequired("publisher")
	_ = pluginGroupAddCmd.MarkFlagRequired("manifest")

	return pluginGroupAddCmd
}

type inventoryPluginGroupActivateDeactivateFlags struct {
	GroupName         string
	Repository        string
	InventoryImageTag string
	ManifestFile      string
	Publisher         string
	Vendor            string
}

func newInventoryPluginGroupActivateCmd() *cobra.Command {
	pluginGroupActivateCmd, flags := getPluginGroupActivateDeactivateBaseCmd()
	pluginGroupActivateCmd.Use = "activate"
	pluginGroupActivateCmd.Short = "Activate the existing plugin-group in the inventory database available on the remote repository"
	pluginGroupActivateCmd.Example = ""
	pluginGroupActivateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		pguOptions := inventory.InventoryPluginGroupUpdateOptions{
			GroupName:             flags.GroupName,
			Repository:            flags.Repository,
			InventoryImageTag:     flags.InventoryImageTag,
			Vendor:                flags.Vendor,
			Publisher:             flags.Publisher,
			DeactivatePluginGroup: false,
			ImgpkgOptions:         imgpkg.NewImgpkgCLIWrapper(),
		}
		return pguOptions.UpdatePluginGroupActivationState()
	}
	return pluginGroupActivateCmd
}

func newInventoryPluginGroupDeactivateCmd() *cobra.Command {
	pluginGroupDeactivateCmd, flags := getPluginGroupActivateDeactivateBaseCmd()
	pluginGroupDeactivateCmd.Use = "deactivate"
	pluginGroupDeactivateCmd.Short = "Deactivate the existing plugin-group in the inventory database available on the remote repository"
	pluginGroupDeactivateCmd.Example = ""
	pluginGroupDeactivateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		pguOptions := inventory.InventoryPluginGroupUpdateOptions{
			GroupName:             flags.GroupName,
			Repository:            flags.Repository,
			InventoryImageTag:     flags.InventoryImageTag,
			Vendor:                flags.Vendor,
			Publisher:             flags.Publisher,
			DeactivatePluginGroup: true,
			ImgpkgOptions:         imgpkg.NewImgpkgCLIWrapper(),
		}
		return pguOptions.UpdatePluginGroupActivationState()
	}
	return pluginGroupDeactivateCmd
}

func getPluginGroupActivateDeactivateBaseCmd() (*cobra.Command, *inventoryPluginGroupActivateDeactivateFlags) { // nolint:dupl
	var flags = &inventoryPluginGroupActivateDeactivateFlags{}

	var activateDeactivateCmd = &cobra.Command{}
	activateDeactivateCmd.SilenceUsage = true

	activateDeactivateCmd.Flags().StringVarP(&flags.GroupName, "name", "", "", "name of the plugin group")
	activateDeactivateCmd.Flags().StringVarP(&flags.Repository, "repository", "", "", "repository to publish plugin inventory image")
	activateDeactivateCmd.Flags().StringVarP(&flags.InventoryImageTag, "plugin-inventory-image-tag", "", "latest", "tag to which plugin inventory image needs to be published")
	activateDeactivateCmd.Flags().StringVarP(&flags.Vendor, "vendor", "", "", "name of the vendor")
	activateDeactivateCmd.Flags().StringVarP(&flags.Publisher, "publisher", "", "", "name of the publisher")

	_ = activateDeactivateCmd.MarkFlagRequired("name")
	_ = activateDeactivateCmd.MarkFlagRequired("repository")
	_ = activateDeactivateCmd.MarkFlagRequired("vendor")
	_ = activateDeactivateCmd.MarkFlagRequired("publisher")

	return activateDeactivateCmd, flags
}
