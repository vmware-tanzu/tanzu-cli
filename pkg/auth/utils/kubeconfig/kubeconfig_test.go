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
			deleteTempFile(kubeconfigFilePath3)
		})
		It("should merge with existing kubeconf file without switching context", func() {
			copyFile(kubeconfigFilePath2, kubeconfigFilePath3)
			validateKubeconfig(kubeconfigFilePath, 2, "foo-context")
			validateKubeconfig(kubeconfigFilePath3, 1, "baz-context")
			kubeconfFileContent, _ := os.ReadFile(kubeconfigFilePath)
			err := MergeKubeConfigWithoutSwitchContext(kubeconfFileContent, kubeconfigFilePath3)
			validateKubeconfig(kubeconfigFilePath3, 3, "baz-context")
			Expect(err).To(BeNil())
		})
		It("should merge with existing empty kubeconf file, using same current context from source", func() {
			kubeconfFileContent, _ := os.ReadFile(kubeconfigFilePath)
			err := MergeKubeConfigWithoutSwitchContext(kubeconfFileContent, kubeconfigFilePath3)
			validateKubeconfig(kubeconfigFilePath3, 2, "foo-context")
			Expect(err).To(BeNil())
		})
		It("should return value for default kubeconfig file", func() {
			defKubeConf := GetDefaultKubeConfigFile()
			Expect(defKubeConf).ToNot(BeNil())
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
