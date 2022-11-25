## tanzu cluster completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(cluster completion bash)

To load completions for every new session, execute once:

#### Linux:

	cluster completion bash > /etc/bash_completion.d/cluster

#### macOS:

	cluster completion bash > $(brew --prefix)/etc/bash_completion.d/cluster

You will need to start a new shell for this setup to take effect.


```
tanzu cluster completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --log-file string   Log file path
  -v, --verbose int32     Number for the log level verbosity(0-9)
```

### SEE ALSO

* [tanzu cluster completion](tanzu_cluster_completion.md)	 - Generate the autocompletion script for the specified shell

###### Auto generated by spf13/cobra on 14-Sep-2022