// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package configpaths

import (
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "configpaths Suite")
}

var _ = Describe("Unit tests for filepaths utils", func() {
	home, err := os.UserHomeDir()
	Expect(err).To(BeNil())
	It("GetRegistryCertFile should return the registry certificate file path", func() {
		regCertFilePath, err := GetRegistryCertFile()
		Expect(err).To(BeNil())
		Expect(regCertFilePath).To(Equal(path.Join(home, constants.TKGRegistryCertFile)))
	})

	It("GetRegistryTrustedCACertFileForWindows should return the registry certificate file path for windows os", func() {
		regCertFilePath, err := GetRegistryTrustedCACertFileForWindows()
		Expect(err).To(BeNil())
		Expect(regCertFilePath).To(Equal(path.Join(home, constants.TKGRegistryTrustedRootCAFileForWindows)))
	})
})
