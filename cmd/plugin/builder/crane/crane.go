// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package crane

import (
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/cmd/crane/cmd"
	"github.com/google/go-containerregistry/pkg/crane"

	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

// CraneOptions implements the CraneWrapper interface by using `crane` library
type CraneOptions struct{}

// SaveImage image as an tar.gz archive file
func (co *CraneOptions) SaveImage(imageName, pluginTarGZFilePath string) error {
	err := os.MkdirAll(filepath.Dir(pluginTarGZFilePath), 0755)
	if err != nil {
		return err
	}

	pluginTarFile, err := os.CreateTemp("", "*.tar")
	if err != nil {
		return err
	}
	defer os.Remove(pluginTarFile.Name())

	cranePullCmd := cmd.NewCmdPull(&[]crane.Option{})
	err = cranePullCmd.RunE(cranePullCmd, []string{imageName, pluginTarFile.Name()})
	if err != nil {
		return err
	}

	// convert the tar file into the tar.gz file
	return utils.Gzip(pluginTarFile.Name(), pluginTarGZFilePath)
}

// PushImage publish the archive file to remote container registry
func (co *CraneOptions) PushImage(pluginTarGZFilePath, image string) error {
	pluginTarFile, err := os.CreateTemp("", "*.tar")
	if err != nil {
		return err
	}
	defer os.Remove(pluginTarFile.Name())
	err = utils.UnGzip(pluginTarGZFilePath, pluginTarFile.Name())
	if err != nil {
		return err
	}

	cranePushCmd := cmd.NewCmdPush(&[]crane.Option{})
	return cranePushCmd.RunE(cranePushCmd, []string{pluginTarFile.Name(), image})
}
