# The Tanzu CLI Plugin Shared Taxonomy

To encourage consistency among CLI commands across all plugins. The Tanzu CLI
project strives to main a taxonomy of nouns, verbs, and flag names used through
the commands implemented thus far.

Being a resource for plugin developers, the Shared Taxonomy, in the form of a
word list YAML file, is maintained in the tanzu-plugin-runtime repository, and
evolves (almost always cumulatively) with each new version of the runtime
release.

[LINK to Taxomony YAML](https://github.com/vmware-tanzu/tanzu-plugin-runtime/blob/main/plugin/lint/cli-wordlist.yml)

It is highly recommended that the introduction of any new noun, verb or flag in a
new plugin command be accompanied by a PR to update the YAML (the label
`shared-taxonomy` should be applied to the issue and pull request).

Every plugin usage can be validated against the taxonomy associated with the version of the runtime the plugin is compiled with. To see taxonomy violations and identify potential inclusions into the taxonomy, run:

```shell
tanzu <pluginname> lint
```
