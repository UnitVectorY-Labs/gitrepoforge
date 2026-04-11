---
layout: default
title: gitrepoforge apply
parent: Commands
nav_order: 2
permalink: /commands/apply
---

# apply

Applies the desired state to repos by writing files and then optionally running the shared Git automation from `.gitrepoforge-config`.

```
gitrepoforge apply [flags]
```

## Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--repo <name>` | no | Target a single repo by its directory name. |
| `--json` | no | Output results as JSON instead of human-readable text. |

## Behavior

1. Same discovery and validation as [`validate`](VALIDATE.md).
2. For each repo with findings:
   - Applies file changes (`create`, `update`, `delete`).
   - If root Git automation is enabled, requires a clean working tree before running Git commands.
   - If `create_branch` is `true`, creates the configured branch from the repo's current branch.
   - If `commit` is `true`, stages and commits the changes.
   - If `push` is `true`, pushes the active branch to `remote`.
   - If `pull_request` is `GITHUB_CLI`, opens a PR via `gh pr create --fill`.
   - If `return_to_original_branch` is `true`, switches back to the original branch.
   - If `delete_branch` is `true`, deletes the created branch after returning.

## Statuses

| Status | Meaning |
|--------|---------|
| `clean` | Already compliant and nothing needs to change. |
| `skipped` | No `.gitrepoforge` file was found. |
| `invalid` | Validation errors prevented apply. |
| `applied` | Changes were written successfully, including any configured Git automation. |
| `failed` | An error occurred during Git operations. |

When `commit` is not enabled and a repo is `clean`, the same `compliant (not staged)` or `compliant (staged, not committed)` warnings described in the [validate](VALIDATE.md) section apply here as well.
