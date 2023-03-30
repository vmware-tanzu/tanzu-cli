# Transitioning from tanzu-framework to tanzu-cli/tanzu-plugin-runtime

This section covers information relevant to plugin developers looking to
transition their Tanzu CLI plugins developed using the legacy Tanzu CLI
codebase in
[tanzu-framework](https://github.com/vmware-tanzu/tanzu-framework/tree/main/cli)
to make use of [tanzu-plugin-runtime](https://github.com/vmware-tanzu/tanzu-plugin-runtime)
project and this repository.

## Updating the plugin code and dependencies

Include [plugin-tooling.mk](https://github.com/vmware-tanzu/tanzu-cli/blob/main/plugin-tooling.mk) in your make file. It will provide make targets that are useful during the plugin build, test, and publishing process

Using tanzu-plugin-runtime as a `go.mod` dependency
github.com/vmware-tanzu/tanzu-plugin-runtime v0.90.0-alpha.0

VVV: let's give more concrete steps on how we want developers to update this dep (now, then later)

(until we have the v0.90.0-alpha.0 release out developers can point to this tag: v0.0.2-0.20230324033521-a110e57a60b9)

Updating the import references to use `tanzu-plugin-runtime`

1. The main change is to update the import references: "github.com/vmware-tanzu/tanzu-framework/cli/runtime" => "github.com/vmware-tanzu/tanzu-plugin-runtime"
1. Additional required changes are based on the following things:
    - PluginDescriptor has moved from "github.com/vmware-tanzu/tanzu-framework/cli/runtime/apis/cli/v1alpha1" => "github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"
"github.com/vmware-tanzu/tanzu-framework/cli/runtime/buildinfo" => "github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo"
    - Plugins are required to provide Target information with the PluginDescriptor.

Here is the [sample change](https://github.com/anujc25/tanzu-framework/commit/cdd1239b863ef3e0e00ad5868b17966a28cacfa0)
which includes the updates to the `isolated-cluster` and `feature` plugins to use the new tanzu-plugin-runtime and use new tooling to build plugins.
