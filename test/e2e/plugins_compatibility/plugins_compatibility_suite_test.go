// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// plugincompatibility provides plugins compatibility E2E test cases
package plugincompatibility_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPluginsCompatibility(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PluginsCompatibility Suite")
}
