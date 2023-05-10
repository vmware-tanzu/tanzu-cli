// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

var _ = Describe("EULA command tests", func() {
	Describe("config eula show tests", func() {
		var (
			tanzuConfigFile   *os.File
			tanzuConfigFileNG *os.File
			err               error
		)
		BeforeEach(func() {
			tanzuConfigFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", tanzuConfigFile.Name())

			tanzuConfigFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", tanzuConfigFileNG.Name())

			featureArray := strings.Split(constants.FeatureContextCommand, ".")
			err = config.SetFeature(featureArray[1], featureArray[2], "true")
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tanzuConfigFile.Name())
			os.RemoveAll(tanzuConfigFileNG.Name())
		})
		// TODO(vuil) : add tests for show command
		Context("When invoking the accept command", func() {
			It("should return successfully with agreement acceptance registered", func() {
				eulaCmd := newEULACmd()
				eulaCmd.SetArgs([]string{"accept"})
				err = eulaCmd.Execute()
				Expect(err).To(BeNil())

				eulaStatus, err := config.GetEULAStatus()
				Expect(err).To(BeNil())
				Expect(eulaStatus).To(Equal(config.EULAStatusAccepted))
			})
		})
	})
})
