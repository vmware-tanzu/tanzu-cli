// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
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

func TestCompletionCeip(t *testing.T) {
	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		// =====================
		// tanzu ceip set
		// =====================
		{
			test: "completion of true/false for the ceip set command",
			args: []string{"__complete", "ceip", "set", ""},
			// ":36" is the value of the ShellCompDirectiveNoFileComp | ShellCompDirectiveKeepOrder
			expected: "true\tAccept to participate\n" +
				"false\tRefuse to participate\n" + ":36\n",
		},
		{
			test: "no completion after the first arg for the ceip set command",
			args: []string{"__complete", "ceip", "set", "true", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		// =====================
		// tanzu ceip get
		// =====================
		{
			test: "no completion for the ceip get command",
			args: []string{"__complete", "ceip", "get", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
	}

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			rootCmd, err := NewRootCmd()
			assert.Nil(err)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Nil(err)

			assert.Equal(spec.expected, out.String())
		})
	}
}
