// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

// DockerOptions implements the DockerWrapper interface by using `docker` binary internally
type DockerOptions struct{}

// BuildImage invokes `docker build -t <image> -f <template> <dirpath>` command
func (do *DockerOptions) BuildImage(image, template, dirPath string) error {
	output, err := exec.Command("docker", "build", "-t", image, "-f", template, dirPath).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

// SaveImage invokes `docker save <image> | gzip -c > <archivefile>` command
func (do *DockerOptions) SaveImage(image, archiveFile string) error {
	// Create archive directory if doesn't exists
	err := os.MkdirAll(filepath.Dir(archiveFile), 0755)
	if err != nil {
		return errors.Wrap(err, "unable to create directory")
	}

	cmd := fmt.Sprintf("docker save %s | gzip -c > %s", image, archiveFile)
	output, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

// LoadImage invokes `docker load -i <architefile>` command
func (do *DockerOptions) LoadImage(archiveFile string) error {
	loadCMD := fmt.Sprintf("docker load -i %s", archiveFile)
	output, err := exec.Command("bash", "-c", loadCMD).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

// TagImage invokes `docker tag <existingImage> <newImage>` command
func (do *DockerOptions) TagImage(existingImage, newImage string) error {
	tagCMD := fmt.Sprintf("docker tag %s %s", existingImage, newImage)
	output, err := exec.Command("bash", "-c", tagCMD).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}

// PushImage invokes `docker push <image>` command
func (do *DockerOptions) PushImage(image string) error {
	pushCMD := fmt.Sprintf("docker push %s", image)
	output, err := exec.Command("bash", "-c", pushCMD).CombinedOutput()
	return errors.Wrapf(err, "output: %s", string(output))
}
