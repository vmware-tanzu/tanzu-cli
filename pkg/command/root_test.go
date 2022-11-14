// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCmd(t *testing.T) {
	assert := assert.New(t)
	rootCmd, err := NewRootCmd()
	assert.Nil(err)
	err = rootCmd.Execute()
	assert.Nil(err)
}

func TestExecute(t *testing.T) {
	assert := assert.New(t)
	err := Execute()
	assert.Nil(err)
}
