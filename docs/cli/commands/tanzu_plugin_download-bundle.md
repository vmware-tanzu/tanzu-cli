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
    tanzu plugin download-bundle --to-tar /tmp/plugin_bundle_vmware_tkg_default_v1.0.0.tar.gz --group vmware-tkg/default:v1.0.0

    # Download a plugin bundle with the entire plugin repository from a custom discovery source
    tanzu plugin download-bundle --image custom.registry.vmware.com/tkg/tanzu-plugins/plugin-inventory:latest --to-tar /tmp/plugin_bundle_complete.tar.gz
```

### Options

```
      --group strings   only download the plugins specified in the plugin-group version (can specify multiple)
  -h, --help            help for download-bundle
      --image string    URI of the plugin discovery image providing the plugins (default "projects.packages.broadcom.com/tanzu_cli/plugins/plugin-inventory:latest")
      --to-tar string   local tar file path to store the plugin images
```

### SEE ALSO

* [tanzu plugin](tanzu_plugin.md)	 - Manage CLI plugins

