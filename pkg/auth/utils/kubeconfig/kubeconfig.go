// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package kubeconfig provides kubeconfig access functions.
package kubeconfig

import (
	"os"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
)

func GetDefaultKubeConfigFile() string {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	return rules.GetDefaultFilename()
}

// MergeKubeConfigWithoutSwitchContext merges kubeconfig without updating kubecontext
func MergeKubeConfigWithoutSwitchContext(kubeConfig []byte, mergeFile string) error {
	if mergeFile == "" {
		mergeFile = GetDefaultKubeConfigFile()
	}
	newConfig, err := clientcmd.Load(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "unable to load kubeconfig")
	}

	if _, err := os.Stat(mergeFile); os.IsNotExist(err) {
		return clientcmd.WriteToFile(*newConfig, mergeFile)
	}

	dest, err := clientcmd.LoadFromFile(mergeFile)
	if err != nil {
		return errors.Wrap(err, "unable to load kube config")
	}

	context := dest.CurrentContext
	err = mergo.Merge(dest, newConfig)

	if err != nil {
		return errors.Wrap(err, "failed to merge config")
	}
	dest.CurrentContext = context

	return clientcmd.WriteToFile(*dest, mergeFile)
}

// SetCurrentContext updates the current context of a kubeconfig file to one of
// the contexts in said file.
func SetCurrentContext(kubeConfigPath, contextName string) error {
	config, err := clientcmd.LoadFromFile(kubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "unable to load kubeconfig")
	}

	if contextName == "" {
		return errors.New("context is not provided")
	}

	for name := range config.Contexts {
		if name == contextName {
			config.CurrentContext = contextName
			return clientcmd.WriteToFile(*config, kubeConfigPath)
		}
	}

	return errors.Errorf("context %q does not exist", contextName)
}

// DeleteContextFromKubeConfig deletes the context,user and the cluster information from give kubeconfigPath
func DeleteContextFromKubeConfig(kubeconfigPath, context string) error {
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return errors.Wrap(err, "unable to load kubeconfig")
	}

	clusterName := ""
	// if the context is not present in the kubeconfigPath, nothing to do
	c, ok := config.Contexts[context]
	if !ok {
		return nil
	}
	clusterName = c.Cluster
	userName := c.AuthInfo

	delete(config.Contexts, context)
	delete(config.Clusters, clusterName)
	delete(config.AuthInfos, userName)

	if config.CurrentContext == context {
		config.CurrentContext = ""
	}
	err = clientcmd.WriteToFile(*config, kubeconfigPath)
	if err != nil {
		return errors.Wrapf(err, "failed to delete the kubeconfig context '%s' ", context)
	}

	return nil
}
