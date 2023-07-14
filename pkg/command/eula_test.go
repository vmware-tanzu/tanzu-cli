// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

// Executes the command and verify if prompt is expected to be shown or not
func checkForPromptOnExecute(cmd *cobra.Command, expectPrompt bool) {
	err := cmd.Execute()
	// Survey prompts when run without attached tty will fail. Use this fact
	// to help detect that a prompt is indeed presented.
	if expectPrompt {
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("prompt failed"))
	} else {
		Expect(err).To(BeNil())
	}
}

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

			os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
			os.RemoveAll(tanzuConfigFile.Name())
			os.RemoveAll(tanzuConfigFileNG.Name())
		})

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
		Context("When invoking the show command", func() {
			It("should invoke the eula prompt", func() {
				eulaCmd := newEULACmd()
				eulaCmd.SetArgs([]string{"show"})
				checkForPromptOnExecute(eulaCmd, true)
			})

			It("should invoke the eula prompt even if EULA is accepted", func() {
				acceptCmd := newEULACmd()
				acceptCmd.SetArgs([]string{"accept"})
				err = acceptCmd.Execute()
				Expect(err).To(BeNil())

				eulaCmd := newEULACmd()
				eulaCmd.SetArgs([]string{"show"})
				checkForPromptOnExecute(eulaCmd, true)
			})
		})
		Context("When invoking an arbitrary command", func() {
			It("should invoke the eula prompt if EULA has not been accepted", func() {
				os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "No")
				cmd, err := NewRootCmd()
				Expect(err).To(BeNil())
				cmd.SetArgs([]string{"context", "list"})
				err = cmd.Execute()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("terms not accepted"))
				os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
			})

			It("should not invoke the eula prompt if EULA has been accepted", func() {
				os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "yes")
				cmd, err := NewRootCmd()
				Expect(err).To(BeNil())
				cmd.SetArgs([]string{"context", "list"})
				err = cmd.Execute()
				Expect(err).To(BeNil())
				os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
			})
		})
	})
})

func TestCompletionEULA(t *testing.T) {
	tests := []struct {
		test     string
		args     []string
		expected string
	}{
		// =====================
		// tanzu config eula accept
		// =====================
		{
			test: "no completion for the eula accept command",
			args: []string{"__complete", "config", "eula", "accept", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: ":4\n",
		},
		// =====================
		// tanzu config eula show
		// =====================
		{
			test: "no completion for the eula show command",
			args: []string{"__complete", "config", "eula", "show", ""},
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
