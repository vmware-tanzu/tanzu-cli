// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package tanzu

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testingDir string

func TestTanzuAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/auth/tanzu Suite")
}

const (
	fakeCAcertPath = "../../fakes/certs/fake-ca.crt"
)

var _ = Describe("Unit tests for tanzu auth", func() {
	var (
		err          error
		endpoint     string
		tanzuContext *configtypes.Context
	)

	const (
		fakeContextName = "fake-tanzu-context"
		fakeAccessToken = "fake-access-token"
		fakeOrgID       = "fake-org-id"
		fakeEndpoint    = "fake.tanzu.cloud.vmware.com"
	)

	Describe("GetTanzuKubeconfig()", func() {
		var kubeConfigPath, kubeContext, clusterAPIServerURL string
		BeforeEach(func() {
			err = createTempDirectory("kubeconfig-test")
			Expect(err).ToNot(HaveOccurred())
			tanzuContext = &configtypes.Context{
				Name: fakeContextName,
				GlobalOpts: &configtypes.GlobalServer{
					Auth: configtypes.GlobalServerAuth{
						AccessToken: fakeAccessToken,
					},
				},
			}
			err = os.Setenv("KUBECONFIG", filepath.Join(testingDir, ".kube", "config"))
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			deleteTempDirectory()
			err = os.Unsetenv("KUBECONFIG")
			Expect(err).ToNot(HaveOccurred())
		})
		Context("When the endpoint caCertPath file doesn't exist", func() {
			BeforeEach(func() {
				nonExistingCACertPath := filepath.Join(testingDir, "non-existing-file")
				_, _, _, err = GetTanzuKubeconfig(tanzuContext, fakeEndpoint, fakeOrgID, nonExistingCACertPath, false)
			})
			It("should return the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("error reading CA certificate file"))
			})
		})
		Context("When the endpoint caCertPath provided exists and skipTLSVerify is set to false", func() {
			BeforeEach(func() {
				kubeConfigPath, kubeContext, clusterAPIServerURL, err = GetTanzuKubeconfig(tanzuContext, fakeEndpoint, fakeOrgID, fakeCAcertPath, false)
			})
			It("should set the 'certificate-authority-data' in kubeconfig and 'insecure-skip-tls-verify' should be unset", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(kubeConfigPath).Should(Equal(filepath.Join(testingDir, ".kube", "config")))
				Expect(kubeContext).Should(Equal(kubeconfigContextName(tanzuContext.Name)))
				config, err := clientcmd.LoadFromFile(kubeConfigPath)
				Expect(err).ToNot(HaveOccurred())

				gotClusterName := config.Contexts[kubeContext].Cluster
				cluster := config.Clusters[config.Contexts[kubeContext].Cluster]
				user := config.AuthInfos[config.Contexts[kubeContext].AuthInfo]

				Expect(cluster.Server).To(Equal(clusterAPIServerURL))
				Expect(config.Contexts[kubeContext].AuthInfo).To(Equal(kubeconfigUserName(tanzuContext.Name)))
				Expect(gotClusterName).To(Equal(kubeconfigClusterName(tanzuContext.Name)))
				Expect(len(cluster.CertificateAuthorityData)).ToNot(Equal(0))
				Expect(user.Exec).To(Equal(getExecConfig(tanzuContext)))
			})
		})
		Context("When endpointCACertPath is not provided and skipTLSVerify is set to true", func() {
			BeforeEach(func() {
				kubeConfigPath, kubeContext, clusterAPIServerURL, err = GetTanzuKubeconfig(tanzuContext, endpoint, fakeOrgID, "", true)
			})
			It("should not set the 'certificate-authority-data' in kubeconfig and 'insecure-skip-tls-verify' should be set", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(kubeConfigPath).Should(Equal(filepath.Join(testingDir, ".kube", "config")))
				Expect(kubeContext).Should(Equal("tanzu-cli-" + tanzuContext.Name))
				config, err := clientcmd.LoadFromFile(kubeConfigPath)
				Expect(err).ToNot(HaveOccurred())

				gotClusterName := config.Contexts[kubeContext].Cluster
				cluster := config.Clusters[config.Contexts[kubeContext].Cluster]
				user := config.AuthInfos[config.Contexts[kubeContext].AuthInfo]

				Expect(cluster.Server).To(Equal(clusterAPIServerURL))
				Expect(config.Contexts[kubeContext].AuthInfo).To(Equal("tanzu-cli-" + tanzuContext.Name + "-user"))
				Expect(gotClusterName).To(Equal("tanzu-cli-" + tanzuContext.Name))
				Expect(len(cluster.CertificateAuthorityData)).To(Equal(0))
				Expect(cluster.InsecureSkipTLSVerify).To(Equal(true))
				Expect(user.Exec).To(Equal(getExecConfig(tanzuContext)))
			})
		})

	})
})

func createTempDirectory(prefix string) error {
	var err error
	testingDir, err = os.MkdirTemp("", prefix)
	if err != nil {
		fmt.Println("Error TempDir: ", err.Error())
		return err
	}
	return nil
}
func deleteTempDirectory() {
	os.Remove(testingDir)
}
