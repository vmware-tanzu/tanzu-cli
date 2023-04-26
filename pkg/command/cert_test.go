// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package command

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gopkg.in/yaml.v3"

	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

var _ = Describe("config cert command tests", func() {

	Describe("config cert add/list command tests", func() {
		var (
			tanzuConfigFile   *os.File
			tanzuConfigFileNG *os.File
			caCertFile        *os.File
			err               error
		)
		const (
			fakeCACertData        = "fake ca cert data"
			fakeCACertDataUpdated = "fake ca cert data updated"
			testHost              = "test.vmware.com"
		)

		BeforeEach(func() {

			tanzuConfigFile, err = os.CreateTemp("", "config")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG", tanzuConfigFile.Name())

			tanzuConfigFileNG, err = os.CreateTemp("", "config_ng")
			Expect(err).To(BeNil())
			os.Setenv("TANZU_CONFIG_NEXT_GEN", tanzuConfigFileNG.Name())

			caCertFile, err = os.CreateTemp("", "cert")
			err = os.WriteFile(caCertFile.Name(), []byte(fakeCACertData), 0600)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			os.Unsetenv("TANZU_CONFIG")
			os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
			os.RemoveAll(tanzuConfigFile.Name())
			os.RemoveAll(tanzuConfigFileNG.Name())
			os.RemoveAll(caCertFile.Name())
			resetCertCommandFlags()
		})
		Context("config cert add with all the options", func() {
			It("should be success and cert list should return the cert successfully", func() {
				certCmd := newCertCmd()
				certCmd.SetArgs([]string{
					"add", "--host", testHost, "--ca-certificate", caCertFile.Name(),
					"--skip-cert-verify", "true", "--insecure", "true"})
				err = certCmd.Execute()
				Expect(err).To(BeNil())

				certs := listCerts()
				Expect(certs).To(ContainElement(certOutputRow{
					Host:                 testHost,
					CACertificate:        "<REDACTED>",
					SkipCertVerification: "true",
					Insecure:             "true",
				}))
			})
			It("should return error if the cert for a host already exists", func() {
				certCmd := newCertCmd()
				certCmd.SetArgs([]string{
					"add", "--host", testHost, "--ca-certificate", caCertFile.Name(),
					"--skip-cert-verify", "true", "--insecure", "true"})
				err = certCmd.Execute()
				Expect(err).To(BeNil())

				certCmd.SetArgs([]string{
					"add", "--host", testHost, "--ca-certificate", caCertFile.Name(),
					"--skip-cert-verify", "true", "--insecure", "false"})
				err = certCmd.Execute()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(`certificate configuration for host "test.vmware.com" already exist`))

			})

			It("should return error if the arguments for 'skip-cert-verify' and 'insecure' are not boolean", func() {
				certCmd := newCertCmd()
				certCmd.SetArgs([]string{
					"add", "--host", testHost, "--ca-certificate", caCertFile.Name(),
					"--skip-cert-verify", "true", "--insecure", "fakeint"})
				err = certCmd.Execute()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(`incorrect boolean argument for '--insecure' option : "fakeint"`))

				certCmd.SetArgs([]string{
					"add", "--host", testHost, "--ca-certificate", caCertFile.Name(),
					"--skip-cert-verify", "fakebool", "--insecure", "false"})
				err = certCmd.Execute()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(`incorrect boolean argument for '--skip-cert-verify' option : "fakebool"`))

			})
		})
		Context("config cert add with some options", func() {
			It("should return success and cert list should return the cert successfully", func() {
				certCmd := newCertCmd()
				certCmd.SetArgs([]string{
					"add", "--host", testHost, "--ca-certificate", caCertFile.Name()})
				err = certCmd.Execute()
				Expect(err).To(BeNil())

				certs := listCerts()
				Expect(certs).To(ContainElement(certOutputRow{
					Host:                 testHost,
					CACertificate:        "<REDACTED>",
					SkipCertVerification: "false",
					Insecure:             "false",
				}))
			})
		})
		Context("config cert update", func() {
			It("should update the host CA cert successfully", func() {
				certCmd := newCertCmd()
				certCmd.SetArgs([]string{
					"add", "--host", testHost, "--ca-certificate", caCertFile.Name()})
				err = certCmd.Execute()
				Expect(err).To(BeNil())

				gotCAData := getConfigCertData(testHost)
				Expect(gotCAData).To(Equal(fakeCACertData))

				// update the ca cert data
				err = os.WriteFile(caCertFile.Name(), []byte(fakeCACertDataUpdated), 0600)
				Expect(err).To(BeNil())
				certCmd.SetArgs([]string{
					"update", testHost, "--ca-certificate", caCertFile.Name()})
				err = certCmd.Execute()
				Expect(err).To(BeNil())

				gotCAData = getConfigCertData(testHost)
				Expect(gotCAData).To(Equal(fakeCACertDataUpdated))

			})
			It("should update the 'skipCertVerify' and 'insecure' config data successfully", func() {
				certCmd := newCertCmd()
				certCmd.SetArgs([]string{
					"add", "--host", testHost, "--ca-certificate", caCertFile.Name(),
					"--skip-cert-verify", "false", "--insecure", "false"})
				err = certCmd.Execute()
				Expect(err).To(BeNil())

				gotCAData := getConfigCertData(testHost)
				Expect(gotCAData).To(Equal(fakeCACertData))

				cert, err := configlib.GetCert(testHost)
				Expect(err).To(BeNil())
				Expect(cert.SkipCertVerify).To(Equal("false"))
				Expect(cert.Insecure).To(Equal("false"))

				// update the SkipCertVerify and Insecure configuration
				certCmd.SetArgs([]string{
					"update", testHost, "--skip-cert-verify", "true", "--insecure", "true"})
				err = certCmd.Execute()
				Expect(err).To(BeNil())

				cert, err = configlib.GetCert(testHost)
				Expect(err).To(BeNil())
				Expect(cert.SkipCertVerify).To(Equal("true"))
				Expect(cert.Insecure).To(Equal("true"))

			})
		})
		Context("config cert delete", func() {
			It("should delete the cert config successfully if configuration for host exists", func() {
				certCmd := newCertCmd()
				certCmd.SetArgs([]string{
					"add", "--host", testHost, "--ca-certificate", caCertFile.Name(),
					"--skip-cert-verify", "true", "--insecure", "true"})
				err = certCmd.Execute()
				Expect(err).To(BeNil())

				certs := listCerts()
				Expect(certs).To(ContainElement(certOutputRow{
					Host:                 testHost,
					CACertificate:        "<REDACTED>",
					SkipCertVerification: "true",
					Insecure:             "true",
				}))

				//delete the cert config
				certCmd.SetArgs([]string{"delete", testHost})
				err = certCmd.Execute()
				Expect(err).To(BeNil())

				certs = listCerts()
				Expect(certs).To(BeEmpty())

				// delete the cert config for host which doesn't exist
				certCmd.SetArgs([]string{"delete", testHost})
				err = certCmd.Execute()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(fmt.Sprintf("cert configuration for %s not found", testHost)))

			})
		})
	})

})

type certOutputRow struct {
	Host                 string `json:"host,omitempty" yaml:"host,omitempty"`
	CACertificate        string `json:"ca-certificate,omitempty" yaml:"ca-certificate,omitempty"`
	Insecure             string `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	SkipCertVerification string `json:"skip-cert-verification,omitempty" yaml:"skip-cert-verification,omitempty"`
}

func listCerts() []certOutputRow {
	var out bytes.Buffer
	certCmd := newCertCmd()
	certCmd.SetOut(&out)
	certCmd.SetArgs([]string{"list", "-o", "yaml"})
	err := certCmd.Execute()
	Expect(err).To(BeNil())
	certs := []certOutputRow{}
	err = yaml.Unmarshal(out.Bytes(), &certs)
	Expect(err).To(BeNil())
	return certs
}

func getConfigCertData(host string) string {
	cert, err := configlib.GetCert(host)
	Expect(err).To(BeNil())

	caData, err := base64.StdEncoding.DecodeString(cert.CACertData)
	Expect(err).To(BeNil())
	return string(caData)
}

func resetCertCommandFlags() {
	outputFormat = ""
	host = ""
	caCertPathForAdd = ""
	skipCertVerifyForAdd = ""
	insecureForAdd = ""
	caCertPathForUpdate = ""
	skipCertVerifyForUpdate = ""
	insecureForUpdate = ""
}
