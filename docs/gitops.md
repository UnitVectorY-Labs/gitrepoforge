# Git Operations

gitrepoforge performs Git operations during `apply` and `bootstrap` when the root config enables them. All operations run inside each target repository using the local `git` CLI and, for pull requests, the GitHub CLI (`gh`). The behavior of each step is controlled by the Git fields in the root config (see [root-config.md](root-config.md)).

If no Git automation is configured, `apply` and `bootstrap` still write the managed files; they simply stop before any Git commands.

## Branch Creation

If `create_branch` is `true`, gitrepoforge:

- Reads the repository's current branch
- Expands `branch_name` using the target repo's placeholder values
- Runs `git checkout -b {branch}`

Branch creation happens from the current branch, not from `default_branch`.

## Commits

If `commit` is `true`, gitrepoforge stages and commits the modified files:

1. **Stage** all changes: `git add -A`
2. **Check** for staged changes: `git diff --cached --quiet`
3. If changes exist, **commit** with the expanded `commit_message`

If there are no staged changes after applying rules, the repo is reported as `clean` and no commit is made.

## Push

If `push` is `true`, gitrepoforge pushes the active branch to the configured remote after a successful commit:

```
git push {remote} {branch}
```

`remote` is required when `push` is enabled.

## Pull Request Creation

If `pull_request` is `GITHUB_CLI`, gitrepoforge opens a pull request after a successful push:

```
gh pr create --fill
```

- If `pull_request` is `NO` (the default), no PR is created.

### Prerequisites

- The `gh` CLI must be installed and authenticated.
- The repo must have a GitHub remote matching the configured `remote`.

## Return To Original Branch

If `return_to_original_branch` is `true`, gitrepoforge checks out back to the branch that was active before it started:

```
git checkout {original_branch}
```

If `return_to_original_branch` is `false`, gitrepoforge leaves the working tree on the feature branch.

## Branch Deletion

If `delete_branch` is `true` and `return_to_original_branch` is also `true`, gitrepoforge deletes the local feature branch after switching back to the original branch:

```
git branch -D {branch}
```

This mirrors the repver `delete_branch` behavior.

## Status Checks

When Git automation is enabled, gitrepoforge uses these checks during operations:

| Check | Command | Purpose |
|-------|---------|---------|
| Clean working tree | `git status --porcelain` | Ensures no uncommitted changes before branching. |
| Current branch | `git rev-parse --abbrev-ref HEAD` | Determines the active branch. |
| Staged changes exist | `git diff --cached --quiet` | Decides whether to commit. |

## Error Handling

If a Git operation fails, the repo is reported with status `failed` and the error is included in the output. gitrepoforge makes a best-effort attempt to switch back to the original branch when `return_to_original_branch` is enabled.
