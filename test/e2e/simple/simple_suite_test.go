// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package simple_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "simple Suite")
}

// BeforeSuite creates KIND cluster needed to test 'tanzu config server' use cases
// initializes the tf
var _ = BeforeSuite(func() {
	 log.Info("BeforeSuite")
})

// AfterSuite deletes the KIND which is created in BeforeSuite
var _ = AfterSuite(func() {
	log.Info("AfterSuite")
})
