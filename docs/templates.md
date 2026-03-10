# Template Files

Templates live in the config repo under `templates/`. Output rules point at files in that folder and choose whether to copy them verbatim or evaluate them as templates.

## Data Available To Templates

When `evaluate: true` is set on a candidate, the file is rendered with:

| Field | Type | Description |
|-------|------|-------------|
| `.Name` | `string` | Repository name from `.gitrepoforge`. |
| `.DefaultBranch` | `string` | Repository default branch from `.gitrepoforge`. |
| `.Config` | `map[string]interface{}` | Repo config values from `.gitrepoforge`. |

## Helper Function

| Function | Description |
|----------|-------------|
| `getConfig` | Looks up a config value by key. |

Go template built-ins such as `if`, `eq`, `ne`, `and`, and `or` are also available.

## Verbatim Copy

If `evaluate` is omitted or false, the template file is copied exactly as-is.

**`outputs/LICENSE.gitrepoforge`**

```yaml
templates:
  - condition: license == "MIT"
    template: LICENSE/MIT
```

This is appropriate for static license text or other files that should not execute template directives.

## Evaluated Template

If `evaluate: true` is set, gitrepoforge renders the file with Go's `text/template`.

**`templates/justfile.tmpl`**

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

**`outputs/justfile.gitrepoforge`**

```yaml
templates:
  - condition: justfile
    template: justfile.tmpl
    evaluate: true
  - absent: true
```

The `template` value is always a path relative to `templates/`.
