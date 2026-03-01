# Templates

gitrepoforge uses Go's `text/template` package to render file content. Templates have access to repo-specific data and custom functions.

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

## Template Sources

File rules can specify template content in two ways:

### External Template File

Reference a file in the config repo using the `template` field:

```yaml
files:
  - path: .github/workflows/ci.yml
    template: templates/ci.yml.tmpl
```

The path is relative to the config repo root.

### Inline Content

Use the `content` field for short templates:

```yaml
files:
  - path: CODEOWNERS
    content: "* @{{getInput .Inputs \"team\"}}"
```

Use `template` or `content`, not both.

## Conditions

File rules support a `condition` field that determines whether the rule applies to a given repo. If the condition is empty, the rule always applies.

### Equality

```yaml
condition: "language == go"
```

Applies when the input `language` equals `go`. Quoted values are also supported:

```yaml
condition: "language == \"node\""
```

### Inequality

```yaml
condition: "language != python"
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

The `partial` action lets you manage specific sections of a file without overwriting the rest. This is useful for files like `README.md` where some content is hand-written and some is generated.

### How It Works

Each block rule specifies a `begin_marker` and `end_marker`. gitrepoforge finds these markers in the file and replaces everything between them with rendered template content.

```yaml
files:
  - path: README.md
    action: partial
    blocks:
      - begin_marker: "<!-- BEGIN MANAGED -->"
        end_marker: "<!-- END MANAGED -->"
        template: templates/readme-section.tmpl
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

A single partial file rule can contain multiple block rules, each managing a different section:

```yaml
files:
  - path: README.md
    action: partial
    blocks:
      - begin_marker: "<!-- BEGIN BADGES -->"
        end_marker: "<!-- END BADGES -->"
        content: "![CI](https://img.shields.io/badge/ci-passing-green)"
      - begin_marker: "<!-- BEGIN FOOTER -->"
        end_marker: "<!-- END FOOTER -->"
        template: templates/footer.tmpl
```
