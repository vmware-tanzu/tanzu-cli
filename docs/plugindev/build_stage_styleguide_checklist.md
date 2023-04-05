# Build-Stage Styleguide Checklist

Follow this checklist while implementing commands to ensure consistency with the [CLI Style Guide](style_guide.md).

Working up initial designs? Please use the [Design-Stage Checklist](design_stage_styleguide_checklist.md) instead.

## Importance Ratings

Each item in the checklist below has a high, medium, or low importance rating to help with prioritization.

* **High** = must fix before release
* **Medium** = should fix before release (or commitment to fix in a fast-follow release)
* **Low** = would be nice to fix before release (or commitment to fix in a near-term release)

## Command Structure

* [ ] **Do the commands follow [the design pattern](style_guide.md#designing-commands) described in the CLI Style Guide?** (importance: high)
  * noun - verb - resource - flags
* [ ] **Is the number of nested noun layers <= 2?** (importance: medium)
* [ ] **Is the number of required flags <= 2?** (importance: medium)
* [ ] **Is the number of optional flags <= 5?** (importance: medium)

## UI Text / Taxonomy

* [ ] **Are there new nouns (resources) that should be added to the [existing taxonomy](taxonomy.md)?** (importance: high)
  * If so, please update the [existing taxonomy](taxonomy.md) to include the new nouns
* [ ] **Are there new verbs (actions) that should be added to the [existing taxonomy](taxonomy.md)?** (importance: high)
  * If so, please update the [existing taxonomy](taxonomy.md) to include the new verbs
* [ ] **Do any commands require adding new flags to the [existing taxonomy](taxonomy.md)?** (importance: low)
  * If so, please update the [existing taxonomy](taxonomy.md) to include the new flags

## Design

### Commands

* [ ] **Does interaction deactivate when not using a tty interface?** (importance: high)
* [ ] **Is there helpful behavior if no required arg or flags are specified?** (importance: medium)
  * For example: show help, show error, or provide an interactive prompt

### Execution

* [ ] **Is each command idempotent?** (importance: high)

### Feedback

* [ ] **Does command feedback adhere to [color guidance](style_guide.md#color)?** (importance:  high)
* [ ] **Do dangerous actions have confirmation prompts?** (importance: high)
* [ ] **Can confirmation prompts be skipped by passing in `--yes` or `--force`?** (importance: high)
* [ ] **Do all commands provide [confirmation feedback](style_guide.md#confirmation-feedback) when run?** (importance: medium)
* [ ] **Do all commands provide [completion feedback](style_guide.md#feedback-when-a-process-completes)?** (importance: medium)
* [ ] **Is verbosity configurable?** (importance: medium)

### Outputs

* [ ] **Is the default output format human friendly?** (importance: high)
  * For example tables, key/value pairs, etc... (not JSON/YAML, etc...)
* [ ] **Do all [date/times](style_guide.md#time-format) follow the ISO 8601 standard?** (importance: high)
  * For example: `2021-03-02T15:43:12.41-0700`
* [ ] **Do [table styles](style_guide.md#tables) align with the styledguide?** (importance: medium)
* [ ] **Does [key:value pair styling](style_guide.md#keyvalue-pairs) align with the style guide?** (importance: medium)

### Help text

* [ ] **Is there help text available for every command?** (importance: high)
* [ ] **For complex commands, are there examples?** (importance: medium)
* [ ] **Does the help text include URL for docs?** (importance: low)

### Errors/warnings

* [ ] **If there are experimental commands, do they include a warning notice in the confirmation feedback when run?** (importance: high)
* [ ] **Are there tailored error messages for known/common error cases?** (importance: medium)
