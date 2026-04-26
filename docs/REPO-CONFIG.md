---
layout: default
title: Per-Repo Config
parent: Configuration
nav_order: 2
permalink: /configuration/per-repo
---

# Per-Repo Config

Each managed repository can opt in with a `.gitrepoforge` file at its root.

## Example

```yaml
name: my-repo
default_branch: main
manifest: .managedfiles
config:
  license: mit
  docs:
    enabled: true
    domain: docs.example.com
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Must match the repository folder name. |
| `default_branch` | yes | Repository metadata made available to templates and root-level Git placeholders. |
| `manifest` | no | Relative path for the generated managed files manifest in this repo. When omitted, gitrepoforge uses the workspace `manifest` value if set, otherwise `.managedfiles`. |
| `config` | no | Values that match keys defined in the config repo's `config/` folder. Missing keys may be filled from definition defaults. |

## Validation Rules

- `name` must match the repository folder name.
- `default_branch` must be present.
- `manifest`, when set, must be a relative path that stays within the repository.
- Required config keys must be present.
- Missing keys use the definition's `default` value when one is provided.
- Reserved top-level field names such as `name`, `default_branch`, and `manifest` cannot appear inside `config:`.
- The repo config does not contain a top-level `git` section; Git automation is configured only in `.gitrepoforge-config`.
- Unknown config keys are rejected.
- Values must match the declared type.
- Object values must be YAML mappings, and nested keys are validated against the matching `config/<key>/` definitions from the config repo.
- String values with `enum` must use one of the allowed values.
