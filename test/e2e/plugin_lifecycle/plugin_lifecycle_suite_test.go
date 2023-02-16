// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugin provides plugin command specific E2E test cases
package plugin

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPluginLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PluginLifecycle Suite")
}