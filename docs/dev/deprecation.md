# Tanzu CLI deprecation policy

## How to deprecate CLI functionality

See [file](https://github.com/vmware-tanzu/tanzu-plugin-runtime/blob/main/command/deprecation.go)
for the helper functions that can be useful when adhering to the [deprecation policy](../full/deprecation.md) laid out for deprecating any aspect of the CLI command.

Example usage to deprecate a command `foo`:

```golang
import "github.com/vmware-tanzu/tanzu-plugin-runtime/command"
//...
command.DeprecateCommand(fooCmd, "1.5.0", "bar")
```

Running the `foo` command will display the following:

```console
Command "foo" is deprecated, will be removed in version as early as "1.5.0". Use "bar" instead.
```

Similarly, to deprecate a flag --use-grouping in a `describe` command:

```golang
import "github.com/vmware-tanzu/tanzu-plugin-runtime/command"
//...
command.DeprecateFlagWithAlternative(describeCmd, "use-grouping", "1.6.0", "--show-group-members")
```
