// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package airgapped

import (
	"testing"

	"github.com/tj/assert"
)

func Test_GetPluginInventoryMetadataImage(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		pluginInventoryImage  string
		expectedMetadataImage string
		errString             string
	}{
		{
			pluginInventoryImage:  "fake.repo.com/plugin/plugin-inventory:latest",
			expectedMetadataImage: "fake.repo.com/plugin/plugin-inventory-metadata:latest",
			errString:             "",
		},
		{
			pluginInventoryImage:  "fake.repo.com/plugin/airgapped:v1.0.0",
			expectedMetadataImage: "fake.repo.com/plugin/airgapped-metadata:v1.0.0",
			errString:             "",
		},
		{
			pluginInventoryImage:  "fake.repo.com/plugin/metadata",
			expectedMetadataImage: "fake.repo.com/plugin/metadata-metadata:latest",
			errString:             "",
		},
		{
			pluginInventoryImage:  "invalid-inventory-image$#",
			expectedMetadataImage: "",
			errString:             "invalid image",
		},
	}

	for _, test := range tests {
		t.Run(test.pluginInventoryImage, func(t *testing.T) {
			actualMetadataImage, err := GetPluginInventoryMetadataImage(test.pluginInventoryImage)
			assert.Equal(actualMetadataImage, test.expectedMetadataImage)
			if test.errString == "" {
				assert.Nil(err)
			} else {
				assert.Contains(err.Error(), test.errString)
			}
		})
	}
}

func Test_GetImageRelativePath(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		image                string
		basePath             string
		withTag              bool
		expectedRelativePath string
	}{
		{
			image:                "fake.repo.com/plugin/plugin-inventory:latest",
			basePath:             "fake.repo.com/plugin/",
			withTag:              true,
			expectedRelativePath: "plugin-inventory:latest",
		},
		{
			image:                "fake.repo.com/plugin/plugin-inventory:latest",
			basePath:             "fake.repo.com/plugin/",
			withTag:              false,
			expectedRelativePath: "plugin-inventory",
		},
		{
			image:                "fake.repo.com/plugin/airgapped:v1.0.0",
			basePath:             "fake.repo.com/",
			withTag:              true,
			expectedRelativePath: "plugin/airgapped:v1.0.0",
		},
		{
			image:                "fake.repo.com/plugin/metadata",
			basePath:             "fake.repo.com/",
			withTag:              false,
			expectedRelativePath: "plugin/metadata",
		},
		{
			image:                "fake.repo.com/plugin/metadata:latest",
			basePath:             "fake.repo.com/plugin/metadata-metadata",
			withTag:              true,
			expectedRelativePath: "fake.repo.com/plugin/metadata:latest",
		},
	}

	for _, test := range tests {
		t.Run(test.image, func(t *testing.T) {
			actualImage := GetImageRelativePath(test.image, test.basePath, test.withTag)
			assert.Equal(actualImage, test.expectedRelativePath)
		})
	}
}
