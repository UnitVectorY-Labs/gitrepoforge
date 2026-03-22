# Commands

gitrepoforge provides three commands: `validate`, `apply`, and `bootstrap`.

## validate

Dry-run audit. Discovers repos, validates configs, computes findings, and reports drift — without making any changes.

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
| `clean` | Repo is compliant — no changes needed. |
| `skipped` | Repo has no `.gitrepoforge` file. |
| `invalid` | Validation errors (missing config values, type mismatches, etc.). |
| `drift` | Findings detected — files differ from desired state. |

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
| `clean` | Already compliant — nothing to do. |
| `skipped` | No `.gitrepoforge` file. |
| `invalid` | Validation errors. |
| `applied` | Changes were written successfully, with any configured Git automation completed. |
| `failed` | An error occurred during Git operations. |

## bootstrap

Initializes a repo for the first time. It uses the same Git behavior as `apply`, but requires `--repo` so you target a single repository explicitly.

```
gitrepoforge bootstrap --repo <name> [flags]
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--repo <name>` | **yes** | Target repo (required for bootstrap). |
| `--json` | no | Output results as JSON instead of human-readable text. |

### Behavior

Same as `apply`, but limited to the explicitly named repo.

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
