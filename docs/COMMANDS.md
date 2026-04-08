---
layout: default
title: Commands
nav_order: 2
permalink: /commands
---

# Commands

gitrepoforge provides three commands: `validate`, `apply`, and `report`.

## validate

Dry-run audit. Discovers repos, validates configs, computes findings, and reports drift without making any changes.

```
gitrepoforge validate [flags]
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--repo <name>` | no | Target a single repo by its directory name. |
| `--json` | no | Output results as JSON instead of human-readable text. |
| `--verbose` | no | In human-readable mode, print colorized git-style line diffs for each finding. |

### Behavior

1. Loads the root config (`.gitrepoforge-config`) and config repo (`config/`, `outputs/`, `templates/`).
2. Discovers Git repos in the workspace (or targets the single `--repo`).
3. For each repo:
   - If no `.gitrepoforge` file exists, the repo is **skipped**.
   - Validates the per-repo config, including `default_branch`, against the shared config schema.
   - Selects template files, renders them, and compares them to the current file state.
4. Reports each repo's status.

When `--verbose` is set, drift findings also include per-file diffs showing removed lines in red and added lines in green.

### Statuses

| Status | Meaning |
|--------|---------|
| `clean` | Repo is compliant and no changes are needed. |
| `skipped` | Repo has no `.gitrepoforge` file. |
| `invalid` | Validation errors such as missing config values or type mismatches. |
| `drift` | Findings were detected and files differ from the desired state. |

When `commit` is not enabled in the root config and a repo is `clean`, the output may include an additional detail if the repo has uncommitted changes: `compliant (not staged)` or `compliant (staged, not committed)`.

When `ignore_missing` is `true` in the root config, repos with the `skipped` status are not shown in the human-readable output.

## apply

Applies the desired state to repos by writing files and then optionally running the shared Git automation from `.gitrepoforge-config`.

```
gitrepoforge apply [flags]
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--repo <name>` | no | Target a single repo by its directory name. |
| `--json` | no | Output results as JSON instead of human-readable text. |

### Behavior

1. Same discovery and validation as `validate`.
2. For each repo with findings:
   - Applies file changes (`create`, `update`, `delete`).
   - If root Git automation is enabled, requires a clean working tree before running Git commands.
   - If `create_branch` is `true`, creates the configured branch from the repo's current branch.
   - If `commit` is `true`, stages and commits the changes.
   - If `push` is `true`, pushes the active branch to `remote`.
   - If `pull_request` is `GITHUB_CLI`, opens a PR via `gh pr create --fill`.
   - If `return_to_original_branch` is `true`, switches back to the original branch.
   - If `delete_branch` is `true`, deletes the created branch after returning.

### Statuses

| Status | Meaning |
|--------|---------|
| `clean` | Already compliant and nothing needs to change. |
| `skipped` | No `.gitrepoforge` file was found. |
| `invalid` | Validation errors prevented apply. |
| `applied` | Changes were written successfully, including any configured Git automation. |
| `failed` | An error occurred during Git operations. |

When `commit` is not enabled and a repo is `clean`, the same `compliant (not staged)` or `compliant (staged, not committed)` warnings described in the validate section apply here as well.

## Output

### Human-Readable (default)

Prints a summary per repo with status, validation errors, and findings.

### JSON (`--json`)

Returns a structured report:

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

## report

Generates a markdown report showing what changes `apply` would make, without actually making them. Changes are grouped by output file path and deduplicated so identical diffs across repos appear only once.

```
gitrepoforge report [flags]
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--repo <name>` | no | Target a single repo by its directory name. |
| `--output <path>` | no | Write the markdown report to a file instead of stdout. |

### Behavior

1. Loads the root config (`.gitrepoforge-config`) and config repo.
2. Discovers Git repos in the workspace (or targets the single `--repo`).
3. For each repo with a valid `.gitrepoforge` config, computes the findings that `apply` would produce.
4. Aggregates findings by output file path across all repos.
5. Deduplicates diffs so that repos receiving the same change share a single diff block.
6. Outputs a markdown report to stdout or to the file specified by `--output`.

### Report Structure

The generated markdown report contains:

1. **Repository Summary** – a table listing each repository that has changes and the number of changes.
2. **File Summary** – a table listing each output file, its operation, and the number of affected repositories.
3. **File Sections** – one section per output file path, showing the affected repositories and the deduplicated diffs.

Diffs are rendered using the markdown diff code block syntax (` ```diff `). When `collapse_diffs` is `true` (the default), each diff is wrapped in a collapsible `<details>` block. See [Root Config](ROOT-CONFIG.md) for how to configure this.

When multiple repos have the same change to a file, the diffs are deduplicated based on the added and removed lines. Context lines that differ between repos (due to surrounding content) are ignored for deduplication purposes. If different repos have different changes to the same file, each unique change is shown separately with its own list of affected repositories.
