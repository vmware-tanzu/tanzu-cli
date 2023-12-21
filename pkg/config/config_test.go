// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
)

var (
	prevValue string
)

const envVar = "test-conf-env"

var _ = Describe("config env variables", func() {
	Context("get env from config", func() {
		BeforeEach(func() {
			cc := &fakes.FakeConfigClientWrapper{}
			configClient = cc
			prevValue = os.Getenv(envVar)
			confEnvMap := map[string]string{envVar: envVar}
			cc.GetEnvConfigurationsReturns(confEnvMap)
		})
		It("env variable should be set with config env", func() {
			ConfigureEnvVariables()
			Expect(os.Getenv(envVar)).To(Equal(envVar))
			os.Setenv(envVar, prevValue)
		})
		It("env variable should not be changed if it already exists", func() {
			existingVal := "existing"
			os.Setenv(envVar, existingVal)
			ConfigureEnvVariables()
			Expect(os.Getenv(envVar)).To(Equal(existingVal))
			os.Setenv(envVar, prevValue)
		})
		It("env variable should not be changed if it already exists even if it is set to an empty value", func() {
			emptyVal := ""
			os.Setenv(envVar, emptyVal)
			ConfigureEnvVariables()
			Expect(os.Getenv(envVar)).To(Equal(emptyVal))
			os.Setenv(envVar, prevValue)
		})
	})
	Context("config return nil map", func() {
		BeforeEach(func() {
			cc := &fakes.FakeConfigClientWrapper{}
			configClient = cc
			cc.GetEnvConfigurationsReturns(nil)
		})
		It("execute without error", func() {
			ConfigureEnvVariables()
		})
	})
})
