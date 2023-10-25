## tanzu config set

Set config values at the given PATH

### Synopsis

Set config values at the given PATH. Supported PATH values: [features.global.<feature>, features.<plugin>.<feature>, env.<variable>]

```
tanzu config set PATH <value> [flags]
```

### Examples

```

    # Sets a custom CA cert for a proxy that requires it
    tanzu config set env.PROXY_CA_CERT b329baa034afn3.....
    # Enables a specific plugin feature
    tanzu config set features.management-cluster.custom_nameservers true
    # Enables a general CLI feature
    tanzu config set features.global.abcd true
```

### Options

```
  -h, --help   help for set
```

### SEE ALSO

* [tanzu config](tanzu_config.md)	 - Configuration for the CLI

