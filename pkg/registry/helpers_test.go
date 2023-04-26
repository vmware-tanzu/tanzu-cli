// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package registry

import (
	"encoding/base64"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/configpaths"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

func TestClientConfigHelperSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/clientconfighelpers suite")
}

var _ = Describe("config cert command tests", func() {

	Describe("config cert add/list command tests", func() {
		var (
			tanzuConfigFile   *os.File
			tanzuConfigFileNG *os.File
			caCertDataOpt     string
			skipCertVerifyOpt string
			insecureOpt       string
			err               error
		)
		const (
			fakeCACertData = "fake ca cert data"
			testHost       = "test.vmware.com"
			trueStr        = "true"
		)

		BeforeEach(func() {
			tanzuConfigFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", tanzuConfigFile.Name())

			tanzuConfigFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", tanzuConfigFileNG.Name())

		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tanzuConfigFile.Name())
			os.RemoveAll(tanzuConfigFileNG.Name())
		})
		JustBeforeEach(func() {
			caCertDataOptB64 := ""
			if len(caCertDataOpt) > 0 {
				caCertDataOptB64 = base64.StdEncoding.EncodeToString([]byte(caCertDataOpt))
			}
			cert := &configtypes.Cert{
				Host:           testHost,
				CACertData:     caCertDataOptB64,
				SkipCertVerify: skipCertVerifyOpt,
				Insecure:       insecureOpt,
			}
			err := configlib.SetCert(cert)
			Expect(err).To(BeNil())
		})

		Context("When only custom CA cert data is provided for the registry host in the config", func() {
			BeforeEach(func() {
				caCertDataOpt = fakeCACertData
			})
			It("should return success and cert options should have registry cert path updated", func() {
				certOptions, err := GetRegistryCertOptions(testHost)
				Expect(err).To(BeNil())
				Expect(certOptions).ToNot(BeNil())

				regFilePath, err := configpaths.GetRegistryCertFile()
				Expect(err).To(BeNil())
				Expect(certOptions.CACertPaths).To(ContainElement(regFilePath))
				Expect(certOptions.SkipCertVerify).To(Equal(false))
				Expect(certOptions.Insecure).To(Equal(false))
			})
		})

		Context("When the custom CA cert data, skipCertVerify and Insecure options are provided for the registry host in the config", func() {
			BeforeEach(func() {
				caCertDataOpt = fakeCACertData
				skipCertVerifyOpt = trueStr
				insecureOpt = trueStr
			})
			It("should return success and cert options should have registry cert path updated and skipCertVerify and Insecure options are updated", func() {
				certOptions, err := GetRegistryCertOptions(testHost)
				Expect(err).To(BeNil())
				Expect(certOptions).ToNot(BeNil())

				regFilePath, err := configpaths.GetRegistryCertFile()
				Expect(err).To(BeNil())
				Expect(certOptions.CACertPaths).To(ContainElement(regFilePath))
				Expect(certOptions.SkipCertVerify).To(Equal(true))
				Expect(certOptions.Insecure).To(Equal(true))
			})
			It("should return defaults if the registry name doesn't match with the host existing in the config", func() {
				certOptions, err := GetRegistryCertOptions("NonExistingRegistryName")
				Expect(err).To(BeNil())
				Expect(certOptions).ToNot(BeNil())
				// check the cert options returned are default values
				Expect(certOptions.CACertPaths).To(BeEmpty())
				Expect(certOptions.SkipCertVerify).To(Equal(false))
				Expect(certOptions.Insecure).To(Equal(false))

			})
		})
	})
})
