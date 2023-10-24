// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package frameworkfunctionality implements CLI E2E framework API usage test cases
package frameworkfunctionality

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI E2E Framework API Suite")
}

var (
	tf *framework.Framework
)

// BeforeSuite performs setup before the suite is started
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()
})

// AfterSuite performs teardown after the suite is complete
var _ = AfterSuite(func() {
})
