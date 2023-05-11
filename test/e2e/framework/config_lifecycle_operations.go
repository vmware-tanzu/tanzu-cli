// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	configapi "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

const (
	ConfigFileName   = "config.yaml"
	ConfigNGFileName = "config-ng.yaml"
	ConfigFileDir    = ".config/tanzu/"
)

// ConfigLifecycleOps performs "tanzu config" command operations
type ConfigLifecycleOps interface {
	// ConfigSetFeatureFlag sets the tanzu config feature flag
	ConfigSetFeatureFlag(path, value string, opts ...E2EOption) error
	// ConfigGetFeatureFlag gets the tanzu config feature flag
	ConfigGetFeatureFlag(path string, opts ...E2EOption) (string, error)
	// ConfigUnsetFeature un-sets the tanzu config feature flag
	ConfigUnsetFeature(path string, opts ...E2EOption) error
	// ConfigInit performs "tanzu config init"
	ConfigInit(opts ...E2EOption) error
	// GetConfig gets the tanzu config
	GetConfig(opts ...E2EOption) (*configapi.ClientConfig, error)
	// ConfigServerList returns the server list
	ConfigServerList(opts ...E2EOption) ([]*Server, error)
	// ConfigServerDelete deletes given server from tanzu config
	ConfigServerDelete(serverName string, opts ...E2EOption) error
	// DeleteCLIConfigurationFiles deletes cli configuration files
	DeleteCLIConfigurationFiles() error
	// IsCLIConfigurationFilesExists checks the existence of cli configuration files
	IsCLIConfigurationFilesExists() bool
}

// ConfigCertOps performs "tanzu config cert" command operations
type ConfigCertOps interface {
	// ConfigCertAdd adds cert config for a host, and returns stdOut and error info
	ConfigCertAdd(certAddOpts *CertAddOptions, opts ...E2EOption) (string, error)

	// ConfigCertDelete deletes cert config for a host, and returns error info
	ConfigCertDelete(host string, opts ...E2EOption) error

	// ConfigCertList list cert
	ConfigCertList(opts ...E2EOption) ([]*CertDetails, error)
}

type ConfigCmdOps interface {
	ConfigLifecycleOps
	ConfigCertOps
}

// configOps is the implementation of ConfOps interface
type configOps struct {
	cmdExe CmdOps
}

func NewConfOps() ConfigCmdOps {
	return &configOps{
		cmdExe: NewCmdOps(),
	}
}

type CertAddOptions struct {
	Host              string
	CACertificatePath string
	SkipCertVerify    string
	Insecure          string
}

// GetConfig gets the tanzu config
func (co *configOps) GetConfig(opts ...E2EOption) (*configapi.ClientConfig, error) {
	out, _, err := co.cmdExe.TanzuCmdExec(ConfigGet, opts...)
	var cnf *configapi.ClientConfig
	if err != nil {
		return cnf, err
	}
	err = yaml.Unmarshal(out.Bytes(), &cnf)
	if err != nil {
		return cnf, errors.Wrap(err, "failed to construct yaml node from config get output")
	}
	return cnf, nil
}

// ConfigSetFeatureFlag sets the given tanzu config feature flag
func (co *configOps) ConfigSetFeatureFlag(path, value string, opts ...E2EOption) (err error) {
	confSetCmd := ConfigSet + path + " " + value
	_, _, err = co.cmdExe.TanzuCmdExec(confSetCmd, opts...)
	return err
}

// ConfigGetFeatureFlag gets the given tanzu config feature flag
func (co *configOps) ConfigGetFeatureFlag(path string, opts ...E2EOption) (string, error) {
	cnf, err := co.GetConfig(opts...)
	if err != nil {
		return "", err
	}
	featureName := strings.Split(path, ".")[len(strings.Split(path, "."))-1]
	pluginName := strings.Split(path, ".")[len(strings.Split(path, "."))-2]
	if cnf != nil && cnf.ClientOptions.Features[pluginName] != nil {
		return cnf.ClientOptions.Features[pluginName][featureName], nil
	}
	return "", err
}

// ConfigUnsetFeature un-sets the tanzu config feature flag
func (co *configOps) ConfigUnsetFeature(path string, opts ...E2EOption) (err error) {
	unsetFeatureCmd := ConfigUnset + path
	_, _, err = co.cmdExe.TanzuCmdExec(unsetFeatureCmd, opts...)
	return
}

// ConfigInit performs "tanzu config init"
func (co *configOps) ConfigInit(opts ...E2EOption) (err error) {
	_, _, err = co.cmdExe.TanzuCmdExec(ConfigInit, opts...)
	return
}

// ConfigServerList returns the server list
func (co *configOps) ConfigServerList(opts ...E2EOption) ([]*Server, error) {
	ConfigServerListWithJSONOutput := ConfigServerList + JSONOutput
	list, _, _, err := ExecuteCmdAndBuildJSONOutput[Server](co.cmdExe, ConfigServerListWithJSONOutput, opts...)
	return list, err
}

// ConfigServerDelete deletes a server from tanzu config
func (co *configOps) ConfigServerDelete(serverName string, opts ...E2EOption) error {
	configDelCmd := fmt.Sprintf(ConfigServerDelete, "%s", serverName)
	_, _, err := co.cmdExe.TanzuCmdExec(configDelCmd, opts...)
	if err != nil {
		log.Infof("failed to delete config server: %s", serverName)
		log.Error(err, "error while running: "+configDelCmd)
	} else {
		log.Infof(ConfigServerDeleted, serverName)
	}
	return err
}

// DeleteCLIConfigurationFiles deletes cli configuration files
func (co *configOps) DeleteCLIConfigurationFiles() error {
	homeDir, _ := os.UserHomeDir()
	configFile := filepath.Join(homeDir, ConfigFileDir, ConfigFileName)
	_, err := os.Stat(configFile)
	if err == nil {
		if fileErr := os.Remove(configFile); fileErr != nil {
			return fileErr
		}
	}
	configNGFile := filepath.Join(homeDir, ConfigFileDir, ConfigNGFileName)
	if _, err := os.Stat(configNGFile); err == nil {
		if fileErr := os.Remove(configNGFile); fileErr != nil {
			return fileErr
		}
	}
	return nil
}

// IsCLIConfigurationFilesExists checks the existence of cli configuration files
func (co *configOps) IsCLIConfigurationFilesExists() bool {
	homeDir, _ := os.UserHomeDir()
	configFilePath := filepath.Join(homeDir, ConfigFileDir, ConfigFileName)
	configNGFilePath := filepath.Join(homeDir, ConfigFileDir, ConfigNGFileName)
	_, err1 := os.Stat(configFilePath)
	_, err2 := os.Stat(configNGFilePath)
	if err1 == nil && err2 == nil {
		return true
	}
	return false
}

func (co *configOps) ConfigCertAdd(certAddOpts *CertAddOptions, opts ...E2EOption) (string, error) {
	certAddCmd := fmt.Sprintf(ConfigCertAdd, "%s", certAddOpts.Host, certAddOpts.CACertificatePath, certAddOpts.SkipCertVerify, certAddOpts.Insecure)
	out, _, err := co.cmdExe.TanzuCmdExec(certAddCmd, opts...)
	return out.String(), err
}

func (co *configOps) ConfigCertDelete(host string, opts ...E2EOption) error {
	certDeleteCmd := fmt.Sprintf(ConfigCertDelete, "%s", host)
	_, _, err := co.cmdExe.TanzuCmdExec(certDeleteCmd, opts...)
	return err
}

func (co *configOps) ConfigCertList(opts ...E2EOption) ([]*CertDetails, error) {
	list, _, _, err := ExecuteCmdAndBuildJSONOutput[CertDetails](co.cmdExe, ConfigCertList, opts...)
	return list, err
}
