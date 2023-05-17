## tanzu plugin upload-bundle

Upload plugin bundle to a repository

### Synopsis

Upload plugin bundle to a repository

```
tanzu plugin upload-bundle [flags]
```

### Examples

```

# Upload the plugin bundle to the remote repository
tanzu plugin upload-bundle --tar /tmp/plugin_bundle_vmware_tkg_default_v1.0.0.tar.gz --to-repo custom.registry.company.com/tanzu-plugins/
tanzu plugin upload-bundle --tar /tmp/plugin_bundle_complete.tar.gz --to-repo custom.registry.company.com/tanzu-plugins/
```

### Options

```
  -h, --help             help for upload-bundle
      --tar string       source tar file
      --to-repo string   destination repository for publishing plugins
```

### SEE ALSO

* [tanzu plugin](tanzu_plugin.md)	 - Manage CLI plugins

