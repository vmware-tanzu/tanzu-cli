## tanzu plugin search

Search for available plugins

### Synopsis

Search provides the ability to search for plugins that can be installed.
The command lists all plugins currently available for installation.
The search command also provides flags to limit the scope of the search.


```
tanzu plugin search [flags]
```

### Options

```
  -h, --help            help for search
  -l, --local string    path to local plugin source
  -n, --name string     limit the search to plugins with the specified name
  -o, --output string   output format (yaml|json|table)
      --show-details    show the details of the specified plugin, including all available versions
  -t, --target string   limit the search to plugins of the specified target (kubernetes[k8s]/mission-control[tmc]/global)
```

### SEE ALSO

* [tanzu plugin](tanzu_plugin.md)	 - Manage CLI plugins

