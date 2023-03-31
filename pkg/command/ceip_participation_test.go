// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
)

var _ = Describe("ceip-participation command tests", func() {

	Describe("ceip-participation command set/get tests", func() {
		var (
			tkgConfigFile   *os.File
			tkgConfigFileNG *os.File
			err             error
		)

		BeforeEach(func() {
			tkgConfigFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), tkgConfigFile.Name())
			Expect(err).To(BeNil(), "Error while copying tanzu config file for testing")
			os.Setenv("TANZU_CONFIG", tkgConfigFile.Name())

			tkgConfigFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", tkgConfigFileNG.Name())
			err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng.yaml"), tkgConfigFileNG.Name())
			Expect(err).To(BeNil(), "Error while copying tanzu config_ng.yaml file for testing")
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tkgConfigFile.Name())
			os.RemoveAll(tkgConfigFileNG.Name())
			resetLoginCommandFlags()
		})
		Context("ceip-participation set to true", func() {
			It("ceip-participation set should be successful and get should return status as 'Opt-in' status", func() {
				ceipSetCmd := newCEIPParticipationSetCmd()
				ceipSetCmd.SetArgs([]string{"true"})
				err = ceipSetCmd.Execute()
				Expect(err).To(BeNil())

				ceipGetCmd := newCEIPParticipationGetCmd()
				var out bytes.Buffer
				ceipGetCmd.SetOut(&out)
				err = ceipGetCmd.Execute()
				Expect(err).To(BeNil())
				Expect(out.String()).To(ContainSubstring("Opt-in"))

				ceipSetCmd = newCEIPParticipationSetCmd()
				ceipSetCmd.SetArgs([]string{"True"})
				err = ceipSetCmd.Execute()
				Expect(err).To(BeNil())

				ceipGetCmd = newCEIPParticipationGetCmd()
				out.Reset()
				ceipGetCmd.SetOut(&out)
				err = ceipGetCmd.Execute()
				Expect(err).To(BeNil())
				Expect(out.String()).To(ContainSubstring("Opt-in"))

			})
		})
		Context("ceip-participation set to false", func() {
			It("ceip-participation set should be successful and get should return status as 'Opt-out' status", func() {
				ceipSetCmd := newCEIPParticipationSetCmd()
				ceipSetCmd.SetArgs([]string{"false"})
				err = ceipSetCmd.Execute()
				Expect(err).To(BeNil())

				ceipGetCmd := newCEIPParticipationGetCmd()
				var out bytes.Buffer
				ceipGetCmd.SetOut(&out)
				err = ceipGetCmd.Execute()
				Expect(err).To(BeNil())
				Expect(out.String()).To(ContainSubstring("Opt-out"))

				ceipSetCmd = newCEIPParticipationSetCmd()
				ceipSetCmd.SetArgs([]string{"False"})
				err = ceipSetCmd.Execute()
				Expect(err).To(BeNil())

				ceipGetCmd = newCEIPParticipationGetCmd()
				out.Reset()
				ceipGetCmd.SetOut(&out)
				err = ceipGetCmd.Execute()
				Expect(err).To(BeNil())
				Expect(out.String()).To(ContainSubstring("Opt-out"))
			})
		})
		Context("ceip-participation set to invalid boolean argument", func() {
			It("ceip-participation set should fail", func() {
				ceipSetCmd := newCEIPParticipationSetCmd()
				ceipSetCmd.SetArgs([]string{"fakebool"})
				err = ceipSetCmd.Execute()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("incorrect boolean argument:"))
			})
		})
		Context("ceip-participation set without argument", func() {
			It("ceip-participation set should fail", func() {
				ceipSetCmd := newCEIPParticipationSetCmd()
				err = ceipSetCmd.Execute()
				Expect(err).To(HaveOccurred())
			})
		})
	})

})
