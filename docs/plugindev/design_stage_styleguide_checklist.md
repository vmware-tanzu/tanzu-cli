# Design-Stage Styleguide Checklist

Follow this checklist to design commands that are consistent with the [CLI Style Guide](style_guide.md).

Follow the [Build-Stage Checklist](build_stage_styleguide_checklist.md) when implementing in code.

VVV

## Command Structure

- [ ] **Do the commands follow the pattern described in the [CLI Style Guide](style_guide.md#designing-commands)?**  (importance: high)
  - noun - verb - resource - flags
- [ ] **Is the number of nested noun layers <= 2?** (importance: medium)
- [ ] **Is the number of required flags <= 2?** (importance: medium)
- [ ] **Is the number of optional flags <= 5?** (importance: medium)

## UI Text / Taxonomy

- [ ] **Do any commands require adding new nouns (resources) to the [existing taxonomy](taxonomy.md)?** (importance: high)
  - Consider raising awareness sooner by [filing an issue](https://github.com/vmware-tanzu/tanzu-plugin-runtime/issues/new?assignees=&labels=shared-taxonomy%2C+kind%2Ffeature&template=feature_request.md) proposing an update to the taxonomy
- [ ] **Are the nouns in each command used in a manner consistent with usage in existing commands?** (importance: high)
- [ ] **Do any commands require adding new verbs (actions) to the [existing taxonomy](taxonomy.md)?**    (importance: medium)
  - Not a problem, but if there is a verb in the [existing taxonomy](taxonomy.md) that could be used, please use it
- [ ] **Are the verbs in each command used in a manner consistent with usage in existing commands?** (importance: medium)
- [ ] **Do any commands require adding new flags to the [existing taxonomy](taxonomy.md)?** (importance: low)
  - Not a problem, but if there is a flag in the [existing taxonomy](taxonomy.md) that could be used, please use it
