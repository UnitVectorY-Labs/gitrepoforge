# Git Operations

gitrepoforge performs Git operations during `apply` and `bootstrap` commands. All operations run inside each target repository using the local `git` CLI and the GitHub CLI (`gh`). The behavior of each step is controlled by the `git` section in the root config (see [root-config.md](root-config.md)).

## Branching

Before making changes, gitrepoforge creates a dedicated branch:

| Command | Branch Name |
|---------|-------------|
| `apply` | `{branch_prefix}update` |
| `bootstrap` | `{branch_prefix}bootstrap` |

The `branch_prefix` is set in the root config `git` section (default: `gitrepoforge/`), so the default branch names are `gitrepoforge/update` and `gitrepoforge/bootstrap`.

### Branch Creation Flow

1. If the branch already exists locally, check it out.
2. If not, create it from the current HEAD with `git checkout -b`.

## Commits

After file changes are applied:

1. **Stage** all changes: `git add -A`
2. **Check** for staged changes: `git diff --cached --quiet`
3. If changes exist, **commit** with the configured message:
   - `apply` uses `git.commit_message` (default: `"gitrepoforge: apply desired state"`)
   - `bootstrap` uses `git.bootstrap_commit_message` (default: `"gitrepoforge: bootstrap repo"`)

If there are no staged changes after applying rules, the repo is reported as `clean` and no commit is made.

## Push

If `git.push` is `true` (the default), changes are pushed to the configured remote:

```
git push {remote} {branch}
```

The `remote` defaults to `origin` and can be overridden in the root config `git` section.

If `git.push` is `false`, changes are committed locally but not pushed. Pull request creation is also skipped.

## Pull Request Creation

If `git.pull_request` is `GITHUB_CLI` **and** the remote branch did not already exist before pushing, gitrepoforge opens a pull request using the GitHub CLI:

```
gh pr create --head {branch} --base {default_branch} --title {title} --body {body}
```

- The base branch is the `default_branch` from the repo's `.gitrepoforge` file.
- The title and body are configurable via `git.pr_title` / `git.pr_body` for `apply` and `git.bootstrap_pr_title` / `git.bootstrap_pr_body` for `bootstrap`.
- PRs are only created for new remote branches to avoid duplicate PRs on subsequent runs.
- If `git.pull_request` is `NO` (the default), no PR is created.

### Prerequisites

- The `gh` CLI must be installed and authenticated.
- The repo must have a GitHub remote matching the configured `git.remote`.

## Checkout Restore

If `git.return_to_original_branch` is `true` (the default), gitrepoforge checks out back to the `default_branch` after pushing:

```
git checkout {default_branch}
```

This leaves the working tree on the default branch regardless of success or failure.

If `git.return_to_original_branch` is `false`, gitrepoforge leaves the working tree on the feature branch.

## Branch Deletion

If `git.delete_branch` is `true` and `git.return_to_original_branch` is also `true`, gitrepoforge deletes the local feature branch after switching back to the default branch:

```
git branch -D {branch}
```

This mirrors the behavior of the repver tool's `delete_branch` option. The branch must have already been pushed to the remote (if `push` is enabled) before deletion.

## Status Checks

gitrepoforge uses these checks during operations:

| Check | Command | Purpose |
|-------|---------|---------|
| Clean working tree | `git status --porcelain` | Ensures no uncommitted changes before branching. |
| Current branch | `git rev-parse --abbrev-ref HEAD` | Determines the active branch. |
| Local branch exists | `git rev-parse --verify {branch}` | Decides whether to create or check out. |
| Remote branch exists | `git ls-remote --heads origin {branch}` | Decides whether to create a PR. |
| Staged changes exist | `git diff --cached --quiet` | Decides whether to commit. |

## Error Handling

If a Git operation fails, the repo is reported with status `failed` and the error is included in the output. gitrepoforge attempts to check out back to the default branch even after a failure (when `return_to_original_branch` is enabled).
