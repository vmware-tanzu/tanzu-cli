# Builder

Scaffolds and builds Tanzu plugin repositories

## Usage

### Init

`tanzu builder init <repo-name>` will initialize a new plugin repository with scaffolding for:

* Tanzu Framework CLI integration
* GolangCI linting config
* GitHub or GitLab CI config
* A Makefile

For more details, this command supports a `--dry-run` flag which will show everything created:

```sh
tanzu builder init <repo-name> --dry-run
```

### Add-plugin

`tanzu builder cli add-plugin <plugin-name>` adds a new plugin to your repository. The plugins command will live in the `./cmd/plugin/<plugin-name>` directory.

### Compile

`tanzu builder cli compile` will compile a repository and create the artifacts to be used with tanzu cli.

The artifact output directory structure will be created to match the expected layout. This will include some plugin
metadata used in the publishing and installation of plugins in a `manifest.yaml` file and a `plugin.yaml` file for
each included plugin.

Plugins will find that their `make build` command will suffice for most compile cases, but there are many flags at your disposal as well:

```txt
--artifacts string   path to output artifacts (default "artifacts")
--ldflags string     ldflags to set on build
--match string       match a plugin name to build, supports globbing (default "*")
--path string        path of the plugins directory (default "./cmd/cli/plugin")
--target string      only compile for a specific target, use 'local' to compile for host os (default "all")
--version string     version of the root cli (required)
```

### Build-plugins

`tanzu builder plugin build` can be used to build the plugins and create artifacts that can be used with tanzu cli.

The artifacts output directory structure will be different compared to what we have with `tanzu builder cli compile`
command in that the `tanzu builder plugin build` command will automatically group plugins based on the given os_arch
and `target`.
This command will also generate a plugin manifest file (`plugin_manifest.yaml`) that can be used in the publishing and
installation of plugins from the local directory with `tanzu plugin install --local`.

Below are the flags available with this command:

```txt
      --binary-artifacts string                path to output artifacts directory (default "./artifacts")
  -h, --help                                   help for build
      --ldflags string                         ldflags to set on build
      --match string                           match a plugin name to build, supports globbing (default "*")
      --os-arch stringArray                    compile for specific os-arch, use 'local' for host os, use '<os>_<arch>' for specific (default [all])
      --path string                            path of plugin directory (default "./cmd/plugin")
      --plugin-scope-association-file string   file specifying plugin scope association
  -v, --version string                         version of the plugins
```

Below are the examples:

```shell
  # Build all plugins under the 'cmd/plugin' directory for the local host os and arch
  tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch local

  # Build all plugins under the 'cmd/plugin' directory for os-arch 'darwin_amd64', 'linux_amd64', 'windows_amd64'
  tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch darwin_amd64 --os-arch linux_amd64 --os-arch windows_amd64

  # Build only foo plugin under the 'cmd/plugin' directory for all supported os-arch
  tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch all --match foo
```

The `tanzu builder plugin build` command provides a convenient way to create a [plugin-group manifest file](#inventory-plugin-group-add) (`plugin_group_manifest.yaml`) containing plugin-group metadata by providing the `--plugin-scope-association-file` flag. The purpose of a plugin-group is to define a product-release-specific set of plugins for users to easily install plugins for the specific product release. More details are provided in the [inventory-plugin-group-add](#inventory-plugin-group-add) section.

Using the `--plugin-scope-association-file` flag is a convenient way to generate a plugin-group manifest file consisting of the plugins built in the `artifacts` directory.  However, if any external plugins or different versions of plugins need to be included in the plugin-group manifest file, the developer will need to manually create this file. When the `--plugin-scope-association-file` flag is provided, the tooling will generate the `plugin_group_manifest.yaml` file within the same binary artifacts directory.

Below is a sample `plugin-scope-association.yaml`:

```yaml
plugins:
- name: foo
  target: global
  isContextScoped: false
- name: bar
  target: kubernetes
  isContextScoped: true
- name: baz
  target: mission-control
  isContextScoped: true
```

Below is the `plugin_group_manifest.yaml` that will get generated based on the above `plugin-scope-association.yaml`:

```yaml
plugins:
- name: foo
  target: global
  isContextScoped: false
  version: v0.0.2
- name: bar
  target: kubernetes
  isContextScoped: true
  version: v0.0.2
- name: baz
  target: mission-control
  isContextScoped: true
  version: v0.0.2
```

Here is the artifacts directory structure that gets generated by running the `tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch darwin_amd64 --os-arch linux_amd64 --plugin-scope-association-file ./cmd/plugin/plugin-scope-association.yaml` command. Assuming we have 2 plugins `foo` with a `global` target and `bar` with a `kubernetes` target available under `./cmd/plugin` directory.

```shell
artifacts
├── darwin
│   └── amd64
│       ├── global
│       │   └── foo
│       │       └── v0.0.2
│       │           └── tanzu-foo-darwin_amd64
│       ├── kubernetes
│       │   └── bar
│       │       └── v0.0.2
│       │           └── tanzu-bar-darwin_amd64
│       └── plugin_manifest.yaml
├── linux
│   └── amd64
│       ├── global
│       │   └── foo
│       │       └── v0.0.2
│       │           └── tanzu-foo-linux_amd64
│       ├── kubernetes
│       │   └── bar
│       │       └── v0.0.2
│       │           └── tanzu-bar-linux_amd64
│       └── plugin_manifest.yaml
├── plugin_manifest.yaml
└── plugin_group_manifest.yaml

```

Note: `tanzu builder plugin build` command expects plugin to met one of following two condition:

* Each plugin advertises the `Target` information as part of the PluginDescriptor.
* Each plugin directory contains a `metadata.yaml` file which describes the name and the target of the plugin.

### Publish-plugins

`tanzu builder plugin build-package` and `tanzu builder plugin publish-package` can be used to build the plugin packages
and publish these package to the remote repository as OCI image.

The `tanzu builder plugin build-package` command takes binary artifacts directory generates as part of `tanzu builder plugin build` command
as an input and generates the plugin package as archive files(`<plugin>.tar.gz`). It also uses a `plugin_manifest.yaml` to parse available plugins from the artifact directory.

Below are the flags available with `tanzu builder plugin build-package` command:

```txt
      --binary-artifacts string    plugin binary artifact directory (default "./artifacts/plugins")
  -h, --help                       help for build-package
      --oci-registry string        local oci-registry to use for generating packages
      --package-artifacts string   plugin package artifacts directory (default "./artifacts/packages")
```

Below are the examples:

```shell
  # Build all plugin packages available under the './artifacts/plugins' directory
  tanzu builder plugin build-package --oci-registry localhost:5001 --binary-artifacts ./artifacts/plugins
```

Once user generate the plugin packages, user can use `tanzu builder plugin publish-package` command to actually publish the generate packages to the remote repository as OCI image.

Below are the flags available with `tanzu builder plugin publish-package` this command:

```txt
      --dry-run                    show commands without publishing plugin packages
  -h, --help                       help for publish-package
      --package-artifacts string   plugin package artifacts directory (default "./artifacts/packages")
      --publisher string           name of the publisher
      --repository string          repository to publish plugins
      --vendor string              name of the vendor
```

Below are the examples:

```shell
  # Publish all plugin packages available under the './artifacts/packages' directory
  tanzu builder plugin publish-package
                --repository gcr.io/repository/cli-plugins
                --package-artifacts ./artifacts/packages
                --vendor vmware
                --publisher tkg

  # Run without publishing plugin packages and just log commands with `--dry-run` flag
    tanzu builder plugin publish-package
                --repository gcr.io/repository/cli-plugins
                --package-artifacts ./artifacts/packages
                --vendor vmware
                --publisher tkg
                --dry-run
```

### Inventory-init

As part of the central repository for plugins implementation, The Tanzu CLI is leveraging an sqlite based inventory database published as an OCI image to discover available plugins. The builder plugin implements `tanzu builder inventory init` command to generate this sqlite based inventory database and publish it as an OCI image.

Below are the flags available with `tanzu builder inventory init` command:

```txt
  -h, --help                                help for init
      --override                            override the inventory database image if already exists
      --plugin-inventory-image-tag string   tag to which plugin inventory image needs to be published (default "latest")
      --repository string                   repository to publish plugin inventory image
```

Below are the examples:

```shell
  # Command initialises inventory database at 'project-stg.registry.vmware.com/test/v1/tanzu-cli/plugins/plugin-inventory:latest'
  tanzu builder inventory init --repository project-stg.registry.vmware.com/test/v1/tanzu-cli/plugins --plugin-inventory-image-tag latest
```

### Inventory-plugin-add

Once the inventory database has been initialized within the repository by publishing it as an OCI image the next thing would be to add plugin entries to the database. The builder plugin implements `tanzu builder inventory plugin add` command to add plugin entries to the sqlite based inventory database.

Below are the flags available with `tanzu builder inventory plugin add` command:

```txt
  -h, --help                                help for add
      --manifest string                     manifest file specifying plugin details that needs to be processed
      --plugin-inventory-image-tag string   tag to which plugin inventory image needs to be published (default "latest")
      --publisher string                    name of the publisher
      --repository string                   repository to publish plugin inventory image
      --validate                            validate whether plugins already exists in the plugin inventory or not
      --vendor string                       name of the vendor
```

Below are the examples:

```shell
  # Add plugin entries to the inventory database based on the specified manifest file
  tanzu builder inventory plugin add --repository project-stg.registry.vmware.com/test/v1/tanzu-cli/plugins --vendor vmware --publisher tkg --manifest ./artifacts/packages/plugin_manifest.yaml
```

### Inventory-plugin-activate-deactivate

Once the plugins are added to the inventory database, there might be a scenario where publisher need to mark the plugin as hidden or in deactive state so that users do not discover these plugins from the central repository. To support this scenario builder plugin implements `tanzu builder inventory plugin activate` and `tanzu builder inventory plugin deactivate` commands.

Below are the flags available with `tanzu builder inventory plugin activate` and `tanzu builder inventory plugin deactivate` commands:

```txt
  -h, --help                                help for activate/deactivate
      --manifest string                     manifest file specifying plugin details that needs to be processed
      --plugin-inventory-image-tag string   tag to which plugin inventory image needs to be published (default "latest")
      --publisher string                    name of the publisher
      --repository string                   repository to publish plugin inventory image
      --vendor string                       name of the vendor
```

Below are the examples:

```shell
  # Activate plugins in the inventory database based on the specified manifest file
  tanzu builder inventory plugin activate --repository localhost:5002/test/v1/tanzu-cli/plugins --vendor vmware --publisher tkg1 --manifest ./artifacts/packages/plugin_manifest.yaml

  # Deactivate plugins in the inventory database based on the specified manifest file
  tanzu builder inventory plugin deactivate --repository localhost:5002/test/v1/tanzu-cli/plugins --vendor vmware --publisher tkg1 --manifest ./artifacts/packages/plugin_manifest.yaml
```

### Inventory-plugin-group-add

Once the plugins are published and added to the inventory database the next thing would be to add/create plugin-groups. The purpose of a plugin-group is to define a product-release-specific set of plugins for users to easily install plugins for the specific product release. To support this use-case the `builder` plugin provides a `tanzu builder inventory plugin-group add` command.

Below are the flags available with the `tanzu builder inventory plugin-group add` command:

```txt
      --deactivate                          mark plugin-group as deactivated
  -h, --help                                help for add
      --manifest string                     manifest file specifying plugin-group details that needs to be processed
      --name string                         name of the plugin-group
      --override                            override the plugin-group if already exists
      --plugin-inventory-image-tag string   tag to which plugin inventory image needs to be published (default "latest")
      --publisher string                    name of the publisher
      --repository string                   repository to publish plugin inventory image
      --vendor string                       name of the vendor
```

Below are some examples:

```shell
  # Add plugin-group entries to the inventory database based on the specified plugin-group manifest file
  tanzu builder inventory plugin-group add --name v1.0.0 --repository project-stg.registry.vmware.com/test/v1/tanzu-cli/plugins --vendor vmware --publisher tkg --manifest ./artifacts/plugins/plugin_group_manifest.yaml
```

Here the `--manifest` flag is used to provide metadata about the plugin-group including which plugins to associate with the plugin-group.

### Inventory-plugin-group-activate-deactivate

In some scenarios, such as preparing for a new product release, a plugin-group may need to be created and added to the inventory database but kept "deactivated".  A "deactivated" plugin-group is not visible to the Tanzu CLI and therefore will not be discovered by users before the official product release, however testers can configure the CLI to discover "deactivated" plugins. To support this scenario the `builder` plugin implements the `tanzu builder inventory plugin-group activate` and `tanzu builder inventory plugin-group deactivate` commands.

Below are the flags available with `tanzu builder inventory plugin-group activate` and `tanzu builder inventory plugin-group deactivate` commands:

```txt
  -h, --help                                help for activate
      --name string                         name of the plugin group
      --plugin-inventory-image-tag string   tag to which plugin inventory image needs to be published (default "latest")
      --publisher string                    name of the publisher
      --repository string                   repository to publish plugin inventory image
      --vendor string                       name of the vendor
```

Below are some examples:

```shell
  # Activate plugin-group in the inventory database
  tanzu builder inventory plugin-group activate --name v1.0.0 --repository localhost:5002/test/v1/tanzu-cli/plugins --vendor vmware --publisher tkg1

  # Dectivate plugin-group in the inventory database
  tanzu builder inventory plugin-group deactivate --name v1.0.0 --repository localhost:5002/test/v1/tanzu-cli/plugins --vendor vmware --publisher tkg1
```
