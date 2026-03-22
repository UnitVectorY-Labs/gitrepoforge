# Per-Repo Config

Each managed repository can opt in with a `.gitrepoforge` file at its root.

## Example

```yaml
name: my-repo
default_branch: main
config:
  license: mit
  enable_license: true
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Must match the repository folder name. |
| `default_branch` | yes | Repository metadata made available to templates and root-level Git placeholders. |
| `config` | no | Values that match keys defined in the config repo's `config/` folder. Missing keys may be filled from definition defaults. |

## Validation Rules

- `name` must match the repository folder name.
- `default_branch` must be present.
- Required config keys must be present.
- Missing keys use the definition's `default` value when one is provided.
- Reserved top-level field names such as `name` and `default_branch` cannot appear inside `config:`.
- The repo config does not contain a top-level `git` section; Git automation is configured only in `.gitrepoforge-config`.
- Unknown config keys are rejected.
- Values must match the declared type.
- String values with `enum` must use one of the allowed values.
