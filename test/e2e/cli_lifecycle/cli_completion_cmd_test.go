// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package clilifecycle provides cli E2E test cases for basic commands like init, version, completion
package clilifecycle

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Command-completion]", func() {
	Context("tests for tanzu completion ommand", func() {
		When("completion command executed", func() {
			It("When the completion command is executed without specifying a shell input value", func() {
				_, _, err := tf.CliOps.CompletionCmd("")
				Expect(err).NotTo(BeNil(), "There should be an error when running the completion command without specifying the shell input.")
				Expect(err.Error()).To(ContainSubstring(framework.CompletionWithoutShell))
			})
			It("When the completion command is executed with bash as the input", func() {
				out, _, err := tf.CliOps.CompletionCmd("bash")
				Expect(err).To(BeNil(), "There should be no errors when using the completion command with bash as the shell input.")
				Expect(out).To(ContainSubstring(framework.CompletionOutputForBash))
			})
			It("When the completion command is executed with zsh as the input", func() {
				out, _, err := tf.CliOps.CompletionCmd("zsh")
				Expect(err).To(BeNil(), "There should be no errors when using the completion command with zsh as the shell input.")
				Expect(out).To(ContainSubstring(framework.CompletionOutputForZsh))
			})
			It("When the completion command is executed with fish as the input", func() {
				out, _, err := tf.CliOps.CompletionCmd("fish")
				Expect(err).To(BeNil(), "There should be no errors when using the completion command with fish as the shell input.")
				Expect(out).To(ContainSubstring(framework.CompletionOutputForFish))
			})
			It("When the completion command is executed with powershell as the input", func() {
				out, _, err := tf.CliOps.CompletionCmd("powershell")
				Expect(err).To(BeNil(), "There should be no errors when using the completion command with powershell as the shell input.")
				Expect(out).To(ContainSubstring(framework.CompletionOutputForPowershell))
			})
			It("When the completion command is executed with pwsh as the input", func() {
				out, _, err := tf.CliOps.CompletionCmd("pwsh")
				Expect(err).To(BeNil(), "There should be no errors when using the completion command with powershell as the shell input.")
				Expect(out).To(ContainSubstring(framework.CompletionOutputForPowershell))
			})
			It("When the cobra __complete command is executed", func() {
				out, _, err := tf.Exec.TanzuCmdExec(framework.CobraCompleteCmd)
				Expect(err).To(BeNil(), "There should be no errors when running cobra __complete command")
				Expect(out).To(ContainSubstring("Completion ended with directive: ShellCompDirectiveNoFileComp"))
			})
		})
	})
})
