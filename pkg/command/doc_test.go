// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
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

func TestCompletionGenerateDocs(t *testing.T) {
	// This is global logic and needs not be tested for each
	// command.  Let's deactivate it.
	os.Setenv("TANZU_ACTIVE_HELP", "no_short_help")

	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		{
			test: "no completion for the generate-all-docs command",
			args: []string{"__complete", "generate-all-docs", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		{
			test: "file completion for the generate-all-docs --docs-dir flag",
			args: []string{"__complete", "generate-all-docs", "--docs-dir", ""},
			// ":0" is the value of the ShellCompDirectiveDefault
			expected: ":0\n",
		},
	}

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Nil(err)

			assert.Equal(spec.expected, out.String())
		})
	}

	os.Unsetenv("TANZU_ACTIVE_HELP")
}
