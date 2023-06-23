// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package carvelhelpers

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
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

// ResolveImage invokes `imgpkg tag resolve -i <image>` command
func (i *ImageOperationOptions) ResolveImage(image string) error {
	registryName, err := registry.GetRegistryName(image)
	if err != nil {
		return err
	}
	reg, err := newRegistry(registryName)
	if err != nil {
		return errors.Wrapf(err, "unable to initialize registry")
	}
	return reg.ResolveImage(image)
}

// GetFileDigestFromImage invokes `DownloadImageAndSaveFilesToDir` to fetch the image and returns the digest of the specified file
func (i *ImageOperationOptions) GetFileDigestFromImage(image, fileName string) (string, error) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temporary directory")
	}
	defer os.RemoveAll(tempDir)

	// Pull image to the temporary directory
	err = i.DownloadImageAndSaveFilesToDir(image, tempDir)
	if err != nil {
		return "", errors.Wrapf(err, "unable to find image at %q", image)
	}

	// find the digest of the specified file
	digest, err := helpers.GetDigest(filepath.Join(tempDir, fileName))
	if err != nil {
		return "", errors.Wrapf(err, "unable to calculate digest for path %v", filepath.Join(tempDir, fileName))
	}
	return digest, nil
}
