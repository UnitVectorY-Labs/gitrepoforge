---
layout: default
title: Commands
nav_order: 2
permalink: /commands
has_children: true
---

# Commands

gitrepoforge provides four commands: [`validate`](commands/VALIDATE.md), [`apply`](commands/APPLY.md), [`report`](commands/REPORT.md), and [`schema`](commands/SCHEMA.md).

| Command | Description |
|---------|-------------|
| [`validate`](commands/VALIDATE.md) | Dry-run audit. Reports drift without making changes. |
| [`apply`](commands/APPLY.md) | Applies the desired state and optionally runs shared Git automation. |
| [`report`](commands/REPORT.md) | Generates a markdown report of what `apply` would change. |
| [`schema`](commands/SCHEMA.md) | Generates a JSON Schema for the `.gitrepoforge` per-repo config file. |

## Common Flags

Several flags are shared across commands:

| Flag | Used by | Description |
|------|---------|-------------|
| `--repo <name>` | `validate`, `apply`, `report` | Target a single repo by its directory name. |
| `--json` | `validate`, `apply`, `schema` | Output results as JSON instead of the default format. |
| `--output <path>` | `report`, `schema` | Write output to a file instead of stdout. |
| `--action <name>` | `apply` | Named action from the `apply` config to use for Git automation. |

## Output

### Human-Readable (default)

`validate` and `apply` print a summary per repo with status, validation errors, and findings.

### JSON (`--json`)

`validate` and `apply` return a structured report when `--json` is passed:

```json
{
  "tool": {
    "name": "gitrepoforge",
    "version": "...",
    "timestamp": "2024-01-15T10:30:00Z",
    "command": "validate"
  },
  "root_config": "/path/to/.gitrepoforge-config",
  "config_repo": "/path/to/config-repo",
  "repos": [
    {
      "name": "my-repo",
      "status": "drift",
      "validation_errors": [],
      "findings": [
        {
          "file_path": ".github/workflows/ci.yml",
          "operation": "create",
          "message": "file should exist"
        }
      ]
    }
  ]
}
```

### Finding Operations

| Operation | Description |
|-----------|-------------|
| `create` | File should exist but is missing. |
| `update` | File exists but content differs from desired state. |
| `delete` | File should not exist but is present. |
