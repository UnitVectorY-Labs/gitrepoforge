---
layout: default
title: Config Repository
parent: Configuration
nav_order: 3
permalink: /configuration/config-repo
---

# Config Repository

The config repo contains the shared definitions and file rules used across repositories.

## Layout

```text
config-repo/
├── config/
│   ├── docs.yaml
│   ├── docs/
│   │   ├── domain.yaml
│   │   └── enabled.yaml
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

Object definitions can also declare nested attributes from a same-named folder.

**`config/docs.yaml`**

```yaml
type: object
required: true
description: Documentation settings.
```

**`config/docs/enabled.yaml`**

```yaml
type: boolean
default: true
description: Whether documentation hosting is enabled.
```

**`config/docs/domain.yaml`**

```yaml
type: string
required: true
description: Canonical docs hostname.
```

### Config Definition Fields

| Field | Required | Description |
|-------|----------|-------------|
| `type` | yes | Supported values are `string`, `boolean`, `number`, `list`, and `object`. |
| `required` | no | If true, the repo must provide the key unless a `default` is defined. |
| `default` | no | Typed default value used when the repo omits the key. |
| `enum` | no | Allowed values for `string` definitions. |
| `pattern` | no | Regex with named capture groups for `string` definitions. Used for input validation and to extract named parts accessible in templates via the `capture` function. |
| `description` | no | Human-readable description. |

### Type Notes

- `string` accepts YAML strings and may also use `enum` and/or `pattern`.
- `boolean` accepts `true` or `false`.
- `number` accepts numeric YAML values.
- `list` accepts YAML sequences.
- `object` accepts YAML mappings. Nested attributes are loaded from `config/<key>/` using the same per-file format as top-level definitions.

### Pattern Matching

The `pattern` field accepts a regular expression with named capture groups using the `(?P<name>...)` syntax. It is only supported for `string` definitions.

When a pattern is defined:

- The value provided by the repo (or the default) must match the pattern. A validation error is produced if it does not.
- Named capture groups are extracted from the matched value and made available to templates through the `capture` function.

The pattern must contain at least one named capture group. If a `default` is also set, the default value must match the pattern.

**`config/goversion.yaml`**

```yaml
type: string
required: false
default: "1.26.1"
pattern: "^(?P<major>\\d+)\\.(?P<minor>\\d+)\\.(?P<patch>\\d+)$"
description: "The version of Go."
```

With the value `1.26.1`, the named groups resolve to:

| Group | Value |
|-------|-------|
| `major` | `1` |
| `minor` | `26` |
| `patch` | `1` |

Templates can then use `capture` to access individual groups. See [Template Files](/reference/templates#capture-function) for usage details.

`pattern` and `enum` may be combined. The value must satisfy both constraints.

### Reserved Config Keys

These names are reserved because they already exist as top-level fields in `.gitrepoforge`:

- `name`
- `default_branch`
- `config`

They cannot be declared under `config/` and must not appear inside the repo's `config:` map.

## `outputs/`

Each output rule maps to one target file. The relative path under `outputs/`, without the `.gitrepoforge` suffix, becomes the managed file path in the repo.
Every file under `outputs/` must end with `.gitrepoforge`; unexpected filenames are treated as config errors so typos are not silently ignored.

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
| `condition` | no | Boolean selector for the candidate. Empty means always matches, which is useful for a fallthrough entry. Supported forms are documented in [Condition Syntax](/reference/conditions). |
| `template` | yes unless `absent` is true | Path to a file under `templates/`. |
| `evaluate` | no | If true, render the template file with template data. If false or omitted, copy the file verbatim. |
| `template_mode` | no | Controls how template delimiters are recognized when `evaluate` is true. `DOUBLE_BRACKET` is the default. `DOUBLE_BRACKET_STRICT` only recognizes {% raw %}`{{`{% endraw %} when it is at the start of the file or preceded by whitespace. |
| `absent` | no | If true, the selected result is that the target file must not exist. |

### Selection Rules

- Candidates are checked in order.
- The first candidate whose `condition` matches is selected.
- A candidate with no `condition` is unconditional and usually belongs at the end as a fallback.
- `absent: true` is the fallback form for "the file should not exist".

### Common Patterns

Use a single unconditional candidate when the same template should always apply:

```yaml
templates:
  - template: .github/workflows/add-to-project.yml
```

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

Use strict delimiter matching when the file also contains other {% raw %}`{{ ... }}`{% endraw %}-style syntax, such as GitHub Actions expressions:

```yaml
templates:
  - template: .github/workflows/ci.yml.tmpl
    evaluate: true
    template_mode: DOUBLE_BRACKET_STRICT
```

With `DOUBLE_BRACKET_STRICT`, {% raw %}`${{ runner.os }}`{% endraw %} remains literal because the {% raw %}`{{`{% endraw %} is preceded by `$`, while template directives like {% raw %}`{{- if eq .Config.codecov true }}`{% endraw %} still evaluate when they begin on their own lines.

The evaluated template can branch on other config values internally:

{% raw %}
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
{% endraw %}

Nested config values can also be referenced in conditions with dotted keys:

```yaml
templates:
  - condition: docs.enabled
    template: docs/CNAME.tmpl
    evaluate: true
  - absent: true
```

When a value must be emitted as a quoted scalar, use `quote_double` or `quote_single` instead of adding literal quotes around the interpolation:

{% raw %}
```yaml
description: {{ .Config.description | quote_double }}
summary: {{ .Config.summary | quote_single }}
go-version: {{ .Config.versions.go | quote_double }}
```
{% endraw %}

Use `exists` when selection should depend on whether a repo explicitly set a value rather than whether a default resolved value is available:

```yaml
templates:
  - condition: exists docs.domain
    template: docs/CNAME.tmpl
    evaluate: true
  - absent: true
```

`exists docs.domain` only matches when `config.docs.domain` is present in the repo's `.gitrepoforge` file. It does not match a value that only came from a schema default.

Conditions can also be combined with `&&` and `||`:

```yaml
templates:
  - condition: docs.enabled && exists docs.domain
    template: docs/CNAME.tmpl
    evaluate: true
  - absent: true
```

Use parentheses when grouping is needed, for example `(docs.enabled || preview_docs) && exists docs.domain`.

`mode: delete` is still available when a file is always forbidden and does not need conditional selection.

### Section-Based Patterns

Templates can use section directives to manage only part of a file instead of replacing the entire file. The directives are placed in the template file itself. See [Template Files](/reference/templates#section-directives) for the full directive reference.

Manage a README header while letting users write the body:

**`outputs/README.md.gitrepoforge`**

```yaml
templates:
  - condition: readme
    template: README.md.tmpl
```

**`templates/README.md.tmpl`**

{% raw %}
```text
{{ section start=start_of_file end=contains("<!-- END MANAGED -->") }}
# My Project
<!-- END MANAGED -->
{{ end }}
```
{% endraw %}

Ensure a file exists without managing its content. Useful for files like `go.sum` that should be present but are maintained by other tools:

**`outputs/go.sum.gitrepoforge`**

```yaml
templates:
  - condition: manage_gosum
    template: go.sum.tmpl
```

**`templates/go.sum.tmpl`**

{% raw %}
```text
{{ bootstrap }}
{{ end }}
```
{% endraw %}

Manage both a header and footer while preserving user content in between:

**`templates/README.md.tmpl`**

{% raw %}
```text
{{ section start=start_of_file end=content("<!-- END HEADER -->") }}
# Managed Header
<!-- END HEADER -->
{{ end }}
{{ section start=contains("<!-- START FOOTER -->") end=end_of_file }}
<!-- START FOOTER -->
Managed Footer Content
{{ end }}
```
{% endraw %}

Provide default body content on first creation, then manage only the header afterwards:

**`templates/README.md.tmpl`**

{% raw %}
```text
{{ section start=start_of_file end=contains("<!-- END MANAGED -->") }}
# Managed Header
<!-- END MANAGED -->
{{ end }}
{{ bootstrap }}
Default body content goes here.
{{ end }}
```
{% endraw %}

## `templates/`

The `templates/` folder stores the file content referenced by output rules. See [Template Files](/reference/templates) for rendering details.
