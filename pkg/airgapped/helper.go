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

// GetImageRelativePath returns the relative path of the image with respect to `basePath`
// E.g. If the image is `fake.repo.com/plugin/database/plugin-inventory:latest` with
// basePath as `fake.repo.com/plugin` it should return
// `database/plugin-inventory:latest` if withTag is true and
// `database/plugin-inventory` if withTag is false
// This function also supports tags in the form of digests (SHAs),
// e.g., fake.repo.com/plugin/database/vmware/tkg/darwin/amd64/mission-control/account@sha256:69dc17b84e77d0844c36c11f1191f47bb3cec4ca61e06950a3884e34b3ecb6eb
func GetImageRelativePath(image, basePath string, withTag bool) string {
	relativePath := strings.TrimPrefix(image, basePath)
	if withTag {
		return relativePath
	}
	if idx := strings.LastIndex(relativePath, "@"); idx != -1 {
		return relativePath[:idx]
	}
	if idx := strings.LastIndex(relativePath, ":"); idx != -1 {
		return relativePath[:idx]
	}
	return relativePath
}
