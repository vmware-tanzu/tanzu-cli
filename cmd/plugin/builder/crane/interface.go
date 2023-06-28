// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package crane implements helper function for crane library
package crane

// CraneWrapper defines the crane command wrapper functions
type CraneWrapper interface {
	// SaveImage image as an tar file
	SaveImage(image, pluginTarFilePath string) error
	// PushImage publish the tar file to remote container registry
	PushImage(pluginTarFilePath, image string) error
}

// NewCraneWrapper creates new CraneWrapper instance
func NewCraneWrapper() CraneWrapper {
	return &CraneOptions{}
}
