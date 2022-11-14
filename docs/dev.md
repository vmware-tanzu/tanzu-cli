# Development

## Building

default target to update dependencies, build test and lint the CLI:

```sh
make all
```

NOTE: Until tanzu-plugin-runtime is public, to avoid checksum issues when accessing
said dependency, run this prior to build:

```sh
go env -w GOPRIVATE=github.com/vmware-tanzu/tanzu-plugin-runtime
```

## Source Code Changes

## Source Code Structure

### Tests
