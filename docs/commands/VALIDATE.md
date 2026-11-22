---
layout: default
title: gitrepoforge validate
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
| `--file <pattern>` | no | Include only managed files matching the glob pattern (supports `*`, `?`, `[...]`, and `**`). Repeatable. |
| `--exclude-file <pattern>` | no | Exclude managed files matching the glob pattern. Exclusions take precedence over `--file`. Repeatable. |
| `--json` | no | Output results as JSON instead of human-readable text. |
| `--verbose` | no | In human-readable mode, print git-style unified diffs for each finding. |

## Behavior

1. Loads the root config (`.gitrepoforge-config`) and config repo (`config/`, `outputs/`, `templates/`).
2. Discovers Git repos in the workspace (or targets the single `--repo`).
3. For each repo:
   - If no `.gitrepoforge` file exists, the repo is **skipped**.
   - Validates the per-repo config, including `default_branch` and any configured `manifest` path, against the shared config schema.
   - Selects template files, renders them, and compares them to the current file state. When `--file` or `--exclude-file` is set, only matching managed files are evaluated, and the managed-files manifest is skipped unless explicitly matched by `--file` (and not excluded).
4. Reports each repo's status.

When `--verbose` is set, drift findings also include per-file git-style unified diffs. Diff output includes hunk headers with old and new line ranges and uses the standard three lines of surrounding context where available. Removed lines are red and added lines are green unless the `NO_COLOR` environment variable is set.

## Statuses

| Status | Meaning |
|--------|---------|
| `clean` | Repo is compliant and no changes are needed. |
| `skipped` | Repo has no `.gitrepoforge` file. |
| `invalid` | Validation errors such as missing config values or type mismatches. |
| `drift` | Findings were detected and files differ from the desired state. |

When `commit` is not enabled in the root config and a repo is `clean`, the output may include an additional detail if the repo has uncommitted changes: `compliant (not staged)` or `compliant (staged, not committed)`.

When `ignore_missing` is `true` in the root config, repos with the `skipped` status are not shown in the human-readable output.
