// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

func readOutput(t *testing.T, r io.Reader, c chan<- []byte) {
	data, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}
	c <- data
}

func TestVersion(t *testing.T) {
	assert := assert.New(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Error(err)
	}
	c := make(chan []byte)
	go readOutput(t, r, c)

	// Set up for our test
	stdout := os.Stdout
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()
	os.Stdout = w
	os.Stderr = w

	buildinfo.Version = "1.2.3"
	buildinfo.Date = "today"
	buildinfo.SHA = "cafecafe"
	originalArch := cli.GOARCH
	cli.GOARCH = "amd64"
	defer func() {
		buildinfo.Version = ""
		buildinfo.Date = ""
		buildinfo.SHA = ""
		cli.GOARCH = originalArch
	}()

	err = newVersionCmd().Execute()
	assert.Nil(err)
	w.Close()

	got := <-c
	expected := "version: 1.2.3\nbuildDate: today\nsha: cafecafe\narch: amd64\n"
	assert.Equal(expected, string(got))
}

func TestCompletionVersion(t *testing.T) {
	// This is global logic and needs not be tested for each
	// command.  Let's deactivate it.
	os.Setenv("TANZU_ACTIVE_HELP", "no_short_help")

	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		{
			test: "no completion for the version command",
			args: []string{"__complete", "version", ""},
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

	os.Unsetenv("TANZU_ACTIVE_HELP")
}
