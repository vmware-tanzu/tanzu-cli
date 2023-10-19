// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompletionGenerateDocs(t *testing.T) {
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
}
