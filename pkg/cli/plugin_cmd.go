// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
)

// K8s-targeted plugins command group shows up as top level commands and under
// the kubernetes group. Any remap location that points to one should also map
// to the other to be consistent.
// This function returns the alternate location, if applicable.
func alternateRemapLocation(p *PluginInfo, remapLocation string) string {
	if p.Target == configtypes.TargetK8s {
		cmdHierarchy := strings.Split(remapLocation, " ")
		if len(cmdHierarchy) == 1 {
			return fmt.Sprintf("kubernetes %s", cmdHierarchy[0])
		}
		if len(cmdHierarchy) == 2 && cmdHierarchy[0] == "kubernetes" {
			return cmdHierarchy[1]
		}
	}
	return ""
}

// GetCommandMapForPlugin returns how the plugin's commands should be mapped
func GetCommandMapForPlugin(p *PluginInfo) map[string]*cobra.Command {
	cmdMap := map[string]*cobra.Command{}

	for _, invokedAsItem := range p.InvokedAs {
		remapLocation := strings.TrimSpace(invokedAsItem)
		cmdHierarchy := strings.Split(remapLocation, " ")

		if len(cmdHierarchy) > 0 {
			cmdMap[remapLocation] = getCmdForPluginEx(p, cmdHierarchy[len(cmdHierarchy)-1])
		}
		if alternateLocation := alternateRemapLocation(p, remapLocation); alternateLocation != "" {
			cmdMap[alternateLocation] = cmdMap[remapLocation]
		}
	}

	return cmdMap
}

// GetCmdForPlugin returns a cobra command for the plugin.
func GetCmdForPlugin(p *PluginInfo) *cobra.Command {
	return getCmdForPluginEx(p, p.Name)
}

// GetUnmappedCmdForPlugin returns a cobra command for the plugin unless there
// are remapping directives in the plugin info, in which case it will return
// nil instead.
func GetUnmappedCmdForPlugin(p *PluginInfo) *cobra.Command {
	if len(p.InvokedAs) > 0 {
		return nil
	}
	return getCmdForPluginEx(p, p.Name)
}

// getCmdForPluginEx returns a cobra command for the plugin.
func getCmdForPluginEx(p *PluginInfo, cmdGroupName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdGroupName,
		Short: p.Description,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := NewRunner(p.Name, p.InstallationPath, args)
			ctx := context.Background()
			setupPluginEnv()
			return runner.Run(ctx)
		},
		DisableFlagParsing: true,
		Annotations: map[string]string{
			"group":                  string(p.Group),
			"scope":                  p.Scope,
			"type":                   common.CommandTypePlugin,
			"pluginInstallationPath": p.InstallationPath,
		},
		Hidden:  p.Hidden,
		Aliases: p.Aliases,
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Parses the completion info provided by cobra.Command. This should be formatted similar to:
		//   help	Help about any command
		//   :4
		//   Completion ended with directive: ShellCompDirectiveNoFileComp
		completion := []string{"__complete"}
		completion = append(completion, args...)
		completion = append(completion, toComplete)

		runner := NewRunner(p.Name, p.InstallationPath, completion)
		ctx := context.Background()
		output, _, err := runner.RunOutput(ctx)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		lines := strings.Split(strings.Trim(output, "\n"), "\n")

		directive := cobra.ShellCompDirectiveError
		if len(lines) > 0 {
			lastLine := lines[len(lines)-1]
			lines = lines[:len(lines)-1]

			if lastLine[0] == ':' {
				// Special :(integer) marker at end of output to indicate the
				// outcome of the delegated completion command
				marker, err := strconv.Atoi(lastLine[1:])
				if err == nil {
					directive = cobra.ShellCompDirective(marker)
				}
			}
		}
		return lines, directive
	}

	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		// Plugin commands don't provide full details to the default "help" cmd.
		// To get around this, we need to intercept and send the help request
		// out to the plugin.
		// Cobra also doesn't pass along any additional args since it has parsed
		// the command structure, and as far as it knows, there are no subcommands
		// below the top level plugin command. To get around this to support help
		// calls such as "tanzu help cluster list", we need to do some argument
		// parsing ourselves and modify what gets passed along to the plugin.
		helpArgs := getHelpArguments()

		// Pass this new command in to our plugin to have it handle help output
		runner := NewRunner(p.Name, p.InstallationPath, helpArgs)
		ctx := context.Background()
		err := runner.Run(ctx)
		if err != nil {
			log.Errorf("Help output for '%s' is not available.", c.Name())
		}
	})
	return cmd
}

// getHelpArguments extracts the command line to pass along to help calls.
// The help function is only ever called for help commands in the format of
// "tanzu help cmd", so we can assume anything two after "help" should get
// passed along (this also accounts for aliases).
func getHelpArguments() []string {
	cliArgs := os.Args
	helpArgs := []string{}
	for i := range cliArgs {
		if cliArgs[i] == "help" {
			// Found the "help" argument, now capture anything after the plugin name/alias
			argLen := len(cliArgs)
			if (i + 1) < argLen {
				helpArgs = cliArgs[i+2:]
			}
			break
		}
	}

	// Then add the -h flag for whatever we found
	return append(helpArgs, "-h")
}

// setupPluginEnv prepares some extra environment variables
// that communicate certain information to plugins.
func setupPluginEnv() {
	env := make(map[string]string, 10)

	// The location of the tanzu binary
	env["TANZU_BIN"] = os.Args[0]

	for key, val := range env {
		os.Setenv(key, val)
	}
}

// GetTestCmdForPlugin returns a cobra command for the test plugin.
func GetTestCmdForPlugin(p *PluginInfo) *cobra.Command {
	cmd := &cobra.Command{
		Use:   p.Name,
		Short: p.Description,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := NewRunner(p.Name, p.InstallationPath, args)
			ctx := context.Background()
			return runner.RunTest(ctx)
		},
		DisableFlagParsing: true,
	}
	return cmd
}
