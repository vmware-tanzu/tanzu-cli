// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package telemetry collects the CLI metrics and sends the telemetry data to supercollider
package telemetry

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugincmdtree"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

var once sync.Once

var client MetricsHandler

type MetricsHandler interface {
	// SetInstalledPlugins adds the installed plugins to the handler used to retrieve
	// the plugin information
	SetInstalledPlugins(plugins []cli.PluginInfo)
	// UpdateCmdPreRunMetrics updates the metrics collected before running the command
	UpdateCmdPreRunMetrics(cmd *cobra.Command, args []string) error
	// UpdateCmdPostRunMetrics updates the metrics collected after command execution is completed
	UpdateCmdPostRunMetrics(metrics *PostRunMetrics) error
	// SaveMetrics saves the metrics to the metrics store/DB
	SaveMetrics() error
	// SendMetrics sends the metrics to the destination(metrics data lake)
	SendMetrics() error
}

type telemetryClient struct {
	currentOperationMetrics *OperationMetricsPayload
	installedPlugins        []cli.PluginInfo
	metricsDB               MetricsDB
	cmdTreeCacheGetter      func() (plugincmdtree.Cache, error)
}

type PostRunMetrics struct {
	ExitCode int
}
type OperationMetricsPayload struct {
	CliID         string
	StartTime     time.Time
	EndTime       time.Time
	Args          []string
	NameArg       string
	CommandName   string
	ExitStatus    int
	PluginName    string
	Flags         string
	CliVersion    string
	PluginVersion string
	Target        string
	Endpoint      string
	Error         string
}

func Client() MetricsHandler {
	once.Do(func() {
		client = newTelemetryClient()
	})
	return client
}

func newTelemetryClient() MetricsHandler {
	opMetrics := &OperationMetricsPayload{}
	metricsDB := newSQLiteMetricsDB()
	return &telemetryClient{
		currentOperationMetrics: opMetrics,
		metricsDB:               metricsDB,
		cmdTreeCacheGetter:      pluginCommandTreeCacheGetter,
	}
}

func (tc *telemetryClient) SetInstalledPlugins(plugins []cli.PluginInfo) {
	tc.installedPlugins = plugins
}

func (tc *telemetryClient) UpdateCmdPreRunMetrics(cmd *cobra.Command, args []string) error {
	if err := ensureMetricsSource(); err != nil {
		return errors.Wrap(err, "failed to ensure metrics source in the configuration file")
	}

	cliID, err := cliInstanceID()
	if err != nil {
		return errors.Wrap(err, "unable to get CLI Instance ID")
	}

	if isCoreCommand(cmd) {
		return tc.updateMetricsForCoreCommand(cmd, args, cliID)
	}

	return tc.updateMetricsForPlugin(cmd, args, cliID)
}

func (tc *telemetryClient) UpdateCmdPostRunMetrics(metrics *PostRunMetrics) error {
	if metrics == nil {
		return errors.New("post metrics data is required for update")
	}
	tc.currentOperationMetrics.ExitStatus = metrics.ExitCode
	tc.currentOperationMetrics.EndTime = time.Now()
	return nil
}

func (tc *telemetryClient) SaveMetrics() error {
	// If cli command fail cobra validation, the PersistentPreRunE() wouldn't be invoked where initialization is done
	// so, it is safe to ignore the metrics for user errors(like typos) at least to an extent where cobra can validate.
	if tc.currentOperationMetrics.StartTime.IsZero() {
		return nil
	}

	err := tc.metricsDB.CreateSchema()
	if err != nil {
		return errors.Wrap(err, "unable to create the telemetry schema")
	}

	return tc.metricsDB.SaveOperationMetric(tc.currentOperationMetrics)
}

// SendMetrics sends the local stored metrics to super collider
// TODO: to be implemented
func (tc *telemetryClient) SendMetrics() error {
	return nil
}

func isCoreCommand(cmd *cobra.Command) bool {
	if cmd.Annotations != nil && cmd.Annotations["type"] == common.CommandTypePlugin {
		return false
	}
	return true
}
func (tc *telemetryClient) updateMetricsForCoreCommand(cmd *cobra.Command, args []string, cliID string) error {
	tc.currentOperationMetrics.CliID = cliID
	tc.currentOperationMetrics.CliVersion = buildinfo.Version
	tc.currentOperationMetrics.StartTime = time.Now()
	tc.currentOperationMetrics.CommandName = strings.Join(strings.Split(cmd.CommandPath(), " ")[1:], " ")

	// CLI recommendation is to have a single argument for a command
	if len(args) != 0 {
		tc.currentOperationMetrics.NameArg = hashString(args[0])
	}

	flagMap := make(map[string]string)
	hashRequired := isHashRequiredForCmdFlags(cmd.CommandPath())
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		// capture the boolean and empty flag values as is
		if !hashRequired || flag.Value.String() == "" || flag.Value.Type() == "bool" {
			flagMap[flag.Name] = flag.Value.String()
		} else {
			flagMap[flag.Name] = hashString(flag.Value.String())
		}
	})
	if len(flagMap) != 0 {
		jsonString, _ := json.Marshal(flagMap)
		tc.currentOperationMetrics.Flags = string(jsonString)
	}

	return nil
}

func (tc *telemetryClient) updateMetricsForPlugin(cmd *cobra.Command, args []string, cliID string) error {
	tc.currentOperationMetrics.CliID = cliID
	tc.currentOperationMetrics.CliVersion = buildinfo.Version
	tc.currentOperationMetrics.StartTime = time.Now()

	flagNames := TraverseFlagNames(args)
	if len(flagNames) > 0 {
		tc.currentOperationMetrics.Flags = flagNamesToJSONString(flagNames)
	}

	plugin := tc.pluginInfoFromCommand(cmd)
	if plugin != nil {
		tc.currentOperationMetrics.PluginName = plugin.Name
		tc.currentOperationMetrics.PluginVersion = plugin.Version
		tc.currentOperationMetrics.Target = string(plugin.Target)
		tc.currentOperationMetrics.Endpoint = getEndpointSHA(plugin)
		// for plugins, cobra can only parse the command upto the plugin name,
		// and the rest of the subcommands and args would be captured as args
		// ex: tanzu cluster kubeconfig get testCluster --export-file /path/to/file
		//   the above command after parsing cobra will provide the below
		//    ==>   cmd.CommandPath() would return "tanzu cluster"
		//          args = ["kubeconfig","get","testCluster","--export-file","/path/to/file"]
		// So, use the plugin command parser to figure out(best-effort) the command path using command tree as reference
		cobraParsedCMDPath := strings.Join(strings.Split(cmd.CommandPath(), " ")[1:], " ")
		cmdPath, err := tc.parsePluginCommandPath(plugin, args)
		if err != nil {
			LogError(err, "")
			// assign the default plugin path
			tc.currentOperationMetrics.CommandName = cobraParsedCMDPath
		} else {
			tc.currentOperationMetrics.CommandName = cobraParsedCMDPath + cmdPath
		}
	}

	return nil
}
func (tc *telemetryClient) pluginInfoFromCommand(cmd *cobra.Command) *cli.PluginInfo {
	var plugin *cli.PluginInfo
	if cmd.Annotations == nil || cmd.Annotations["pluginInstallationPath"] == "" {
		return nil
	}
	for i := range tc.installedPlugins {
		if cmd.Annotations["pluginInstallationPath"] == tc.installedPlugins[i].InstallationPath {
			return &tc.installedPlugins[i]
		}
	}

	return plugin
}

// parsePluginCommandPath parses the args provided by the cobra and uses the best-effort strategy to
// map to the plugin command tree and would return the command path
func (tc *telemetryClient) parsePluginCommandPath(plugin *cli.PluginInfo, args []string) (string, error) {
	pctCache, err := tc.cmdTreeCacheGetter()
	if err != nil {
		return "", err
	}
	pct, err := pctCache.GetTree(plugin)
	if err != nil {
		return "", err
	}
	cmdPath := ""
	current := pct
	for _, arg := range args {
		if current == nil || current.Subcommands == nil || len(current.Subcommands) == 0 {
			return cmdPath, nil
		}
		switch {
		// "--" terminates the flags (everything after is an argument)
		case arg == doubleHyphen:
			return cmdPath, nil
		// A flag without a value, or with an `=` separated value
		case isFlagArg(arg):
			continue
		default:
			if subCMD := subCommandMatchingArg(current, arg); subCMD != nil {
				cmdPath = cmdPath + " " + arg
				current = subCMD
			}
		}
	}

	return cmdPath, nil
}
func subCommandMatchingArg(current *plugincmdtree.CommandNode, arg string) *plugincmdtree.CommandNode {
	if current.Subcommands == nil {
		return nil
	}
	if subCMD, exists := current.Subcommands[arg]; exists {
		return subCMD
	}
	for _, subCMD := range current.Subcommands {
		if subCMD.Aliases != nil {
			if _, exists := subCMD.Aliases[arg]; exists {
				return subCMD
			}
		}
	}
	return nil
}
func cliInstanceID() (string, error) {
	cliID, err := configlib.GetCLIId()
	if err != nil {
		return "", err
	}
	return cliID, nil
}

func ensureMetricsSource() error {
	telemetryOptions, _ := configlib.GetCLITelemetryOptions()
	dbFile := filepath.Join(common.DefaultCLITelemetryDir, SQliteDBFileName)
	if telemetryOptions != nil && telemetryOptions.Source == dbFile {
		return nil
	}

	err := configlib.SetCLITelemetryOptions(&configtypes.TelemetryOptions{Source: dbFile})
	if err != nil {
		return err
	}
	return nil
}

// isHashRequiredForCmdFlags determines if hashing is required for a core command read from command path
// currently, for each command we are either hashing all the values or none. A possible enhancement would be
// to return list of flags whose values need to be hashed.
func isHashRequiredForCmdFlags(cmdPath string) bool {
	coreCommandsAllowedWithFlagValues := map[string]struct{}{
		"plugin": struct{}{},
	}

	cmds := strings.Split(cmdPath, " ")
	if len(cmds) < 2 {
		return false
	}

	if _, exists := coreCommandsAllowedWithFlagValues[cmds[1]]; exists {
		return false
	}
	return true
}

func pluginCommandTreeCacheGetter() (plugincmdtree.Cache, error) {
	pctCache, err := plugincmdtree.NewCache()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plugin command tree cache")
	}
	return pctCache, nil
}
