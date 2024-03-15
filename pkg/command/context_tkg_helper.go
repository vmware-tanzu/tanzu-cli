// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	tkgauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/tkg"
	wcpauth "github.com/vmware-tanzu/tanzu-cli/pkg/auth/wcp"
	"github.com/vmware-tanzu/tanzu-cli/pkg/pluginsupplier"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const (
	pinnipedAuthPluginName = "pinniped-auth"
)

type TKGKubeconfigFetcherOptions struct {
	Endpoint              string
	InsecureSkipTLSVerify bool
	EndpointCACertPath    string
	CmdExecutor           func(command *cobra.Command, args []string) (stdOut []byte, stderr []byte, err error)
}

type TKGKubeconfigFetcher interface {
	GetPinnipedKubeconfig() (mergeFilePath, currentContext string, err error)
}

func NewTKGKubeconfigFetcher(endpoint, endpointCACertPath string, insecureSkipTLSVerify bool) TKGKubeconfigFetcher {
	return &TKGKubeconfigFetcherOptions{
		Endpoint:              endpoint,
		EndpointCACertPath:    endpointCACertPath,
		InsecureSkipTLSVerify: insecureSkipTLSVerify,
		CmdExecutor:           RunCommandAndGetStdOutAndErr,
	}
}

func (tkfo *TKGKubeconfigFetcherOptions) GetPinnipedKubeconfig() (string, string, error) {
	exists, err := tkfo.isPinnipedPluginInstalled()
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to determine if the 'pinniped-auth' plugin is installed")
	}
	if !exists {
		return "", "", errors.New("the 'pinniped-auth' plugin is not installed. This plugin is required to authenticate with TKG/vSphere with Kubernetes(TKGs), please install the plugin and retry")
	}

	// get kubeconfig using the 'pinniped-auth' plugin. If the pinniped-auth plugin doesn't support `kubeconfig get` command,
	// use the existing logic to fetch the kubeconfig for prior versions of TKG/TKGs
	kubeconfigBytes, err := tkfo.getKubeconfigUsingPinnipedAuthPlugin()
	if err != nil {
		if strings.Contains(err.Error(), `unknown command "kubeconfig" for "pinniped-auth"`) {
			return tkfo.getKubeconfigForPriorTKGVersion()
		}
		return "", "", err
	}

	return tkgauth.MergeAndSaveKubeconfigBytes(kubeconfigBytes, nil)
}

func (tkfo *TKGKubeconfigFetcherOptions) getKubeconfigUsingPinnipedAuthPlugin() ([]byte, error) {
	rootCmd, err := NewRootCmd()
	if err != nil {
		return nil, err
	}

	args := []string{pinnipedAuthPluginName, "kubeconfig", "get", "--endpoint", tkfo.Endpoint}

	if tkfo.EndpointCACertPath != "" {
		fileBytes, err := os.ReadFile(tkfo.EndpointCACertPath)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading CA certificate file %s", tkfo.EndpointCACertPath)
		}
		args = append(args, "--endpoint-ca-bundle-b64", base64.StdEncoding.EncodeToString(fileBytes))
	}

	if tkfo.InsecureSkipTLSVerify {
		args = append(args, "--insecure-skip-tls-verify")
	}
	// TODO: Instead of executing the cobra command using root, evaluate the merits using exec.Command() by passing the
	// installed plugin binary path(cli.PluginInfo.InstallationPath)
	sOut, sErr, err := tkfo.CmdExecutor(rootCmd, args)
	if err != nil {
		return sErr, errors.Wrap(err, string(sErr))
	}
	return sOut, nil
}

func (tkfo *TKGKubeconfigFetcherOptions) isPinnipedPluginInstalled() (bool, error) {
	standalonePlugins, err := pluginsupplier.GetInstalledPlugins()
	if err != nil {
		return false, err
	}

	for i := range standalonePlugins {
		if standalonePlugins[i].Name == pinnipedAuthPluginName {
			return true, nil
		}
	}
	return false, nil
}

func (tkfo *TKGKubeconfigFetcherOptions) getKubeconfigForPriorTKGVersion() (string, string, error) {
	var kubeCfg, kubeCtx string
	// While this would add an extra HTTP round trip, it avoids the need to
	// add extra provider specific login flags.
	tlsConfig, err := tkgauth.GetTLSConfig(tkfo.EndpointCACertPath, tkfo.InsecureSkipTLSVerify)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get TLS config")
	}

	isVSphereSupervisor, err := wcpauth.IsVSphereSupervisor(endpoint, getDiscoveryHTTPClient(tlsConfig))
	// Fall back to assuming non vSphere supervisor.
	if err != nil {
		err := fmt.Errorf("error creating kubeconfig with tanzu pinniped-auth login plugin: %v", err)
		log.Error(err, "")
		return "", "", err
	}

	if isVSphereSupervisor {
		log.Info("Detected a vSphere Supervisor being used")
		kubeCfg, kubeCtx, err = vSphereSupervisorLogin(endpoint)
		if err != nil {
			err := fmt.Errorf("error login into the vSphere Supervisor: %v", err)
			log.Error(err, "")
			return "", "", err
		}
	} else {
		kubeCfg, kubeCtx, err = tkgauth.KubeconfigWithPinnipedAuthLoginPlugin(endpoint, nil,
			tkgauth.DiscoveryStrategy{ClusterInfoConfigMap: tkgauth.DefaultClusterInfoConfigMap}, tkfo.EndpointCACertPath, tkfo.InsecureSkipTLSVerify)
		if err != nil {
			err := fmt.Errorf("error creating kubeconfig with tanzu pinniped-auth login plugin: %v", err)
			log.Error(err, "")
			return "", "", err
		}
	}
	return kubeCfg, kubeCtx, nil
}

// RunCommandAndGetStdOutAndErr executes the cobra command and returns the command's stdout,stderr and error
func RunCommandAndGetStdOutAndErr(cmd *cobra.Command, args []string) ([]byte, []byte, error) {
	// Create a pipe to read stdout
	ro, wo, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	c := make(chan []byte)
	go fetchOutput(ro, c)

	// Create a pipe to read stderr
	re, we, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	e := make(chan []byte)
	go fetchOutput(re, e)

	stdout := os.Stdout
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()
	os.Stdout = wo
	os.Stderr = we
	cmd.SetArgs(args)
	err = cmd.Execute()
	cmd.ErrOrStderr()
	wo.Close()
	we.Close()
	sout := <-c
	serr := <-e
	if err != nil {
		return sout, serr, err
	}
	return sout, serr, nil
}

func fetchOutput(r io.Reader, c chan<- []byte) {
	data, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err, "failed reading the pinniped-auth output")
	}
	c <- data
}
