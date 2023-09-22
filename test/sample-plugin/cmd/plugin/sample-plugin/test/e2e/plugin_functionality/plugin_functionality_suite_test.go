// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package pluginfunctionality implements plugin functionality specific E2E test cases
package pluginfunctionality

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sample-Plugin functionality Suite")
}

var (
	tf *framework.Framework
)

// BeforeSuite performs setup before the suite is started
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()
	home := os.Getenv("HOME")
	os.Setenv("HOME", home+"/..")
})

// AfterSuite performs teardown after the suite is complete
var _ = AfterSuite(func() {
})
