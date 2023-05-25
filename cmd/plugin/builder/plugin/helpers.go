// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package plugin implements plugin specific publishing functions
package plugin

import (
	"os"
	"path/filepath"

	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

const (
	localRegistry  = "localhost"
	dockerTemplate = `FROM scratch
COPY . ./`
)

func getDockerTemplateFileForPluginPackageBuild() (string, error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}
	templateFile := filepath.Join(tmpDir, "Dockerfile")
	err = utils.SaveFile(templateFile, []byte(dockerTemplate))
	if err != nil {
		return "", err
	}
	return templateFile, nil
}
