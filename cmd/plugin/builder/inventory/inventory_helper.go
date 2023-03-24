// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/imgpkg"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
)

func inventoryDBDownload(imgpkgOptions imgpkg.ImgpkgWrapper, pluginInventoryDBImage, tempDir string) (string, error) {
	err := imgpkgOptions.PullImage(pluginInventoryDBImage, tempDir)
	if err != nil {
		return "", errors.Wrapf(err, "error while pulling database from the image: %q", pluginInventoryDBImage)
	}
	return filepath.Join(tempDir, plugininventory.SQliteDBFileName), nil
}

func inventoryDBUpload(imgpkgOptions imgpkg.ImgpkgWrapper, pluginInventoryDBImage, dbFile string) error {
	err := imgpkgOptions.PushImage(pluginInventoryDBImage, dbFile)
	if err != nil {
		return errors.Wrapf(err, "error while publishing inventory database to the repository as image: %q", pluginInventoryDBImage)
	}
	return nil
}
