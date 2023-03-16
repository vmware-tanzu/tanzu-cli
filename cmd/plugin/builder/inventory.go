// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/imgpkg"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/inventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// newInventoryCmd creates a new command for inventory operations.
func newInventoryCmd() *cobra.Command {
	var inventoryCmd = &cobra.Command{
		Use:   "inventory",
		Short: "Inventory Operations",
	}

	inventoryCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	inventoryCmd.AddCommand(
		newInventoryInitCmd(),
		newInventoryPluginCmd(),
		newInventoryPluginGroupCmd(),
	)

	return inventoryCmd
}

type inventoryInitFlags struct {
	Repository        string
	InventoryImageTag string
	Override          bool
}

func newInventoryInitCmd() *cobra.Command {
	var piiFlags = &inventoryInitFlags{}

	var pluginInventoryInitCmd = &cobra.Command{
		Use:     "init",
		Short:   "Initialize empty plugin inventory database and publish it to the remote repository",
		Example: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			iiOptions := inventory.InventoryInitOptions{
				Repository:        piiFlags.Repository,
				InventoryImageTag: piiFlags.InventoryImageTag,
				Override:          piiFlags.Override,
				ImgpkgOptions:     imgpkg.NewImgpkgCLIWrapper(),
			}
			return iiOptions.InitializeInventory()
		},
	}

	pluginInventoryInitCmd.Flags().StringVarP(&piiFlags.Repository, "repository", "", "", "repository to publish plugin inventory image")
	pluginInventoryInitCmd.Flags().StringVarP(&piiFlags.InventoryImageTag, "plugin-inventory-image-tag", "", "latest", "tag to which plugin inventory image needs to be published")
	pluginInventoryInitCmd.Flags().BoolVarP(&piiFlags.Override, "override", "", false, "override the inventory database image if already exists")
	_ = pluginInventoryInitCmd.MarkFlagRequired("repository")

	return pluginInventoryInitCmd
}
