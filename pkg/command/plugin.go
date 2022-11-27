// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/pkg/catalog"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

var (
	local        string
	outputFormat string
)

func newPluginCmd(ps catalog.PluginSupplier) *cobra.Command {
	var pluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Manage CLI plugins",
		Annotations: map[string]string{
			"group": string(plugin.SystemCmdGroup),
		},
	}

	pluginCmd.SetUsageFunc(cli.SubCmdUsageFunc)
	listPluginCmd, err := NewListCmd(ps)
	if err == nil {
		pluginCmd.AddCommand(listPluginCmd)
	}
	listPluginCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (yaml|json|table)")
	listPluginCmd.Flags().StringVarP(&local, "local", "l", "", "path to local plugin source")

	return pluginCmd
}

func NewListCmd(ps catalog.PluginSupplier) (*cobra.Command, error) {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Not handling plugin discovery yet, so only showing installed plugins

			descriptors, err := ps.GetInstalledPlugins()
			if err != nil {
				return err
			}

			data := [][]string{}
			for _, desc := range descriptors {
				var exists bool
				for _, d := range data {
					if desc.Name == d[0] {
						exists = true
						break
					}
				}
				if !exists {
					data = append(data, []string{desc.Name, desc.Description, desc.Version, "installed"})
				}
			}

			// sort plugins based on their names
			sort.SliceStable(data, func(i, j int) bool {
				return strings.ToLower(data[i][0]) < strings.ToLower(data[j][0])
			})

			output := component.NewOutputWriter(cmd.OutOrStdout(), outputFormat, "Name", "Description", "Version", "Status")
			for _, row := range data {
				vals := make([]interface{}, len(row))
				for i, val := range row {
					vals[i] = val
				}
				output.AddRow(vals...)
			}
			output.Render()
			return nil
		},
	}

	return listCmd, nil
}
