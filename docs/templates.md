# Template Files

Templates live in the config repo under `templates/`. Output rules do not store inline content anymore; they point at template files in that folder.

## Data Available To Templates

Each template is rendered with:

| Field | Type | Description |
|-------|------|-------------|
| `.Name` | `string` | Repository name from `.gitrepoforge`. |
| `.Config` | `map[string]interface{}` | Repo config values from `.gitrepoforge`. |

## Helper Function

| Function | Description |
|----------|-------------|
| `getConfig` | Looks up a config value by key. |

## Example

**`templates/licenses/mit.tmpl`**

```text
MIT License

Copyright (c) {{.Name}}
```

**`outputs/LICENSE.gitrepoforge`**

```yaml
templates:
  - condition: license == "mit"
    template: licenses/mit.tmpl
```

The `template` value is always a path relative to `templates/`.
