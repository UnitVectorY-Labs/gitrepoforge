# Per-Repo Config

Each managed repository can opt in with a `.gitrepoforge` file at its root.

## Example

```yaml
name: my-repo
default_branch: main
config:
  license: mit
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Must match the repository folder name. |
| `default_branch` | yes | Branch used as the checkout/PR base for this specific repository. |
| `config` | no | Values that match keys defined in the config repo's `config/` folder. |

## Validation Rules

- `name` must match the repository folder name.
- `default_branch` must be present.
- Required config keys must be present.
- Unknown config keys are rejected.
- Values must match the declared type.
- String values with `enum` must use one of the allowed values.
