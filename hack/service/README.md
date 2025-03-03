# Test CLI-Svc

This mock allows to run an nginx docker image which serves the same endpoints as
the ones the real CLI-Svc serves.  It allows to test the CLI without using the
real CLI-Svc.

The endpoints being mocked use SSL and are:

- localhost:9443/cli/v1/install
- localhost:9443/cli/v1/plugin/discovery
- localhost:9443/cli/v1/binary

## Using the test CLI-Svc

From the root directory of the tanzu-cli repo, run `make start-test-cli-service`.
This will start the test CLI-Svc as a configured nginx docker image.

To access the endpoints manually, e.g.,:

```console
# NOTE: the trailing / is essential
curl https://localhost:9443/cli/v1/plugin/discovery/ --cacert hack/central-repo/certs/localhost.crt
# or
curl https://localhost:9443/cli/v1/plugin/discovery/ -k
```

## Testing plugin discovery

If testing plugin discovery (localhost:9443/cli/v1/plugin/discovery/), the
test CLI-Svc will randomly serve different discovery data which is configured in
`hack/service/cli-service.conf`.

To tell the CLI to use the test CLI-Svc we must execute:

```console
export TANZU_CLI_PLUGIN_DISCOVERY_HOST_FOR_TANZU_CONTEXT=http://localhost:9443
```

To allow testing using different central repositories the endpoint serves some
discoveries using both:

- projects.packages.broadcom.com/tanzu_cli/plugins/plugin-inventory:latest
- localhost:9876/tanzu-cli/plugins/central:small

Therefore, it is required to also start the test central repo by doing:

```console
export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=localhost:9876/tanzu-cli/plugins/central:small
make start-test-central-repo

# If the certificates should be handled automatically by the CLI itself
# we should remove their configuration done by the make file so that the
# testing is more appropriate.  This would be done by doing:
tanzu config cert delete localhost:9876
```
