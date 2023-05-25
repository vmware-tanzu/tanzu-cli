// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package clilifecycle

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

func TestCliLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI lifecycle E2E Test Suite")
}

var (
	tf *framework.Framework
)

// This suite has below e2e tests:
// 1. version and init commands
// 2. completion command
var _ = BeforeSuite(func() {
	tf = framework.NewFramework()

	// delete config files
	err := tf.Config.DeleteCLIConfigurationFiles()
	Expect(err).To(BeNil())
	// call init
	err = tf.Config.ConfigInit()
	Expect(err).To(BeNil())
	// should create config files
	Expect(tf.Config.IsCLIConfigurationFilesExists()).To(BeTrue())
})
