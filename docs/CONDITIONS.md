---
layout: default
title: Condition Syntax
parent: Reference
nav_order: 2
permalink: /reference/conditions
---

# Condition Syntax

Conditions are only used to decide whether a template candidate matches. The result is always boolean.

## Supported Forms

| Syntax | Meaning |
|--------|---------|
| `license == "mit"` | Match when the config value equals the given value. |
| `license != "apache-2.0"` | Match when the config value does not equal the given value. |
| `enabled` | Match when the config value is the boolean `true`. |
| `!enabled` | Match when the config value is the boolean `false`. |
| `exists docs.domain` | Match when the key is explicitly present in the repo's `.gitrepoforge` config. |
| `!exists docs.domain` | Match when the key is not explicitly present in the repo's `.gitrepoforge` config. |
| `docs.enabled && exists docs.domain` | Match when both subconditions are true. |
| `docs.enabled || exists docs.domain` | Match when either subcondition is true. |
| `(docs.enabled || preview) && exists docs.domain` | Use parentheses to group expressions. |
| empty | Always matches. |

## Notes

- Bare conditions such as `enabled` and `!enabled` are only valid for boolean config values.
- `exists` and `!exists` check the repo's explicit config before defaults are applied.
- Equality, inequality, and bare boolean conditions still evaluate against resolved config values after defaults are applied.
- `&&` has higher precedence than `||`.
- Parentheses can be used to group compound expressions.
- Template candidates are evaluated in order.
- The first matching candidate is selected.
- If no candidate matches, the output rule fails.

## Example

```yaml
templates:
  - condition: docs.enabled && exists docs.domain
    template: docs/CNAME.tmpl
    evaluate: true
  - absent: true
```
