## tanzu plugin source update

Update a discovery source configuration

```
tanzu plugin source update SOURCE_NAME --uri <URI>
```

### Examples

```

    # Update the discovery source for an air-gapped scenario. The URI must be an OCI image.
    tanzu plugin source update default --uri registry.example.com/tanzu/plugin-inventory:latest
```

### Options

```
  -h, --help         help for update
  -u, --uri string   URI for discovery source. The URI must be of an OCI image
```

### SEE ALSO

* [tanzu plugin source](tanzu_plugin_source.md)	 - Manage plugin discovery sources

