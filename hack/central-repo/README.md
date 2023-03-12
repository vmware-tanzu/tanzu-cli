# Test Central Repository

## Using the test Central Repo

From the root directory of the tanzu-cli repo, run `make start-test-central-repo`.  This will start an OCI registry
containing two test central repositories:

1. a small repo with a few plugins with names that match real product plugins.  Such a repo can be simpler to test with.  The OCI image for this repo is: localhost:9876/tanzu-cli/plugins/central:small
1. a large repo with the same plugins as the small repo plus extra stub plugins making the total number of plugins 100. This repo aims to mimic a full Central Repository. The OCI image for this repo is: localhost:9876/tanzu-cli/plugins/central:large

Limitations:

For efficiency in storage and generation, only certain plugins have a binary in the test repos
and therefore only those can be installed.
Plugins named `stubXY` cannot be installed.
Also, only versions `v0.0.1` and `v9.9.9` for the other plugins can be installed.

The steps to follow to use the test central repo are:

1. Start the test repo with `make start-test-central-repo`.
1. Enable the central repository using the temporary feature flag: `tz config set features.global.central-repository true`
1. Configure the plugin source for the test central repo: `tz plugin source add -n default -t oci -u localhost:9876/tanzu-cli/plugins/central:small`
1. Allow the use of a local registry: `export ALLOWED_REGISTRY=localhost:9876`

Here are the exact commands:

```bash
cd tanzu-cli
make build
make start-test-central-repo
tz config set features.global.central-repository true
tz plugin source add -n default -t oci -u localhost:9876/tanzu-cli/plugins/central:small
export ALLOWED_REGISTRY=localhost:9876

tz plugin search
tz plugin install cluster --target tmc
```

It is possible to test installing plugins that are recommended from a TMC context:

```bash
tz context create --name tmc-unstable --endpoint unstable.tmc-dev.cloud.vmware.com --staging
tz plugin sync
```

The above `tz plugin sync` will install the plugins versions recommended by the TMC context (`v0.0.1`), but will install
them from the test Central Repository.

To use the large test central repo instead:

```bash
tz plugin source update default -t oci -u localhost:9876/tanzu-cli/plugins/central:large
```

To stop the central repos: `make stop-test-central-repo`.

Note that the registry is pre-configured through the existing `hack/central-repo/registry-content` directory.

## Generating the test Central Repo

This should only be done if a new version of the registry should be generated.
Normally, the content of the repo is persisted on disk under `hack/central-repo/registry-content`
which avoids having to regenerate the repo.  When using the Makefile target `make start-test-central-repo`
the directory `registry-content` is extracted from the `registry-content.tgz` tarball.
A tarball is used to dramatically reduce the size saved in git.

If it is necessary to re-generate a new test central repo, it took around 4 minutes on a Mac M1.
The procedure follows:

```bash
cd tanzu-cli
make stop-test-central-repo
cd hack/central-repo
\rm -rf registry-content registry-content.tgz
./generate-central.sh
tar czf registry-content.tgz registry-content
git add registry-content.tgz
git commit -m "Regenerated the test central repos"
```
