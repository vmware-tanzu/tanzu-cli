# Context Recommended Plugin Installation

## Abstract

The Tanzu CLI is an amalgamation of all the Tanzu infrastructure elements under
one unified core CLI experience. The core CLI supports a plugin model where the
developers of different Tanzu services (bundled or SaaS) can distribute plugins
that target the functionalities of the services they own. When users switch between
different services via the CLI context, we want to surface only the relevant
plugins for the given context for a crisp user experience.

When a user is working with multiple instances of a product, we want to
automatically select the right set of plugins and plugin versions for use
based on the active context the user is connected to.

The goals of the Context-Recommended Plugin Installation feature are:

As a plugin developer, I want to,

- Recommend relevant plugins and their versions that might be needed by the user once the user creates a context.

As a user, I want to,

- Install the recommended version of all plugins for the active context during context create
- Install the recommended version of any missing plugin for the active context via a `tanzu plugin sync` command
- Upgrade all installed plugins to newer versions via a `tanzu plugin sync` command, if the installed versions are not supported anymore
- Avoid re-downloading a plugin version if it was already installed previously (e.g., if the same version of the `package` plugin is provided by two management clusters, do not re-download that plugin)

## Plugin Discovery and Distribution

Discovery is the interface to fetch the list of available plugins, their
supported versions, and how to download them. The Tanzu CLI has a
plugin discovery source configured by default which returns the list
of all available plugins. In the future, this can be made configurable
to allow more than one discovery source.

Distribution is the interface to download a plugin binary for a given OS
and architecture combination. A discovery source provides details about
the distribution regarding where to fetch the plugin binary.

Plugin availability is solely dependent on the configured discovery sources in the
tanzu configuration file. Each discovery source points to a plugin repository
which can contain one or more plugins.

## Context-Recommended Plugins

As mentioned above in the abstract section, there might be a scenario when a user
is working with multiple contexts at a time and wants to automatically select the
right set of plugins and plugin versions based on the currently active context.
The context-recommended plugin implementation is useful in this scenario.

When the CLI user creates a new context for the Tanzu CLI using the
`tanzu context create` command, the CLI adds a context in the tanzu configuration file
and marks the newly created context as an active context for the specified target.

Now, this newly created context can also recommend the list of plugins and their versions
that are needed to be installed on the user's machine to interact with the created context.
The Tanzu CLI automatically detects the list of recommended plugins and their versions and
installs them as part of the `tanzu context create` or `tanzu context use` commands. Below
is the workflow of context-recommended plugin installation:

- The user runs the `tanzu context create` or `tanzu context use` commands to create a new context or switch active context
- The Tanzu CLI gets the list of recommended plugins and their version from the created context
- The Tanzu CLI finds the plugins and their metadata in the available list of plugins generated from the configured discovery sources
- The Tanzu CLI fetches the plugin binary for these plugins from the specified location and installs the plugin

If the user switches the context to a different context using the `tanzu context use` command,
the CLI will automatically install/update the recommended plugins based on the new context.

Note: Users should understand that these plugins (installed based on a recommendation from a context) are
installed as normal plugins and will not be automatically deleted when a user deletes the context or switches
the context to a different context. Commands associated with those plugins will remain available
to be used but will likely throw an error if those plugins do not work with the active context.

## Plugin Recommendations from a Context

This section provides more details on how a context can provide
recommended plugins to automatically install when a user creates or activates the context.

### When the context is of type Kubernetes

When the context is of type kubernetes, the Tanzu CLI uses a kubernetes discovery to fetch the
list of recommended plugins and their versions. Using the kubernetes discovery implementation
the Tanzu CLI queries the `CLIPlugin` resources available on the kubernetes cluster.

For example, if the user is expected to use the plugins `cluster:v1.0.0` and `feature:v1.2.0`
when talking to the kubernetes cluster `test-cluster` then the cluster should have the below
`CLIPlugin` resources defined:

```yaml
apiVersion: cli.tanzu.vmware.com/v1alpha1
kind: CLIPlugin
metadata:
  name: cluster
spec:
  recommendedVersion: v1.0.0
  description: Kubernetes cluster operations
```

```yaml
apiVersion: cli.tanzu.vmware.com/v1alpha1
kind: CLIPlugin
metadata:
  name: feature
spec:
  recommendedVersion: v1.2.0
  description: Feature plugin operations
```

Note: Starting with `v1.1.0` version of Tanzu CLI, it supports providing shortened
version (`vMAJOR` or `vMAJOR.MINOR`) as part of the recommendedVersion.
Using shortened version as above, will install the latest available minor.patch of
`vMAJOR` and latest patch version of `vMAJOR.MINOR` respectively.

For Tanzu CLI to read these `CLIPlugin` resources available on the kubernetes
cluster `get` and `list` RBAC permission needs to be given to all the users.
To do that please configure below RBAC rules on your kubernetes cluster.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: read-cli-plugins
rules:
- apiGroups: ["cli.tanzu.vmware.com"]
  resources: ["cliplugins"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
 name: read-cli-plugins-rolebinding
subjects:
- kind: Group
  name: system:authenticated
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: read-cli-plugins
  apiGroup: rbac.authorization.k8s.io
```

### When the context is of type Mission-Control

When the context is of type mission control, the Tanzu CLI uses a REST discovery to fetch the
list of recommended plugins and their versions. Using the REST discovery implementation
the Tanzu CLI queries the `<server-url>/v1alpha1/system/binaries/plugins` REST API that
should return a list of `CLIPlugin` information.
