# Test Central Repository

## Using the test Central Repo

From the root directory of the tanzu-cli repo, run `make start-test-central-repo`.  This will start an OCI registry
containing two test central repositories:

1. a small repo with 4 plugins which can be simpler to test with: localhost:9876/tanzu-cli/plugins/central:small
1. a large one with 99 plugins: localhost:9876/tanzu-cli/plugins/central:large

The steps to follow to use this test central repo are:

1. start the test repo
1. enable the central repository feature flag (this is a temporary flag which will be removed eventually)
1. configure the plugin source for the test central repo
1. allow the use of a local registry with the `ALLOWED_REGISTRY` variable

```bash
cd tanzu-cli
make build
make start-test-central-repo
tz config set features.global.central-repository true
tz plugin source add -n default -t oci -u localhost:9876/tanzu-cli/plugins/central:small || true
tz plugin source update default -t oci -u localhost:9876/tanzu-cli/plugins/central:small
export ALLOWED_REGISTRY=localhost:9876

tz plugin list
tz plugin install twotargets1 --target tmc
```

To use the large test central repo instead:

```bash
tz plugin source update default -t oci -u localhost:9876/tanzu-cli/plugins/central:large
```

To stop the central repos: `make stop-test-central-repo`.

Note that the registry is pre-configured through the existing `hack/central-repo/registry-content` directory.

Limitations:

1. Only certain plugins can be installed.  This avoids having to push binaries for all plugins, for efficiency.
1. The plugins that can be installed are `twotargets1` for two targets, `twotargets2` for two targets, and `plugin0` and `plugin1`

## Generate the test Central Repo

This should only be done if a new version of the registry should be generated.
Normally, the content of the repo is persisted on disk under `hack/central-repo/registry-content`
which avoids having to regenerate the repo.  When using the Makefile target `make start-test-central-repo`
the directory `registry-content` is extracted from the `registry-content.bz2` tarball.
A tarball is used to dramatically reduce the size saved in git.

If it is necessary to re-generate a new test central repo, it took around 1 minute on a Mac M1.
The procedure follows:

```bash
cd tanzu-cli
make stop-test-central-repo
cd hack/central-repo
\rm -rf registry-content registry-content.bz2
./generate-central.sh
tar cjf registry-content.bz2 registry-content
git add registry-content.bz2
git commit -m "Regenerated the test central repos"
```
