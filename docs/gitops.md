# Git Operations

gitrepoforge performs Git operations during `apply` and `bootstrap` commands. All operations run inside each target repository using the local `git` CLI and the GitHub CLI (`gh`).

## Branching

Before making changes, gitrepoforge creates a dedicated branch:

| Command | Branch Name |
|---------|-------------|
| `apply` | `{branch_prefix}update` |
| `bootstrap` | `{branch_prefix}bootstrap` |

The `branch_prefix` is set in the root config (default: `gitrepoforge/`), so the default branch names are `gitrepoforge/update` and `gitrepoforge/bootstrap`.

### Branch Creation Flow

1. If the branch already exists locally, check it out.
2. If not, create it from the current HEAD with `git checkout -b`.

## Commits

After file changes are applied:

1. **Stage** all changes: `git add -A`
2. **Check** for staged changes: `git diff --cached --quiet`
3. If changes exist, **commit**:
   - `apply`: `"gitrepoforge: apply desired state"`
   - `bootstrap`: `"gitrepoforge: bootstrap repo"`

If there are no staged changes after applying rules, the repo is reported as `clean` and no commit is made.

## Push

Changes are pushed to the remote:

```
git push origin {branch}
```

## Pull Request Creation

If `create_pr` is `true` in the root config **and** the remote branch did not already exist before pushing, gitrepoforge opens a pull request using the GitHub CLI:

```
gh pr create --head {branch} --base {default_branch} --title {title} --body {body}
```

- The base branch is the `default_branch` from the repo's `.gitrepoforge` file.
- PRs are only created for new remote branches to avoid duplicate PRs on subsequent runs.

### Prerequisites

- The `gh` CLI must be installed and authenticated.
- The repo must have a GitHub remote named `origin`.

## Checkout Restore

After pushing (and optionally creating a PR), gitrepoforge checks out back to the `default_branch`:

```
git checkout {default_branch}
```

This leaves the working tree on the default branch regardless of success or failure.

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

If a Git operation fails, the repo is reported with status `failed` and the error is included in the output. gitrepoforge attempts to check out back to the default branch even after a failure.
