## tanzu completion

Output shell completion code

### Synopsis

Output shell completion code for the specified shell (bash, zsh, fish, powershell).

The shell completion code must be evaluated to provide completion. See Examples
for how to perform this for your given shell.

Note for bash users: make sure the bash-completions package has been installed.

```
tanzu completion [bash|zsh|fish|powershell]
```

### Examples

```

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
  Register-ArgumentCompleter -CommandName 'tz' -ScriptBlock ${__tanzuCompleterBlock}
```

### Options

```
  -h, --help   help for completion
```

### SEE ALSO

* [tanzu](tanzu.md)	 - The Tanzu CLI

