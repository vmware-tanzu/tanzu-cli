// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package carvelhelpers

//go:generate counterfeiter -o ../fakes/imageoperationsimpl.go --fake-name ImageOperationsImpl . ImageOperationsImpl

// ImageOperationsImpl defines the helper functions for downloading, copying and processing oci images
type ImageOperationsImpl interface {
	// CopyImageToTar downloads the image as tar file
	// This is equivalent to `imgpkg copy --image <image> --to-tar <tar-file-path>` command
	CopyImageToTar(sourceImageName, destTarFile string) error
	// CopyImageFromTar publishes the image to destination repository from specified tar file
	// This is equivalent to `imgpkg copy --tar <file> --to-repo <dest-repo>` command
	CopyImageFromTar(sourceTarFile, destImageRepo string) error
	// DownloadImageAndSaveFilesToDir reads a plain OCI image and saves its
	// files to the specified location.
	DownloadImageAndSaveFilesToDir(imageWithTag, destinationDir string) error
	// GetFilesMapFromImage returns map of files metadata
	// It takes os environment variables for custom repository and proxy
	// configuration into account while downloading image from repository
	GetFilesMapFromImage(imageWithTag string) (map[string][]byte, error)
	// GetImageDigest gets digest of the image
	GetImageDigest(imageWithTag string) (string, string, error)
	// PushImage publishes the image to the specified location
	// This is equivalent to `imgpkg push -i <image> -f <filepath>`
	PushImage(imageWithTag string, filePaths []string) error
	// ResolveImage invokes `imgpkg tag resolve -i <image>` command
	ResolveImage(image string) error
	// GetFileDigestFromImage invokes `DownloadImageAndSaveFilesToDir` to fetch the image and returns the digest of the specified file
	GetFileDigestFromImage(image, fileName string) (string, error)
}
