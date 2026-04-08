---
layout: default
title: validate
parent: Commands
nav_order: 1
permalink: /commands/validate
---

# validate

Dry-run audit. Discovers repos, validates configs, computes findings, and reports drift without making any changes.

```
gitrepoforge validate [flags]
```

## Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--repo <name>` | no | Target a single repo by its directory name. |
| `--json` | no | Output results as JSON instead of human-readable text. |
| `--verbose` | no | In human-readable mode, print colorized git-style line diffs for each finding. |

## Behavior

1. Loads the root config (`.gitrepoforge-config`) and config repo (`config/`, `outputs/`, `templates/`).
2. Discovers Git repos in the workspace (or targets the single `--repo`).
3. For each repo:
   - If no `.gitrepoforge` file exists, the repo is **skipped**.
   - Validates the per-repo config, including `default_branch`, against the shared config schema.
   - Selects template files, renders them, and compares them to the current file state.
4. Reports each repo's status.

When `--verbose` is set, drift findings also include per-file diffs showing removed lines in red and added lines in green.

## Statuses

| Status | Meaning |
|--------|---------|
| `clean` | Repo is compliant and no changes are needed. |
| `skipped` | Repo has no `.gitrepoforge` file. |
| `invalid` | Validation errors such as missing config values or type mismatches. |
| `drift` | Findings were detected and files differ from the desired state. |

When `commit` is not enabled in the root config and a repo is `clean`, the output may include an additional detail if the repo has uncommitted changes: `compliant (not staged)` or `compliant (staged, not committed)`.

When `ignore_missing` is `true` in the root config, repos with the `skipped` status are not shown in the human-readable output.
