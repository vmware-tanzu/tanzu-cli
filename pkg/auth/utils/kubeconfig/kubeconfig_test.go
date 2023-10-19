// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package kubeconfig provides kubeconfig access functions.
package kubeconfig

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/tools/clientcmd"
)

func TestAuthUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/auth/tkg/util/kubeconfig Suite")
}

var (
	kubeconfigFilePath  string
	kubeconfigFilePath2 string
	kubeconfigFilePath3 string
)

const ConfigFilePermissions = 0o600

var _ = Describe("Unit tests for kubeconfig use cases", func() {
	Context("when valid kubeconfig file is provided", func() {
		BeforeEach(func() {
			kubeconfigFilePath = "../../../fakes/config/kubeconfig1.yaml"
			kubeconfigFilePath2 = "../../../fakes/config/kubeconfig2.yaml"
			kubeconfigFilePath3 = "../../../fakes/config/kubeconfig3_temp_rnhwe.yaml"
			validateKubeconfig(kubeconfigFilePath, 3, "foo-context")
			validateKubeconfig(kubeconfigFilePath2, 1, "baz-context")
		})
		AfterEach(func() {
			deleteTempFile(kubeconfigFilePath3)
		})
		It("should merge with existing kubeconf file without switching context", func() {
			copyFile(kubeconfigFilePath2, kubeconfigFilePath3)
			validateKubeconfig(kubeconfigFilePath3, 1, "baz-context")
			kubeconfFileContent, _ := os.ReadFile(kubeconfigFilePath)
			err := MergeKubeConfigWithoutSwitchContext(kubeconfFileContent, kubeconfigFilePath3)
			validateKubeconfig(kubeconfigFilePath3, 4, "baz-context")
			Expect(err).To(BeNil())
		})
		It("should merge with existing empty kubeconf file, using same current context from source", func() {
			kubeconfFileContent, _ := os.ReadFile(kubeconfigFilePath)
			err := MergeKubeConfigWithoutSwitchContext(kubeconfFileContent, kubeconfigFilePath3)
			validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
			Expect(err).To(BeNil())
		})
		It("should return value for default kubeconfig file", func() {
			defKubeConf := GetDefaultKubeConfigFile()
			Expect(defKubeConf).ToNot(BeNil())
		})

		Context("Setting current context", func() {
			Context("when context is not present in kubeconfig file", func() {
				It("should fail", func() {
					copyFile(kubeconfigFilePath, kubeconfigFilePath3)
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
					err := SetCurrentContext(kubeconfigFilePath3, "MISSING-context")
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("context \"MISSING-context\" does not exist"))
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
				})
			})

			Context("when context is not provided", func() {
				It("should fail", func() {
					copyFile(kubeconfigFilePath, kubeconfigFilePath3)
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
					err := SetCurrentContext(kubeconfigFilePath3, "")
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("context is not provided"))
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
				})
			})

			Context("when kubeconfig is not loadable", func() {
				It("should fail", func() {
					err := SetCurrentContext("MISSING-file", "bar-context")
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("unable to load kubeconfig:"))
				})
			})

			Context("when context is present in kubeconfig file", func() {
				It("should update current context to it", func() {
					copyFile(kubeconfigFilePath, kubeconfigFilePath3)
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
					err := SetCurrentContext(kubeconfigFilePath3, "bar-context")
					Expect(err).To(BeNil())
					validateKubeconfig(kubeconfigFilePath3, 3, "bar-context")
				})
			})

		})
		Context("Deleting Context related information from Kubeconfig", func() {
			Context("when context is not present in kubeconfig file", func() {
				It("should not fail", func() {
					copyFile(kubeconfigFilePath, kubeconfigFilePath3)
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
					err := DeleteContextFromKubeConfig(kubeconfigFilePath3, "MISSING-context")
					Expect(err).To(BeNil())
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
				})
			})

			Context("when context is not provided", func() {
				It("should not fail", func() {
					copyFile(kubeconfigFilePath, kubeconfigFilePath3)
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
					err := DeleteContextFromKubeConfig(kubeconfigFilePath3, "")
					Expect(err).To(BeNil())
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
				})
			})

			Context("when kubeconfig is not loadable", func() {
				It("should fail", func() {
					err := DeleteContextFromKubeConfig("MISSING-file", "bar-context")
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("unable to load kubeconfig:"))
				})
			})

			Context("when context is present in kubeconfig file", func() {
				It("should delete the context,cluster and user and also delete the current-context if the deleted context is current", func() {
					copyFile(kubeconfigFilePath, kubeconfigFilePath3)
					validateKubeconfig(kubeconfigFilePath3, 3, "foo-context")
					err := DeleteContextFromKubeConfig(kubeconfigFilePath3, "foo-context")
					Expect(err).To(BeNil())
					kubecfg, err := clientcmd.LoadFromFile(kubeconfigFilePath3)
					Expect(err).To(BeNil())
					Expect(kubecfg.Clusters["foo-cluster"]).To(BeNil())
					Expect(kubecfg.AuthInfos["blue-user"]).To(BeNil())
					Expect(kubecfg.Contexts["foo-context"]).To(BeNil())
					Expect(kubecfg.CurrentContext).To(BeEmpty())
				})
			})

		})
	})

})

func validateKubeconfig(kubeconfigFile string, numContexts int, currentContext string) {
	kubecfg, err := clientcmd.LoadFromFile(kubeconfigFile)
	Expect(err).To(BeNil())
	Expect(numContexts).To(Equal(len(kubecfg.Contexts)))
	Expect(currentContext).To(Equal(kubecfg.CurrentContext))
}

func copyFile(sourceFile, destFile string) {
	input, err := os.ReadFile(sourceFile)
	Expect(err).To(BeNil())
	err = os.WriteFile(destFile, input, ConfigFilePermissions)
	Expect(err).To(BeNil())
}

func deleteTempFile(filename string) {
	os.Remove(filename)
}
