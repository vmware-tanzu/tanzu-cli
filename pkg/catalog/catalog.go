// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/rogpeppe/go-internal/lockedfile" //nolint:depguard

	"gopkg.in/yaml.v3"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

const (
	// catalogCacheFileName is the name of the file which holds Catalog cache
	catalogCacheFileName = "catalog.yaml"
)

var (
	// PluginRoot is the plugin root where plugins are installed
	pluginRoot = common.DefaultPluginRoot
)

// ContextCatalog denotes a local plugin catalog for a given context or
// stand-alone.
type ContextCatalog struct {
	sharedCatalog *Catalog
	plugins       PluginAssociation
	lockedFile    *lockedfile.File
}

// NewContextCatalog creates context-aware catalog for reading the catalog
func NewContextCatalog(context string) (PluginCatalogReader, error) {
	return newContextCatalog(context, false)
}

// NewContextCatalogUpdater creates context-aware catalog for reading/updating the catalog
// When using this API invoker needs to call `Unlock` API to unlock the WriteLock
// acquired to update the catalog
// After Unlock() is called, the ContextCatalog object can no longer be used,
// and a new one must be obtained for any further operation on the catalog
func NewContextCatalogUpdater(context string) (PluginCatalogUpdater, error) {
	return newContextCatalog(context, true)
}

// newContextCatalog creates a new context-aware catalog object
func newContextCatalog(context string, lockCatalog bool) (*ContextCatalog, error) {
	sc, lockedFile, err := getCatalogCache(lockCatalog)
	if err != nil {
		return nil, err
	}

	var plugins PluginAssociation
	if context == "" {
		plugins = sc.StandAlonePlugins
	} else {
		var ok bool
		plugins, ok = sc.ServerPlugins[context]
		if !ok {
			plugins = make(PluginAssociation)
			sc.ServerPlugins[context] = plugins
		}
	}

	return &ContextCatalog{
		sharedCatalog: sc,
		plugins:       plugins,
		lockedFile:    lockedFile,
	}, nil
}

// Upsert inserts/updates the given plugin.
func (c *ContextCatalog) Upsert(plugin *cli.PluginInfo) error {
	if c.lockedFile == nil {
		return errors.Errorf("cannot complete the upsert plugin operation for plugin %q. catalog is not locked", plugin.Name)
	}

	pluginNameTarget := PluginNameTarget(plugin.Name, plugin.Target)

	c.plugins[pluginNameTarget] = plugin.InstallationPath
	c.sharedCatalog.IndexByPath[plugin.InstallationPath] = *plugin

	if !utils.ContainsString(c.sharedCatalog.IndexByName[pluginNameTarget], plugin.InstallationPath) {
		c.sharedCatalog.IndexByName[pluginNameTarget] = append(c.sharedCatalog.IndexByName[pluginNameTarget], plugin.InstallationPath)
	}

	// The "unknown" target was previously used in two scenarios:
	// 1- to represent the global target (>= v0.28 and < v0.90)
	// 2- to represent either the global or kubernetes target (< v0.28)
	// When inserting the "global" or "k8s" target we should remove any similar plugin using
	// the "unknown" target and vice versa.
	if plugin.Target == configtypes.TargetGlobal || plugin.Target == configtypes.TargetK8s {
		delete(c.plugins, PluginNameTarget(plugin.Name, configtypes.TargetUnknown))
	} else if plugin.Target == configtypes.TargetUnknown {
		// An older plugin binary may not specify a target (through its 'info' command).
		// Therefore the plugin could be a global plugin or a k8s plugin (but not both).
		// We need to delete either pre-existing entries from the catalog to avoid having
		// a double entry.
		delete(c.plugins, PluginNameTarget(plugin.Name, configtypes.TargetGlobal))
		delete(c.plugins, PluginNameTarget(plugin.Name, configtypes.TargetK8s))
	}
	return saveCatalogCache(c.sharedCatalog, c.lockedFile)
}

// Get looks up the descriptor of a plugin given its name.
func (c *ContextCatalog) Get(plugin string) (cli.PluginInfo, bool) {
	pd := cli.PluginInfo{}
	path, ok := c.plugins[plugin]
	if !ok {
		return pd, false
	}

	pd, ok = c.sharedCatalog.IndexByPath[path]
	if !ok {
		return pd, false
	}

	return pd, true
}

// List returns the list of active plugins.
// Active plugin means the plugin that are available to the user
// based on the current logged-in server.
func (c *ContextCatalog) List() []cli.PluginInfo {
	pds := make([]cli.PluginInfo, 0)
	for _, installationPath := range c.plugins {
		pd := c.sharedCatalog.IndexByPath[installationPath]
		pds = append(pds, pd)
	}
	return pds
}

// Delete deletes the given plugin from the catalog, but it does not delete
// the installation.
func (c *ContextCatalog) Delete(plugin string) error {
	if c.lockedFile == nil {
		return errors.Errorf("cannot complete the delete plugin operation for plugin %q. catalog is not locked", plugin)
	}
	_, ok := c.plugins[plugin]
	if ok {
		delete(c.plugins, plugin)
	}
	return saveCatalogCache(c.sharedCatalog, c.lockedFile)
}

// Unlock unlocks the catalog for other process to read/write
// After Unlock() is called, the ContextCatalog object can no longer be used,
// and a new one must be obtained for any further operation on the catalog
func (c *ContextCatalog) Unlock() {
	if c.lockedFile != nil {
		c.lockedFile.Close()
		c.lockedFile = nil
	}
}

// getCatalogCacheDir returns the local directory in which tanzu state is stored.
func getCatalogCacheDir() (path string) {
	// NOTE: TEST_CUSTOM_CATALOG_CACHE_DIR is only for test purpose
	customCacheDirForTest := os.Getenv("TEST_CUSTOM_CATALOG_CACHE_DIR")
	if customCacheDirForTest != "" {
		return customCacheDirForTest
	}
	return common.DefaultCacheDir
}

// newSharedCatalog creates an instance of the shared catalog file.
func newSharedCatalog() (*Catalog, error) {
	c := &Catalog{
		IndexByPath:       map[string]cli.PluginInfo{},
		IndexByName:       map[string][]string{},
		StandAlonePlugins: map[string]string{},
		ServerPlugins:     map[string]PluginAssociation{},
	}

	err := ensureRoot()
	if err != nil {
		return nil, err
	}
	return c, nil
}

// getCatalogCache retrieves the catalog from the local directory along with locking the catalog file
// If `setWriteLock` is false, it will read the catalog file with ReadLock and release the lock at the same time
// If `setWriteLock` is true, it will apply WriteLock to the catalog file, read the catalog file
// and keep the WriteLock to the file along with returning `lockedFile` object. It is caller's
// responsibility to unlock the WriteLock after the catalog update
func getCatalogCache(setWriteLock bool) (*Catalog, *lockedfile.File, error) {
	b, lockedFile, err := getCatalogCacheBytes(setWriteLock)
	if err != nil {
		if os.IsNotExist(err) {
			catalog, err := newSharedCatalog()
			if err != nil {
				return nil, lockedFile, err
			}
			return catalog, lockedFile, nil
		}
		return nil, lockedFile, err
	}

	var c Catalog
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return nil, lockedFile, errors.Wrap(err, "could not decode catalog file")
	}

	if c.IndexByPath == nil {
		c.IndexByPath = map[string]cli.PluginInfo{}
	}
	if c.IndexByName == nil {
		c.IndexByName = map[string][]string{}
	}
	if c.StandAlonePlugins == nil {
		c.StandAlonePlugins = map[string]string{}
	}
	if c.ServerPlugins == nil {
		c.ServerPlugins = map[string]PluginAssociation{}
	}

	return &c, lockedFile, nil
}

func getCatalogCacheBytes(setWriteLock bool) ([]byte, *lockedfile.File, error) {
	var lockedFile *lockedfile.File
	var err error
	var b []byte

	if setWriteLock {
		if !utils.PathExists(getCatalogCachePath()) {
			// Create directory path if missing before locking the file
			_ = os.MkdirAll(getCatalogCacheDir(), 0755)
		}
		lockedFile, err = lockedfile.Edit(getCatalogCachePath())
		if err != nil {
			return nil, lockedFile, err
		}
		b, err = io.ReadAll(lockedFile)
	} else {
		b, err = lockedfile.Read(getCatalogCachePath())
	}
	return b, lockedFile, err
}

// saveCatalogCache saves the catalog in the local directory.
func saveCatalogCache(catalog *Catalog, lockedCatalogFile *lockedfile.File) error {
	if lockedCatalogFile == nil {
		return errors.New("cannot save the catalog file. catalog is not locked")
	}

	catalogCachePath := getCatalogCachePath()
	_, err := os.Stat(catalogCachePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(getCatalogCacheDir(), 0755)
		if err != nil {
			return errors.Wrap(err, "could not make tanzu cache directory")
		}
	} else if err != nil {
		return errors.Wrap(err, "could not create catalog cache path")
	}

	out, err := yaml.Marshal(catalog)
	if err != nil {
		return errors.Wrap(err, "failed to encode catalog cache file")
	}

	if err := lockedCatalogFile.Truncate(0); err != nil {
		return errors.Wrap(err, "failed to write catalog cache file. truncate failed")
	}
	if _, err := lockedCatalogFile.Seek(0, 0); err != nil {
		return errors.Wrap(err, "failed to write catalog cache file. seek failed")
	}
	if _, err := lockedCatalogFile.Write(out); err != nil {
		return errors.Wrap(err, "failed to write catalog cache file")
	}
	return nil
}

// CleanCatalogCache cleans the catalog cache
func CleanCatalogCache() error {
	if err := os.Remove(getCatalogCachePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// getCatalogCachePath gets the catalog cache path
func getCatalogCachePath() string {
	return filepath.Join(getCatalogCacheDir(), catalogCacheFileName)
}

// Ensure the root directory exists.
func ensureRoot() error {
	_, err := os.Stat(testPath())
	if os.IsNotExist(err) {
		err := os.MkdirAll(testPath(), 0755)
		return errors.Wrap(err, "could not make root plugin directory")
	}
	return err
}

// Returns the test path relative to the plugin root
func testPath() string {
	return filepath.Join(pluginRoot, "test")
}

// PluginNameTarget constructs a string to uniquely refer to a plugin associated
// with a specific target when target is provided.
func PluginNameTarget(pluginName string, target configtypes.Target) string {
	if target == "" {
		return pluginName
	}
	return fmt.Sprintf("%s_%s", pluginName, target)
}
