// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	topLevelFileName = "tanzu.md"
)

func getGeneratedFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, info.Name())
		}
		if err != nil {
			return err
		}
		return nil
	})

	return files, err
}

func topLevelHelp(docsDir string) string {
	topLevelFilePath := filepath.Join(docsDir, topLevelFileName)

	b, err := os.ReadFile(topLevelFilePath)
	if err != nil {
		fmt.Print(err)
		return ""
	}

	return string(b)
}

func TestGenDocs(t *testing.T) {
	assert := assert.New(t)

	docsDir, err := os.MkdirTemp("", "tanzu-cli-gendocs")
	assert.Nil(err)
	defer os.RemoveAll(docsDir)

	rootCmd, err := NewRootCmd()
	assert.Nil(err)
	rootCmd.SetArgs([]string{"generate-all-docs", "--docs-dir", docsDir})
	err = rootCmd.Execute()
	assert.Nil(err)

	files, err := getGeneratedFiles(docsDir)
	assert.Nil(err)
	assert.Contains(files, "tanzu.md")
	// expects multi level generation of core commands too
	assert.Contains(files, "tanzu_config.md")
	assert.Contains(files, "tanzu_config_set.md")
	assert.Contains(files, "tanzu_plugin_group_search.md")

	// expect only non-hidden commands to be referenced
	topLevelHelpText := topLevelHelp(docsDir)
	assert.Contains(topLevelHelpText, "tanzu_context.md")
	assert.Contains(topLevelHelpText, "[tanzu version]")
	assert.NotContains(topLevelHelpText, "tanzu_generate")
}
