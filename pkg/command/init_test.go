// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompletionInit(t *testing.T) {
	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		{
			test: "no completion for the init command",
			args: []string{"__complete", "init", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
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
