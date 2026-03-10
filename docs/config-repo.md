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
default: mit
enum:
  - mit
  - apache-2.0
description: License template to apply.
```

**`config/enable_license.yaml`**

```yaml
type: boolean
default: true
description: Whether a LICENSE file should be managed.
```

### Config Definition Fields

| Field | Required | Description |
|-------|----------|-------------|
| `type` | yes | Supported values are `string`, `boolean`, `number`, and `list`. |
| `required` | no | If true, the repo must provide the key unless a `default` is defined. |
| `default` | no | Typed default value used when the repo omits the key. |
| `enum` | no | Allowed values for `string` definitions. |
| `description` | no | Human-readable description. |

### Type Notes

- `string` accepts YAML strings and may also use `enum`.
- `boolean` accepts `true` or `false`.
- `number` accepts numeric YAML values.
- `list` accepts YAML sequences.

### Reserved Config Keys

These names are reserved because they already exist as top-level fields in `.gitrepoforge`:

- `name`
- `default_branch`
- `config`

They cannot be declared under `config/` and must not appear inside the repo's `config:` map.

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

**`outputs/justfile.gitrepoforge`**

```yaml
templates:
  - condition: justfile
    template: justfile.tmpl
    evaluate: true
  - absent: true
```

### Output Fields

| Field | Required | Description |
|-------|----------|-------------|
| `mode` | no | `create` or `delete`. Defaults to `create`. |
| `templates` | yes for `create` | Ordered list of candidates. The first matching candidate is selected and evaluation stops. |

### Candidate Fields

| Field | Required | Description |
|-------|----------|-------------|
| `condition` | no | Boolean selector for the candidate. Empty means always matches, which is useful for a fallthrough entry. |
| `template` | yes unless `absent` is true | Path to a file under `templates/`. |
| `evaluate` | no | If true, render the template file with template data. If false or omitted, copy the file verbatim. |
| `absent` | no | If true, the selected result is that the target file must not exist. |

### Selection Rules

- Candidates are checked in order.
- The first candidate whose `condition` matches is selected.
- A candidate with no `condition` is unconditional and usually belongs at the end as a fallback.
- `absent: true` is the fallback form for "the file should not exist".

### Common Patterns

Use verbatim copy for static assets such as license files:

```yaml
templates:
  - condition: license == "MIT"
    template: LICENSE/MIT
  - condition: license == "Apache-2.0"
    template: LICENSE/Apache-2.0
```

Use evaluation for generated files:

```yaml
templates:
  - condition: justfile
    template: justfile.tmpl
    evaluate: true
  - absent: true
```

The evaluated template can branch on other config values internally:

```text
# Commands for {{.Name}}
default:
  @just --list

{{- if eq .Config.language "go" }}
# Build {{.Name}} with Go
build:
  go build ./...
{{- end }}
{{- if eq .Config.language "java" }}
# Build {{.Name}} with Maven
build:
  mvn package
{{- end }}
```

`mode: delete` is still available when a file is always forbidden and does not need conditional selection.

## `templates/`

The `templates/` folder stores the file content referenced by output rules. See [Template Files](/Users/jaredhatfield/github/gitrepoforge/docs/templates.md) for rendering details.
