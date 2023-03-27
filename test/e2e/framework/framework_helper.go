// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package framework defines the integration and end-to-end test case for cli core
package framework

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/pkg/errors"

	configapi "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// SliceToSet converts the given slice to set type
func SliceToSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{})
	exists := struct{}{}
	for _, ele := range slice {
		set[ele] = exists
	}
	return set
}

// PluginListToSet converts the given PluginInfo slice to set type, key is combination  of plugin's name_target_version
func PluginListToSet(pluginsToInstall []*PluginInfo) map[string]struct{} {
	set := make(map[string]struct{})
	exists := struct{}{}
	for _, plugin := range pluginsToInstall {
		set[fmt.Sprintf(PluginKey, plugin.Name, plugin.Target, plugin.Version)] = exists
	}
	return set
}

// PluginListToMap converts the given PluginInfo slice to map type, key is combination  of plugin's name_target_version and value is PluginInfo
func PluginListToMap(pluginsList []*PluginInfo) map[string]*PluginInfo {
	m := make(map[string]*PluginInfo)
	for i := range pluginsList {
		m[fmt.Sprintf(PluginKey, (pluginsList)[i].Name, (pluginsList)[i].Target, (pluginsList)[i].Version)] = pluginsList[i]
	}
	return m
}

// PluginGroupToMap converts the given slice of PluginGroups to map (PluginGroup name is the key) and PluginGroup is the value
func PluginGroupToMap(pluginGroups []*PluginGroup) map[string]*PluginGroup {
	m := make(map[string]*PluginGroup)
	for i := range pluginGroups {
		m[(pluginGroups)[i].Group] = pluginGroups[i]
	}
	return m
}

// RandomString generates random string of given length
func RandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[int(n.Int64())]
	}
	return string(b)
}

// RandomNumber generates random string of given length
func RandomNumber(length int) string {
	charset := "1234567890"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[int(n.Int64())]
	}
	return string(b)
}

func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorf("error while getting user home directory, error:%s", err.Error())
	}
	return home
}

// ExecuteCmdAndBuildJSONOutput is generic function to execute given command and build JSON output and return
func ExecuteCmdAndBuildJSONOutput[T PluginInfo | PluginSearch | PluginGroup | PluginSourceInfo | configapi.ClientConfig | Server | ContextListInfo](cmdExe CmdOps, cmd string) ([]*T, error) {
	out, stdErr, err := cmdExe.Exec(cmd)

	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrAndStdErr, cmd, err.Error(), stdErr.String())
		return nil, err
	}
	jsonStr := out.String()
	var list []*T
	err = json.Unmarshal([]byte(jsonStr), &list)
	if err != nil {
		log.Errorf(FailedToConstructJSONNodeFromOutputAndErrInfo, jsonStr, err.Error())
		return nil, errors.Wrapf(err, FailedToConstructJSONNodeFromOutput, jsonStr)
	}
	return list, nil
}

// GetMapKeys takes map[K]any and returns the slice of all map keys
func GetMapKeys[K string, V *PluginInfo](m map[K][]V) []*K {
	keySet := make([]*K, 0)
	for key := range m {
		entry := key
		keySet = append(keySet, &entry)
	}
	return keySet
}
