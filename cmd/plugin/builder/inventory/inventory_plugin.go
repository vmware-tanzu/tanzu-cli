// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package inventory implements inventory specific init and update functionalities
package inventory

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/distribution"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

// InventoryPluginUpdateOptions defines options for inserting plugin to the inventory database
type InventoryPluginUpdateOptions struct {
	Repository        string
	InventoryImageTag string
	ManifestFile      string
	Publisher         string
	Vendor            string
	InventoryDBFile   string
	DeactivatePlugins bool
	ValidateOnly      bool

	ImageOperationsImpl carvelhelpers.ImageOperationsImpl
}

// PluginAdd add plugin entry to the inventory database by downloading the database from the repository, updating it locally
// and publishing the inventory database as OCI image on the remote repository
func (ipuo *InventoryPluginUpdateOptions) PluginAdd() error {
	pluginAddFunc := func(dbFile string, entry *plugininventory.PluginInventoryEntry) error {
		db := plugininventory.NewSQLiteInventory(dbFile, "")
		err := db.InsertPlugin(entry)
		if err != nil {
			return errors.Wrapf(err, "error while inserting plugin '%s_%s'", entry.Name, entry.Target)
		}
		return nil
	}
	return ipuo.genericInventoryUpdater(pluginAddFunc)
}

// UpdatePluginActivationState updates plugin entry in the inventory database by downloading the
// database from the repository, updating it locally and publishing the inventory database
// as OCI image on the remote repository
func (ipuo *InventoryPluginUpdateOptions) UpdatePluginActivationState() error {
	activateDeactivateFunc := func(dbFile string, entry *plugininventory.PluginInventoryEntry) error {
		db := plugininventory.NewSQLiteInventory(dbFile, "")
		err := db.UpdatePluginActivationState(entry)
		if err != nil {
			return errors.Wrapf(err, "error while updating plugin '%s_%s'", entry.Name, entry.Target)
		}
		return nil
	}
	return ipuo.genericInventoryUpdater(activateDeactivateFunc)
}

func (ipuo *InventoryPluginUpdateOptions) genericInventoryUpdater(inventoryUpdater func(string, *plugininventory.PluginInventoryEntry) error) error {
	// Get inventory database file
	dbFile, err := ipuo.getInventoryDBFile()
	if err != nil {
		return err
	}

	// Create plugin inventory entries and update the database
	pluginInventoryEntries, err := ipuo.preparePluginInventoryEntriesFromManifest()
	if err != nil {
		return errors.Wrap(err, "error while updating plugin inventory database")
	}
	for i := range pluginInventoryEntries {
		err := inventoryUpdater(dbFile, pluginInventoryEntries[i])
		if err != nil {
			return err
		}
	}

	// Publish inventory database file if needed and return
	return ipuo.putInventoryDBFile(dbFile)
}

func (ipuo *InventoryPluginUpdateOptions) preparePluginInventoryEntriesFromManifest() ([]*plugininventory.PluginInventoryEntry, error) {
	pluginManifest, err := helpers.ReadPluginManifest(ipuo.ManifestFile)
	if err != nil {
		return nil, err
	}

	var pluginInventoryEntries []*plugininventory.PluginInventoryEntry

	for i := range pluginManifest.Plugins {
		var pluginInventoryEntry *plugininventory.PluginInventoryEntry

		for _, osArch := range cli.MinOSArch {
			for _, version := range pluginManifest.Plugins[i].Versions {
				pluginInventoryEntry, err = ipuo.updatePluginInventoryEntry(pluginInventoryEntry, pluginManifest.Plugins[i], osArch, version)
				if err != nil {
					return nil, err
				}
			}
		}

		pluginInventoryEntries = append(pluginInventoryEntries, pluginInventoryEntry)
	}

	return pluginInventoryEntries, nil
}

func (ipuo *InventoryPluginUpdateOptions) updatePluginInventoryEntry(pluginInventoryEntry *plugininventory.PluginInventoryEntry, plugin cli.Plugin, osArch cli.Arch, version string) (*plugininventory.PluginInventoryEntry, error) {
	var err error
	var digest string

	log.Infof("validating plugin '%s_%s_%s_%s'", plugin.Name, plugin.Target, osArch.String(), version)

	pluginImageBasePath := fmt.Sprintf("%s/%s/%s/%s/%s/%s:%s", ipuo.Vendor, ipuo.Publisher, osArch.OS(), osArch.Arch(), plugin.Target, plugin.Name, version)
	if !ipuo.ValidateOnly {
		// If we are only validating the plugin's existence, we don't need to waste
		// resources downloading the image to get the digest which won't actually be used.
		pluginImage := fmt.Sprintf("%s/%s", ipuo.Repository, pluginImageBasePath)
		digest, err = ipuo.ImageOperationsImpl.GetFileDigestFromImage(pluginImage, cli.MakeArtifactName(plugin.Name, osArch))
		if err != nil {
			return nil, errors.Wrapf(err, "error while getting plugin binary digest from the image %q", pluginImage)
		}
	}

	if pluginInventoryEntry == nil {
		pluginInventoryEntry = &plugininventory.PluginInventoryEntry{
			Name:        plugin.Name,
			Target:      configtypes.Target(plugin.Target),
			Description: plugin.Description,
			Publisher:   ipuo.Publisher,
			Vendor:      ipuo.Vendor,
			Artifacts:   make(map[string]distribution.ArtifactList),
			Hidden:      ipuo.DeactivatePlugins,
		}
	}
	_, exists := pluginInventoryEntry.Artifacts[version]
	if !exists {
		pluginInventoryEntry.Artifacts[version] = make([]distribution.Artifact, 0)
	}

	artifact := distribution.Artifact{
		OS:     osArch.OS(),
		Arch:   osArch.Arch(),
		Digest: digest,
		Image:  pluginImageBasePath,
	}
	pluginInventoryEntry.Artifacts[version] = append(pluginInventoryEntry.Artifacts[version], artifact)
	return pluginInventoryEntry, nil
}

func (ipuo *InventoryPluginUpdateOptions) getPluginInventoryDBImagePath() string {
	return fmt.Sprintf("%s/%s:%s", ipuo.Repository, helpers.PluginInventoryDBImageName, ipuo.InventoryImageTag)
}

func (ipuo *InventoryPluginUpdateOptions) getInventoryDBFile() (string, error) {
	if ipuo.InventoryDBFile != "" {
		log.Infof("using local plugin inventory database file: %q", ipuo.InventoryDBFile)
		if ipuo.ValidateOnly {
			tempFile, err := os.CreateTemp("", "*.db")
			if err != nil {
				return "", err
			}
			err = utils.CopyFile(ipuo.InventoryDBFile, tempFile.Name())
			if err != nil {
				return "", err
			}
			return tempFile.Name(), nil
		}
		return ipuo.InventoryDBFile, nil
	}

	// get plugin inventory database image path
	pluginInventoryDBImage := ipuo.getPluginInventoryDBImagePath()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temporary directory")
	}

	log.Infof("pulling plugin inventory database from: %q", pluginInventoryDBImage)
	err = ipuo.ImageOperationsImpl.DownloadImageAndSaveFilesToDir(pluginInventoryDBImage, dir)
	if err != nil {
		return "", errors.Wrapf(err, "error while pulling database from the image: %q", pluginInventoryDBImage)
	}
	return filepath.Join(dir, plugininventory.SQliteDBFileName), nil
}

func (ipuo *InventoryPluginUpdateOptions) putInventoryDBFile(dbFile string) error {
	pluginInventoryDBImage := ipuo.getPluginInventoryDBImagePath()

	// If validateOnly option was provided return validation as successful
	if ipuo.ValidateOnly {
		log.Info("validation successful")
		return nil
	}

	// If local inventory database file was provided nothing to publish just return
	if ipuo.InventoryDBFile != "" {
		log.Infof("successfully updated plugin inventory database file at: %q", ipuo.InventoryDBFile)
		return nil
	}

	// Publish the database to the remote repository
	log.Info("publishing plugin inventory database")
	err := ipuo.ImageOperationsImpl.PushImage(pluginInventoryDBImage, []string{dbFile})
	if err != nil {
		return errors.Wrapf(err, "error while publishing inventory database to the repository as image: %q", pluginInventoryDBImage)
	}
	log.Infof("successfully published plugin inventory database at: %q", pluginInventoryDBImage)
	return nil
}
