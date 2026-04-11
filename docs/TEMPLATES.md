---
layout: default
title: Template Files
parent: Reference
nav_order: 1
permalink: /reference/templates
---

# Template Files

Templates live in the config repo under `templates/`. Output rules point at files in that folder and choose whether to copy them verbatim or evaluate them as templates.

## Template Directives

{% raw %}

All directives use the `{{ }}` delimiter syntax. The following table documents every directive available in templates.

### Value and Output Directives

| Directive | Description |
|-----------|-------------|
| `{{ .Name }}` | Print the repository name. |
| `{{ .DefaultBranch }}` | Print the repository default branch. |
| `{{ .Config.key }}` | Print a config value. Nested keys use dots, e.g. `{{ .Config.docs.domain }}`. |
| `{{ value \| quote_double }}` | Pipe a value through the `quote_double` function. |
| `{{ value \| quote_single }}` | Pipe a value through the `quote_single` function. |
| `{{ capture "key" "group" }}` | Extract a named capture group from a config value with a `pattern`. |

### Control Flow Directives

| Directive | Description |
|-----------|-------------|
| `{{ if condition }}...{{ end }}` | Conditional block. Content is included only when condition is true. |
| `{{ if condition }}...{{ else }}...{{ end }}` | Conditional with fallback. |
| `{{ range .Items }}...{{ end }}` | Iterate over a list. |
| `{{ with value }}...{{ end }}` | Set dot to value if non-empty. |
| `{{- ... }}` | Trim whitespace before the directive. |
| `{{ ... -}}` | Trim whitespace after the directive. |
| `{{ /* comment */ }}` | Template comment, not included in output. |

### Section Management Directives

Section directives let a template manage specific regions of a file instead of replacing the entire file. When a template contains section directives, all content must be inside section, bootstrap, or join blocks.

| Directive | Description |
|-----------|-------------|
| `{{ section start=<boundary> end=<boundary> }}` | Define a managed section with both boundaries. |
| `{{ section start=<boundary> }}` | Define a managed section from a boundary to the end of the file. |
| `{{ section end=<boundary> }}` | Define a managed section from the start of the file to a boundary. |
| `{{ bootstrap }}` | Content only added when the file is first created. |
| `{{ join }}` | Join enclosed lines into a single line by removing newlines. |
| `{{ end }}` | End a section, bootstrap, or join block. |

{% endraw %}

### Helper Functions

| Function | Description |
|----------|-------------|
| `getConfig` | Looks up a config value by key. |
| `quote_double` | Returns a double-quoted string with escaping applied. |
| `quote_single` | Returns a single-quoted string with escaping applied by doubling embedded single quotes. |
| `capture` | Extracts a named capture group from a config value with a `pattern` defined. Takes a dotted key path and a group name. |

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

## Capture Function

The `capture` function extracts named capture groups from config values that have a `pattern` defined in their [config definition](/configuration/config-repo#pattern-matching). It takes a dotted key path and a group name:

{% raw %}
```text
{{ capture "<key>" "<group>" }}
```
{% endraw %}

Given a config definition:

```yaml
# config/versions/go.yaml
type: string
pattern: "^(?P<major>\\d+)\\.(?P<minor>\\d+)\\.(?P<patch>\\d+)$"
```

And a repo config value of `1.26.1`, the full value and individual groups can both be used:

{% raw %}
```yaml
go-version: {{ .Config.versions.go | quote_single }}
go: {{ capture "versions.go" "major" }}.{{ capture "versions.go" "minor" }}
```
{% endraw %}

This renders as:

```yaml
go-version: '1.26.1'
go: 1.26
```

The key path uses dots to reference nested config values, matching the same notation used in conditions. For a top-level config key named `version`, the key path is simply `"version"`. For a nested key like `versions.go`, it is `"versions.go"`.

Template execution fails with an error if:

- The key path does not reference a config value with a pattern.
- The named group does not exist in the pattern.

The `capture` function works in both whole-file templates and section-based templates when `evaluate: true` is set.

{% raw %}
Use `capture` in section directives to control specific parts of a file, such as the Go version line in `go.mod`:

```text
{{ section end=line(4) }}
module github.com/my-org/{{ .Name }}

go {{ capture "versions.go" "major" }}.{{ capture "versions.go" "minor" }}

{{ end }}
```
{% endraw %}

## Section Directives

Section directives let a template manage specific regions of a file instead of replacing the entire file. When a template contains section directives, all content must be inside section, bootstrap, or join blocks.

### Boundary Types

Boundaries define where a managed section starts and ends in the target file:

| Boundary | Description |
|----------|-------------|
| `start_of_file` | Beginning of the file. |
| `end_of_file` | End of the file. |
| `line(N)` | Specific line number (1-based). |
| `content("text")` | Exact line match after trimming whitespace. |
| `contains("text")` | Line that contains the given text. |

A section directive requires at least one boundary. When only `start=` is specified, the end defaults to `end_of_file`. When only `end=` is specified, the start defaults to `start_of_file`.

### Behavior

**New files:** All section contents are concatenated in order and any bootstrap content is appended. The resulting content becomes the new file.

**Existing files:** Each managed section in the file is located using its boundary markers and replaced with the section content from the template. Content outside managed sections is preserved. Bootstrap content is ignored for existing files.

**Join blocks:** Lines inside a join block are concatenated with newlines removed, producing a single line. Join blocks can appear inside section blocks.

{% raw %}
The `{{ end }}` directive that closes a section does not produce a trailing newline in the section content. This means the last line of a section block is included without an extra newline appended after it. For example, a section containing two lines produces content equivalent to `"line1\nline2"` rather than `"line1\nline2\n"`.
{% endraw %}

### Examples

Manage the header of a README while preserving user content below it:

{% raw %}
```text
{{ section start=start_of_file end=contains("<!-- END MANAGED -->") }}
# My Project
<!-- END MANAGED -->
{{ end }}
```
{% endraw %}

Create a file only on first run (bootstrap), then leave it alone:

{% raw %}
```text
{{ bootstrap }}
{{ end }}
```
{% endraw %}

Join badge images onto a single line:

{% raw %}
```text
{{ section start=start_of_file end=contains("<!-- END BADGES -->") }}
{{ join }}
[![Build](https://img.shields.io/badge/build-passing-green)]
[![Coverage](https://img.shields.io/badge/coverage-100%25-green)]
{{ end }}
<!-- END BADGES -->
{{ end }}
```
{% endraw %}

Manage multiple sections (header and footer) while preserving everything in between:

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

{% raw %}
Use `evaluate: true` with section directives for template rendering inside sections:

```text
{{ section start=start_of_file end=contains("<!-- END MANAGED -->") }}
# {{ .Name }}
<!-- END MANAGED -->
{{ end }}
```
{% endraw %}

Manage only the footer of a file, preserving user content above:

{% raw %}
```text
{{ section start=contains("<!-- START FOOTER -->") end=end_of_file }}
<!-- START FOOTER -->
Managed Footer
{{ end }}
```
{% endraw %}

{% raw %}
Use `line(N)` to manage a fixed number of lines at the top of a file:

```text
{{ section start=start_of_file end=line(3) }}
Line 1 managed
Line 2 managed
Line 3 managed
{{ end }}
```
{% endraw %}

{% raw %}
Manage the first four lines of a `go.mod` to control the module path and Go version while preserving the `require` block:

```text
{{ section start=start_of_file end=line(4) }}
module github.com/my-org/{{ .Name }}

go 1.21

{{ end }}
```
{% endraw %}

With `evaluate: true` in the output configuration, `{{ "{{ .Name }}" }}` is replaced with the repository name.

{% raw %}
Combine a managed header section with bootstrap content that only appears when the file is first created:

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

When this template creates a new file, the result includes both the header section and the bootstrap text. On subsequent runs, only the header section is managed and the bootstrap text is ignored, allowing the user to replace the default body with their own content.

{% raw %}
Use only `start=` when the managed section should extend to the end of the file:

```text
{{ section start=contains("<!-- START FOOTER -->") }}
<!-- START FOOTER -->
Managed Footer
{{ end }}
```

Use only `end=` when the managed section should start at the beginning of the file:

```text
{{ section end=contains("<!-- END MANAGED -->") }}
# Managed Header
<!-- END MANAGED -->
{{ end }}
```
{% endraw %}

### Error Handling

If a section directive references a boundary that cannot be found in an existing file, the operation fails with an error. This ensures that managed sections are only applied when the file structure matches expectations.

Content outside of section, bootstrap, or join blocks in a template that uses directives is not allowed and produces an error. Blank lines between blocks are permitted.
