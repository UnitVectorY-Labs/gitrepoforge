---
layout: default
title: Template Files
nav_order: 6
permalink: /templates
---

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
| `quote_double` | Returns a double-quoted string with escaping applied. |
| `quote_single` | Returns a single-quoted string with escaping applied by doubling embedded single quotes. |

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

The optional `template_mode` field controls how template delimiters are recognized:

| `template_mode` | Description |
|-----------------|-------------|
| `DOUBLE_BRACKET` | Default behavior. Any {% raw %}`{{ ... }}`{% endraw %} sequence is treated as a Go template action. |
| `DOUBLE_BRACKET_STRICT` | Only treats {% raw %}`{{ ... }}`{% endraw %} as a Go template action when the {% raw %}`{{`{% endraw %} appears at the start of the file or is immediately preceded by whitespace. |

**`templates/justfile.tmpl`**

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

**`outputs/justfile.gitrepoforge`**

```yaml
templates:
  - condition: justfile
    template: justfile.tmpl
    evaluate: true
  - absent: true
```

The `template` value is always a path relative to `templates/`.

Use `DOUBLE_BRACKET_STRICT` when a file needs to preserve other {% raw %}`{{ ... }}`{% endraw %}-style syntax such as GitHub Actions expressions:

**`outputs/.github/workflows/ci.yml.gitrepoforge`**

```yaml
templates:
  - template: .github/workflows/ci.yml.tmpl
    evaluate: true
    template_mode: DOUBLE_BRACKET_STRICT
```

**`templates/.github/workflows/ci.yml.tmpl`**

{% raw %}
```yaml
key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
{{- if eq .Config.codecov true }}
- uses: codecov/codecov-action@v4
{{- end }}
```
{% endraw %}

In strict mode, the {% raw %}`${{ ... }}`{% endraw %} expressions stay literal because the {% raw %}`{{`{% endraw %} is preceded by `$`, while the control blocks still execute because they start on their own lines.

Use `quote_double` or `quote_single` when a rendered value must become a quoted string literal without manually adding quotes in the template:

{% raw %}
```yaml
description: {{ .Config.description | quote_double }}
summary: {{ .Config.summary | quote_single }}
go-version: {{ .Config.versions.go | quote_double }}
```
{% endraw %}

## Section Directives

Section directives let a template manage specific regions of a file instead of replacing the entire file. When a template contains section directives, all content must be inside section, bootstrap, or join blocks.

### Directive Syntax

| Directive | Description |
|-----------|-------------|
| `#!gitrepoforge:section start=<boundary> end=<boundary>` | Defines a managed section of the file. |
| `#!gitrepoforge:bootstrap` | Content only added when the file is first created. |
| `#!gitrepoforge:join` | Joins enclosed lines into a single line by removing newlines. |
| `#!gitrepoforge:end` | Ends a section, bootstrap, or join block. |

### Boundary Types

Boundaries define where a managed section starts and ends in the target file:

| Boundary | Description |
|----------|-------------|
| `start_of_file` | Beginning of the file. |
| `end_of_file` | End of the file. |
| `line("N")` | Specific line number (1-based). |
| `content("text")` | Exact line match after trimming whitespace. |
| `contains("text")` | Line that contains the given text. |

### Behavior

**New files:** All section contents are concatenated in order and any bootstrap content is appended. The resulting content becomes the new file.

**Existing files:** Each managed section in the file is located using its boundary markers and replaced with the section content from the template. Content outside managed sections is preserved. Bootstrap content is ignored for existing files.

**Join blocks:** Lines inside a join block are concatenated with newlines removed, producing a single line. Join blocks can appear inside section blocks.

The `#!gitrepoforge:end` directive does not produce a trailing newline in the section content.

### Examples

Manage the header of a README while preserving user content below it:

```text
#!gitrepoforge:section start=start_of_file end=contains("<!-- END MANAGED -->")
# My Project
<!-- END MANAGED -->
#!gitrepoforge:end
```

Create a file only on first run (bootstrap), then leave it alone:

```text
#!gitrepoforge:bootstrap
#!gitrepoforge:end
```

Join badge images onto a single line:

```text
#!gitrepoforge:section start=start_of_file end=contains("<!-- END BADGES -->")
#!gitrepoforge:join
[![Build](https://img.shields.io/badge/build-passing-green)]
[![Coverage](https://img.shields.io/badge/coverage-100%25-green)]
#!gitrepoforge:end
<!-- END BADGES -->
#!gitrepoforge:end
```

Manage multiple sections (header and footer) while preserving everything in between:

```text
#!gitrepoforge:section start=start_of_file end=content("<!-- END HEADER -->")
# Managed Header
<!-- END HEADER -->
#!gitrepoforge:end
#!gitrepoforge:section start=contains("<!-- START FOOTER -->") end=end_of_file
<!-- START FOOTER -->
Managed Footer Content
#!gitrepoforge:end
```

{% raw %}
Use `evaluate: true` with section directives for template rendering inside sections:

```text
#!gitrepoforge:section start=start_of_file end=contains("<!-- END MANAGED -->")
# {{ .Name }}
<!-- END MANAGED -->
#!gitrepoforge:end
```
{% endraw %}
