---
layout: default
title: gitrepoforge apply
parent: Commands
nav_order: 2
permalink: /commands/apply
---

# apply

Applies the desired state to repos when a named `--action` is selected. Without `--action`, `apply` behaves like `validate` and reports drift without writing files.

```
gitrepoforge apply [flags]
```

## Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--repo <name>` | no | Target a single repo by its directory name. |
| `--file <pattern>` | no | Include only managed files matching the glob pattern (supports `*`, `?`, `[...]`, and `**`). Repeatable. |
| `--exclude-file <pattern>` | no | Exclude managed files matching the glob pattern. Exclusions take precedence over `--file`. Repeatable. |
| `--json` | no | Output results as JSON instead of human-readable text. |
| `--action <name>` | no | Named action from the `action` config to use for Git automation. |
| `--branch <name>` | no | Override the `branch_name` configured on the selected action (supports {% raw %}`{{param}}`{% endraw %} placeholders or a literal shared branch name). Requires `--action` with `create_branch: true`. |

## Behavior

1. Same discovery and validation as [`validate`](VALIDATE.md).
2. If `--file` or `--exclude-file` is provided, only matching managed file rules are evaluated and applied. The generated managed-files manifest is skipped during filtered runs unless it is explicitly matched by `--file` (and not excluded by `--exclude-file`).
3. If `--action` is omitted:
   - Reports `drift` findings without writing files.
4. If `--action` is provided:
    - Resolves the named action from the `action` config in `.gitrepoforge-config`.
    - If `--branch` is provided, overrides the action's `branch_name` for this run.
    - For each repo with findings, applies file changes (`create`, `update`, `delete`), including the generated managed-files manifest when unfiltered or explicitly matched.
    - If Git automation is enabled for the selected action, requires a clean working tree before running Git commands.
    - If `on_default_branch` is `true`, fails the action unless the repo is currently on the branch named by that repo's `.gitrepoforge` `default_branch`.
    - If `create_branch` is `true`, creates the configured (or `--branch` overridden) branch from the repo's current branch.
    - If `commit` is `true`, stages only the applied finding paths and commits the changes.
    - If `push` is `true`, pushes the active branch to `remote`.
    - If `pull_request` is `GITHUB_CLI`, opens a PR via `gh pr create --fill`.
   - If `return_to_original_branch` is `true`, switches back to the original branch.
   - If `delete_branch` is `true`, deletes the created branch after returning.

`--action` must match a key under the root config's `action` object.

## Selective Bulk PR Example

Apply only a subset of managed files across all repos onto a single shared branch name for review and merging:

```
gitrepoforge apply --action pr --file '.github/workflows/*.yml' --exclude-file '.github/workflows/release*.yml' --branch chore/update-ci-workflows
```

## Statuses

| Status | Meaning |
|--------|---------|
| `clean` | Already compliant and nothing needs to change. |
| `skipped` | No `.gitrepoforge` file was found. |
| `invalid` | Validation errors prevented apply. |
| `drift` | `--action` was omitted and the repo differs from the desired state. |
| `applied` | Changes were written successfully, including any configured Git automation. |
| `failed` | An error occurred during Git operations. |

When `commit` is not enabled in the selected action and a repo is `clean`, the same `compliant (not staged)` or `compliant (staged, not committed)` warnings described in the [validate](VALIDATE.md) section apply here as well.

See [Managed Files Manifest](/reference/managed-files-manifest) for the generated manifest file format and behavior.
