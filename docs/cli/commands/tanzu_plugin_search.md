## tanzu plugin search

Search for a keyword or regex in the list of available plugins

### Synopsis

Search provides the ability to search for plugins that can be installed.
Without an argument, the command lists all plugins currently available.
The search command can also be used with a keyword filter to filter the
list of available plugins. If the filter is flanked with slashes, the
filter will be treated as a regex.


```
tanzu plugin search [keyword|/regex/] [flags]
```

### Options

```
  -h, --help            help for search
      --list-versions   show the long listing, with each available version of plugins
  -l, --local string    path to local plugin source
  -o, --output string   Output format (yaml|json|table)
  -t, --target string   list plugins for the specified target (kubernetes[k8s]/mission-control[tmc])
```

### SEE ALSO

* [tanzu plugin](tanzu_plugin.md)	 - Manage CLI plugins

