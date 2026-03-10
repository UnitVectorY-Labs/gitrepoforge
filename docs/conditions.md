# Condition Syntax

Conditions are only used to decide whether a template candidate matches. The result is always boolean.

## Supported Forms

| Syntax | Meaning |
|--------|---------|
| `license == "mit"` | Match when the config value equals the given value. |
| `license != "apache-2.0"` | Match when the config value does not equal the given value. |
| `enabled` | Match when the config value is the boolean `true`. |
| `!enabled` | Match when the config value is the boolean `false`. |
| empty | Always matches. |

## Notes

- Bare conditions such as `enabled` and `!enabled` are only valid for boolean config values.
- Template candidates are evaluated in order.
- The first matching candidate is selected.
- If no candidate matches, the output rule fails.

## Example

```yaml
templates:
  - condition: license == "mit"
    template: licenses/mit.tmpl
  - condition: license == "apache-2.0"
    template: licenses/apache-2.0.tmpl
```
