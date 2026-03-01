# Configuration

gitrepoforge uses three configuration files at different levels.

## Root Config — `.gitrepoforge-config`

Located at the workspace root. Tells gitrepoforge where to find the central config and how to behave.

```yaml
config_repo: "config-repo"        # Required. Relative path to the central config repo.
default_branch: "main"            # Required. Default branch name for all repos.
excludes:                          # Optional. Glob patterns for repos to skip.
  - "archived-*"
  - "legacy/*"
branch_prefix: "gitrepoforge/"    # Optional. Prefix for created branches. Default: "gitrepoforge/"
create_pr: true                    # Optional. Create a pull request after push. Default: false.
```

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `config_repo` | string | yes | — | Path to the repo containing `gitrepoforge.yaml` and templates. |
| `default_branch` | string | yes | — | The default branch name used when checking out repos. |
| `excludes` | list of strings | no | `[]` | Glob patterns matched against repo directory names. |
| `branch_prefix` | string | no | `"gitrepoforge/"` | Prefix prepended to branch names created by apply/bootstrap. |
| `create_pr` | bool | no | `false` | Whether to open a PR via `gh pr create` after pushing. |

## Central Config — `gitrepoforge.yaml`

Located in the config repo (the repo referenced by `config_repo`). Defines the input schema and file rules that apply to all managed repos.

```yaml
inputs:
  - name: language
    type: string
    required: true
    enum: ["go", "python", "node"]
    description: "Primary language of the repo."
  - name: enable_ci
    type: boolean
    default: "true"
    description: "Whether to generate CI config."
  - name: team
    type: string
    required: false
    description: "Owning team name."

files:
  - path: .github/workflows/ci.yml
    condition: "enable_ci"
    template: templates/ci.yml.tmpl
  - path: CODEOWNERS
    content: "* @{{getInput .Inputs \"team\"}}"
  - path: .eslintrc.json
    condition: "language == node"
    template: templates/eslintrc.json.tmpl
  - path: legacy.txt
    action: delete
  - path: README.md
    action: partial
    blocks:
      - begin_marker: "<!-- BEGIN MANAGED -->"
        end_marker: "<!-- END MANAGED -->"
        template: templates/readme-block.tmpl
```

### Input Definitions (`inputs`)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Identifier used in conditions and templates. |
| `type` | string | yes | One of `string`, `boolean`, `number`, `list`. |
| `required` | bool | no | If `true`, every repo must provide this input. |
| `enum` | list of strings | no | Allowed values (only for `string` type). |
| `default` | string | no | Default value when the input is omitted. |
| `description` | string | no | Human-readable description. |

### File Rules (`files`)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | yes | Target file path relative to the repo root. |
| `action` | string | no | `create` (default), `delete`, or `partial`. |
| `condition` | string | no | Condition that must be true for this rule to apply. See [templates.md](templates.md#conditions). |
| `template` | string | no | Path to a template file in the config repo. |
| `content` | string | no | Inline template string. Use `template` or `content`, not both. |
| `blocks` | list | no | Block rules for `partial` action only. |

### Block Rules (`blocks`)

Used with `action: partial` to manage sections within an existing file.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `begin_marker` | string | yes | Start marker for the managed block. |
| `end_marker` | string | yes | End marker for the managed block. |
| `template` | string | no | Path to a template file for block content. |
| `content` | string | no | Inline template string for block content. |

If the markers are not found in the target file, the block (including markers and content) is appended to the end of the file.

## Per-Repo Config — `.gitrepoforge`

Located at the root of each managed repository. Declares the repo name and provides input values.

```yaml
name: my-repo
inputs:
  language: go
  enable_ci: true
  team: platform
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Must match the repository's directory name. |
| `inputs` | map | no | Key-value pairs matching the input definitions in the central config. |

### Validation Rules

- `name` must match the repo's folder name.
- All `required` inputs must be present.
- Unknown inputs (not defined in the central config) are rejected.
- Input values must match their declared `type`.
- String inputs with `enum` must use one of the allowed values.
