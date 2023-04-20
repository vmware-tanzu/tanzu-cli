// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package airgapped

import (
	"fmt"
	"strings"

	dockerparser "github.com/novln/docker-parser"
	"github.com/pkg/errors"
)

// GetPluginInventoryMetadataImage returns the plugin inventory metadata
// image based on plugin inventory image.
// E.g. if plugin inventory image is `fake.repo.com/plugin/plugin-inventory:latest`
// it returns metadata image as `fake.repo.com/plugin/plugin-inventory-metadata:latest`
func GetPluginInventoryMetadataImage(pluginInventoryImage string) (string, error) {
	ref, err := dockerparser.Parse(pluginInventoryImage)
	if err != nil {
		return "", errors.Wrapf(err, "invalid image %q", pluginInventoryImage)
	}
	return fmt.Sprintf("%s-metadata:%s", ref.Repository(), ref.Tag()), nil
}

// GetImageRelativePath returns relative path of the image based on `basePath`
// E.g. If image is `fake.repo.com/plugin/plugin-inventory:latest` with
// basePath as `fake.repo.com/plugin` it should return
// `plugin-inventory:latest` if withTag is true and
// `plugin-inventory` if withTag is false
func GetImageRelativePath(image, basePath string, withTag bool) string {
	relativePath := strings.TrimPrefix(image, basePath)
	if withTag {
		return relativePath
	}
	if idx := strings.LastIndex(relativePath, ":"); idx != -1 {
		return relativePath[:idx]
	}
	return relativePath
}
