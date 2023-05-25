// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package imgpkg implements helper function for imgpkg cli
package imgpkg

//go:generate counterfeiter -o ../fakes/imgpkgwrapper.go --fake-name ImgpkgWrapper . ImgpkgWrapper

// ImgpkgWrapper defines the imgpkg command wrapper functions
type ImgpkgWrapper interface {
	// ResolveImage invokes `imgpkg tag resolve -i <image>` command
	ResolveImage(image string) error
	// PushImage invokes `imgpkg push -i <image> -f <filepath>` command
	PushImage(image, filePath string) error
	// PullImage invokes `imgpkg pull -i <image> -o <dirPath>` command
	PullImage(image, dirPath string) error
	// GetFileDigestFromImage invokes `PullImage` to fetch the image and returns the digest of the specified file
	GetFileDigestFromImage(image, fileName string) (string, error)
}

// NewImgpkgCLIWrapper creates new ImgpkgWrapper instance
func NewImgpkgCLIWrapper() ImgpkgWrapper {
	return &ImgpkgOptions{}
}
