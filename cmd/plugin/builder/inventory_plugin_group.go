// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/inventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
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
	GroupVersion          string
	Description           string
	Repository            string
	InventoryImageTag     string
	ManifestFile          string
	Publisher             string
	Vendor                string
	InventoryDBFile       string
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
				GroupVersion:            ipgaFlags.GroupVersion,
				Description:             ipgaFlags.Description,
				Repository:              ipgaFlags.Repository,
				InventoryImageTag:       ipgaFlags.InventoryImageTag,
				PluginGroupManifestFile: ipgaFlags.ManifestFile,
				Vendor:                  ipgaFlags.Vendor,
				Publisher:               ipgaFlags.Publisher,
				InventoryDBFile:         ipgaFlags.InventoryDBFile,
				DeactivatePluginGroup:   ipgaFlags.DeactivatePluginGroup,
				Override:                ipgaFlags.Override,

				ImageOperationsImpl: carvelhelpers.NewImageOperationsImpl(),
			}
			return pgaOptions.PluginGroupAdd()
		},
	}

	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.GroupName, "name", "", "", "name of the plugin-group")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.GroupVersion, "version", "", "", "version of the plugin-group")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.Description, "description", "", "", "a description for the plugin-group")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.Repository, "repository", "", "", "repository to publish plugin inventory image")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.InventoryImageTag, "plugin-inventory-image-tag", "", "latest", "tag to which plugin inventory image needs to be published")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.ManifestFile, "manifest", "", "", "manifest file specifying plugin-group details that needs to be processed")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.Vendor, "vendor", "", "", "name of the vendor")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.Publisher, "publisher", "", "", "name of the publisher")
	pluginGroupAddCmd.Flags().StringVarP(&ipgaFlags.InventoryDBFile, "plugin-inventory-db-file", "", "", "local file for the inventory database")
	pluginGroupAddCmd.Flags().BoolVarP(&ipgaFlags.DeactivatePluginGroup, "deactivate", "", false, "mark plugin-group as deactivated")
	pluginGroupAddCmd.Flags().BoolVarP(&ipgaFlags.Override, "override", "", false, "overwrite the plugin-group version if it already exists")

	_ = pluginGroupAddCmd.MarkFlagRequired("name")
	_ = pluginGroupAddCmd.MarkFlagRequired("version")
	_ = pluginGroupAddCmd.MarkFlagRequired("vendor")
	_ = pluginGroupAddCmd.MarkFlagRequired("publisher")
	_ = pluginGroupAddCmd.MarkFlagRequired("manifest")

	return pluginGroupAddCmd
}

type inventoryPluginGroupActivateDeactivateFlags struct {
	GroupName         string
	GroupVersion      string
	Repository        string
	InventoryImageTag string
	ManifestFile      string
	Publisher         string
	Vendor            string
	InventoryDBFile   string
}

func newInventoryPluginGroupActivateCmd() *cobra.Command { //nolint:dupl
	pluginGroupActivateCmd, flags := getPluginGroupActivateDeactivateBaseCmd()
	pluginGroupActivateCmd.Use = "activate"
	pluginGroupActivateCmd.Short = "Activate the existing plugin-group in the inventory database available on the remote repository"
	pluginGroupActivateCmd.Example = ""
	pluginGroupActivateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		pguOptions := inventory.InventoryPluginGroupUpdateOptions{
			GroupName:             flags.GroupName,
			GroupVersion:          flags.GroupVersion,
			Repository:            flags.Repository,
			InventoryImageTag:     flags.InventoryImageTag,
			Vendor:                flags.Vendor,
			Publisher:             flags.Publisher,
			InventoryDBFile:       flags.InventoryDBFile,
			DeactivatePluginGroup: false,
			ImageOperationsImpl:   carvelhelpers.NewImageOperationsImpl(),
		}
		return pguOptions.UpdatePluginGroupActivationState()
	}
	return pluginGroupActivateCmd
}

func newInventoryPluginGroupDeactivateCmd() *cobra.Command { //nolint:dupl
	pluginGroupDeactivateCmd, flags := getPluginGroupActivateDeactivateBaseCmd()
	pluginGroupDeactivateCmd.Use = "deactivate"
	pluginGroupDeactivateCmd.Short = "Deactivate the existing plugin-group in the inventory database available on the remote repository"
	pluginGroupDeactivateCmd.Example = ""
	pluginGroupDeactivateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		pguOptions := inventory.InventoryPluginGroupUpdateOptions{
			GroupName:             flags.GroupName,
			GroupVersion:          flags.GroupVersion,
			Repository:            flags.Repository,
			InventoryImageTag:     flags.InventoryImageTag,
			Vendor:                flags.Vendor,
			Publisher:             flags.Publisher,
			InventoryDBFile:       flags.InventoryDBFile,
			DeactivatePluginGroup: true,
			ImageOperationsImpl:   carvelhelpers.NewImageOperationsImpl(),
		}
		return pguOptions.UpdatePluginGroupActivationState()
	}
	return pluginGroupDeactivateCmd
}

func getPluginGroupActivateDeactivateBaseCmd() (*cobra.Command, *inventoryPluginGroupActivateDeactivateFlags) {
	var flags = &inventoryPluginGroupActivateDeactivateFlags{}

	var activateDeactivateCmd = &cobra.Command{}
	activateDeactivateCmd.SilenceUsage = true

	activateDeactivateCmd.Flags().StringVarP(&flags.GroupName, "name", "", "", "name of the plugin-group")
	activateDeactivateCmd.Flags().StringVarP(&flags.GroupVersion, "version", "", "", "version of the plugin-group")
	activateDeactivateCmd.Flags().StringVarP(&flags.Repository, "repository", "", "", "repository to publish plugin inventory image")
	activateDeactivateCmd.Flags().StringVarP(&flags.InventoryImageTag, "plugin-inventory-image-tag", "", "latest", "tag to which plugin inventory image needs to be published")
	activateDeactivateCmd.Flags().StringVarP(&flags.Vendor, "vendor", "", "", "name of the vendor")
	activateDeactivateCmd.Flags().StringVarP(&flags.Publisher, "publisher", "", "", "name of the publisher")
	activateDeactivateCmd.Flags().StringVarP(&flags.InventoryDBFile, "plugin-inventory-db-file", "", "", "local file for the inventory database")

	_ = activateDeactivateCmd.MarkFlagRequired("name")
	_ = activateDeactivateCmd.MarkFlagRequired("version")
	_ = activateDeactivateCmd.MarkFlagRequired("vendor")
	_ = activateDeactivateCmd.MarkFlagRequired("publisher")

	return activateDeactivateCmd, flags
}
