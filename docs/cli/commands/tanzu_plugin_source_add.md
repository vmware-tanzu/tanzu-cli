## tanzu plugin source add

Add a discovery source

### Synopsis

Add a discovery source. Supported discovery types are: oci, local

```
tanzu plugin source add [flags]
```

### Examples

```

    # Add a local discovery source. If URI is relative path,
    # $HOME/.config/tanzu-plugins will be considered based path
    tanzu plugin source add --name standalone-local --type local --uri path/to/local/discovery

    # Add an OCI discovery source. URI should be an OCI image.
    tanzu plugin source add --name standalone-oci --type oci --uri projects.registry.vmware.com/tkg/tanzu-plugins/standalone:latest
```

### Options

```
  -h, --help          help for add
  -n, --name string   name of discovery source
  -t, --type string   type of discovery source
  -u, --uri string    URI for discovery source. URI format might be different based on the type of discovery source
```

### SEE ALSO

* [tanzu plugin source](tanzu_plugin_source.md)	 - Manage plugin discovery sources

