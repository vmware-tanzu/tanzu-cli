## tanzu builder plugin build

Build plugins

```
tanzu builder plugin build [flags]
```

### Examples

```
# Build all plugins under 'cmd/plugin' directory for local host os and arch
  tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch local

  # Build all plugins under 'cmd/plugin' directory for os-arch 'darwin_amd64', 'linux_amd64', 'windows_amd64'
  tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch darwin_amd64 --os-arch linux_amd64 --os-arch windows_amd64

  # Build only foo plugin under 'cmd/plugin' directory for all supported os-arch
  tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch all --match foo
```

### Options

```
      --binary-artifacts string                path to output artifacts directory (default "./artifacts")
  -h, --help                                   help for build
      --ldflags string                         ldflags to set on build
      --match string                           match a plugin name to build, supports globbing (default "*")
      --os-arch stringArray                    compile for specific os-arch, use 'local' for host os, use '<os>_<arch>' for specific (default [all])
      --path string                            path of plugin directory (default "./cmd/plugin")
      --plugin-scope-association-file string   file specifying plugin scope association
  -v, --version string                         version of the plugins
```

### SEE ALSO

* [tanzu builder plugin](tanzu_builder_plugin.md)	 - Plugin Operations

