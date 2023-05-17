## tanzu builder inventory plugin-group add

Add the plugin-group to the inventory database available on the remote repository

```
tanzu builder inventory plugin-group add [flags]
```

### Options

```
      --deactivate                          mark plugin-group as deactivated
      --description string                  a description for the plugin-group
  -h, --help                                help for add
      --manifest string                     manifest file specifying plugin-group details that needs to be processed
      --name string                         name of the plugin-group
      --override                            overwrite the plugin-group version if it already exists
      --plugin-inventory-image-tag string   tag to which plugin inventory image needs to be published (default "latest")
      --publisher string                    name of the publisher
      --repository string                   repository to publish plugin inventory image
      --vendor string                       name of the vendor
      --version string                      version of the plugin-group
```

### SEE ALSO

* [tanzu builder inventory plugin-group](tanzu_builder_inventory_plugin-group.md)	 - Plugin-Group Inventory Operations

