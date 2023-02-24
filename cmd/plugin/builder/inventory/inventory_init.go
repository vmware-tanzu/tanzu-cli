// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aunum/log"
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/imgpkg"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
)

// InventoryInitOptions defines options for inventory init
type InventoryInitOptions struct {
	Repository        string
	InventoryImageTag string
	Override          bool

	ImgpkgOptions imgpkg.ImgpkgWrapper
}

// InitializeInventory initializes the repository with the empty inventory database
func (iio *InventoryInitOptions) InitializeInventory() error {
	// create plugin inventory database image path
	pluginInventoryDBImage := fmt.Sprintf("%s/%s:%s", iio.Repository, helpers.PluginInventoryDBImageName, iio.InventoryImageTag)

	if !iio.Override {
		// check if the image already exists or not
		err := iio.ImgpkgOptions.ResolveImage(pluginInventoryDBImage)
		if err == nil {
			return errors.Errorf("%q image already exists on the repository. Use `--override` flag to override the content", pluginInventoryDBImage)
		}
	}

	// Create plugin inventory database
	dbFile := filepath.Join(os.TempDir(), plugininventory.SQliteDBFileName)
	_ = os.Remove(dbFile)
	err := plugininventory.NewSQLiteInventory(dbFile, "").CreateSchema()
	if err != nil {
		return errors.Wrap(err, "error while creating database")
	}
	log.Infof("Create database locally at: %q", dbFile)

	// Publish the database to the remote repository
	log.Infof("Publishing database at: %q", pluginInventoryDBImage)
	err = iio.ImgpkgOptions.PushImage(pluginInventoryDBImage, dbFile)
	if err != nil {
		return errors.Wrapf(err, "error while publishing database to the repository as image: %q", pluginInventoryDBImage)
	}
	log.Infof("Successfully published plugin inventory database")

	return nil
}
