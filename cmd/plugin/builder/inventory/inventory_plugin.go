// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package inventory implements inventory specific init and update functionalities
package inventory

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

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

	pluginBinaryDigestMap := map[string]string{}
	if !ipuo.ValidateOnly {
		pluginBinaryDigestMap, err = ipuo.fetchPluginBinaryDigest(pluginManifest)
		if err != nil {
			return nil, err
		}
	}

	for i := range pluginManifest.Plugins {
		var pluginInventoryEntry *plugininventory.PluginInventoryEntry

		for _, osArch := range cli.MinOSArch {
			for _, version := range pluginManifest.Plugins[i].Versions {
				pluginInventoryEntry, err = ipuo.updatePluginInventoryEntry(pluginInventoryEntry, pluginManifest.Plugins[i], osArch, version, pluginBinaryDigestMap)
				if err != nil {
					return nil, err
				}
			}
		}

		pluginInventoryEntries = append(pluginInventoryEntries, pluginInventoryEntry)
	}

	return pluginInventoryEntries, nil
}

func (ipuo *InventoryPluginUpdateOptions) fetchPluginBinaryDigest(pluginManifest *cli.Manifest) (map[string]string, error) {
	pluginBinaryDigestMap := map[string]string{}

	// Limit the number of concurrent operations we perform so we don't overwhelm the system.
	maxConcurrent := helpers.GetMaxParallelism()
	guard := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	fatalErrors := make(chan helpers.ErrInfo, helpers.GetNumberOfIndividualPluginBinariesFromManifest(pluginManifest))
	var mutex = &sync.RWMutex{}

	fetchPluginBinaryDigestFromImage := func(threadID string, pluginImage string, filename string) {
		defer func() {
			<-guard
			wg.Done()
		}()

		log.Infof("%s getting plugin digest from image: '%s'", threadID, pluginImage)

		digest, err := ipuo.ImageOperationsImpl.GetFileDigestFromImage(pluginImage, filename)
		if err != nil {
			fatalErrors <- helpers.ErrInfo{Err: errors.Wrapf(err, "error while getting plugin binary digest from the image %q", pluginImage), ID: threadID, Path: pluginImage}
		}
		mutex.Lock()
		pluginBinaryDigestMap[pluginImage] = digest
		mutex.Unlock()
	}

	if !ipuo.ValidateOnly {
		id := 0
		for i := range pluginManifest.Plugins {
			for _, osArch := range cli.MinOSArch {
				for _, version := range pluginManifest.Plugins[i].Versions {
					wg.Add(1)
					guard <- struct{}{}
					pluginImageBasePath := fmt.Sprintf("%s/%s/%s/%s/%s/%s:%s", ipuo.Vendor, ipuo.Publisher, osArch.OS(), osArch.Arch(), pluginManifest.Plugins[i].Target, pluginManifest.Plugins[i].Name, version)
					pluginImage := fmt.Sprintf("%s/%s", ipuo.Repository, pluginImageBasePath)
					go fetchPluginBinaryDigestFromImage(helpers.GetID(id), pluginImage, cli.MakeArtifactName(pluginManifest.Plugins[i].Name, osArch))
					id++
				}
			}
		}
		wg.Wait()
		close(fatalErrors)

		errList := []error{}
		for err := range fatalErrors {
			log.Errorf("%s - error while getting plugin binary digest - %v", err.ID, err.Err)
			errList = append(errList, err.Err)
		}
		if len(errList) > 0 {
			return pluginBinaryDigestMap, kerrors.NewAggregate(errList)
		}
	}
	return pluginBinaryDigestMap, nil
}

// Take the image download logic to get the digest out of the updatePluginInventoryEntry and run it in parallel
// Pass the digest map to this function to update the plugin inventory entry in sync operation
func (ipuo *InventoryPluginUpdateOptions) updatePluginInventoryEntry(pluginInventoryEntry *plugininventory.PluginInventoryEntry, plugin cli.Plugin, osArch cli.Arch, version string, pluginBinaryDigestMap map[string]string) (*plugininventory.PluginInventoryEntry, error) {
	var digest string

	pluginImageBasePath := fmt.Sprintf("%s/%s/%s/%s/%s/%s:%s", ipuo.Vendor, ipuo.Publisher, osArch.OS(), osArch.Arch(), plugin.Target, plugin.Name, version)
	if !ipuo.ValidateOnly {
		// If we are only validating the plugin's existence, we don't need to waste
		// resources downloading the image to get the digest which won't actually be used.
		pluginImage := fmt.Sprintf("%s/%s", ipuo.Repository, pluginImageBasePath)
		digest = pluginBinaryDigestMap[pluginImage]
		if digest == "" {
			return nil, errors.Errorf("plugin binary digest cannot be empty for image %q", pluginImage)
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
