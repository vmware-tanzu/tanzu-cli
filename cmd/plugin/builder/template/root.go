// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package template

import "github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/template/plugintemplates"

const gitignore = `artifacts
tools/bin`

// GoMod target
var GoMod = Target{
	Filepath: "go.mod",
	Template: plugintemplates.Gomod,
}

// GitIgnore target
var GitIgnore = Target{
	Filepath: ".gitignore",
	Template: gitignore,
}

// GitLabCI target
var GitLabCI = Target{
	Filepath: ".gitlab-ci.yaml",
	Template: plugintemplates.GitlabCI,
}

// GitHubCI target
var GitHubCI = Target{
	Filepath: ".github/workflows/build.yaml",
	Template: plugintemplates.GithubWorkflowBuild,
}

// CommonMK MK4 target
var CommonMK = Target{
	Filepath: "common.mk",
	Template: plugintemplates.CommonMK,
}

// Makefile target
var Makefile = Target{
	Filepath: "Makefile",
	Template: plugintemplates.Makefile,
}

// PluginToolingMK target
var PluginToolingMK = Target{
	Filepath: "plugin-tooling.mk",
	Template: plugintemplates.PluginToolingMK,
}

// Codeowners target
var Codeowners = Target{
	Filepath: "CODEOWNERS",
	Template: `* # edit as appropriate`,
}

// MainReadMe target
var MainReadMe = Target{
	Filepath: "README.md",
	Template: plugintemplates.PluginReadme,
}

// GolangCIConfig target.
var GolangCIConfig = Target{
	Filepath: ".golangci.yaml",
	Template: plugintemplates.GolangCIConfig,
}
