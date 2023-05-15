// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package inventory implements inventory specific init and update functionalities
package inventory

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/imgpkg"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// InventoryPluginGroupUpdateOptions defines options for updating plugin-group to the inventory database
type InventoryPluginGroupUpdateOptions struct {
	Repository              string
	InventoryImageTag       string
	PluginGroupManifestFile string
	Publisher               string
	Vendor                  string
	GroupName               string
	GroupVersion            string
	Description             string
	InventoryDBFile         string
	DeactivatePluginGroup   bool
	Override                bool

	ImgpkgOptions imgpkg.ImgpkgWrapper
}

// PluginGroupAdd add plugin-group entry to the inventory database by downloading
// the database from the repository, updating it locally and
// publishing the inventory database as OCI image on the remote repository
func (ipuo *InventoryPluginGroupUpdateOptions) PluginGroupAdd() error {
	dbFile, err := ipuo.getInventoryDBFile()
	if err != nil {
		return err
	}

	// Get the PluginGroup object from the plugin-group manifest file
	pg, err := ipuo.getPluginGroupFromManifest()
	if err != nil {
		return errors.Wrapf(err, "error while reading plugin group")
	}

	// Insert PluginGroup to the database
	log.Info("updating plugin inventory database with plugin group entry")
	db := plugininventory.NewSQLiteInventory(dbFile, "")
	err = db.InsertPluginGroup(pg, ipuo.Override)
	if err != nil {
		return errors.Wrapf(err, "error while inserting plugin group '%s'", pg.Name)
	}

	return ipuo.putInventoryDBFile(dbFile)
}

func (ipuo *InventoryPluginGroupUpdateOptions) getPluginGroupFromManifest() (*plugininventory.PluginGroup, error) {
	pg := plugininventory.PluginGroup{
		Vendor:      ipuo.Vendor,
		Publisher:   ipuo.Publisher,
		Name:        ipuo.GroupName,
		Description: ipuo.Description,
		Hidden:      ipuo.DeactivatePluginGroup,
		Versions:    make(map[string][]*plugininventory.PluginGroupPluginEntry, 0),
	}

	pluginGroupManifest, err := helpers.ReadPluginGroupManifest(ipuo.PluginGroupManifestFile)
	if err != nil {
		return nil, err
	}

	var plugins []*plugininventory.PluginGroupPluginEntry
	for _, plugin := range pluginGroupManifest.Plugins {
		pge := plugininventory.PluginGroupPluginEntry{
			PluginIdentifier: plugininventory.PluginIdentifier{
				Name:    plugin.Name,
				Target:  types.Target(plugin.Target),
				Version: plugin.Version,
			},
			Mandatory: !plugin.IsContextScoped,
		}
		plugins = append(plugins, &pge)
	}
	pg.Versions[ipuo.GroupVersion] = plugins

	return &pg, nil
}

// UpdatePluginGroupActivationState updates plugin-group entry in the inventory database by
// downloading the database from the repository, updating it locally and publishing the
// inventory database as OCI image on the remote repository
func (ipuo *InventoryPluginGroupUpdateOptions) UpdatePluginGroupActivationState() error {
	dbFile, err := ipuo.getInventoryDBFile()
	if err != nil {
		return err
	}

	// Create plugin-group object
	pg := &plugininventory.PluginGroup{
		Vendor:    ipuo.Vendor,
		Publisher: ipuo.Publisher,
		Name:      ipuo.GroupName,
		Hidden:    ipuo.DeactivatePluginGroup,
		Versions:  map[string][]*plugininventory.PluginGroupPluginEntry{ipuo.GroupVersion: {}},
	}

	// Insert PluginGroup to the database
	log.Info("updating plugin inventory database with plugin group entry")
	db := plugininventory.NewSQLiteInventory(dbFile, "")
	err = db.UpdatePluginGroupActivationState(pg)
	if err != nil {
		return errors.Wrapf(err, "error while updating activation state of plugin group '%s'", pg.Name)
	}

	return ipuo.putInventoryDBFile(dbFile)
}

func (ipuo *InventoryPluginGroupUpdateOptions) getPluginInventoryDBImage() string {
	return fmt.Sprintf("%s/%s:%s", ipuo.Repository, helpers.PluginInventoryDBImageName, ipuo.InventoryImageTag)
}

func (ipuo *InventoryPluginGroupUpdateOptions) getInventoryDBFile() (string, error) {
	if ipuo.InventoryDBFile != "" {
		log.Infof("using local plugin inventory database file: %q", ipuo.InventoryDBFile)
		return ipuo.InventoryDBFile, nil
	}

	// get plugin inventory database image path
	pluginInventoryDBImage := ipuo.getPluginInventoryDBImage()

	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temporary directory")
	}

	log.Infof("pulling plugin inventory database from: %q", pluginInventoryDBImage)
	dbFile, err := inventoryDBDownload(ipuo.ImgpkgOptions, pluginInventoryDBImage, tempDir)
	if err != nil {
		return "", errors.Wrapf(err, "error while downloading inventory database from the repository as image: %q", pluginInventoryDBImage)
	}

	return dbFile, nil
}

func (ipuo *InventoryPluginGroupUpdateOptions) putInventoryDBFile(dbFile string) error {
	pluginInventoryDBImage := ipuo.getPluginInventoryDBImage()

	// If local inventory database file was provided nothing to publish just return
	if ipuo.InventoryDBFile != "" {
		log.Infof("successfully updated plugin inventory database file at: %q", ipuo.InventoryDBFile)
		return nil
	}

	// Publish the database to the remote repository
	log.Info("publishing plugin inventory database")
	err := inventoryDBUpload(ipuo.ImgpkgOptions, pluginInventoryDBImage, dbFile)
	if err != nil {
		return errors.Wrapf(err, "error while publishing inventory database to the repository as image: %q", pluginInventoryDBImage)
	}
	log.Infof("successfully published plugin inventory database at: %q", pluginInventoryDBImage)
	return nil
}
