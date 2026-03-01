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

### Behavior

1. Loads the root config (`.gitrepoforge-config`) and central config (`gitrepoforge.yaml`).
2. Discovers Git repos in the workspace (or targets the single `--repo`).
3. For each repo:
   - If no `.gitrepoforge` file exists, the repo is **skipped**.
   - Validates the per-repo config against the central input schema.
   - Renders templates and compares to the current file state.
4. Reports each repo's status.

### Statuses

| Status | Meaning |
|--------|---------|
| `clean` | Repo is compliant — no changes needed. |
| `skipped` | Repo has no `.gitrepoforge` file. |
| `invalid` | Validation errors (missing inputs, type mismatches, etc.). |
| `drift` | Findings detected — files differ from desired state. |

## apply

Applies the desired state to repos by creating a branch, writing files, committing, pushing, and optionally opening a pull request.

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
   - Creates branch `{branch_prefix}update` (e.g. `gitrepoforge/update`).
   - Applies file changes (create, update, delete, block replace).
   - Stages all changes with `git add -A`.
   - Commits with message `"gitrepoforge: apply desired state"`.
   - Pushes to `origin`.
   - If `create_pr` is enabled and the remote branch did not already exist, opens a PR via `gh pr create`.
   - Checks out back to the default branch.

### Statuses

| Status | Meaning |
|--------|---------|
| `clean` | Already compliant — nothing to do. |
| `skipped` | No `.gitrepoforge` file. |
| `invalid` | Validation errors. |
| `applied` | Changes committed and pushed. |
| `failed` | An error occurred during Git operations. |

## bootstrap

Initializes a repo for the first time. Behaves like `apply` but uses a different branch name and commit message.

```
gitrepoforge bootstrap --repo <name> [flags]
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--repo <name>` | **yes** | Target repo (required for bootstrap). |
| `--json` | no | Output results as JSON instead of human-readable text. |

### Behavior

Same as `apply` with these differences:

- Branch name: `{branch_prefix}bootstrap` (e.g. `gitrepoforge/bootstrap`).
- Commit message: `"gitrepoforge: bootstrap repo"`.

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
| `block_replace` | A managed block within a file differs from desired content. |
