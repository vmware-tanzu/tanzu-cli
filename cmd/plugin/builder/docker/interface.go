// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package docker implements helper function for docker cli
package docker

// DockerWrapper defines the docker command wrapper functions
type DockerWrapper interface {
	// BuildImage invokes `docker build -t <image> -f <template> <dirpath>` command
	BuildImage(image, template, dirPath string) error
	// SaveImage invokes `docker save <image> | gzip -c > <archivefile>` command
	SaveImage(image, archiveFile string) error
	// LoadImage invokes `docker load -i <architefile>` command
	LoadImage(archiveFile string) error
	// TagImage invokes `docker tag <existingImage> <newImage>` command
	TagImage(existingImage, newImage string) error
	// PushImage invokes `docker push <image>` command
	PushImage(image string) error
}

// NewDockerCLIWrapper creates new DockerWrapper instance
func NewDockerCLIWrapper() DockerWrapper {
	return &DockerOptions{}
}
