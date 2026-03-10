# Root Config

The root config lives at the workspace root, outside the managed repos, in `.gitrepoforge-config`.

## Example

```yaml
config_repo: config-repo
excludes:
  - archived-*
branch_prefix: gitrepoforge/
create_pr: false
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `config_repo` | yes | Relative or absolute path to the config repo. |
| `excludes` | no | Repo folder globs to skip during discovery. |
| `branch_prefix` | no | Prefix for branches created by `apply` and `bootstrap`. Defaults to `gitrepoforge/`. |
| `create_pr` | no | If true, open a pull request after a successful push. |
