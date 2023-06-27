// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package crane

import (
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/cmd/crane/cmd"
	"github.com/google/go-containerregistry/pkg/crane"
)

// CraneOptions implements the CraneWrapper interface by using `crane` library
type CraneOptions struct{}

// SaveImage image as an tar file
func (co *CraneOptions) SaveImage(imageName, pluginTarFilePath string) error {
	err := os.MkdirAll(filepath.Dir(pluginTarFilePath), 0755)
	if err != nil {
		return err
	}

	cranePullCmd := cmd.NewCmdPull(&[]crane.Option{})
	return cranePullCmd.RunE(cranePullCmd, []string{imageName, pluginTarFilePath})
}

// PushImage publish the tar file to remote container registry
func (co *CraneOptions) PushImage(pluginTarFilePath, image string) error {
	cranePushCmd := cmd.NewCmdPush(&[]crane.Option{})
	return cranePushCmd.RunE(cranePushCmd, []string{pluginTarFilePath, image})
}
