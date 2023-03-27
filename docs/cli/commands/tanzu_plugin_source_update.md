## tanzu plugin source update

Update a discovery source configuration

```
tanzu plugin source update [name] [flags]
```

### Examples

```

    # Update a local discovery source. If URI is relative path, 
    # $HOME/.config/tanzu-plugins will be considered base path
    tanzu plugin source update standalone-local --type local --uri new/path/to/local/discovery

    # Update an OCI discovery source. URI should be an OCI image.
    tanzu plugin source update standalone-oci --type oci --uri projects.registry.vmware.com/tkg/tanzu-plugins/standalone:v1.0
```

### Options

```
  -h, --help          help for update
  -t, --type string   type of discovery source
  -u, --uri string    URI for discovery source. URI format might be different based on the type of discovery source
```

### SEE ALSO

* [tanzu plugin source](tanzu_plugin_source.md)	 - Manage plugin discovery sources

