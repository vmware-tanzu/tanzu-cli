// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package tanzu provides functionality related to authentication for the Tanzu control plane
package tanzu

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	clientauthenticationv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	kubeutils "github.com/vmware-tanzu/tanzu-cli/pkg/auth/utils/kubeconfig"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const (
	// tanzuLocalKubeDir is the local config directory
	tanzuLocalKubeDir = "kube"

	// tanzuKubeconfigFile is the name the of the kubeconfig file
	tanzuKubeconfigFile = "config"
)

// GetTanzuKubeconfig constructs and returns the kubeconfig that points to
// Tanzu Org. The constructed kubeconfig will incorporate any explicitly certdata or
// skip verify flag or rely on the same information captured in the certmap
// otherwise.
func GetTanzuKubeconfig(c *configtypes.Context, endpoint, orgID, endpointCACertPath string, skipTLSVerify bool) (string, string, string, error) {
	var endpointCACertBytes []byte
	var err error

	clusterAPIServerURL := strings.TrimSpace(endpoint)
	if !strings.HasPrefix(clusterAPIServerURL, "https://") && !strings.HasPrefix(clusterAPIServerURL, "http://") {
		clusterAPIServerURL = "https://" + clusterAPIServerURL
	}
	clusterAPIServerURL = clusterAPIServerURL + "/org/" + orgID

	if endpointCACertPath != "" {
		endpointCACertBytes, err = os.ReadFile(endpointCACertPath)
		if err != nil {
			return "", "", "", errors.Wrapf(err, "error reading CA certificate file %s", endpointCACertPath)
		}
	} else if !skipTLSVerify {
		// When there is no explicit provision to use custom cert or skip TLS
		// verification, fall back to information captured in the cert map, if
		// any

		if val, ok := c.AdditionalMetadata[config.TanzuAuthEndpointKey]; ok {
			endpoint := val.(string)
			certInfo, _ := config.GetCert(endpoint)
			if certInfo != nil {
				val, err := strconv.ParseBool(certInfo.SkipCertVerify)
				if err == nil {
					skipTLSVerify = val
				}
				decodedCACertData, err := base64.StdEncoding.DecodeString(certInfo.CACertData)
				if err == nil {
					endpointCACertBytes = decodedCACertData
					log.V(7).Infof("Using cert data for endpoint: %s", endpoint)
				}
			}
		}
	}

	contextName := kubeconfigContextName(c.Name)
	clusterName := kubeconfigClusterName(c.Name)
	username := kubeconfigUserName(c.Name)
	execConfig := getExecConfig(c)
	kcfg := &clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: clientcmdapi.SchemeGroupVersion.Version,
		Clusters: map[string]*clientcmdapi.Cluster{clusterName: {
			CertificateAuthorityData: endpointCACertBytes,
			InsecureSkipTLSVerify:    skipTLSVerify,
			Server:                   clusterAPIServerURL,
		}},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{username: {Exec: execConfig}},
		Contexts:       map[string]*clientcmdapi.Context{contextName: {Cluster: clusterName, AuthInfo: username}},
		CurrentContext: contextName,
	}

	kubeconfigBytes, err := json.Marshal(kcfg)
	if err != nil {
		return "", "", "", errors.Wrap(err, "failed to marshal the tanzu kubeconfig")
	}

	kubeconfigPath, err := tanzuLocalKubeConfigPath()
	if err != nil {
		return "", "", "", errors.Wrap(err, "unable to get the Tanzu local kubeconfig path")
	}
	err = kubeutils.MergeKubeConfigWithoutSwitchContext(kubeconfigBytes, kubeconfigPath)
	if err != nil {
		return "", "", "", errors.Wrap(err, "failed to merge the tanzu kubeconfig")
	}

	return kubeconfigPath, contextName, clusterAPIServerURL, nil
}

func kubeconfigContextName(tanzuContextName string) string {
	return "tanzu-cli-" + tanzuContextName
}

func kubeconfigClusterName(tanzuContextName string) string {
	return "tanzu-cli-" + tanzuContextName
}

func kubeconfigUserName(tanzuContextName string) string {
	return "tanzu-cli-" + tanzuContextName + "-user"
}

func getExecConfig(c *configtypes.Context) *clientcmdapi.ExecConfig {
	execConfig := &clientcmdapi.ExecConfig{
		APIVersion:      clientauthenticationv1.SchemeGroupVersion.String(),
		Args:            []string{},
		Env:             []clientcmdapi.ExecEnvVar{},
		InteractiveMode: clientcmdapi.IfAvailableExecInteractiveMode,
	}

	execConfig.Command = "tanzu"
	execConfig.Args = append([]string{"context", "get-token"}, c.Name)
	return execConfig
}

// tanzuLocalKubeConfigPath returns the local tanzu kubeconfig path
func tanzuLocalKubeConfigPath() (path string, err error) {
	localDir, err := config.LocalDir()
	if err != nil {
		return path, errors.Wrap(err, "could not locate local tanzu dir")
	}
	path = filepath.Join(localDir, tanzuLocalKubeDir)
	// create tanzu kubeconfig directory
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	configFilePath := filepath.Join(path, tanzuKubeconfigFile)

	return configFilePath, nil
}
