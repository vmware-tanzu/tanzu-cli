// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	cliconfig "github.com/vmware-tanzu/tanzu-cli/pkg/config"
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
			It("should return successfully with agreement acceptance and version registered", func() {
				// run twice, to show second accept is idempotent, with no
				// effect to the status or accepted EULA version
				for i := 0; i < 2; i++ {
					eulaCmd := newEULACmd()
					eulaCmd.SetArgs([]string{"accept"})
					err = eulaCmd.Execute()
					Expect(err).To(BeNil())

					eulaStatus, err := config.GetEULAStatus()
					Expect(err).To(BeNil())
					Expect(eulaStatus).To(Equal(config.EULAStatusAccepted))

					acceptedVersions, err := config.GetEULAAcceptedVersions()
					Expect(err).To(BeNil())

					if cliconfig.CurrentEULAVersion != "" {
						Expect(acceptedVersions).To(Equal([]string{cliconfig.CurrentEULAVersion}))
					} else {
						Expect(acceptedVersions).To(Equal([]string{}))
					}
				}
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
				cmd, err := NewRootCmdForTest()
				Expect(err).To(BeNil())
				cmd.SetArgs([]string{"context", "list"})
				err = cmd.Execute()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("agreement not accepted"))
				os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
			})

			It("should not invoke the eula prompt if EULA has been accepted", func() {
				os.Setenv("TANZU_CLI_EULA_PROMPT_ANSWER", "yes")
				cmd, err := NewRootCmdForTest()
				Expect(err).To(BeNil())
				cmd.SetArgs([]string{"context", "list"})
				err = cmd.Execute()
				Expect(err).To(BeNil())
				os.Unsetenv("TANZU_CLI_EULA_PROMPT_ANSWER")
			})
		})
	})
})

var _ = Describe("EULA version checking tests", func() {
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

		os.Setenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER", "No")
	})
	AfterEach(func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.Unsetenv("TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER")
		os.RemoveAll(tanzuConfigFile.Name())
		os.RemoveAll(tanzuConfigFileNG.Name())
	})

	DescribeTable("Running arbitrary command when EULA status is accepted", func(versionToAccept string, versionsAlreadyAccepted []string, expectToPrompt bool) {
		cliconfig.CurrentEULAVersion = versionToAccept

		err := config.SetEULAAcceptedVersions(versionsAlreadyAccepted)
		Expect(err).To(BeNil())

		err = config.SetEULAStatus(config.EULAStatusAccepted)
		Expect(err).To(BeNil())

		cmd, err := NewRootCmdForTest()
		Expect(err).To(BeNil())
		cmd.SetArgs([]string{"context", "list"})
		checkForPromptOnExecute(cmd, expectToPrompt)
	},
		Entry("still prompt when no accepted version found",
			"v1.0.0", []string{}, true),
		Entry("still prompt when no accepted version matches in major.minor",
			"v1.0.0", []string{"v1.1.0", "v0.9.9"}, true),
		Entry("do not prompt when accepted version found",
			"v1.6.0", []string{"v1.6.0", "v0.9.9"}, false),
		Entry("do not prompt when found accepted older version matching major.minor",
			"v2.0.1", []string{"v2.0.0"}, false),
		Entry("do not prompt when found accepted newer version matching major.minor",
			"v2.0.0", []string{"v2.0.1"}, false),
		Entry("do not prompt when no current EULA version is set",
			"", []string{"v2.0.0"}, false),
		Entry("do not prompt when no current EULA version is set, even with no accepted versions",
			"", []string{}, false),
	)
})

func TestCompletionEULA(t *testing.T) {
	// This is global logic and needs not be tested for each
	// command.  Let's deactivate it.
	os.Setenv("TANZU_ACTIVE_HELP", "no_short_help")

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
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
		// =====================
		// tanzu config eula show
		// =====================
		{
			test: "no completion for the eula show command",
			args: []string{"__complete", "config", "eula", "show", ""},
			// ":4" is the value of the ShellCompDirectiveNoFileComp
			expected: "_activeHelp_ " + compNoMoreArgsMsg + "\n:4\n",
		},
	}

	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			rootCmd, err := NewRootCmdForTest()
			assert.Nil(err)

			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetArgs(spec.args)

			err = rootCmd.Execute()
			assert.Nil(err)

			assert.Equal(spec.expected, out.String())
		})
	}

	os.Unsetenv("TANZU_ACTIVE_HELP")
}
