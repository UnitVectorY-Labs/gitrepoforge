---
layout: default
title: Workspace Config
parent: Configuration
nav_order: 1
permalink: /configuration/workspace
---

# Workspace Config

The root config lives at the workspace root, outside the managed repos, in `.gitrepoforge-config`.

## Example

{% raw %}
```yaml
config_repo: config-repo
ignore_missing: false
excludes:
  - archived-*
report:
  collapse_diffs: true
action:
  stage: {}
  commit:
    commit: true
    commit_message: "gitrepoforge: apply desired state for {{name}}"
    push: true
    remote: origin
  pr:
    create_branch: true
    branch_name: "gitrepoforge/{{name}}"
    commit: true
    commit_message: "gitrepoforge: apply desired state for {{name}}"
    push: true
    remote: origin
    pull_request: GITHUB_CLI
    return_to_original_branch: true
    delete_branch: true
```
{% endraw %}

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `config_repo` | yes | Relative or absolute path to the config repo. |
| `excludes` | no | Repo folder globs to skip during discovery. |
| `ignore_missing` | no | When `true`, suppresses the warning for repos that have no `.gitrepoforge` file. Defaults to `false`. |

## Actions

The optional `action` section defines named **actions** that control how `apply` interacts with Git. Each key under `action` is an action name; its value is a set of Git fields for that action.

```yaml
action:
  <action-name>:
    <git-fields>
```

Pass the action name at the command line with `--action`:

```
gitrepoforge apply --action pr
```

If `--action` is omitted, `apply` behaves like `validate`: it reports drift but does not write files. To make changes, pass a named action from the `action` object.

Multiple actions may be defined to support different workflows — for example a `stage` action that only writes files, a `commit` action that commits directly, and a `pr` action that branches and opens a pull request.

### Git Fields

Each action supports the following fields:

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `create_branch` | boolean | No | Create a new branch before making changes. |
| `branch_name` | string | Yes* | Name for the new branch. Supports {% raw %}`{{param}}`{% endraw %} placeholders. *Required if `create_branch` is true. |
| `commit` | boolean | No | Commit the changes after modification. |
| `commit_message` | string | Yes* | Commit message. Supports {% raw %}`{{param}}`{% endraw %} placeholders. *Required if `commit` is true. |
| `push` | boolean | No | Push the branch to the remote repository. |
| `remote` | string | Yes* | Git remote name (for example `origin`). *Required if `push` is true. |
| `pull_request` | string | No | Create a pull request. Values: `NO` (default), `GITHUB_CLI`. |
| `return_to_original_branch` | boolean | No | Switch back to the original branch after operations. Requires `create_branch` to be true. |
| `delete_branch` | boolean | No | Delete the new branch locally after operations. Requires `return_to_original_branch` to be true. |

### Placeholder Values

`branch_name` and `commit_message` may use {% raw %}`{{param}}`{% endraw %} placeholders. In gitrepoforge those placeholders are resolved from the target repo's `.gitrepoforge` values:

- `name`
- `default_branch`
- Any key under `config:`

### Validation Rules

- `pull_request` must be `NO` or `GITHUB_CLI` (case-insensitive).
- `branch_name` is required when `create_branch` is `true`.
- `commit_message` is required when `commit` is `true`.
- `remote` is required when `push` is `true`.
- `pull_request` requires `push` to be `true`.
- `return_to_original_branch` requires `create_branch` to be `true`.
- `delete_branch` requires `return_to_original_branch` to be `true`.
- Unknown placeholders in `branch_name` or `commit_message` are rejected for the affected repo.

### Compliant Status Warnings

When `commit` is `false` (or not set) in the selected action, the tool applies file changes without committing them. If a repo is compliant (files match the desired state) but has uncommitted changes in the working tree, the console output includes an additional warning:

- **not staged** – the repo has changes that are not staged with git.
- **staged, not committed** – the repo has changes staged in the index but not yet committed.

These warnings help identify repos where the desired state has been applied but the changes have not been persisted in git.

## Report Fields

The optional `report` section controls the behavior of the `report` command.

```yaml
report:
  collapse_diffs: true
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `collapse_diffs` | boolean | `true` | When `true`, diffs in the generated markdown report are wrapped in collapsible `<details>` blocks. Set to `false` to show diffs expanded by default. |
