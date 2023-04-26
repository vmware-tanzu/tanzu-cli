// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package carvelhelpers

import (
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/registry"
)

// ImageOperationOptions implements the ImageOperationsImpl interface by using `imgpkg` library
type ImageOperationOptions struct{}

// NewImageOperationsImpl creates new ImgpkgWrapper instance
func NewImageOperationsImpl() ImageOperationsImpl {
	return &ImageOperationOptions{}
}

// CopyImageToTar downloads the image as tar file
// This is equivalent to `imgpkg copy --image <image> --to-tar <tar-file-path>` command
func (i *ImageOperationOptions) CopyImageToTar(sourceImageName, destTarFile string) error {
	registryName, err := registry.GetRegistryName(sourceImageName)
	if err != nil {
		return err
	}
	reg, err := newRegistry(registryName)
	if err != nil {
		return errors.Wrapf(err, "unable to initialize registry")
	}
	return reg.CopyImageToTar(sourceImageName, destTarFile)
}

// CopyImageFromTar publishes the image to destination repository from specified tar file
// This is equivalent to `imgpkg copy --tar <file> --to-repo <dest-repo>` command
func (i *ImageOperationOptions) CopyImageFromTar(sourceTarFile, destImageRepo string) error {
	registryName, err := registry.GetRegistryName(destImageRepo)
	if err != nil {
		return err
	}
	reg, err := newRegistry(registryName)
	if err != nil {
		return errors.Wrapf(err, "unable to initialize registry")
	}
	return reg.CopyImageFromTar(sourceTarFile, destImageRepo)
}

// DownloadImageAndSaveFilesToDir reads a plain OCI image and saves its
// files to the specified location.
func (i *ImageOperationOptions) DownloadImageAndSaveFilesToDir(imageWithTag, destinationDir string) error {
	registryName, err := registry.GetRegistryName(imageWithTag)
	if err != nil {
		return err
	}
	reg, err := newRegistry(registryName)
	if err != nil {
		return errors.Wrapf(err, "unable to initialize registry")
	}
	err = reg.DownloadImage(imageWithTag, destinationDir)
	if err != nil {
		return errors.Wrap(err, "error downloading image")
	}
	return nil
}

// GetFilesMapFromImage returns map of files metadata
// It takes os environment variables for custom repository and proxy
// configuration into account while downloading image from repository
func (i *ImageOperationOptions) GetFilesMapFromImage(imageWithTag string) (map[string][]byte, error) {
	registryName, err := registry.GetRegistryName(imageWithTag)
	if err != nil {
		return nil, err
	}
	reg, err := newRegistry(registryName)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to initialize registry")
	}
	return reg.GetFiles(imageWithTag)
}

// GetImageDigest gets digest of the image
func (i *ImageOperationOptions) GetImageDigest(imageWithTag string) (string, string, error) {
	registryName, err := registry.GetRegistryName(imageWithTag)
	if err != nil {
		return "", "", err
	}
	reg, err := newRegistry(registryName)
	if err != nil {
		return "", "", errors.Wrapf(err, "unable to initialize registry")
	}

	hashAlgorithm, hashHexVal, err := reg.GetImageDigest(imageWithTag)
	if err != nil {
		return "", "", errors.Wrap(err, "error getting the image digest")
	}

	return hashAlgorithm, hashHexVal, nil
}

// PushImage publishes the image to the specified location
func (i *ImageOperationOptions) PushImage(imageWithTag string, filePaths []string) error {
	registryName, err := registry.GetRegistryName(imageWithTag)
	if err != nil {
		return err
	}
	reg, err := newRegistry(registryName)
	if err != nil {
		return errors.Wrapf(err, "unable to initialize registry")
	}
	return reg.PushImage(imageWithTag, filePaths)
}
