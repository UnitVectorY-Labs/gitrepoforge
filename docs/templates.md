# Templates

gitrepoforge uses Go's `text/template` package to render file content. Templates are always written inline in the output YAML files and have access to repo-specific data and custom functions.

## Template Data

Every template receives a data object with two fields:

| Field | Type | Description |
|-------|------|-------------|
| `.Name` | `string` | The repo name (from `.gitrepoforge`). |
| `.Inputs` | `map[string]interface{}` | The repo's input values. |

### Accessing Values

```
Repository: {{.Name}}
Language:   {{.Inputs.language}}
```

## Custom Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `getInput` | `getInput(inputs, key)` | Retrieves an input value by key. Returns the value or `nil`. |
| `contains` | `contains(list, item)` | Returns `true` if the list contains the item (values are compared as strings). |
| `join` | `join(list, separator)` | Joins a list into a single string with the given separator (values are converted to strings). |

### Examples

```
Owner: {{getInput .Inputs "team"}}

{{if contains .Inputs.tags "production"}}
  Production deployment enabled.
{{end}}

Tags: {{join .Inputs.tags ", "}}
```

## Inline Templates

Templates are always specified inline using the `template` field in output YAML files. Both full-file output rules and block rules within partial files use inline templates.

### Full-File Template

The `template` field in an output YAML file contains the Go template string for the entire file:

**`outputs/CODEOWNERS.gitrepoforge`**
```yaml
template: "* @{{getInput .Inputs \"team\"}}"
```

For multi-line content, use YAML block scalars:

**`outputs/.github/workflows/ci.yml.gitrepoforge`**
```yaml
condition: "enable_ci"
template: |
  name: CI
  on: [push]
  jobs:
    build:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
```

### Block Template

Block rules within `partial` mode outputs also use the `template` field for inline content:

**`outputs/README.md.gitrepoforge`**
```yaml
mode: partial
blocks:
  - begin_marker: "<!-- BEGIN MANAGED -->"
    end_marker: "<!-- END MANAGED -->"
    template: |
      Managed by gitrepoforge.
      Language: {{.Inputs.language}}
```

## Conditions

Output rules support a `condition` field that determines whether the rule applies to a given repo. If the condition is empty, the rule always applies.

### Equality

```yaml
condition: "language == go"
```

Applies when the input `language` equals `go`. Quoted values are also supported:

```yaml
condition: "language == \"go\""
```

### Inequality

```yaml
condition: "language != \"python\""
```

Applies when the input `language` does not equal `python`.

### Boolean

```yaml
condition: "enable_ci"
```

Applies when the input `enable_ci` is truthy.

### Summary

| Syntax | Meaning |
|--------|---------|
| `key == value` | Input `key` equals `value`. |
| `key != value` | Input `key` does not equal `value`. |
| `key` | Input `key` is truthy. |
| *(empty)* | Always applies. |

## Managed Blocks (Partial Files)

The `partial` mode lets you manage specific sections of a file without overwriting the rest. This is useful for files like `README.md` where some content is hand-written and some is generated.

### How It Works

Each block rule specifies a `begin_marker` and `end_marker`. gitrepoforge finds these markers in the file and replaces everything between them with rendered template content.

**`outputs/README.md.gitrepoforge`**
```yaml
mode: partial
blocks:
  - begin_marker: "<!-- BEGIN MANAGED -->"
    end_marker: "<!-- END MANAGED -->"
    template: |
      This content is replaced by gitrepoforge.
```

The target file would look like:

```markdown
# My Project

Hand-written introduction.

<!-- BEGIN MANAGED -->
This content is replaced by gitrepoforge.
<!-- END MANAGED -->

More hand-written content.
```

### Marker Not Found

If the begin and end markers are not found in the target file, the block (markers + rendered content) is appended to the end of the file.

### Multiple Blocks

A single partial output rule can contain multiple block rules, each managing a different section:

**`outputs/README.md.gitrepoforge`**
```yaml
mode: partial
blocks:
  - begin_marker: "<!-- BEGIN BADGES -->"
    end_marker: "<!-- END BADGES -->"
    template: "![CI](https://img.shields.io/badge/ci-passing-green)"
  - begin_marker: "<!-- BEGIN FOOTER -->"
    end_marker: "<!-- END FOOTER -->"
    template: |
      ---
      Maintained by {{getInput .Inputs "team"}}
```
