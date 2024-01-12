// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package tanzu provides functionality related to authentication for the Tanzu control plane
package tanzu

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"

	"github.com/pkg/errors"
	clientauthenticationv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	kubeutils "github.com/vmware-tanzu/tanzu-cli/pkg/auth/utils/kubeconfig"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// GetTanzuKubeconfig constructs and returns the kubeconfig that points to Tanzu Org and
func GetTanzuKubeconfig(c *configtypes.Context, endpoint, orgID, endpointCACertPath string, skipTLSVerify bool) (string, string, string, error) {
	clusterAPIServerURL := strings.TrimSpace(endpoint)
	if !strings.HasPrefix(clusterAPIServerURL, "https://") && !strings.HasPrefix(clusterAPIServerURL, "http://") {
		clusterAPIServerURL = "https://" + clusterAPIServerURL
	}
	clusterAPIServerURL = clusterAPIServerURL + "/org/" + orgID

	clusterCACertData := ""
	if endpointCACertPath != "" {
		fileBytes, err := os.ReadFile(endpointCACertPath)
		if err != nil {
			return "", "", "", errors.Wrapf(err, "error reading CA certificate file %s", endpointCACertPath)
		}
		clusterCACertData = base64.StdEncoding.EncodeToString(fileBytes)
	}

	contextName := kubeconfigContextName(c.Name)
	clusterName := kubeconfigClusterName(c.Name)
	username := kubeconfigUserName(c.Name)
	execConfig := getExecConfig(c)
	config := &clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: clientcmdapi.SchemeGroupVersion.Version,
		Clusters: map[string]*clientcmdapi.Cluster{clusterName: {
			CertificateAuthorityData: []byte(clusterCACertData),
			InsecureSkipTLSVerify:    skipTLSVerify,
			Server:                   clusterAPIServerURL,
		}},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{username: {Exec: execConfig}},
		Contexts:       map[string]*clientcmdapi.Context{contextName: {Cluster: clusterName, AuthInfo: username}},
		CurrentContext: contextName,
	}

	kubeconfigByes, err := json.Marshal(config)
	if err != nil {
		return "", "", "", errors.Wrap(err, "failed to marshal the tanzu kubeconfig")
	}
	kubeconfigPath := kubeutils.GetDefaultKubeConfigFile()
	err = kubeutils.MergeKubeConfigWithoutSwitchContext(kubeconfigByes, kubeconfigPath)
	if err != nil {
		return "", "", "", errors.Wrap(err, "failed to merge the tanzu kubeconfig")
	}

	return kubeconfigPath, contextName, clusterAPIServerURL, nil
}

func kubeconfigContextName(tanzuContextName string) string {
	return "tanzu-cli-" + tanzuContextName
}

func kubeconfigClusterName(tanzuContextName string) string {
	return "tanzu-cli-" + tanzuContextName + "/current"
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
