// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package sigverifier

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/pkg/configpaths"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cosignhelper"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

func TestSigVerifierSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "sigverifier package Suite")
}

var _ = Describe("Unit tests for discovery image signature verification", func() {
	var (
		err          error
		configFile   *os.File
		configFileNG *os.File
	)

	Describe("Verify inventory image signature", func() {
		var (
			cosignVerifier *fakes.Cosignhelperfake
			image          string
		)
		BeforeEach(func() {
			configFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", configFile.Name())

			configFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

			image = "test-image:latest"
		})
		AfterEach(func() {
			os.Unsetenv(constants.PluginDiscoveryImageSignatureVerificationSkipList)
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(configFile.Name())
			os.RemoveAll(configFileNG.Name())
		})
		Context("Cosign signature verification is success", func() {
			It("should return success", func() {
				cosignVerifier = &fakes.Cosignhelperfake{}
				cosignVerifier.VerifyReturns(nil)
				err = verifyInventoryImageSignature(image, cosignVerifier)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When Cosign signature verification failed and TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST environment variable is set", func() {
			It("should skip signature verification and return success", func() {
				cosignVerifier = &fakes.Cosignhelperfake{}
				cosignVerifier.VerifyReturns(fmt.Errorf("signature verification fake error"))
				os.Setenv(constants.PluginDiscoveryImageSignatureVerificationSkipList, image)
				err = verifyInventoryImageSignature(image, cosignVerifier)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("Cosign signature verification failed", func() {
			It("should return error", func() {
				cosignVerifier = &fakes.Cosignhelperfake{}
				cosignVerifier.VerifyReturns(fmt.Errorf("signature verification fake error"))
				err = verifyInventoryImageSignature(image, cosignVerifier)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("signature verification fake error"))
			})
		})
	})

	Describe("getCosignVerifier tests", func() {
		var (
			cosignVerifier cosignhelper.Cosignhelper
		)
		const (
			fakeCACertData = "fake ca cert data"
			testHost       = "test.vmware.com"
			image          = "test.vmware.com/tanzu/test-image:latest"
		)
		BeforeEach(func() {
			configFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", configFile.Name())

			configFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.Unsetenv(constants.PublicKeyPathForPluginDiscoveryImageSignature)
			os.RemoveAll(configFile.Name())
			os.RemoveAll(configFileNG.Name())
		})
		Context("When no custom cert data is provided for registry endpoint/host", func() {
			BeforeEach(func() {
				cert := &configtypes.Cert{
					Host:           testHost,
					CACertData:     base64.StdEncoding.EncodeToString([]byte(fakeCACertData)),
					SkipCertVerify: "true",
					Insecure:       "true",
				}
				err := configlib.SetCert(cert)
				Expect(err).To(BeNil())
			})
			It("should create cosign verifier successfully with registryOptions updated with configured custom cert data", func() {
				cosignVerifier, err = getCosignVerifier(image)
				Expect(err).ToNot(HaveOccurred())
				cvo, ok := cosignVerifier.(*cosignhelper.CosignVerifyOptions)
				Expect(ok).To(BeTrue())

				regFilePath, err := configpaths.GetRegistryCertFile()
				Expect(err).To(BeNil())
				Expect(cvo.RegistryOpts.CACertPaths).To(ContainElement(regFilePath))
				Expect(cvo.RegistryOpts.SkipCertVerify).To(BeTrue())
				Expect(cvo.RegistryOpts.AllowInsecure).To(BeTrue())
			})
			It("should create cosign verifier successfully with Image signature custom public key path if provided using the environment variable", func() {
				keyPath := "fake/path/to/publickey"
				os.Setenv(constants.PublicKeyPathForPluginDiscoveryImageSignature, keyPath)
				cosignVerifier, err = getCosignVerifier(image)
				Expect(err).ToNot(HaveOccurred())
				cvo, ok := cosignVerifier.(*cosignhelper.CosignVerifyOptions)
				Expect(ok).To(BeTrue())

				Expect(cvo.PublicKeyPath).To(Equal(keyPath))

			})
		})
		Context("When custom cert data is not provided for registry endpoint/host in the config file", func() {
			It("cosign verifier should be created successfully with default registryOptions", func() {
				cosignVerifier, err = getCosignVerifier(image)
				Expect(err).ToNot(HaveOccurred())
				cvo, ok := cosignVerifier.(*cosignhelper.CosignVerifyOptions)
				Expect(ok).To(BeTrue())

				regFilePath, err := configpaths.GetRegistryCertFile()
				Expect(err).To(BeNil())
				Expect(cvo.RegistryOpts.CACertPaths).ToNot(ContainElement(regFilePath))
				Expect(cvo.RegistryOpts.SkipCertVerify).To(BeFalse())
				Expect(cvo.RegistryOpts.AllowInsecure).To(BeFalse())
			})
		})
	})
})
