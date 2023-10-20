// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"github.com/spf13/cobra"
)

const (
	// Completion strings for the values of the --target flag
	compGlobalTarget = "global\tApplicable globally"
	compK8sTarget    = "k8s\tFor interactions with a Kubernetes cluster"
	compTMCTarget    = "tmc\tFor interactions with a Tanzu Mission Control endpoint"

	// Completion strings for the values of the --type flag
	compK8sContextType   = "k8s\tContext for a Kubernetes cluster"
	compTanzuContextType = "tanzu\tContext for a Tanzu endpoint"
	compTMCContextType   = "tmc\tContext for a Tanzu Mission Control endpoint"

	// Completion strings for the values of the --output flag
	compTableOutput = "table\tOutput results in human-readable format"
	compJSONOutput  = "json\tOutput results in JSON format"
	compYAMLOutput  = "yaml\tOutput results in YAML format"
)

// TODO(khouzam): move this to tanzu-plugin-runtime to be usable by plugins
func completionGetOutputFormats(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{compTableOutput, compJSONOutput, compYAMLOutput}, cobra.ShellCompDirectiveNoFileComp
}
