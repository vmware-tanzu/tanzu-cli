# Test

Test the CLI plugins. If a CLI plugin X has a test plugin test-X installed
alongside it, the latter will be detected and invocable with this plugin.

## Usage

`tanzu test fetch` will fetch all the test plugins for the currently installed
plugins to be installed alongside them.

Note: this command will only support installing test plugins from a local
artifacts directory, e.g.

`tanzu test fetch --local ./artifacts/darwin/amd64/cli`

`tanzu test plugin <plugin-name>` will test a specific plugin by invoking the
companion test plugin.
