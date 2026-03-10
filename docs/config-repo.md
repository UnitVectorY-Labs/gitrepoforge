# Config Repo

The config repo contains the shared definitions and file rules used across repositories.

## Layout

```text
config-repo/
├── config/
│   └── license.yaml
├── outputs/
│   └── LICENSE.gitrepoforge
└── templates/
    └── licenses/
        ├── apache-2.0.tmpl
        └── mit.tmpl
```

## `config/`

Each file defines one allowed config key. The filename, without `.yaml`, is the key name.

**`config/license.yaml`**

```yaml
type: string
required: true
enum:
  - mit
  - apache-2.0
description: License template to apply.
```

Supported types are `string`, `boolean`, `number`, and `list`.

## `outputs/`

Each output rule maps to one target file. The relative path under `outputs/`, without the `.gitrepoforge` suffix, becomes the managed file path in the repo.

**`outputs/LICENSE.gitrepoforge`**

```yaml
templates:
  - condition: license == "mit"
    template: licenses/mit.tmpl
  - condition: license == "apache-2.0"
    template: licenses/apache-2.0.tmpl
```

### Output Fields

| Field | Required | Description |
|-------|----------|-------------|
| `mode` | no | `create` or `delete`. Defaults to `create`. |
| `templates` | yes for `create` | Ordered list of template candidates. First match wins. |

### Template Candidate Fields

| Field | Required | Description |
|-------|----------|-------------|
| `condition` | no | Boolean selector for the template. Empty means always matches. |
| `template` | yes | Path to a file under `templates/`. |

Use `mode: delete` when a file should be removed instead of rendered.

## `templates/`

The `templates/` folder stores the file content referenced by output rules. See [Template Files](/Users/jaredhatfield/github/gitrepoforge/docs/templates.md) for rendering details.
