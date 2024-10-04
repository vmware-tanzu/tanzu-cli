## tanzu plugin download-bundle

Download plugin bundle to the local system

### Synopsis

Download a plugin bundle to the local file system to be used when migrating plugins
to an internet-restricted environment. Please also see the "upload-bundle" command.

```
tanzu plugin download-bundle [flags]
```

### Examples

```

    # Download a plugin bundle for a specific group version from the default discovery source
    tanzu plugin download-bundle --group vmware-tkg/default:v1.0.0 --to-tar /tmp/plugin_bundle_vmware_tkg_default_v1.0.0.tar.gz

    # To download plugin bundle with a specific plugin from the default discovery source
    #     --plugin name                 : Downloads the latest available version of the plugin. (Returns an error if the specified plugin name is available across multiple targets)
    #     --plugin name:version         : Downloads the specified version of the plugin. (Returns an error if the specified plugin name is available across multiple targets)
    #     --plugin name@target:version  : Downloads the specified version of the plugin for the specified target.
    #     --plugin name@target          : Downloads the latest available version of the plugin for the specified target.
    tanzu plugin download-bundle --plugin cluster:v1.0.0 --to-tar /tmp/plugin_bundle_cluster.tar.gz

    # Download a plugin bundle with the entire plugin repository from a custom discovery source
    tanzu plugin download-bundle --image custom.registry.vmware.com/tkg/tanzu-plugins/plugin-inventory:latest --to-tar /tmp/plugin_bundle_complete.tar.gz
```

### Options

```
      --group strings                only download the plugins specified in the plugin-group version (can specify multiple)
  -h, --help                         help for download-bundle
      --image string                 URI of the plugin discovery image providing the plugins (default "projects.packages.broadcom.com/tanzu_cli/plugins/plugin-inventory:latest")
      --plugin strings               only download plugins matching specified pluginID. Format: name/name:version/name@target:version (can specify multiple)
      --refresh-configuration-only   only refresh the central configuration data
      --to-tar string                local tar file path to store the plugin images
```

### SEE ALSO

* [tanzu plugin](tanzu_plugin.md)	 - Manage CLI plugins

