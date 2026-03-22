# Root Config

The root config lives at the workspace root, outside the managed repos, in `.gitrepoforge-config`.

## Example

```yaml
config_repo: config-repo
excludes:
  - archived-*
git:
  branch_prefix: gitrepoforge/
  commit_message: "gitrepoforge: apply desired state"
  bootstrap_commit_message: "gitrepoforge: bootstrap repo"
  push: true
  remote: origin
  pull_request: GITHUB_CLI
  pr_title: "gitrepoforge: apply desired state"
  pr_body: "Automated changes applied by gitrepoforge."
  bootstrap_pr_title: "gitrepoforge: bootstrap repo"
  bootstrap_pr_body: "Automated bootstrap by gitrepoforge."
  return_to_original_branch: true
  delete_branch: false
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `config_repo` | yes | Relative or absolute path to the config repo. |
| `excludes` | no | Repo folder globs to skip during discovery. |
| `git` | no | Git automation options (see below). |

## Git Section

The `git` section controls how `apply` and `bootstrap` interact with Git. All fields are optional and have sensible defaults.

| Field | Default | Description |
|-------|---------|-------------|
| `branch_prefix` | `gitrepoforge/` | Prefix for branches created by `apply` and `bootstrap`. The suffix `update` or `bootstrap` is appended automatically. |
| `commit_message` | `gitrepoforge: apply desired state` | Commit message used by `apply`. |
| `bootstrap_commit_message` | `gitrepoforge: bootstrap repo` | Commit message used by `bootstrap`. |
| `push` | `true` | Push the branch to the remote after committing. |
| `remote` | `origin` | Git remote to push to. |
| `pull_request` | `NO` | Pull request creation method. `NO` disables PR creation. `GITHUB_CLI` creates a PR using the `gh` CLI. |
| `pr_title` | value of `commit_message` | Title for pull requests opened by `apply`. |
| `pr_body` | `Automated changes applied by gitrepoforge.` | Body for pull requests opened by `apply`. |
| `bootstrap_pr_title` | value of `bootstrap_commit_message` | Title for pull requests opened by `bootstrap`. |
| `bootstrap_pr_body` | `Automated bootstrap by gitrepoforge.` | Body for pull requests opened by `bootstrap`. |
| `return_to_original_branch` | `true` | Check out back to the default branch after pushing. |
| `delete_branch` | `false` | Delete the local branch after returning to the original branch. Requires `return_to_original_branch` to be `true`. |

### Validation Rules

- `pull_request` must be `NO` or `GITHUB_CLI` (case-insensitive).
- `pull_request` cannot be `GITHUB_CLI` when `push` is `false`.
- `delete_branch` requires `return_to_original_branch` to be `true`.

## Backward Compatibility

The legacy top-level fields `branch_prefix` and `create_pr` are still accepted for backward compatibility. If both a legacy field and the corresponding `git` section field are present, the `git` section takes precedence.

| Legacy field | Equivalent `git` field |
|--------------|----------------------|
| `branch_prefix` | `git.branch_prefix` |
| `create_pr: true` | `git.pull_request: GITHUB_CLI` |
| `create_pr: false` | `git.pull_request: NO` |

### Legacy example

```yaml
config_repo: config-repo
excludes:
  - archived-*
branch_prefix: gitrepoforge/
create_pr: false
```
