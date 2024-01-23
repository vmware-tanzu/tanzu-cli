// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package framework defines the integration and end-to-end test case for cli core
package framework

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// WithTanzuBinary is to set the tanzu binary location; default is tanzu from PATH variable
func WithTanzuBinary(tanzuBinary string) E2EOption {
	return func(opts *E2EOptions) {
		opts.TanzuBinary = tanzuBinary
	}
}

// WithFilePath is the installation file path location
func WithFilePath(filePath string) E2EOption {
	return func(opts *E2EOptions) {
		opts.FilePath = filePath
	}
}

// WithOverride is to provide whether new Tanzu CLI overrides the installation of legacy Tanzu CLI
func WithOverride(override bool) E2EOption {
	return func(opts *E2EOptions) {
		opts.Override = override
	}
}

// AddAdditionalFlagAndValue is to add any additional flag with value (if any) to the end of tanzu command
func AddAdditionalFlagAndValue(flagWithValue string) E2EOption {
	return func(opts *E2EOptions) {
		opts.AdditionalFlags = opts.AdditionalFlags + " " + flagWithValue
	}
}

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
		m[GetMapKeyForPlugin((pluginsList)[i])] = pluginsList[i]
	}
	return m
}

// GetMapKeyForPlugin takes the plugin and returns the map key for the plugin
func GetMapKeyForPlugin(pluginsList *PluginInfo) string {
	return fmt.Sprintf(PluginKey, pluginsList.Name, pluginsList.Target, pluginsList.Version)
}

// LegacyPluginListToMap converts the given PluginInfo slice to map type, key is combination  of plugin's name_target_version and value is PluginInfo
func LegacyPluginListToMap(pluginsList []*PluginInfo) map[string]*PluginInfo {
	m := make(map[string]*PluginInfo)
	for i := range pluginsList {
		m[fmt.Sprintf(LegacyPluginKey, (pluginsList)[i].Name, (pluginsList)[i].Version)] = pluginsList[i]
	}
	return m
}

// PluginGroupToMap converts the given slice of PluginGroups to map (PluginGroup name:version is the key) and PluginGroup is the value
func PluginGroupToMap(pluginGroups []*PluginGroup) map[string]*PluginGroup {
	m := make(map[string]*PluginGroup)
	for i := range pluginGroups {
		for j := range pluginGroups[i].Versions {
			m[pluginGroups[i].Group+":"+pluginGroups[i].Versions[j]] = pluginGroups[i]
		}
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

func GetE2EHomeDir() string {
	if TestHomeDir == "" {
		currentHomeDir := GetHomeDir()
		if strings.HasSuffix(currentHomeDir, TestDir) {
			return currentHomeDir
		}
		TestHomeDir = filepath.Join(TestHomeDir, TestDir)
		return TestHomeDir
	}
	return TestHomeDir
}

func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorf("error while getting user home directory, error:%s", err.Error())
	}
	return home
}

// ExecuteCmdAndBuildJSONOutput is generic function to execute given command and build JSON output and return
// the result, stdOut, stdErr and error
func ExecuteCmdAndBuildJSONOutput[T PluginInfo | PluginSearch | PluginGroup | PluginGroupGet | PluginSourceInfo | types.ClientConfig | Server | ContextListInfo | CertDetails | PluginDescribe](cmdExe CmdOps, cmd string, opts ...E2EOption) ([]*T, string, string, error) {
	out, stdErr, err := cmdExe.TanzuCmdExec(cmd, opts...)
	outStr := ""
	stdErrStr := ""
	if out != nil {
		outStr = out.String()
	}
	if stdErr != nil {
		stdErrStr = stdErr.String()
	}

	var list []*T
	if outStr != "" {
		unmarshalErr := json.Unmarshal([]byte(outStr), &list)
		if unmarshalErr != nil {
			log.Errorf(FailedToConstructJSONNodeFromOutputAndErrInfo, outStr, unmarshalErr.Error())
			log.Errorf("trying with yaml unmarshal")
			// try with yaml format unmarshal
			err2 := yaml.Unmarshal([]byte(outStr), &list)
			if err2 != nil {
				return nil, outStr, stdErrStr, errors.Wrapf(unmarshalErr, FailedToConstructJSONNodeFromOutput, outStr)
			}
		}
	}
	if err != nil {
		log.Errorf(ErrorLogForCommandWithErrStdErrAndStdOut, cmd, err.Error(), stdErr.String(), out.String())
	}
	return list, outStr, stdErrStr, err
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

// CreateKindCluster create the k8s KIND cluster in the local Docker environment
func CreateKindCluster(tf *Framework, name string) (*ClusterInfo, error) {
	ci := &ClusterInfo{Name: name}
	stdOut, stdErr, err := tf.KindCluster.CreateCluster(name)
	if err != nil {
		log.Errorf("kind cluster creation failed, stdOut:%s, stdErr:%s", stdOut, stdErr)
		return nil, errors.Wrapf(err, "error while creating kind cluster: %s", name)
	}
	endpoint, stdOut, stdErr, err := tf.KindCluster.GetClusterEndpoint(name)
	if err != nil {
		log.Errorf("error while getting kind cluster status stdOut:%s, stdErr:%s", stdOut, stdErr)
		return nil, errors.Wrapf(err, "error while getting kind cluster %s endpoint", name)
	}
	ci.EndPoint = endpoint
	ci.ClusterKubeContext = tf.KindCluster.GetClusterContext(name)
	ci.KubeConfigPath = tf.KindCluster.GetKubeconfigPath()
	return ci, nil
}

// IsContextExists checks the given context is exists in the config file by listing the existing contexts in the config file
func IsContextExists(tf *Framework, contextName string, opts ...E2EOption) bool {
	list, _, _, err := tf.ContextCmd.ListContext(opts...)
	gomega.Expect(err).To(gomega.BeNil(), "list context should not return any error")
	for _, context := range list {
		if context.Name == contextName {
			return true
		}
	}
	return false
}

// IsAllPluginGroupsExists takes the two list of PluginGroups (super list and sub list), check if all sub list PluginGroup are exists in super list PluginGroup
func IsAllPluginGroupsExists(superList, subList []*PluginGroup) bool {
	superMap := PluginGroupToMap(superList)
	subMap := PluginGroupToMap(subList)
	for ele := range subMap {
		_, exists := superMap[ele]
		if !exists {
			return false
		}
	}
	return true
}

// CreateTemporaryCRsFromPluginInfos takes list of Plugins info and generates temporary CR files(under $FullPathForTempDir), and returns plugins list, CR files and error if any while creating the CR files
func CreateTemporaryCRsFromPluginInfos(plugins []*PluginInfo) ([]*PluginInfo, []string, error) {
	pluginsList := make([]*PluginInfo, 0)
	filePaths := make([]string, 0)
	for _, plugin := range plugins {
		absoluteCRFilePath := filepath.Join(FullPathForTempDir, fmt.Sprintf(PluginCRFileName, plugin.Name, plugin.Target, plugin.Version))
		err := os.WriteFile(absoluteCRFilePath, []byte(fmt.Sprintf(CRTemplate, plugin.Name, plugin.Version)), 0644)
		if err != nil {
			return pluginsList, filePaths, err
		}
		filePaths = append(filePaths, absoluteCRFilePath)
		pluginsList = append(pluginsList, plugin)
	}
	return pluginsList, filePaths, nil
}

// GetPluginFromFirstListButNotExistsInSecondList returns a plugin which is exists in first plugin list but not in second plugin list
func GetPluginFromFirstListButNotExistsInSecondList(first, second []*PluginInfo) (*PluginInfo, error) {
	m1 := PluginListToMap(first)
	m2 := PluginListToMap(second)
	for plugin := range m1 {
		if _, ok := m2[plugin]; !ok {
			return m1[plugin], nil
		}
	}
	return nil, fmt.Errorf("there is no plugin which is not common in the given pluginInfo's")
}

// IsPluginSourceExists checks the sourceName is exists in the given list of PluginSourceInfo's
func IsPluginSourceExists(list []*PluginSourceInfo, sourceName string) bool {
	for _, val := range list {
		if val.Name == sourceName {
			return true
		}
	}
	return false
}

// CheckAllPluginsExists checks all PluginInfo's in subList are available in superList
// superList is the super set, subList is sub set
func CheckAllPluginsExists(superList, subList []*PluginInfo) bool {
	superSet := PluginListToMap(superList)
	subSet := PluginListToMap(subList)
	for key := range subSet {
		// val2, ok := superSet[key]
		// Plugin's Name, Target and Version are part of map Key, so no need to compare/validate again here if different then we can not find the plugin in superSet map
		// TODO: cpamuluri: currently the plugin's description in 'tanzu plugin search' output and 'tanzu plugin list' (after install) are different, ignore comparing description field for now, until we fix the description fields in local test central repository
		// if !ok || val1.Description != val2.Description {
		// 	return false
		// }
		_, ok := superSet[key]
		if !ok {
			return false
		}
	}
	return true
}

// GetInstalledPlugins takes list of plugins and returns installed only list of plugins
func GetInstalledPlugins(pluginList []*PluginInfo) []*PluginInfo {
	installedPlugin := make([]*PluginInfo, 0)
	for i := range pluginList {
		if pluginList[i].Status == Installed {
			installedPlugin = append(installedPlugin, pluginList[i])
		}
	}
	return installedPlugin
}

// IsPluginExists validates the given plugin (with plugin status) is exists in the plugins list or not
func IsPluginExists(pluginList []*PluginInfo, plugin *PluginInfo, pluginInstallationStatus string) bool {
	isExist := CheckAllPluginsExists(pluginList, append(make([]*PluginInfo, 0), plugin))
	if isExist {
		return plugin.Status == pluginInstallationStatus
	}
	return isExist
}

// GetGivenPluginFromTheGivenPluginList takes the plugin list and a plugin
// checks the given plugin exists in the plugin list, if exists then returns the plugin
// otherwise returns nil
func GetGivenPluginFromTheGivenPluginList(pluginList []*PluginInfo, requiredPlugin *PluginInfo) *PluginInfo {
	superSet := PluginListToMap(pluginList)
	return superSet[GetMapKeyForPlugin(requiredPlugin)]
}

// LogConfigFiles logs info level, contents of files config.yaml and config-ng.yaml
func LogConfigFiles() error {
	err := LogFile(ConfigFilePath)
	if err != nil {
		return err
	}
	err = LogFile(ConfigNGFilePath)
	if err != nil {
		return err
	}
	return nil
}

// LogFile logs in info level, the given file content
func LogFile(file string) error {
	dat, err := os.ReadFile(file)
	if err != nil {
		log.Infof("error while reading file: %s error:%s", file, err.Error())
		return err
	}
	log.Infof(FileContent, ConfigFilePath, string(dat))
	return err
}

// GetPluginsList returns a list of plugins, either installed or both installed and uninstalled, based on the value of the installedOnly parameter.
func GetPluginsList(tf *Framework, installedOnly bool) ([]*PluginInfo, error) {
	out := make([]*PluginInfo, 0)
	pluginListOutput, _, _, err := tf.PluginCmd.ListPlugins()
	if err != nil {
		return out, nil
	}
	for _, pluginInfo := range pluginListOutput {
		if pluginInfo.Status == Installed {
			out = append(out, pluginInfo)
		}
	}
	return out, nil
}

// GetPluginGroupWhichStartsWithGivenPrefix takes plugin groups list and prefix string
// returns first plugin group which starts with the given prefix
func GetPluginGroupWhichStartsWithGivenPrefix(pgs []*PluginGroup, prefix string) string {
	for _, pg := range pgs {
		groupID := pg.Group + ":" + pg.Latest
		if strings.Contains(groupID, prefix) {
			return groupID
		}
	}
	return ""
}

// StartMockServer starts the http mock server (rodolpheche/wiremock)
func StartMockServer(tf *Framework, mappingDir, containerName string) error {
	err := StopContainer(tf, containerName)
	if err != nil && !strings.Contains(err.Error(), "No such container") {
		return err
	}
	startServerCmd := fmt.Sprintf(WiredMockHTTPServerStartCmd, containerName, mappingDir)
	_, _, err = tf.Exec.Exec(startServerCmd)
	// tests are failing randomly if not wait for some time after HTTP mock server started
	time.Sleep(2 * time.Second)
	return err
}

// StopContainer stops the given docker container
func StopContainer(tf *Framework, containerName string) error {
	cmd := fmt.Sprintf(HTTPMockServerStopCmd, containerName)
	_, _, err := tf.Exec.Exec(cmd)
	return err
}

// ConvertPluginsInfoToTMCEndpointMockResponse takes the plugins info and converts to TMC endpoint response to mock http calls
func ConvertPluginsInfoToTMCEndpointMockResponse(plugins []*PluginInfo) (*TMCPluginsMockRequestResponseMapping, error) {
	tmcPlugins := &TMCPluginsResponse{}
	tmcPlugins.PluginsInfo = TMCPluginsInfo{}
	tmcPlugins.PluginsInfo.Plugins = make([]TMCPlugin, 0)
	for i := range plugins {
		tmcPlugin := TMCPlugin{}
		tmcPlugin.Name = plugins[i].Name
		tmcPlugin.Description = plugins[i].Description
		tmcPlugin.RecommendedVersion = plugins[i].Version
		tmcPlugins.PluginsInfo.Plugins = append(tmcPlugins.PluginsInfo.Plugins, tmcPlugin)
	}
	m := &TMCPluginsMockRequestResponseMapping{}
	m.Request.Method = "GET"
	m.Request.URL = TMCEndpointForPlugins
	m.Response.Status = 200
	m.Response.Headers.ContentType = HTTPContentType
	m.Response.Headers.Accept = HTTPContentType
	content, err := json.Marshal(tmcPlugins.PluginsInfo)
	if err != nil {
		log.Error(err, "error while processing input type to json")
		return m, err
	}
	m.Response.Body = string(content)
	return m, nil
}

// WriteToFileInJSONFormat creates (if not exists) and writes the given input type to given file in json format
func WriteToFileInJSONFormat(input any, filePath string) error {
	content, err := json.Marshal(input)
	if err != nil {
		log.Error(err, "error while processing input type to json")
		return err
	}
	err = CreateOrTruncateFile(filePath)
	if err != nil {
		log.Error(err, fmt.Sprintf("error while creating truncating file %s", filePath))
		return err
	}
	f, err := os.OpenFile(filePath, os.O_WRONLY, 0644)
	if err != nil {
		log.Error(err, fmt.Sprintf("error while opening file %s", filePath))
		return err
	}
	defer f.Close()
	_, err = f.Write(content)
	if err != nil {
		log.Error(err, fmt.Sprintf("error while writing to file %s", filePath))
		return err
	}
	return nil
}

// CreateOrTruncateFile creates a given file if not exists
func CreateOrTruncateFile(filePath string) error {
	// check if file exists
	var _, err = os.Stat(filePath)
	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close()
	} else {
		return os.Truncate(filePath, 0)
	}
	return nil
}

// CreateDir creates given directory if not exists
func CreateDir(dir string) error {
	err := os.MkdirAll(dir, 0750)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err, fmt.Sprintf("error while creating directory: %s", dir))
		return err
	}
	return nil
}

// UpdatePluginDiscoverySource updates the plugin discovery source with given url
func UpdatePluginDiscoverySource(tf *Framework, repoURL string) error {
	// setup the test central repo
	_, err := tf.PluginCmd.UpdatePluginDiscoverySource(&DiscoveryOptions{Name: "default", SourceType: SourceType, URI: repoURL})
	return err
}

// ApplyConfigOnKindCluster applies the config files on kind cluster
func ApplyConfigOnKindCluster(tf *Framework, clusterInfo *ClusterInfo, confFilePaths []string) error {
	for _, pluginCRFilePaths := range confFilePaths {
		_, _, err := tf.KindCluster.ApplyConfig(clusterInfo.ClusterKubeContext, pluginCRFilePaths)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetHTTPCall queries http GET call on given url
func GetHTTPCall(url string, v interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", HTTPContentType)
	req.Header.Set("Accept", HTTPContentType)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Error(err, "error for GET call")
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("API error, status code: %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(v); err != nil {
		return err
	}
	return nil
}

// GetAvailableContexts takes list of contexts and returns only which are available in both the 'tanzu context list' command output and the input list
func GetAvailableContexts(tf *Framework, contextNames []string) []string {
	var available []string
	list, _, _, err := tf.ContextCmd.ListContext()
	gomega.Expect(err).To(gomega.BeNil(), "list context should not return any error")
	set := SliceToSet(contextNames)
	for _, context := range list {
		if _, ok := set[context.Name]; ok {
			available = append(available, context.Name)
		}
	}
	return available
}

// GetTMCClusterInfo returns the TMC cluster info by reading environment variables TANZU_CLI_TMC_UNSTABLE_URL and TANZU_API_TOKEN
// Currently we are setting these env variables in GitHub action for local testing these variables need to be set by the developer on their respective machine
func GetTMCClusterInfo() *ClusterInfo {
	return &ClusterInfo{EndPoint: os.Getenv(TanzuCliTmcUnstableURL), APIKey: os.Getenv(TanzuAPIToken)}
}

// CleanConfigFiles deletes the tanzu CLI config files and initializes the tanzu CLI config
func CleanConfigFiles(tf *Framework) error {
	err := tf.Config.DeleteCLIConfigurationFiles()
	if err != nil {
		return err
	}
	// call init
	err = tf.Config.ConfigInit()
	return err
}

// GetJsonOutputFormatAdditionalFlagFunction returns a E2EOption function to add json as output format
func GetJsonOutputFormatAdditionalFlagFunction() E2EOption {
	return AddAdditionalFlagAndValue(JSONOtuput)
}

// ContextInfoToMap takes the contexts list, and returns the map with context name as key and context info as value
func ContextInfoToMap(ctxs []*ContextListInfo) map[string]*ContextListInfo {
	m := make(map[string]*ContextListInfo)
	for i := range ctxs {
		m[ctxs[i].Name] = ctxs[i]
	}
	return m
}
