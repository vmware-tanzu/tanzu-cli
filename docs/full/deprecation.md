# Tanzu CLI deprecation policy

Since the Tanzu CLI offers a varied set of functionality, it is important any breaking changes or removal of functionality follow a clear deprecation policy.

VVV
Add details for deprecation policy

To deprecate a particular piece of CLI functionality,

1. Deprecated CLI elements must display warnings when used.
1. The warning message should include a functional alternative to the
   deprecated command or flag if they exist.
1. The warning message should include the release for when the command/flag
   will be removed.
1. The deprecation should be documented in the Release notes to make users
   aware of the changes.

For instance, should a plugin command `foo` being deprecated be invoked, the
user will be expected to see a message like the following:

```console
Command "foo" is deprecated, will be removed in version as early as "1.5.0". Use "bar" instead.
```

Similar messages will be used to notify the removal of command-line parameters.
