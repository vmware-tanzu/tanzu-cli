// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package clilifecycle

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCliLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI lifecycle E2E Test Suite")
}
