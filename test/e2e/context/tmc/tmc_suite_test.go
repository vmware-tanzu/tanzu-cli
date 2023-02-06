// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// tmc provides context command e2e test cases for tmc target
package tmc

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTmc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Context-TMC Suite")
}
