// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aunum/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	configapi "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

const (
	ConfigFileName   = "config.yaml"
	ConfigNGFileName = "config-ng.yaml"
	ConfigFileDir    = ".config/tanzu/"
)

// ConfigLifecycleOps performs "tanzu config" command operations
type ConfigLifecycleOps interface {
	ConfigSetFeatureFlag(path, value string) error
	ConfigGetFeatureFlag(path string) (string, error)
	ConfigUnsetFeature(path string) error
	ConfigInit() error
	GetConfig() (*configapi.ClientConfig, error)
	ConfigServerList() ([]Server, error)
	ConfigServerDelete(serverName string) error
	DeleteCLIConfigurationFiles() error
	IsCLIConfigurationFilesExists() bool
}

// configOps is the implementation of ConfOps interface
type configOps struct {
	CmdOps
}

func NewConfOps() ConfigLifecycleOps {
	return &configOps{
		CmdOps: NewCmdOps(),
	}
}

// GetConfig gets the tanzu config
func (co *configOps) GetConfig() (*configapi.ClientConfig, error) {
	out, _, err := co.Exec(ConfigGet)
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

// ConfigSetFeature sets the tanzu config feature flag
func (co *configOps) ConfigSetFeatureFlag(path, value string) (err error) {
	confSetCmd := ConfigSet + path + " " + value
	_, _, err = co.Exec(confSetCmd)
	return err
}

// ConfigSetFeature sets the tanzu config feature flag
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
	_, _, err = co.Exec(unsetFeatureCmd)
	return
}

// ConfigInit performs "tanzu config init"
func (co *configOps) ConfigInit() (err error) {
	_, _, err = co.Exec(ConfigInit)
	return
}

// ConfigServerList returns the server list
func (co *configOps) ConfigServerList() ([]Server, error) {
	stdOut, _, err := co.Exec(ConfigServerList)
	if err != nil {
		log.Errorf("error while executing `config server list`, error:%s", err.Error())
		return nil, err
	}
	jsonStr := stdOut.String()
	var list []Server
	err = json.Unmarshal([]byte(jsonStr), &list)
	if err != nil {
		log.Errorf("failed to construct node from config server list output:'%s' error:'%s' ", jsonStr, err.Error())
		return nil, errors.Wrapf(err, "failed to construct node from config server list output:'%s'", jsonStr)
	}
	return list, nil
}

// ConfigServerDelete deletes a server from tanzu config
func (co *configOps) ConfigServerDelete(serverName string) error {
	_, _, err := co.Exec(fmt.Sprintf(ConfigServerDelete, serverName))
	if err != nil {
		log.Error(err)
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
