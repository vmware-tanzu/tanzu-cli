// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
)

const compNoMoreArgsMsg = "This command does not take any more arguments (but may accept flags)."

var (
	completionShells = []string{
		"bash",
		"zsh",
		"fish",
		"powershell",
	}

	completionLongDesc = dedent.Dedent(`
		Output shell completion code for the specified shell %v.

		The shell completion code must be evaluated to provide completion. See Examples
		for how to perform this for your given shell.

		Note for bash users: make sure the bash-completions package has been installed.`)

	completionExamples = dedent.Dedent(`
		# Bash instructions:

		  ## Load only for current session:
		  source <(tanzu completion bash)

		  ## Load for all new sessions:
		  tanzu completion bash > $HOME/.config/tanzu/completion.bash.inc
		  printf "\n# Tanzu shell completion\nsource '$HOME/.config/tanzu/completion.bash.inc'\n" >> $HOME/.bashrc

		  ## NOTE: the bash-completion OS package must also be installed.

		  ## If you invoke the 'tanzu' command using a different name or an alias such as,
		  ## for example, 'tz', you must also include the following in your $HOME/.bashrc
		  complete -o default -F __start_tanzu tz

		# Zsh instructions:

		  ## Load only for current session:
		  autoload -U compinit; compinit
		  source <(tanzu completion zsh)

		  ## Load for all new sessions:
		  echo "autoload -U compinit; compinit" >> $HOME/.zshrc
		  tanzu completion zsh > "${fpath[1]}/_tanzu"

		  ## Aliases are handled automatically, but if you have renamed the actual 'tanzu' binary to,
		  ## for example, 'tz', you must also include the following in your $HOME/.zshrc
		  compdef _tanzu tz

		# Fish instructions:

		  ## Load only for current session:
		  tanzu completion fish | source

		  ## Load for all new sessions:
		  tanzu completion fish > $HOME/.config/fish/completions/tanzu.fish

		  ## Aliases are handled automatically, but if you have renamed the actual 'tanzu' binary to,
		  ## for example, 'tz', you must also include the following in your $HOME/.config/fish/config.fish
		  complete --command tz --wraps tanzu

		# Powershell instructions:

		  ## Load only for current session:
		  tanzu completion powershell | Out-String | Invoke-Expression

		  ## Load for all new sessions:
		  printf "\n# Tanzu shell completion\ntanzu completion powershell | Out-String | Invoke-Expression" >> $PROFILE

		  ## If you invoke the 'tanzu' command using a different name or an alias such as,
		  ## for example, 'tz', you must also include the following in your powershell $PROFILE.
		  Register-ArgumentCompleter -CommandName 'tz' -ScriptBlock ${__tanzuCompleterBlock}`)
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:                   fmt.Sprintf("completion %v", completionShells),
	Short:                 "Output shell completion code",
	Long:                  fmt.Sprintf(completionLongDesc, completionShells),
	Example:               completionExamples,
	DisableFlagsInUseLine: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completionShells, cobra.ShellCompDirectiveNoFileComp
		}
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCompletion(os.Stdout, cmd, args)
	},
	Annotations: map[string]string{
		"group": string(plugin.SystemCmdGroup),
	},
}

func runCompletion(out io.Writer, cmd *cobra.Command, args []string) error {
	if length := len(args); length == 0 {
		return fmt.Errorf("shell not specified, choose one of: %v", completionShells)
	} else if length > 1 {
		return errors.New("too many arguments, expected only the shell type")
	}

	switch strings.ToLower(args[0]) {
	case "bash":
		return cmd.Root().GenBashCompletionV2(out, true)
	case "zsh":
		return cmd.Root().GenZshCompletion(out)
	case "fish":
		return cmd.Root().GenFishCompletion(out, true)
	case "powershell", "pwsh":
		return cmd.Root().GenPowerShellCompletionWithDesc(out)
	default:
		return errors.New("unrecognized shell type specified")
	}
}
