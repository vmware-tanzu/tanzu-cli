## tanzu builder publish

Publish operations

```
tanzu builder publish [flags]
```

### Options

```
  -h, --help                                       help for publish
      --input-artifact-dir string                  Artifact directory which is a output of 'tanzu builder cli compile' command.
      --local-output-discovery-dir string          Local output directory where CLIPlugin resource yamls for discovery will be placed. Applicable to 'local' type.
      --local-output-distribution-dir string       Local output directory where plugin binaries will be placed. Applicable to 'local' type.
      --oci-discovery-image string                 Image path to publish oci image with CLIPlugin resource yamls. Applicable to 'oci' type.
      --oci-distribution-image-repository string   Image path prefix to publish oci image for plugin binaries. Applicable to 'oci' type.
      --os-arch string                             List of OS architectures. (default "darwin-amd64 linux-amd64 windows-amd64")
      --plugins string                             List of plugin names. Example: 'login management-cluster cluster'
      --type string                                Type of discovery and distribution for publishing plugins. Supported: local, oci
      --version string                             Recommended version of the plugins.
```

### SEE ALSO

* [tanzu builder](tanzu_builder.md)	 - Build Tanzu components
