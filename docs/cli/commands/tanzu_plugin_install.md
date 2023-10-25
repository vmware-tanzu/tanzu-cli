## tanzu plugin install

Install a plugin

### Synopsis

Install a specific plugin by name or specify all to install all plugins of a group

```
tanzu plugin install [PLUGIN_NAME] [flags]
```

### Examples

```

    # Install all plugins of the vmware-tkg/default plugin group version v2.1.0
    tanzu plugin install --group vmware-tkg/default:v2.1.0

    # Install all plugins of the latest version of the vmware-tkg/default plugin group
    tanzu plugin install --group vmware-tkg/default

    # Install all plugins from the latest minor and patch of the v1 version of the vmware-tkg/default plugin group
    tanzu plugin install --group vmware-tkg/default:v1

    # Install all plugins from the latest patch of the v1.2 version of the vmware-tkg/default plugin group
    tanzu plugin install --group vmware-tkg/default:v1.2

    # Install the latest version of plugin "myPlugin"
    # If the plugin exists for more than one target, an error will be thrown
    tanzu plugin install myPlugin

    # Install the latest version of plugin "myPlugin" for target kubernetes
    tanzu plugin install myPlugin --target k8s

    # Install version v1.0.0 of plugin "myPlugin"
    tanzu plugin install myPlugin --version v1.0.0

    # Install latest patch version of v1.0 of plugin "myPlugin"
    tanzu plugin install myPlugin --version v1.0

    # Install latest minor and patch version of v1 of plugin "myPlugin"
    tanzu plugin install myPlugin --version v1
```

### Options

```
      --group string     install the plugins specified by a plugin-group version
  -h, --help             help for install
  -t, --target string    target of the plugin (kubernetes[k8s]/mission-control[tmc]/global)
  -v, --version string   version of the plugin (default "latest")
```

### SEE ALSO

* [tanzu plugin](tanzu_plugin.md)	 - Manage CLI plugins

