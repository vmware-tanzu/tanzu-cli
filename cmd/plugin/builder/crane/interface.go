// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package crane implements helper function for crane library
package crane

// CraneWrapper defines the crane command wrapper functions
type CraneWrapper interface {
	// SaveImage image as an archive file
	SaveImage(image, pluginTarGZFilePath string) error
	// PushImage publish the archive file to remote container registry
	PushImage(pluginTarGZFilePath, image string) error
}

// NewCraneWrapper creates new CraneWrapper instance
func NewCraneWrapper() CraneWrapper {
	return &CraneOptions{}
}
