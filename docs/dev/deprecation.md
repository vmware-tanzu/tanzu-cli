# Tanzu CLI deprecation

## How to deprecate CLI functionality

To deprecate a particular piece of CLI functionality,

1. Deprecated CLI elements must display warnings when used.
1. The warning message should include a functional alternative to the
   deprecated command or flag if they exist.
1. The warning message should include the release for when the command/flag
   will be removed.
1. The deprecation should be documented in the Release notes to make users
   aware of the changes.

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

## Tanzu CLI deprecation policy

Any deprecation must adhere to the [deprecation policy](../full/policy.md#tanzu-cli-deprecation) laid out for deprecating any aspect of the CLI command.
