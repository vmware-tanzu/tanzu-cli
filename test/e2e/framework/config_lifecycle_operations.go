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
	ConfigSetFeatureFlag(path, value string) error
	// ConfigGetFeatureFlag gets the tanzu config feature flag
	ConfigGetFeatureFlag(path string) (string, error)
	// ConfigUnsetFeature un-sets the tanzu config feature flag
	ConfigUnsetFeature(path string) error
	// ConfigInit performs "tanzu config init"
	ConfigInit() error
	// GetConfig gets the tanzu config
	GetConfig() (*configapi.ClientConfig, error)
	// ConfigServerList returns the server list
	ConfigServerList() ([]*Server, error)
	// ConfigServerDelete deletes given server from tanzu config
	ConfigServerDelete(serverName string) error
	// DeleteCLIConfigurationFiles deletes cli configuration files
	DeleteCLIConfigurationFiles() error
	// IsCLIConfigurationFilesExists checks the existence of cli configuration files
	IsCLIConfigurationFilesExists() bool
}

// configOps is the implementation of ConfOps interface
type configOps struct {
	cmdExe CmdOps
}

func NewConfOps() ConfigLifecycleOps {
	return &configOps{
		cmdExe: NewCmdOps(),
	}
}

// GetConfig gets the tanzu config
func (co *configOps) GetConfig() (*configapi.ClientConfig, error) {
	out, _, err := co.cmdExe.Exec(ConfigGet)
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
func (co *configOps) ConfigSetFeatureFlag(path, value string) (err error) {
	confSetCmd := ConfigSet + path + " " + value
	_, _, err = co.cmdExe.Exec(confSetCmd)
	return err
}

// ConfigGetFeatureFlag gets the given tanzu config feature flag
func (co *configOps) ConfigGetFeatureFlag(path string) (string, error) {
	cnf, err := co.GetConfig()
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
func (co *configOps) ConfigUnsetFeature(path string) (err error) {
	unsetFeatureCmd := ConfigUnset + path
	_, _, err = co.cmdExe.Exec(unsetFeatureCmd)
	return
}

// ConfigInit performs "tanzu config init"
func (co *configOps) ConfigInit() (err error) {
	_, _, err = co.cmdExe.Exec(ConfigInit)
	return
}

// ConfigServerList returns the server list
func (co *configOps) ConfigServerList() ([]*Server, error) {
	ConfigServerListWithJSONOutput := ConfigServerList + JSONOutput
	return ExecuteCmdAndBuildJSONOutput[Server](co.cmdExe, ConfigServerListWithJSONOutput)
}

// ConfigServerDelete deletes a server from tanzu config
func (co *configOps) ConfigServerDelete(serverName string) error {
	configDelCmd := fmt.Sprintf(ConfigServerDelete, serverName)
	_, _, err := co.cmdExe.Exec(configDelCmd)
	if err != nil {
		log.Error(err, "error while running: "+configDelCmd)
	}
	return err
}

// DeleteCLIConfigurationFiles deletes cli configuration files
func (co *configOps) DeleteCLIConfigurationFiles() error {
	homeDir, _ := os.UserHomeDir()
	configFile := filepath.Join(homeDir, ConfigFileDir, ConfigFileName)
	_, err := os.Stat(configFile)
	if err == nil {
		if ferr := os.Remove(configFile); ferr != nil {
			return ferr
		}
	}
	configNGFile := filepath.Join(homeDir, ConfigFileDir, ConfigNGFileName)
	if _, err := os.Stat(configNGFile); err == nil {
		if ferr := os.Remove(configNGFile); ferr != nil {
			return ferr
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
