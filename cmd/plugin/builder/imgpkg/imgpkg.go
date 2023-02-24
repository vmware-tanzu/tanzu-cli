// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package imgpkg

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/helpers"
)

type ImgpkgOptions struct{}

func (io *ImgpkgOptions) ResolveImage(image string) error {
	output, err := exec.Command("imgpkg", "tag", "resolve", "-i", image).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

func (io *ImgpkgOptions) PushImage(image, filePath string) error {
	output, err := exec.Command("imgpkg", "push", "-i", image, "-f", filePath).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

func (io *ImgpkgOptions) PullImage(image, dirPath string) error {
	output, err := exec.Command("imgpkg", "pull", "-i", image, "-o", dirPath).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

func (io *ImgpkgOptions) CopyArchiveToRepo(imageRepo, archivePath string) error {
	output, err := exec.Command("imgpkg", "copy", "--tar", archivePath, "--to-repo", imageRepo).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

func (io *ImgpkgOptions) CopyImageToArchive(image, archivePath string) error {
	err := os.MkdirAll(filepath.Dir(archivePath), 0755)
	if err != nil {
		return err
	}

	output, err := exec.Command("imgpkg", "copy", "-i", image, "--to-tar", archivePath).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

func (io *ImgpkgOptions) GetFileDigestFromImage(image, fileName string) (string, error) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temporary directory")
	}
	defer os.RemoveAll(tempDir)

	// Pull image to the temporary directory
	err = io.PullImage(image, tempDir)
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
