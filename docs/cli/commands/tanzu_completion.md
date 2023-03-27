## tanzu completion

Output shell completion code

### Synopsis


Output shell completion code for the specified shell [bash zsh fish powershell].

The shell completion code must be evaluated to provide completion. See Examples
for how to perform this for your given shell.

Note for bash users: make sure the bash-completions package has been installed.

```
tanzu completion [bash zsh fish powershell]
```

### Examples

```

# Bash instructions:

  ## Load only for current session:
  source <(tanzu completion bash)

  ## Load for all new sessions:
  tanzu completion bash >  $HOME/.config/tanzu/completion.bash.inc
  printf "\n# Tanzu shell completion\nsource '$HOME/.config/tanzu/completion.bash.inc'\n" >> $HOME/.bash_profile

  ## NOTE: the bash-completion package must be installed.

# Zsh instructions:

  ## Load only for current session:
  autoload -U compinit; compinit
  source <(tanzu completion zsh)
  compdef _tanzu tanzu

  ## Load for all new sessions:
  echo "autoload -U compinit; compinit" >> ~/.zshrc
  tanzu completion zsh > "${fpath[1]}/_tanzu"

# Fish instructions:

  ## Load only for current session:
  tanzu completion fish | source

  ## Load for all new sessions:
  tanzu completion fish > ~/.config/fish/completions/tanzu.fish

# Powershell instructions:

  ## Load only for current session:
  tanzu completion powershell | Out-String | Invoke-Expression

  ## Load for all new sessions:
  Add the output of the above command to your powershell profile.
```

### Options

```
  -h, --help   help for completion
```

### SEE ALSO

* [tanzu](tanzu.md)	 - 

