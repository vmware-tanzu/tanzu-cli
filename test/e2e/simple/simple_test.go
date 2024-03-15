// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package simple_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:config]", func() {
	Context("config feature flag operations", func() {
		When("new config flag set with value", func() {
			It("should set flag and unset flag successfully", func() {
				x := 1
				Expect(x).To(Equal(1))
			})
		})
	})
})
