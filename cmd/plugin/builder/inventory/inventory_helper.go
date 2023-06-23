// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
)

func inventoryDBDownload(imageOperationsImpl carvelhelpers.ImageOperationsImpl, pluginInventoryDBImage, tempDir string) (string, error) {
	err := imageOperationsImpl.DownloadImageAndSaveFilesToDir(pluginInventoryDBImage, tempDir)
	if err != nil {
		return "", errors.Wrapf(err, "error while pulling database from the image: %q", pluginInventoryDBImage)
	}
	return filepath.Join(tempDir, plugininventory.SQliteDBFileName), nil
}

func inventoryDBUpload(imageOperationsImpl carvelhelpers.ImageOperationsImpl, pluginInventoryDBImage, dbFile string) error {
	err := imageOperationsImpl.PushImage(pluginInventoryDBImage, []string{dbFile})
	if err != nil {
		return errors.Wrapf(err, "error while publishing inventory database to the repository as image: %q", pluginInventoryDBImage)
	}
	return nil
}
